package data

import (
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

			spl := strings.Split(k, ":")
			if len(spl) != 3 {
				return true // (continue) is not a root object
			}

			var pk cipher.PubKey
			copy(pk[:], []byte(spl[1]))

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
	return "object:" + string(key[:])
}

func (m *memoryObjects) Set(key cipher.SHA256, value []byte) (err error) {
	_, _, err = m.tx.Set(m.key(key), string(value), nil)
	return
}

func (m *memoryObjects) Del(key cipher.SHA256) (err error) {
	_, err = m.tx.Delete(m.key(key))
	return
}

func (m *memoryObjects) Get(key cipher.SHA256) []byte {
	if val, _ := m.tx.Get(m.key(key)); len(val) != 0 {
		return []byte(val)
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

func (m *memoryObjects) Range(
	fn func(key cipher.SHA256, value []byte) error) (err error) {

	var cp cipher.SHA256

	m.tx.AscendKeys("object:*", func(k, v string) bool {
		copy(cp[:], k)
		if err = fn(cp, []byte(v)); err != nil {
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

	var cp cipher.SHA256
	var del bool

	m.tx.AscendKeys("object:*", func(k, v string) bool {
		copy(cp[:], k)
		if del, err = fn(cp, []byte(v)); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return false // break
		}
		if del {
			if _, err = m.tx.Delete(k); err != nil {
				return false // break
			}
		}
		return true // continue
	})
	return
}

type memoryFeeds struct {
	tx *buntdb.Tx
}

func (m *memoryFeeds) key(pk cipher.PubKey) string {
	return "feed:" + string(pk[:])
}

func (m *memoryFeeds) Add(pk cipher.PubKey) (err error) {
	_, _, err = m.tx.Set(m.key(pk), "", nil)
	return
}

func (m *memoryFeeds) Del(pk cipher.PubKey) (err error) {
	if _, err = m.tx.Delete(m.key(pk)); err != nil {
		return
	}
	m.tx.AscendKeys(m.key(pk)+":*", func(k, _ string) bool {
		if _, err = m.tx.Delete(k); err != nil {
			return false // break
		}
		return true // continue
	})
	return
}

func (m *memoryFeeds) IsExist(pk cipher.PubKey) bool {
	_, err := m.tx.Get(m.key(pk))
	return err != buntdb.ErrNotFound
}

func (m *memoryFeeds) List() (list []cipher.PubKey) {

	var cp cipher.PubKey

	m.tx.AscendKeys("feed:*", func(k, v string) bool {

		if len(v) != 0 {
			return true // continue
		}

		spl := strings.Split(k, ":")
		copy(cp[:], spl[1])

		list = append(list, cp)

		return true // continue
	})

	return
}

func (m *memoryFeeds) Range(fn func(pk cipher.PubKey) error) (err error) {

	var cp cipher.PubKey

	m.tx.AscendKeys("feed:*", func(k, v string) bool {

		if len(v) != 0 {
			return true // continue
		}

		spl := strings.Split(k, ":")
		copy(cp[:], spl[1])

		if err = fn(cp); err != nil {
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

	var cp cipher.PubKey
	var del bool

	m.tx.AscendKeys("feed:*", func(k, v string) bool {

		if len(v) != 0 {
			return true // continue
		}

		spl := strings.Split(k, ":")
		copy(cp[:], spl[1])

		if del, err = fn(cp); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return false // break
		}

		if del {
			if _, err = m.tx.Delete(k); err != nil {
				return false // break
			}
		}

		return true // continue
	})

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
	data := string(encoder.Serialize(rp))
	key := m.prefix + string(utob(rp.Seq)) // feed:pk:seq

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
		if err := encoder.DeserializeRaw([]byte(v), rp); err != nil {
			panic(err) // critical
		}
		return false // break
	})
	return
}

func (m *memoryRoots) Get(seq uint64) (rp *RootPack) {
	seqb := string(utob(seq))
	if val, err := m.tx.Get(m.prefix + seqb); err == nil {
		rp = new(RootPack)
		if err = encoder.DeserializeRaw([]byte(val), rp); err != nil {
			panic(err) // critical
		}
	}
	return
}

func (m *memoryRoots) Del(seq uint64) (err error) {
	seqb := string(utob(seq))
	_, err = m.tx.Delete(m.prefix + seqb)
	return
}

func (m *memoryRoots) Range(fn func(rp *RootPack) error) (err error) {

	var rp *RootPack
	m.tx.AscendKeys(m.prefix+"*", func(k, v string) bool {
		if len(v) == 0 {
			return true // continue
		}
		rp = new(RootPack)
		if err = encoder.DeserializeRaw([]byte(v), rp); err != nil {
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
		if err = encoder.DeserializeRaw([]byte(v), rp); err != nil {
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

	m.tx.AscendKeys(m.prefix+"*", func(k, v string) bool {
		if len(v) == 0 {
			return true // continue
		}
		rp = new(RootPack)
		if err = encoder.DeserializeRaw([]byte(v), rp); err != nil {
			panic(err) // critical
		}
		if del, err = fn(rp); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return false // break
		}
		if del {
			if _, err = m.tx.Delete(k); err != nil {
				return false // break
			}
		}
		return true // continue
	})

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
