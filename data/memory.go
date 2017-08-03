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

	m.bunt.View(func(t *buntdb.Tx) (_ error) {

		// objects

		t.AscendKeys("object:*", func(_, v string) bool {
			s.Objects++
			s.Space += Space(len(v))
			return true // continue
		})

		// feeds (and roots)

		s.Feeds = make(map[cipher.PubKey]FeedStat)

		t.AscendKeys("feed:*", func(k, v string) bool {

			// k is "feed:pub_key:seq" or "feed:pub_key"

			if len(v) == 0 {
				return true // (continue) is not a root object
			}

			pk, err := cipher.PubKeyFromHex(strings.Split(k, ":")[1])
			if err != nil {
				panic(err)
			}

			fs := s.Feeds[pk]
			fs.Roots++
			fs.Space += Space(len(v))

			s.Feeds[pk] = fs

			return true // continue

		})

		if len(s.Feeds) == 0 {
			s.Feeds = nil
		}

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

type memoryTu struct {
	tx *buntdb.Tx
}

func (m *memoryTu) Objects() UpdateObjects {
	return &memoryObjects{m.tx}
}

func (m *memoryTu) Feeds() UpdateFeeds {
	return &memoryFeeds{m.tx}
}

type memoryObjects struct {
	tx *buntdb.Tx
}

func (m *memoryObjects) key(key cipher.SHA256) string {
	return "object:" + key.Hex()
}

func (m *memoryObjects) Set(key cipher.SHA256, value []byte) (err error) {
	_, _, err = m.tx.Set(m.key(key), encValue(value), nil)
	return
}

func (m *memoryObjects) Del(key cipher.SHA256) (err error) {
	if _, err = m.tx.Delete(m.key(key)); err == buntdb.ErrNotFound {
		err = nil
	}
	return
}

func (m *memoryObjects) Get(key cipher.SHA256) (p []byte) {
	if val, _ := m.tx.Get(m.key(key)); len(val) != 0 {
		if p = decValue(val); len(p) == 0 {
			p = nil
		}
		return
	}
	return nil
}

func (m *memoryObjects) GetCopy(key cipher.SHA256) []byte {
	return m.Get(key)
}

func (m *memoryObjects) Add(value []byte) (key cipher.SHA256, err error) {
	key = cipher.SumSHA256(value)
	err = m.Set(key, value)
	return
}

func (m *memoryObjects) IsExist(key cipher.SHA256) bool {
	return m.Get(key) != nil
}

func (m *memoryObjects) SetMap(mp map[cipher.SHA256][]byte) (err error) {
	for k, v := range mp {
		if err = m.Set(k, v); err != nil {
			return
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

func (m *memoryObjects) Range(
	fn func(key cipher.SHA256, value []byte) error) (err error) {

	m.tx.AscendKeys("object:*", func(k, v string) bool {
		if err = fn(m.getKey(k), decValue(v)); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return false // break
		}
		return true // continue
	})
	return
}

func (m *memoryObjects) RangeDel(
	fn func(key cipher.SHA256, value []byte) (bool, error)) (err error) {

	var del bool

	// See TODO note below
	collect := []string{}

	m.tx.AscendKeys("object:*", func(k, v string) bool {
		if del, err = fn(m.getKey(k), decValue(v)); err != nil {
			if err == ErrStopRange {
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

func (m *memoryFeeds) Range(fn func(pk cipher.PubKey) error) (err error) {

	m.tx.AscendKeys("feed:*", func(k, v string) bool {

		if len(v) != 0 {
			return true // continue
		}

		if err = fn(m.getKey(k)); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return false // break
		}

		return true // continue
	})

	return
}

func (m *memoryFeeds) RangeDel(
	fn func(pk cipher.PubKey) (bool, error)) (err error) {

	var del bool

	// See TODO note below
	collect := []string{}

	m.tx.AscendKeys("feed:*", func(k, v string) bool {

		if len(v) != 0 {
			return true // continue
		}

		if del, err = fn(m.getKey(k)); err != nil {
			if err == ErrStopRange {
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

func (m *memoryRoots) MarkFull(seq uint64) (err error) {
	rp := m.Get(seq)
	if rp == nil {
		return ErrNotFound
	}
	rp.IsFull = true
	data := encValue(encoder.Serialize(rp))
	key := m.key(rp.Seq) // feed:pk:seq
	_, _, err = m.tx.Set(key, data, nil)
	return
}

func (m *memoryRoots) Range(fn func(rp *RootPack) error) (err error) {

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
			if err == ErrStopRange {
				err = nil
			}
			return false // break
		}
		return true // continue
	})
	return
}

func (m *memoryRoots) Reverse(fn func(rp *RootPack) error) (err error) {

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
			if err == ErrStopRange {
				err = nil
			}
			return false // break
		}
		return true // continue
	})
	return
}

func (m *memoryRoots) RangeDel(
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
			if err == ErrStopRange {
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

	m.RangeDel(func(rp *RootPack) (del bool, _ error) {
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
