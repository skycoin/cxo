package data

import (
	"encoding/binary"
	"encoding/hex"
	"strings"

	"github.com/tidwall/buntdb"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// buckets:
//  - objects hash -> []byte (including schemas)
//  - feeds   pubkey -> { seq -> RootPack }
//  - misc    key -> value
type memoryDB struct {
	bunt *buntdb.DB
}

// NewMemoryDB creates new database in memory
func NewMemoryDB() (db DB) {
	bunt, err := buntdb.Open(":memory:")
	if err != nil {
		panic(err)
	}
	db = &memoryDB{bunt}
	return
}

func (m *memoryDB) View(fn func(t Tv) error) error {
	return m.bunt.View(func(t *buntdb.Tx) error {
		return fn(&memoryTv{t})
	})
}

func (m *memoryDB) Update(fn func(t Tu) error) error {
	return m.bunt.Update(func(t *buntdb.Tx) error {
		return fn(&memoryTu{t})
	})
}

func (m *memoryDB) Stat() (s Stat) {
	m.View(func(tx Tv) (_ error) {
		objs := tx.Objects()
		s.Objects.Amount = objs.Amount()
		s.Objects.Volume = objs.Volume()

		s.Feeds = make(map[cipher.PubKey]FeedStat)

		feeds := tx.Feeds()
		feeds.Ascend(func(pk cipher.PubKey) (_ error) {
			s.Feeds[pk] = feeds.Stat(pk)
			return
		})
		return
	})
	return
}

func (m *memoryDB) Close() error {
	return m.bunt.Close()
}

type memoryTv struct {
	tx *buntdb.Tx
}

func (m *memoryTv) Objects() ViewObjects {
	return &memoryObjects{m.tx}
}

func (m *memoryTv) Feeds() ViewFeeds {
	return &memoryViewFeeds{memoryFeeds{m.tx}}
}

func (m *memoryTv) Misc() ViewMisc {
	return &memoryMisc{m.tx}
}

type memoryTu struct {
	tx *buntdb.Tx
}

func (m *memoryTu) Objects() UpdateObjects {
	return &memoryObjects{m.tx}
}

func (m *memoryTu) Feeds() UpdateFeeds {
	return &memoryFeeds{m.tx}
}

func (m *memoryTu) Misc() UpdateMisc {
	return &memoryMisc{m.tx}
}

type memoryObjects struct {
	tx *buntdb.Tx
}

func (m *memoryObjects) key(key cipher.SHA256) string {
	return "object:" + key.Hex()
}

func (m *memoryObjects) Set(key cipher.SHA256, obj *Object) (err error) {
	if obj == nil {
		panic("given *data.Object is nil")
	}
	if len(obj.Value) == 0 {
		panic("given *data.Object contains empty Value")
	}
	_, _, err = m.tx.Set(m.key(key), encValue(obj.Encode()), nil)
	return
}

func (m *memoryObjects) Add(obj *Object) (key cipher.SHA256, err error) {
	key = cipher.SumSHA256(obj.Value)
	err = m.Set(key, obj)
	return
}

func (m *memoryObjects) MultiAdd(objs []*Object) (err error) {
	for _, ko := range sortObjs(objs) {
		if err = m.Set(ko.key, ko.obj); err != nil {
			return
		}
	}
	return
}

func (m *memoryObjects) Del(key cipher.SHA256) (err error) {
	if _, err = m.tx.Delete(m.key(key)); err == buntdb.ErrNotFound {
		err = nil
	}
	return
}

func (m *memoryObjects) GetObject(key cipher.SHA256) (obj *Object) {
	if got, _ := m.tx.Get(m.key(key)); len(got) != 0 {
		obj = DecodeObject(decValue(got))
	}
	return
}

func (m *memoryObjects) Get(key cipher.SHA256) (val []byte) {
	if got, _ := m.tx.Get(m.key(key)); len(got) != 0 {
		val = decValue(got)[8:]
	}
	return
}

func (m *memoryObjects) MultiGetObjects(keys []cipher.SHA256) (objs []*Object) {
	if len(keys) == 0 {
		return
	}
	objs = make([]*Object, len(keys))
	for _, key := range sortKeys(keys) {
		obj := m.GetObject(key)
		for i, k := range keys {
			if k == key {
				objs[i] = obj
				break
			}
		}
	}
	return
}

func (m *memoryObjects) MultiGet(keys []cipher.SHA256) (vals [][]byte) {
	if len(keys) == 0 {
		return
	}
	vals = make([][]byte, len(keys))
	for _, key := range sortKeys(keys) {
		val := m.Get(key)
		for i, k := range keys {
			if k == key {
				vals[i] = val
				break
			}
		}
	}
	return
}

func (m *memoryObjects) getKey(k string) cipher.SHA256 {
	cp, err := cipher.SHA256FromHex(strings.TrimPrefix(k, "object:"))
	if err != nil {
		panic(err)
	}
	return cp
}

func (m *memoryObjects) AscendObjects(ascendObjectsFunc func(key cipher.SHA256,
	obj *Object) error) (err error) {

	m.tx.AscendKeys("object:*", func(k, v string) bool {
		obj := DecodeObject(decValue(v))
		if err = ascendObjectsFunc(m.getKey(k), obj); err != nil {
			if err == ErrStopIteration {
				err = nil
			}
			return false // break
		}
		return true // continue
	})
	return
}

func (m *memoryObjects) Ascend(ascendFunc func(key cipher.SHA256,
	value []byte) error) (err error) {

	m.tx.AscendKeys("object:*", func(k, v string) bool {
		if err = ascendFunc(m.getKey(k), decValue(v)[8:]); err != nil {
			if err == ErrStopIteration {
				err = nil
			}
			return false // break
		}
		return true // continue
	})
	return
}

func (m *memoryObjects) AscendDel(ascendDelFunc func(key cipher.SHA256,
	value []byte) (bool, error)) (err error) {

	var del bool

	// See TODO note below
	collect := []string{}

	m.tx.AscendKeys("object:*", func(k, v string) bool {
		if del, err = ascendDelFunc(m.getKey(k), decValue(v)[8:]); err != nil {
			if err == ErrStopIteration {
				err = nil
			}
			return false // break
		}
		if del {
			// TODO (kostyarin): feature was requested, check it out

			// Waiting for
			//
			// https://github.com/tidwall/buntdb/issues/24
			//
			// if _, err = m.tx.Delete(k); err != nil {
			//	return false // break
			// }

			// Until #24 of buntdb is open, use this
			collect = append(collect, k)

		}
		return true // continue
	})

	// temporary (buntdb#24)
	if err != nil {
		return
	}

	for _, k := range collect {
		if _, err = m.tx.Delete(k); err != nil {
			return
		}
	}

	return
}

func (m *memoryObjects) Amount() (amnt uint32) {
	m.tx.AscendKeys("object:*", func(string, string) bool {
		amnt++
		return true // continue
	})
	return
}

func (m *memoryObjects) Volume() (vol Volume) {
	m.tx.AscendKeys("object:*", func(_, v string) bool {
		vol += Volume(len(decValue(v)))
		return true // continue
	})
	return
}

type memoryMisc struct {
	tx *buntdb.Tx
}

func (m *memoryMisc) key(key []byte) string {
	return "misc:" + hex.EncodeToString(key)
}

func (m *memoryMisc) Set(key, value []byte) (err error) {
	_, _, err = m.tx.Set(m.key(key), encValue(value), nil)
	return
}

func (m *memoryMisc) Del(key []byte) (err error) {
	if _, err = m.tx.Delete(m.key(key)); err == buntdb.ErrNotFound {
		err = nil
	}
	return
}

func (m *memoryMisc) Get(key []byte) (p []byte) {
	if val, _ := m.tx.Get(m.key(key)); len(val) != 0 {
		if p = decValue(val); len(p) == 0 {
			p = nil
		}
		return
	}
	return nil
}

func (m *memoryMisc) GetCopy(key []byte) []byte {
	return m.Get(key)
}

func (m *memoryMisc) getKey(k string) []byte {
	key, err := hex.DecodeString(strings.TrimPrefix(k, "misc:"))
	if err != nil {
		panic(err)
	}
	return key
}

func (m *memoryMisc) Ascend(fn func(key, value []byte) error) (err error) {

	m.tx.AscendKeys("misc:*", func(k, v string) bool {
		if err = fn(m.getKey(k), decValue(v)); err != nil {
			if err == ErrStopIteration {
				err = nil
			}
			return false // break
		}
		return true // continue
	})
	return
}

func (m *memoryMisc) AscendDel(
	fn func(key, value []byte) (bool, error)) (err error) {

	var del bool

	// See TODO note below
	collect := []string{}

	m.tx.AscendKeys("misc:*", func(k, v string) bool {
		if del, err = fn(m.getKey(k), decValue(v)); err != nil {
			if err == ErrStopIteration {
				err = nil
			}
			return false // break
		}
		if del {
			// TODO (kostyarin): feature was requested, check it out

			// Waiting for
			//
			// https://github.com/tidwall/buntdb/issues/24
			//
			// if _, err = m.tx.Delete(k); err != nil {
			//	return false // break
			// }

			// Until #24 of buntdb is open, use this
			collect = append(collect, k)

		}
		return true // continue
	})

	// temporary (buntdb#24)
	if err != nil {
		return
	}

	for _, k := range collect {
		if _, err = m.tx.Delete(k); err != nil {
			return
		}
	}

	return
}

type memoryFeeds struct {
	tx *buntdb.Tx
}

func (m *memoryFeeds) key(pk cipher.PubKey) string {
	return "feed:" + pk.Hex()
}

func (m *memoryFeeds) Add(pk cipher.PubKey) (err error) {
	_, _, err = m.tx.Set(m.key(pk), "", nil)
	return
}

func (m *memoryFeeds) Del(pk cipher.PubKey) (err error) {
	if _, err = m.tx.Delete(m.key(pk)); err != nil {
		if err == buntdb.ErrNotFound {
			err = nil
		}
		return
	}

	// see TODO note below
	collect := []string{}

	m.tx.AscendKeys(m.key(pk)+":*", func(k, _ string) bool {
		// TODO (kostyarin): feature was requested, check it out

		// Waiting for
		//
		// https://github.com/tidwall/buntdb/issues/24
		//
		// if _, err = m.tx.Delete(k); err != nil {
		//	return false // break
		// }

		// Until issue 24 of buntdb is open, use this
		collect = append(collect, k)
		return true // continue
	})

	// See TODO note above
	// Until #24 of buntdb is open
	for _, k := range collect {
		if _, err = m.tx.Delete(k); err != nil {
			return
		}
	}

	return
}

func (m *memoryFeeds) IsExist(pk cipher.PubKey) bool {
	_, err := m.tx.Get(m.key(pk))
	return err == nil
}

func (m *memoryFeeds) getKey(k string) cipher.PubKey {
	spl := strings.Split(k, ":")
	pk, err := cipher.PubKeyFromHex(spl[1])
	if err != nil {
		panic(err)
	}
	return pk
}

func (m *memoryFeeds) List() (list []cipher.PubKey) {

	m.tx.AscendKeys("feed:*", func(k, v string) bool {

		if len(v) != 0 {
			return true // continue
		}

		list = append(list, m.getKey(k))

		return true // continue
	})

	return
}

func (m *memoryFeeds) Ascend(fn func(pk cipher.PubKey) error) (err error) {

	// waithing for #24 of buntdb
	collect := []string{}

	m.tx.AscendKeys("feed:*", func(k, v string) bool {

		if len(v) != 0 {
			return true // continue
		}

		collect = append(collect, k)

		// waiting for #24 of buntdb

		// if err = fn(m.getKey(k)); err != nil {
		// 	if err == ErrStopIteration {
		// 		err = nil
		// 	}
		// 	return false // break
		// }

		return true // continue
	})

	for _, k := range collect {
		if err = fn(m.getKey(k)); err != nil {
			if err == ErrStopIteration {
				err = nil
			}
			return
		}
	}

	return
}

func (m *memoryFeeds) AscendDel(
	fn func(pk cipher.PubKey) (bool, error)) (err error) {

	var del bool

	// See TODO note below
	collect := []string{}

	m.tx.AscendKeys("feed:*", func(k, v string) bool {

		if len(v) != 0 {
			return true // continue
		}

		if del, err = fn(m.getKey(k)); err != nil {
			if err == ErrStopIteration {
				err = nil
			}
			return false // break
		}

		// TODO (kostyarin): waiting for #24 of buntdb

		if del {
			// 	if _, err = m.tx.Delete(k); err != nil {
			// 		return false // break
			// 	}
			collect = append(collect, k)
		}

		return true // continue
	})

	// temporary (buntdb#24)
	if err != nil {
		return
	}

	// see TODO above
	for _, k := range collect {
		if _, err = m.tx.Delete(k); err != nil {
			return
		}
	}

	return
}

func (m *memoryFeeds) Roots(pk cipher.PubKey) UpdateRoots {

	if !m.IsExist(pk) {
		return nil
	}

	return &memoryRoots{pk, m.key(pk) + ":", m.tx}
}

func (m *memoryFeeds) Stat(pk cipher.PubKey) (fs FeedStat) {
	roots := m.Roots(pk)
	if roots == nil {
		return
	}
	roots.Ascend(func(rp *RootPack) (_ error) {
		fs = append(fs, RootStat{
			Amount: rp.Amount,
			Volume: rp.Volume,
			Seq:    rp.Seq,
		})
		return
	})
	return
}

type memoryViewFeeds struct {
	memoryFeeds
}

func (m *memoryViewFeeds) Roots(pk cipher.PubKey) ViewRoots {
	return m.memoryFeeds.Roots(pk)
}

type memoryRoots struct {
	feed   cipher.PubKey
	prefix string
	tx     *buntdb.Tx
}

func (m *memoryRoots) Feed() cipher.PubKey {
	return m.feed
}

func (m *memoryRoots) key(seq uint64) string {
	return m.prefix + utos(seq)
}

func (m *memoryRoots) Add(rp *RootPack) (err error) {

	// check

	if rp.Seq == 0 {
		if rp.Prev != (cipher.SHA256{}) {
			err = newRootError(m.feed, rp, "unexpected prev. reference")
			return
		}
	} else if rp.Prev == (cipher.SHA256{}) {
		err = newRootError(m.feed, rp, "missing prev. reference")
		return
	}
	hash := cipher.SumSHA256(rp.Root)
	if hash != rp.Hash {
		err = newRootError(m.feed, rp, "wrong hash of the root")
		return
	}
	data := encValue(encoder.Serialize(rp))
	key := m.key(rp.Seq) // feed:pk:seq

	// find

	if _, err = m.tx.Get(key); err == buntdb.ErrNotFound {

		// not found

		_, _, err = m.tx.Set(key, data, nil)
		return

	} else if err != nil {

		// unknown error

		return

	}

	// found (already exists)

	err = ErrRootAlreadyExists
	return

}

func (m *memoryRoots) Last() (rp *RootPack) {

	m.tx.DescendKeys(m.prefix+"*", func(k, v string) bool {
		rp = new(RootPack)
		if err := encoder.DeserializeRaw(decValue(v), rp); err != nil {
			panic(err) // critical
		}
		return false // break
	})
	return
}

func (m *memoryRoots) Get(seq uint64) (rp *RootPack) {
	if val, err := m.tx.Get(m.key(seq)); err == nil {
		rp = new(RootPack)
		if err = encoder.DeserializeRaw(decValue(val), rp); err != nil {
			panic(err) // critical
		}
	}
	return
}

func (m *memoryRoots) Del(seq uint64) (err error) {
	if _, err = m.tx.Delete(m.key(seq)); err == buntdb.ErrNotFound {
		err = nil
	}
	return
}

func (m *memoryRoots) Update(rp *RootPack) (err error) {
	er := m.Get(rp.Seq)
	if er == nil {
		return ErrNotFound
	}
	if err = canUpdateRoot(er, rp); err != nil {
		return
	}
	data := encValue(encoder.Serialize(rp))
	key := m.key(rp.Seq) // feed:pk:seq
	_, _, err = m.tx.Set(key, data, nil)
	return
}

func (m *memoryRoots) Ascend(fn func(rp *RootPack) error) (err error) {

	var rp *RootPack
	m.tx.AscendKeys(m.prefix+"*", func(k, v string) bool {
		if len(v) == 0 {
			return true // continue
		}
		rp = new(RootPack)
		if err = encoder.DeserializeRaw(decValue(v), rp); err != nil {
			panic(err) // critical
		}
		if err = fn(rp); err != nil {
			if err == ErrStopIteration {
				err = nil
			}
			return false // break
		}
		return true // continue
	})
	return
}

func (m *memoryRoots) Descend(fn func(rp *RootPack) error) (err error) {

	var rp *RootPack
	m.tx.DescendKeys(m.prefix+"*", func(k, v string) bool {
		if len(v) == 0 {
			return true // continue
		}
		rp = new(RootPack)
		if err = encoder.DeserializeRaw(decValue(v), rp); err != nil {
			panic(err) // critical
		}
		if err = fn(rp); err != nil {
			if err == ErrStopIteration {
				err = nil
			}
			return false // break
		}
		return true // continue
	})
	return
}

func (m *memoryRoots) AscendDel(
	fn func(rp *RootPack) (bool, error)) (err error) {

	var rp *RootPack
	var del bool

	// See TODO note below
	collect := []string{}

	m.tx.AscendKeys(m.prefix+"*", func(k, v string) bool {
		if len(v) == 0 {
			return true // continue
		}
		rp = new(RootPack)
		if err = encoder.DeserializeRaw(decValue(v), rp); err != nil {
			panic(err) // critical
		}
		if del, err = fn(rp); err != nil {
			if err == ErrStopIteration {
				err = nil
			}
			return false // break
		}
		if del {
			// TODO (kostyarin): waitng for #24 of buntdb
			// if _, err = m.tx.Delete(k); err != nil {
			// 	return false // break
			// }
			collect = append(collect, k)
		}
		return true // continue
	})

	// temporary (buntdb#24)
	if err != nil {
		return
	}

	// See TODO note above
	for _, k := range collect {
		if _, err = m.tx.Delete(k); err != nil {
			return // break
		}
	}

	return
}

func (m *memoryRoots) DelBefore(seq uint64) (err error) {

	// TODO: optimize (avoid decoding)

	m.AscendDel(func(rp *RootPack) (del bool, _ error) {
		del = rp.Seq < seq
		return
	})

	return
}

//
// utilities
//

func encValue(value []byte) string {
	return hex.EncodeToString(value)
}

func decValue(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func utos(u uint64) string {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, u)
	return hex.EncodeToString(b)
}

// func stou(s string) uint64 {
// 	b, err := hex.DecodeString(s)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return binary.BigEndian.Uint64(b)
// }
