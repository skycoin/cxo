package data

import (
	"bytes"
	"encoding/binary"
	"sync"
	"time"

	"github.com/boltdb/bolt"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

const dbMode = 0644

// names of buckets
var (
	objectsBucket = []byte("objects")
	feedsBucket   = []byte("feeds")
)

// buckets:
//  - objects hash -> []byte (including schemas)
//  - feeds   pubkey -> (roots) { seq -> root }
type driveDB struct {
	bolt   *bolt.DB
	closeo sync.Once // boltdb panics when Close closed database
}

// NewDriveDB creates new database using given path
// to create or use existsing database file
func NewDriveDB(path string) (db DB, err error) {
	var b *bolt.DB
	b, err = bolt.Open(path, dbMode, &bolt.Options{
		Timeout: 500 * time.Millisecond,
	})
	if err != nil {
		return
	}
	err = b.Update(func(t *bolt.Tx) (err error) {
		if _, err = t.CreateBucketIfNotExists(objectsBucket); err != nil {
			return
		}
		_, err = t.CreateBucketIfNotExists(feedsBucket)
		return
	})
	if err != nil {
		return
	}
	db = &driveDB{bolt: b}
	return
}

func (d *driveDB) View(fn func(t Tv) error) (err error) {
	err = d.bolt.View(func(t *bolt.Tx) error {
		tx := new(driveTv)
		tx.tx = t
		return fn(tx)
	})
	return
}

func (d *driveDB) Update(fn func(t Tu) error) (err error) {
	err = d.bolt.Update(func(t *bolt.Tx) error {
		tx := new(driveTu)
		tx.tx = t
		return fn(tx)
	})
	return
}

func (d *driveDB) Stat() (s Stat) {

	d.bolt.View(func(t *bolt.Tx) (_ error) {

		// objects

		objects := t.Bucket(objectsBucket)
		s.Objects = objects.Stats().KeyN

		objects.ForEach(func(_, v []byte) (_ error) {
			s.Space += Space(len(v))
			return
		})

		// feeds (and roots)

		feeds := t.Bucket(feedsBucket)
		fln := feeds.Stats().KeyN

		if fln == 0 {
			return // no feeds
		}

		s.Feeds = make(map[cipher.PubKey]FeedStat, fln)

		feeds.ForEach(func(kk, _ []byte) (_ error) {

			roots := feeds.Bucket(kk)

			var fs FeedStat
			var cp cipher.PubKey

			fs.Roots = roots.Stats().KeyN
			roots.ForEach(func(_, v []byte) (_ error) {
				fs.Space += Space(len(v))
				return
			})

			copy(cp[:], kk)
			s.Feeds[cp] = fs

			return
		})

		return

	})

	return
}

func (d *driveDB) Close() (err error) {
	d.closeo.Do(func() {
		err = d.bolt.Close()
	})
	return
}

type driveTv struct {
	tx *bolt.Tx
}

func (d *driveTv) Objects() ViewObjects {
	o := new(driveObjects)
	o.bk = d.tx.Bucket(objectsBucket)
	return o
}

func (d *driveTv) Feeds() ViewFeeds {
	f := new(driveFeeds)
	f.bk = d.tx.Bucket(feedsBucket)
	return &driveViewFeeds{f}
}

type driveTu struct {
	tx *bolt.Tx
}

func (d *driveTu) Objects() UpdateObjects {
	o := new(driveObjects)
	o.bk = d.tx.Bucket(objectsBucket)
	return o
}

func (d *driveTu) Feeds() UpdateFeeds {
	f := new(driveFeeds)
	f.bk = d.tx.Bucket(feedsBucket)
	return f
}

type driveObjects struct {
	bk *bolt.Bucket
}

func (d *driveObjects) Set(key cipher.SHA256, value []byte) (err error) {
	return d.bk.Put(key[:], value)
}

func (d *driveObjects) Del(key cipher.SHA256) (err error) {
	return d.bk.Delete(key[:])
}

func (d *driveObjects) Get(key cipher.SHA256) []byte {
	return d.bk.Get(key[:])
}

func (d *driveObjects) GetCopy(key cipher.SHA256) (value []byte) {
	if g := d.bk.Get(key[:]); g != nil {
		value = make([]byte, len(g))
		copy(value, g)
	}
	return
}

func (d *driveObjects) Add(value []byte) (key cipher.SHA256, err error) {
	key = cipher.SumSHA256(value)
	err = d.bk.Put(key[:], value)
	return
}

func (d *driveObjects) IsExist(key cipher.SHA256) bool {
	return d.Get(key) != nil
}

func (d *driveObjects) SetMap(m map[cipher.SHA256][]byte) (err error) {
	for _, kv := range sortMap(m) {
		if err = d.bk.Put(kv.key[:], kv.val); err != nil {
			return
		}
	}
	return
}

func (d *driveObjects) Range(
	fn func(key cipher.SHA256, value []byte) error) (err error) {

	c := d.bk.Cursor()

	var ck cipher.SHA256

	for k, v := c.First(); k != nil; k, v = c.Next() {
		copy(ck[:], k)
		if err = fn(ck, v); err != nil {
			break
		}
	}

	if err == ErrStopRange {
		err = nil
	}
	return
}

func (d *driveObjects) RangeDel(
	fn func(key cipher.SHA256, value []byte) (bool, error)) (err error) {

	c := d.bk.Cursor()

	var ck cipher.SHA256
	var del bool

	// seek loop
	for k, v := c.First(); k != nil; k, v = c.Seek(k) {
		// next loop
		for {
			copy(ck[:], k)
			if del, err = fn(ck, v); err != nil {
				if err == ErrStopRange {
					err = nil
				}
				return
			}
			if del {
				if err = c.Delete(); err != nil {
					return
				}
				// coninue seek loop, because after deleting
				// we have got invalid cusor and we need to
				// call Seek to make it valid; the Seek will
				// points to next item, because current one
				// has been deleted
				break
			}
			// just get next item (next loop)
			if k, v = c.Next(); k == nil {
				return // there's nothing more
			}
		}
	}

	return
}

type driveFeeds struct {
	bk *bolt.Bucket
}

func (d *driveFeeds) Add(pk cipher.PubKey) (err error) {
	_, err = d.bk.CreateBucketIfNotExists(pk[:])
	return
}

func (d *driveFeeds) Del(pk cipher.PubKey) (err error) {
	if err = d.bk.DeleteBucket(pk[:]); err != nil {
		if err == bolt.ErrBucketNotFound {
			err = nil
		}
	}
	return
}

func (d *driveFeeds) IsExist(pk cipher.PubKey) bool {
	return d.bk.Bucket(pk[:]) != nil
}

func (d *driveFeeds) List() (list []cipher.PubKey) {

	ln := d.bk.Stats().KeyN
	if ln == 0 {
		return // nil
	}

	list = make([]cipher.PubKey, 0, ln)

	var cp cipher.PubKey
	c := d.bk.Cursor()

	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		copy(cp[:], k)
		list = append(list, cp)
	}

	return
}

