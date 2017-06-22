package data

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/boltdb/bolt"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

const dbMode = 0644

// names of buckets
var (
	objectsBucket = []byte("objects")
	rootsBucket   = []byte("roots")
	feedsBucket   = []byte("feeds")
)

// buckets:
//  - objects hash -> []byte (including schemas)
//  - roots   hash -> encoded RootPack
//  - feeds   pubkey -> { seq -> hash of root }
type driveDB struct {
	bolt *bolt.DB
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
	err = b.Update(func(t *bolt.Tx) (e error) {
		if _, e = t.CreateBucketIfNotExists(objectsBucket); e != nil {
			return
		}
		if _, e = t.CreateBucketIfNotExists(rootsBucket); e != nil {
			return
		}
		_, e = t.CreateBucketIfNotExists(feedsBucket)
		return
	})
	if err != nil {
		return
	}
	db = &driveDB{b}
	return
}

// panic if error
func (d *driveDB) update(fn func(t *bolt.Tx) error) {
	if err := d.bolt.Update(fn); err != nil {
		panic(err)
	}
}

// panic if error
func (d *driveDB) view(fn func(t *bolt.Tx) error) {
	if err := d.bolt.View(fn); err != nil {
		panic(err)
	}
}

func (d *driveDB) Del(key cipher.SHA256) {
	d.update(func(t *bolt.Tx) error {
		return t.Bucket(objectsBucket).Delete(key[:])
	})
}

func (d *driveDB) Get(key cipher.SHA256) (value []byte, ok bool) {
	d.view(func(t *bolt.Tx) (_ error) {
		var temp []byte
		if temp = t.Bucket(objectsBucket).Get(key[:]); temp != nil {
			value, ok = make([]byte, len(temp)), true
			copy(value, temp)
		}
		return
	})
	return
}

func (d *driveDB) Set(key cipher.SHA256, value []byte) {
	d.update(func(t *bolt.Tx) error {
		return t.Bucket(objectsBucket).Put(key[:], value)
	})
}

func (d *driveDB) Add(value []byte) (key cipher.SHA256) {
	key = cipher.SumSHA256(value)
	d.Set(key, value)
	return
}

func (d *driveDB) IsExist(key cipher.SHA256) (ok bool) {
	d.view(func(t *bolt.Tx) (_ error) {
		ok = t.Bucket(objectsBucket).Get(key[:]) != nil
		return
	})
	return
}

func (d *driveDB) Range(fn func(key cipher.SHA256, value []byte) (stop bool)) {
	d.view(func(t *bolt.Tx) (_ error) {
		o := t.Bucket(objectsBucket)
		c := o.Cursor()
		var key cipher.SHA256
		for k, value := c.First(); k != nil; k, value = c.Next() {
			copy(key[:], k)
			if fn(key, value) {
				break // stop
			}
		}
		return
	})
}

func (d *driveDB) RangeDelete(fn func(key cipher.SHA256) (del bool)) {
	d.update(func(t *bolt.Tx) (e error) {
		o := t.Bucket(objectsBucket)
		c := o.Cursor()
		var key cipher.SHA256
		// seek loop
		for k, _ := c.First(); k != nil; k, _ = c.Seek(key[:]) {
			for {
				copy(key[:], k)
				if fn(key) {
					if e = c.Delete(); e != nil {
						return
					}
					// coninue seek loop, because after deleting
					// we have got invalid cusor and we need to
					// call Seek to make it valid; the Seek will
					// points to next item, because current one
					// has been deleted
					break
				}
				// just get next item
				if k, _ = c.Next(); k == nil {
					return
				}
			}
		}
		return
	})
}

//
// Feeds
//

func (d *driveDB) AddFeed(pk cipher.PubKey) {
	d.update(func(t *bolt.Tx) (err error) {
		f := t.Bucket(feedsBucket)
		_, err = f.CreateBucketIfNotExists(pk[:])
		return
	})
}

func (d *driveDB) HasFeed(pk cipher.PubKey) (has bool) {
	d.view(func(t *bolt.Tx) (_ error) {
		has = t.Bucket(feedsBucket).Bucket(pk[:]) != nil
		return
	})
	return
}

