package idxdb

import (
	"encoding/binary"
	"time"

	"github.com/boltdb/bolt"

	"github.com/skycoin/skycoin/src/cipher"
)

var (
	feedsBucket = []byte("f")
)

type driveDB struct {
	b *bolt.DB
}

func NewDriveIdxDB(fileName string) (idx IdxDB, err error) {

	var b *bolt.DB
	b, err = bolt.Open(fileName, 0644, &bolt.Options{
		Timeout: time.Millisecond * 500,
	})
	if err != nil {
		return
	}

	err = b.Update(func(tx *bolt.Tx) (err error) {
		_, err = tx.CreateBucketIfNotExists(feedsBucket)
		return
	})
	if err != nil {
		b.Close()
		return
	}

	idx = &driveDB{b}
	return
}

func (d *driveDB) Tx(txFunc func(feeds Feeds) (err error)) (err error) {
	return d.b.Update(func(tx *bolt.Tx) (err error) {
		return txFunc(&driveFeeds{tx.Bucket(feedsBucket)})
	})
}

func (d *driveDB) Close() (err error) {
	return d.b.Close()
}

type driveFeeds struct {
	bk *bolt.Bucket
}

func (d *driveFeeds) Add(pk cipher.PubKey) (err error) {
	_, err = d.bk.CreateBucketIfNotExists(pk[:])
	return
}

func (d *driveFeeds) Del(pk cipher.PubKey) (err error) {
	if f := d.bk.Bucket(pk[:]); f == nil {
		return // not exists
	} else if f.Stats().KeyN == 0 {
		return d.bk.DeleteBucket(pk[:]) // empty
	}
	return ErrFeedIsNotEmpty // can't remove non-empty feed
}

func (d *driveFeeds) Iterate(iterateFeedsFunc IterateFeedsFunc) (err error) {
	var pk cipher.PubKey
	c := d.bk.Cursor()
	// we have to Seek(next) instead of using Next
	// because we allows mutations during the iteration
	for k, _ := c.First(); k != nil; k, _ = c.Seek(pk[:]) {
		copy(pk[:], k)
		if err = iterateFeedsFunc(pk); err != nil {
			if err == ErrStopIteration {
				err = nil
			}
			return
		}
		incSlice(pk[:])
	}
	return
}

// increment slice
func incSlice(b []byte) {
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] == 0xff {
			b[i] = 0
			continue // increase next byte
		}
		b[i]++
		return
	}
}

func (d *driveFeeds) HasFeed(pk cipher.PubKey) bool {
	return d.bk.Bucket(pk[:]) != nil
}

func (d *driveFeeds) Roots(pk cipher.PubKey) (rs Roots, err error) {
	bk := d.bk.Bucket(pk[:])
	if bk == nil {
		return nil, ErrNoSuchFeed
	}
	return &driveRoots{bk}, nil
}

type driveRoots struct {
	bk *bolt.Bucket
}

func (d *driveRoots) Ascend(iterateRootsFunc IterateRootsFunc) (err error) {

	var seq uint64
	var r *Root = new(Root)
	var sb []byte = make([]byte, 8)

	c := d.bk.Cursor()

	for seqb, er := c.First(); seqb != nil; seqb, er = c.Seek(seqb) {

		seq = binary.LittleEndian.Uint64(seqb)

		if err = r.Decode(er); err != nil {
			panic(err)
		}

		if err = iterateRootsFunc(r); err != nil {
			if err == ErrStopIteration {
				err = nil
			}
			return
		}

		seq++
		binary.LittleEndian.PutUint64(sb, seq)
		seqb = sb
	}
	return
}

func (d *driveRoots) Descend(iterateRootsFunc IterateRootsFunc) (err error) {

	var r *Root = new(Root)

	c := d.bk.Cursor()

	for seqb, er := c.Last(); seqb != nil; {

		if err = r.Decode(er); err != nil {
			panic(err)
		}

		if err = iterateRootsFunc(r); err != nil {
			if err == ErrStopIteration {
				err = nil
			}
			return
		}

		c.Seek(seqb)        // rewind
		seqb, er = c.Prev() // prev
	}
	return
}

func (d *driveRoots) Inc(seq uint64) (err error) {
	var val, seqb []byte
	var r *Root

	seqb = utob(seq)

	if val = d.bk.Get(seqb); len(val) == 0 {
		err = ErrNotFound
		return
	}

	r = new(Root)

	if err = r.Decode(val); err != nil {
		panic(err)
	}

	r.UpdateAccessTime()
	r.RefsCount++

	return d.bk.Put(seqb, r.Encode())
}

func (d *driveRoots) Dec(seq uint64) (err error) {
	var val, seqb []byte
	var r *Root

	seqb = utob(seq)

	if val = d.bk.Get(seqb); len(val) == 0 {
		err = ErrNotFound
		return
	}

	r = new(Root)

	if err = r.Decode(val); err != nil {
		panic(err)
	}

	r.UpdateAccessTime()
	if r.RefsCount >= 1 {
		r.RefsCount--
	}

	return d.bk.Put(seqb, r.Encode())
}

func (d *driveRoots) Set(r *Root) (err error) {

	if err = r.Validate(); err != nil {
		return
	}

	var val, seqb []byte

	seqb = utob(r.Seq)

	if val = d.bk.Get(seqb); len(val) == 0 {
		// not found
		r.UpdateAccessTime()
		r.CreateTime = r.AccessTime
		r.RefsCount = 1
		return d.bk.Put(seqb, r.Encode())
	}

	// found
	nr := new(Root)

	if err = nr.Decode(val); err != nil {
		panic(err)
	}

	nr.UpdateAccessTime()
	nr.RefsCount++ // actually it should panic here (todo)
	nr.IsFull = r.IsFull

	r.AccessTime = nr.AccessTime
	r.RefsCount = nr.RefsCount

	return d.bk.Put(seqb, nr.Encode())
}

func (d *driveRoots) Del(seq uint64) (err error) {
	return d.bk.Delete(utob(seq))
}

func (d *driveRoots) Get(seq uint64) (r *Root, err error) {
	seqb := utob(seq)
	val := d.bk.Get(seqb)
	if len(val) == 0 {
		err = ErrNotFound
		return
	}
	r = new(Root)
	if err := r.Decode(val); err != nil {
		panic(err)
	}
	return
}

func utob(u uint64) (p []byte) {
	p = make([]byte, 8)
	binary.LittleEndian.PutUint64(p, u)
	return
}