func (d *driveFeeds) Range(fn func(pk cipher.PubKey) error) (err error) {
	var cp cipher.PubKey
	c := d.bk.Cursor()

	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		copy(cp[:], k)
		if err = fn(cp); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return
		}
	}
	return
}

func (d *driveFeeds) RangeDel(
	fn func(pk cipher.PubKey) (bool, error)) (err error) {

	var cp cipher.PubKey
	var del bool

	c := d.bk.Cursor()

	// seek loop
	for k, _ := c.First(); k != nil; k, _ = c.Seek(k) {

		// next loop
		for {

			copy(cp[:], k)
			if del, err = fn(cp); err != nil {
				if err == ErrStopRange {
					err = nil
				}
				return
			}
			if del {
				if err = d.bk.DeleteBucket(k); err != nil {
					return
				}
				break // break "next loop" (= continue "seek loop")
			}
			if k, _ = c.Next(); k == nil {
				return // nothing more
			}

		}

	}

	return
}

func (d *driveFeeds) Roots(pk cipher.PubKey) UpdateRoots {
	r := new(driveRoots)
	r.feed = pk
	bk := d.bk.Bucket(pk[:])
	if bk == nil {
		return nil
	}
	r.bk = bk
	return r
}

type driveViewFeeds struct {
	*driveFeeds
}

func (d *driveViewFeeds) Roots(pk cipher.PubKey) ViewRoots {
	return d.driveFeeds.Roots(pk)
}