func (d *driveDB) Feeds() (fs []cipher.PubKey) {
	d.view(func(t *bolt.Tx) (_ error) {
		f := t.Bucket(feedsBucket)
		ln := f.Stats().KeyN
		if ln == 0 {
			return // no feeds
		}
		fs = make([]cipher.PubKey, 0, ln)
		return f.ForEach(func(pkb, _ []byte) (_ error) {
			var pk cipher.PubKey
			copy(pk[:], pkb)
			fs = append(fs, pk)
			return
		})
	})
	return
}

func (d *driveDB) DelFeed(pk cipher.PubKey) {
	d.update(func(t *bolt.Tx) (_ error) {
		fb := t.Bucket(feedsBucket)
		f := fb.Bucket(pk[:])
		if f == nil {
			return
		}
		r := t.Bucket(rootsBucket)
		err := f.ForEach(func(_, hash []byte) error {
			return r.Delete(hash)
		})
		if err != nil {
			panic(err) // critical
		}
		// godoc: https://godoc.org/github.com/boltdb/bolt#Bucket.DeleteBucket
		// > [...] Returns an error if the bucket does not exists, or if
		// > the key represents a non-bucket value.
		fb.DeleteBucket(pk[:]) // Thus, we ignore an error
		return
	})
}

func (d *driveDB) AddRoot(pk cipher.PubKey, rp *RootPack) (err error) {
	// test given rp
	if rp.Seq == 0 {
		if rp.Prev != (cipher.SHA256{}) {
			err = newRootError(pk, rp, "unexpected prev. reference")
			return
		}
	} else if rp.Prev == (cipher.SHA256{}) {
		err = newRootError(pk, rp, "missing prev. reference")
		return
	}
	hash := cipher.SumSHA256(rp.Root)
	if hash != rp.Hash {
		err = newRootError(pk, rp, "wrong hash of the root")
		return
	}
	data := encoder.Serialize(rp)
	seqb := utob(rp.Seq)
	// let's go
	d.update(func(t *bolt.Tx) (_ error) {
		f, e := t.Bucket(feedsBucket).CreateBucketIfNotExists(pk[:])
		if e != nil {
			panic(e) // critical
		}
		sseq, _ := f.Cursor().Seek(seqb)
		if sseq == nil || bytes.Compare(sseq, seqb) != 0 {
			// not found
			if e = f.Put(utob(rp.Seq), hash[:]); e != nil {
				panic(e) // critical
			}
			if e = t.Bucket(rootsBucket).Put(hash[:], data); e != nil {
				panic(e) // critical
			}
			return
		}
		// else => already exists
		err = ErrRootAlreadyExists
		return
	})
	return
}

func (d *driveDB) LastRoot(pk cipher.PubKey) (rp *RootPack, ok bool) {
	d.view(func(t *bolt.Tx) (_ error) {
		f := t.Bucket(feedsBucket).Bucket(pk[:])
		if f == nil {
			return
		}
		_, last := f.Cursor().Last()
		if last == nil {
			return
		}
		value := t.Bucket(rootsBucket).Get(last)
		if value == nil {
			panic("broken database: missing root") // critical
		}
		rp = new(RootPack)
		if err := encoder.DeserializeRaw(value, rp); err != nil {
			panic(err) // critical
		}
		ok = true
		return
	})
	return
}

func (d *driveDB) RangeFeed(pk cipher.PubKey,
	fn func(rp *RootPack) (stop bool)) {

	d.view(func(t *bolt.Tx) (_ error) {
		f := t.Bucket(feedsBucket).Bucket(pk[:])
		if f == nil {
			return
		}
		var rp RootPack
		r := t.Bucket(rootsBucket)
		c := f.Cursor()
		for _, hashb := c.First(); hashb != nil; _, hashb = c.Next() {
			value := r.Get(hashb)
			if value == nil {
				panic("missing root") // critical
			}
			if err := encoder.DeserializeRaw(value, &rp); err != nil {
				panic(err) // critical
			}
			if fn(&rp) {
				break // stop
			}
		}
		return
	})
}

func (d *driveDB) RangeFeedReverse(pk cipher.PubKey,
	fn func(rp *RootPack) (stop bool)) {

	d.view(func(t *bolt.Tx) (_ error) {
		f := t.Bucket(feedsBucket).Bucket(pk[:])
		if f == nil {
			return
		}
		var rp RootPack
		r := t.Bucket(rootsBucket)
		c := f.Cursor()
		for _, hashb := c.Last(); hashb != nil; _, hashb = c.Prev() {
			value := r.Get(hashb)
			if value == nil {
				panic("missing root") // critical
			}
			if err := encoder.DeserializeRaw(value, &rp); err != nil {
				panic(err) // critical
			}
			if fn(&rp) {
				break // stop
			}
		}
		return
	})
}

