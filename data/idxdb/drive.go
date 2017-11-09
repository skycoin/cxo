package idxdb

import (
	"encoding/binary"
	"os"
	"time"

	"github.com/boltdb/bolt"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

var (
	feedsBucket = []byte("f")       // feeds
	metaBucket  = []byte("m")       // meta information
	versionKey  = []byte("version") // encoded version in the meta bucket
)

type driveDB struct {
	b *bolt.DB
}

// NewDriveIdxDB creates data.IdxDB instance that
// keeps its data on drive
func NewDriveIdxDB(fileName string) (idx data.IdxDB, err error) {

	var created bool // true if db file has been created

	_, err = os.Stat(fileName)
	created = os.IsNotExist(err) // set the created var

	var b *bolt.DB

	b, err = bolt.Open(fileName, 0644, &bolt.Options{
		Timeout: time.Millisecond * 500,
	})

	if err != nil {
		return
	}

	err = b.Update(func(tx *bolt.Tx) (err error) {

		// first of all, take a look the meta bucket
		var info = tx.Bucket(metaBucket)

		if info == nil {

			// if the file has not been created, then
			// this DB file seems outdated (version 0)
			if created == false {
				return ErrMissingMetaInfo // report
			}

			// create the bucket and put meta information
			if info, err = tx.CreateBucket(metaBucket); err != nil {
				return
			}

			// put version
			if err = info.Put(versionKey, versionBytes()); err != nil {
				return
			}

		} else {

			// check out the version

			var vb []byte
			if vb = info.Get(versionKey); len(vb) == 0 {
				return ErrMissingVersion
			}

			switch vers := int(binary.BigEndian.Uint32(vb)); {
			case vers == Version: // ok
			case vers < Version:
				return ErrOldVersion
			case vers > Version:
				return ErrNewVersion
			}

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

// Tx performs ACID-transaction
func (d *driveDB) Tx(txFunc func(feeds data.Feeds) (err error)) (err error) {
	return d.b.Update(func(tx *bolt.Tx) (err error) {
		return txFunc(&driveFeeds{tx.Bucket(feedsBucket)})
	})
}

// Close the DB
func (d *driveDB) Close() (err error) {
	return d.b.Close()
}

type driveFeeds struct {
	bk *bolt.Bucket
}

// Add feed or does nothing if its already exists
func (d *driveFeeds) Add(pk cipher.PubKey) (err error) {
	_, err = d.bk.CreateBucketIfNotExists(pk[:])
	return
}

// Del deltes feed if the feed is empty
func (d *driveFeeds) Del(pk cipher.PubKey) (err error) {
	if f := d.bk.Bucket(pk[:]); f == nil {
		return // not exists
	} else if f.Stats().KeyN == 0 {
		return d.bk.DeleteBucket(pk[:]) // empty
	}
	return data.ErrFeedIsNotEmpty // can't remove non-empty feed
}

// Iterate over all feeds
func (d *driveFeeds) Iterate(iterateFunc data.IterateFeedsFunc) (err error) {
	var pk cipher.PubKey
	c := d.bk.Cursor()
	// we have to Seek(next) instead of using Next
	// because we allows mutations during the iteration
	for k, _ := c.First(); k != nil; k, _ = c.Seek(pk[:]) {
		copy(pk[:], k)
		if err = iterateFunc(pk); err != nil {
			if err == data.ErrStopIteration {
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

// Has performs presence check
func (d *driveFeeds) Has(pk cipher.PubKey) bool {
	return d.bk.Bucket(pk[:]) != nil
}

// Roots returns bucket of Root object of given feed
func (d *driveFeeds) Roots(pk cipher.PubKey) (rs data.Roots, err error) {
	bk := d.bk.Bucket(pk[:])
	if bk == nil {
		return nil, data.ErrNoSuchFeed
	}
	return &driveRoots{bk}, nil
}

type driveRoots struct {
	bk *bolt.Bucket
}

// Ascend iternates over all Root objects ascending order
func (d *driveRoots) Ascend(iterateFunc data.IterateRootsFunc) (err error) {

	var seq uint64
	var r = new(data.Root)
	var sb = make([]byte, 8)

	c := d.bk.Cursor()

	for seqb, er := c.First(); seqb != nil; seqb, er = c.Seek(seqb) {

		seq = binary.BigEndian.Uint64(seqb)

		if err = r.Decode(er); err != nil {
			panic(err)
		}

		if err = iterateFunc(r); err != nil {
			if err == data.ErrStopIteration {
				err = nil
			}
			return
		}

		seq++
		binary.BigEndian.PutUint64(sb, seq)
		seqb = sb
	}
	return
}

// Descend iternates over all Root objects descending order
func (d *driveRoots) Descend(iterateFunc data.IterateRootsFunc) (err error) {

	var r = new(data.Root)

	c := d.bk.Cursor()

	for seqb, er := c.Last(); seqb != nil; {

		if err = r.Decode(er); err != nil {
			panic(err)
		}

		if err = iterateFunc(r); err != nil {
			if err == data.ErrStopIteration {
				err = nil
			}
			return
		}

		c.Seek(seqb)        // rewind
		seqb, er = c.Prev() // prev
	}
	return
}

// Set new Root object or does nothing if
// the object already exists
func (d *driveRoots) Set(r *data.Root) (err error) {

	if err = r.Validate(); err != nil {
		return
	}

	var val, seqb []byte

	seqb = utob(r.Seq)

	if val = d.bk.Get(seqb); len(val) == 0 {
		// not found
		r.UpdateAccessTime()
		r.CreateTime = r.AccessTime
		return d.bk.Put(seqb, r.Encode())
	}

	// found
	nr := new(data.Root)

	if err = nr.Decode(val); err != nil {
		panic(err)
	}

	r.AccessTime = nr.AccessTime
	r.CreateTime = nr.CreateTime

	nr.UpdateAccessTime()
	return d.bk.Put(seqb, nr.Encode())
}

// Del deletes Root object by seq
func (d *driveRoots) Del(seq uint64) (err error) {
	return d.bk.Delete(utob(seq))
}

// Get Root object by seq
func (d *driveRoots) Get(seq uint64) (r *data.Root, err error) {
	seqb := utob(seq)
	val := d.bk.Get(seqb)
	if len(val) == 0 {
		err = data.ErrNotFound
		return
	}
	r = new(data.Root)
	if err := r.Decode(val); err != nil {
		panic(err)
	}
	return
}

// Has performs precense check using seq
func (d *driveRoots) Has(seq uint64) (yep bool) {
	return len(d.bk.Get(utob(seq))) > 0
}

// Len returns amount of Root objects
func (d *driveRoots) Len() int {
	return d.bk.Stats().KeyN
}

func utob(u uint64) (p []byte) {
	p = make([]byte, 8)
	binary.BigEndian.PutUint64(p, u)
	return
}