type driveRoots struct {
	feed cipher.PubKey
	bk   *bolt.Bucket
}

func (d *driveRoots) Feed() cipher.PubKey {
	return d.feed
}

func (d *driveRoots) Add(rp *RootPack) (err error) {

	// check

	if rp.Seq == 0 {
		if rp.Prev != (cipher.SHA256{}) {
			err = newRootError(d.feed, rp, "unexpected prev. reference")
			return
		}
	} else if rp.Prev == (cipher.SHA256{}) {
		err = newRootError(d.feed, rp, "missing prev. reference")
		return
	}
	hash := cipher.SumSHA256(rp.Root)
	if hash != rp.Hash {
		err = newRootError(d.feed, rp, "wrong hash of the root")
		return
	}
	data := encoder.Serialize(rp)
	seqb := utob(rp.Seq)

	// find

	if k, _ := d.bk.Cursor().Seek(seqb); bytes.Compare(k, seqb) != 0 {

		// not found

		err = d.bk.Put(seqb, data) // store
		return
	}

	// found (already exists)

	err = ErrRootAlreadyExists
	return
}

func (d *driveRoots) Last() (rp *RootPack) {
	_, last := d.bk.Cursor().Last()
	if last == nil {
		return // nil
	}
	rp = new(RootPack)
	if err := encoder.DeserializeRaw(last, rp); err != nil {
		panic(err) // critical
	}
	return
}

func (d *driveRoots) Get(seq uint64) (rp *RootPack) {
	seqb := utob(seq)
	data := d.bk.Get(seqb)
	if data == nil {
		return // nil
	}
	rp = new(RootPack)
	if err := encoder.DeserializeRaw(data, rp); err != nil {
		panic(err) // critical
	}
	return
}

func (d *driveRoots) Del(seq uint64) error {
	seqb := utob(seq)
	return d.bk.Delete(seqb)
}

func (d *driveRoots) MarkFull(seq uint64) (err error) {
	rp := d.Get(seq)
	if rp == nil {
		return ErrNotFound
	}
	rp.IsFull = true
	return d.bk.Put(utob(seq), encoder.Serialize(rp))
}

func (d *driveRoots) Range(fn func(rp *RootPack) error) (err error) {

	var rp *RootPack
	c := d.bk.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		rp = new(RootPack)
		if err = encoder.DeserializeRaw(v, rp); err != nil {
			panic(err) // critical
		}
		if err = fn(rp); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return
		}
	}
	return
}

func (d *driveRoots) Reverse(fn func(rp *RootPack) error) (err error) {

	var rp *RootPack
	c := d.bk.Cursor()
	for k, v := c.Last(); k != nil; k, v = c.Prev() {
		rp = new(RootPack)
		if err = encoder.DeserializeRaw(v, rp); err != nil {
			panic(err) // critical
		}
		if err = fn(rp); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return
		}
	}
	return
}

func (d *driveRoots) RangeDel(fn func(rp *RootPack) (bool, error)) (err error) {

	var rp *RootPack
	var del bool
	c := d.bk.Cursor()

	// seek loop
	for k, v := c.First(); k != nil; k, v = c.Seek(k) {

		// next loop
		for {

			rp = new(RootPack)
			if err = encoder.DeserializeRaw(v, rp); err != nil {
				panic(err) // critical
			}
			if del, err = fn(rp); err != nil {
				if err == ErrStopRange {
					err = nil
				}
				return
			}
			if del {
				if err = c.Delete(); err != nil {
					return
				}
				break // break "next loop" (= continue "seek loop")
			}
			if k, v = c.Next(); k == nil {
				return
			}

		}

	}

	return
}

func (d *driveRoots) DelBefore(seq uint64) (err error) {

	c := d.bk.Cursor()

	for k, _ := c.First(); k != nil; k, _ = c.Seek(k) {

		if btou(k) >= seq {
			return
		}

		if err = c.Delete(); err != nil {
			return
		}

	}

	return
}

//
// utils
//

func utob(seq uint64) (b []byte) {
	b = make([]byte, 8)
	binary.BigEndian.PutUint64(b, seq)
	return
}

func btou(b []byte) (seq uint64) {
	seq = binary.BigEndian.Uint64(b)
	return
}