func (d *driveDB) RangeFeedDelete(pk cipher.PubKey,
	fn func(rp *RootPack) (del bool)) {

	d.update(func(t *bolt.Tx) (err error) {
		f := t.Bucket(feedsBucket).Bucket(pk[:])
		if f == nil {
			return
		}

		var rp RootPack
		var hash cipher.SHA256

		r := t.Bucket(rootsBucket)
		c := f.Cursor()

		// seek loop
		for _, hashb := c.First(); hashb != nil; _, hashb = c.Seek(hash[:]) {
			for {
				copy(hash[:], hashb)

				value := r.Get(hashb)
				if value == nil {
					panic("missing root") // critical
				}
				if err := encoder.DeserializeRaw(value, &rp); err != nil {
					panic(err) // critical
				}

				if fn(&rp) {
					if err = r.Delete(hashb); err != nil {
						return
					}
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

				// just get next item
				if _, hashb = c.Next(); hashb == nil {
					return // done
				}

			}
		}

		return
	})
}

//
// Roots
//

func (d *driveDB) GetRoot(hash cipher.SHA256) (rp *RootPack, ok bool) {
	d.view(func(t *bolt.Tx) (_ error) {
		var temp []byte
		if temp = t.Bucket(rootsBucket).Get(hash[:]); temp == nil {
			return
		}
		rp = new(RootPack)
		var err error
		if err = encoder.DeserializeRaw(temp, rp); err != nil {
			return
		}
		ok = true
		return
	})
	return
}

// TODO: optimize
func (d *driveDB) DelRootsBefore(pk cipher.PubKey, seq uint64) {
	d.update(func(t *bolt.Tx) (_ error) {
		fb := t.Bucket(feedsBucket)
		f := fb.Bucket(pk[:])
		if f == nil {
			return
		}

		// collect

		type sh struct{ s, h []byte }

		del := []sh{}
		err := f.ForEach(func(seqb, hashb []byte) error {
			if btou(seqb) < seq {
				del = append(del, sh{seqb, hashb})
			}
			return nil
		})
		if err != nil {
			panic(err) // critical
		}

		// delete

		r := t.Bucket(rootsBucket)
		for _, x := range del {

			if e := f.Delete(x.s); e != nil {
				panic(e) // critical
			}
			if e := r.Delete(x.h); e != nil {
				panic(e) // critical
			}
		}

		return
	})
}

func (d *driveDB) Stat() (s Stat) {
	// Objects int
	// Space   Space
	// Feeds   map[cipher.PubKey]struct {
	//     Roots int
	//     Space Space
	// }
	d.view(func(t *bolt.Tx) (_ error) {
		// objects
		o := t.Bucket(objectsBucket)
		s.Objects = o.Stats().KeyN
		e := o.ForEach(func(_, val []byte) (_ error) {
			s.Space += Space(len(val))
			return
		})
		if e != nil {
			panic(e) // critical
		}
		// feeds
		f := t.Bucket(feedsBucket)
		r := t.Bucket(rootsBucket)
		//
		feeds := f.Stats().KeyN
		if feeds == 0 {
			return // no feeds
		}
		s.Feeds = make(map[cipher.PubKey]FeedStat, feeds)
		//
		var pk cipher.PubKey
		// for each feed
		e = f.ForEach(func(pkb, _ []byte) (_ error) {
			// pkb is bucket name
			var fs FeedStat

			fpk := f.Bucket(pkb)
			fs.Roots = fpk.Stats().KeyN

			// for each seq->hash
			e := fpk.ForEach(func(_, hashb []byte) (_ error) {
				// size of encoded RootPack
				fs.Space += Space(len(r.Get(hashb)))
				return
			})
			if e != nil {
				panic(e) // critical
			}

			copy(pk[:], pkb)
			s.Feeds[pk] = fs // store in stat
			return
		})
		if e != nil {
			panic(e) // critical
		}
		return
	})
	return
}

func (d *driveDB) Close() (err error) {
	err = d.bolt.Close()
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
