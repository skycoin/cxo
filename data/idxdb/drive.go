package idxdb

import (
	"encoding/binary"
	"time"

	"github.com/boltdb/bolt"

	"github.com/skycoin/skycoin/src/cipher"
)

var (
	objsBucket  = []byte("o")
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
		if _, err = tx.CreateBucketIfNotExists(objsBucket); err != nil {
			return
		}
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

func (d *driveDB) Tx(txFunc func(tx Tx) (err error)) (err error) {
	return d.b.Update(func(tx *bolt.Tx) (err error) {
		return txFunc(&driveTx{tx})
	})
}

type driveTx struct {
	tx *bolt.Tx
}

func (d *driveTx) Objects() Objects {
	return &driveObjs{d.tx.Bucket(objsBucket)}
}

func (d *driveTx) Feeds() Feeds {
	return &driveFeeds{d.tx.Bucket(feedsBucket)}
}

type driveObjs struct {
	bk *bolt.Bucket
}

func (d *driveObjs) Inc(key cipher.SHA256) (rc uint32, err error) {
	var val []byte
	if val = d.bk.Get(key[:]); len(val) == 0 {
		return
	}
	var o Object
	if err = o.Decode(val); err != nil {
		panic(err) // critical
	}
	o.RefsCount++
	rc = o.RefsCount
	o.UpdateAccessTime()
	err = d.bk.Put(key[:], o.Encode())
	return
}

func (d *driveObjs) Get(key cipher.SHA256) (o *Object, err error) {
	var val []byte
	if val = d.bk.Get(key[:]); len(val) == 0 {
		err = ErrNotFound
		return
	}
	o, err = d.decodeAndUpdateAccessTime(key, val)
	return
}

func (d *driveObjs) decodeAndUpdateAccessTime(key cipher.SHA256,
	val []byte) (o *Object, err error) {

	o = new(Object)
	if err = o.Decode(val); err != nil {
		panic(err) // critical
	}
	acst := o.AccessTime // keep previous AccessTime
	o.UpdateAccessTime()
	if err = d.bk.Put(key[:], o.Encode()); err != nil {
		o = nil
		return
	}
	o.AccessTime = acst // restore previous AccessTime
	return

}

// TODO (kostyarin): use ordered get to speed up the MultiGet,
// because of B+-tree index
func (d *driveObjs) MultiGet(keys []cipher.SHA256) (os []*Object, err error) {
	if len(keys) == 0 {
		return
	}
	os = make([]*Object, len(keys))
	for i, key := range keys {
		var o *Object
		if o, err = d.Get(key); err != nil {
			if err == ErrNotFound {
				err = nil
				os[i] = nil
				continue
			}
			os = nil
			return
		}
		os[i] = o // nil or Object
	}
	return
}

// TODO (kostyarin): use ordered get to speed up the MultiGet,
// because of B+-tree index
func (d *driveObjs) MultiInc(keys []cipher.SHA256) ([]uint32, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	rcs := make([]uint32, 0, len(keys))
	for _, key := range keys {
		if rc, err := d.Inc(key); err != nil {
			return nil, err // don't return rcs if err != nil
		} else {
			rcs = append(rcs, rc)
		}
	}
	return rcs, nil
}

func (d *driveObjs) Iterate(iterateObjectsFunc IterateObjectsFunc) (err error) {
	c := d.bk.Cursor()
	var key cipher.SHA256
	var o *Object
	for k, v := c.First(); k != nil; k, v = c.Next() {
		copy(key[:], k)
		if o, err = d.decodeAndUpdateAccessTime(key, v); err != nil {
			return
		}
		if err = iterateObjectsFunc(key, o); err != nil {
			if err == ErrStopIteration {
				err = nil
			}
			return
		}
	}
	return
}

func (d *driveObjs) Dec(key cipher.SHA256) (rc uint32, err error) {
	var val []byte
	if val = d.bk.Get(key[:]); len(val) == 0 {
		return
	}
	var o Object
	if err = o.Decode(val); err != nil {
		panic(err) // critical
	}
	o.RefsCount--
	if rc = o.RefsCount; rc == 0 {
		err = d.bk.Delete(key[:])
		return
	}
	o.UpdateAccessTime()
	err = d.bk.Put(key[:], o.Encode())
	return
}

func (d *driveObjs) Set(key cipher.SHA256, o *Object) (err error) {
	var val []byte
	if val = d.bk.Get(key[:]); len(val) == 0 {
		o.UpdateAccessTime()
		o.CreateTime = o.AccessTime // created now

		o.RefsCount = 1 // make sure that RefsCount is 1 (no less , no more)
		err = d.bk.Put(key[:], o.Encode())
		return
	}
	if err = o.Decode(val); err != nil {
		panic(err) // critical
	}
	o.RefsCount++
	o.UpdateAccessTime()
	err = d.bk.Put(key[:], o.Encode())
	return
}

func (d *driveObjs) MultiSet(kos []KeyObject) (err error) {
	for _, ko := range kos {
		if err = d.Set(ko.Key, ko.Object); err != nil {
			return
		}
	}
	return
}

func (d *driveObjs) MulitDec(keys []cipher.SHA256) ([]uint32, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	rcs := make([]uint32, 0, len(keys))
	for _, key := range keys {
		if rc, err := d.Dec(key); err != nil {
			return nil, err
		} else {
			rcs = append(rcs, rc)
		}
	}
	return rcs, nil
}

func (d *driveObjs) Amount() (amnt Amount) {
	amnt = Amount(d.bk.Stats().KeyN)
	return
}

func (d *driveObjs) Volume() (vol Volume) {
	d.bk.ForEach(func(k, v []byte) (_ error) {
		o := new(Object)
		if err := o.Decode(v); err != nil {
			panic(err)
		}
		vol += o.Vol
		return
	})
	return
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
