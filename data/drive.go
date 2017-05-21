package data

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/boltdb/bolt"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data/stat"
)

const dbMode = 0644

// names of buckets
var (
	objectsBucket []byte = []byte("objects")
	rootsBucket   []byte = []byte("roots")
	feedsBucket   []byte = []byte("feeds")
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

//
// Feeds
//

func (d *driveDB) DelFeed(pk cipher.PubKey) {
	d.update(func(t *bolt.Tx) (_ error) {
		f := t.Bucket(feedsBucket).Bucket(pk[:])
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
		if e := f.Delete(pk[:]); e != nil {
			panci(e) // critical
		}
		return
	})
}

func (d *driveDB) AddRoot(pk cipher.PubKey, rp RootPack) (err error) {
	// test given rp
	if rp.Seq != 0 && rp.Prev != (cipher.SHA256{}) {
		err = newRootError(pk, &rp, "unexpected prev. reference")
		return
	}
	data := encoder.Serialize(&rp)
	hash := cipher.SumSHA256(rp.Root)
	if hash != rp.Hash {
		err = newRootError(pk, &rp, "wrong hash of the root")
		return
	}
	seqb := utob(rp.Seq)
	//
	d.update(func(t *bolt.Tx) (_ error) {
		f := t.Bucket(feedsBucket).Bucket(pk[:])
		if f == nil {
			var e error
			if f, e = t.CreateBucket(pk[:]); e != nil {
				panic(e) // critical
			}
			if e = f.Put(utob(rp.Seq), hash[:]); e != nil {
				panic(e) // critical
			}
			if e = t.Bucket(rootsBucket).Put(hash[:], data); e != nil {
				panic(e) // critical
			}
			return
		}
		// else => f already exists
		// is given rp older then first
		c := f.Cursor()
		fseq, fhash := c.First()
		if fseq == nil {
			// not found (critical):
			//   a feed must contains at least one root object
			//   (or must be removed from database)
			panic("broken database: feed doesn't contains a root object")
		}
		if cmp := bytes.Compare(fseq, seqb); cmp == +1 {
			// seq number of first rp is greater then seq of given rp
			err = ErrRootIsOld // special error
			return
		} else if cmp == 0 {
			// equal
			err = ErrRootAlreadyExists // special error
			return
		}
		// find the seq (or next if not found)
		sseq, shash := c.Seek(seqb)
		// from godoc: https://godoc.org/github.com/boltdb/bolt#Cursor.Seek
		// > Seek moves the cursor to a given key and returns it.
		// > If the key does not exist then the next key is used.
		// > If no keys follow, a nil key is returned. The returned
		// > key and value are only valid for the life of the transaction.
		if sseq == nil {
			// not found, and there are no next rp
			pseq, phash := c.Prev()
			if pseq == nil {
				// not found (critical):
				//   a feed must contains at least one root object
				//   (or must be removed from database)
				panic("broken database: missing previous object")
			}
			r := t.Bucket(rootsBucket)
			pdata := r.Get(phash)
			if pdata == nil {
				// not found (critical)
				// feed -> { seq -> hash } is just another way
				// to find a root object that must be present in roots
				// bucket
				panic("feeds refers to root doesn't exist")
			}
			var prev RootPack
			var e error
			if e = encoder.DeserializeRaw(pdata, &prev); e != nil {
				panic(err) // critical
			}
			// check rp.prev reference
			if rp.Prev != prev.Hash {
				err = newRootError(pk,
					&rp,
					"previous root points to another next-root")
				return
			}
			// set next reference to prev if it's clear (and update the prev)
			if prev.Next == (cipher.SHA256{}) {
				prev.Next = rp.Hash
				if e = r.Put(phash, encoder.Serialize(&prev)); e != nil {
					panic(err) // critical
				}
			}
			// save the given rp
			if e = f.Put(seqb, hash[:]); e != nil {
				panic(e) // critical
			}
			if e = t.Bucket(rootsBucket).Put(hash[:], data); e != nil {
				panic(e) // critical
			}
			return
		}
		// else => sseq != nil
		// found or next
		switch bytes.Compare(sseq, seqb) {
		// case -1: never happens
		case 0:
			// sseq == seqb (found)
			err = ErrRootAlreadyExists
			return
		case +1:
			// sseq > seqb (next, i.e. we have next but don't have this)
			// databse can only "add next root" or "add first"
			panic("broken database: broken roots chain")
		}
		return
	})
	return
}

func (d *driveDB) LastRoot(pk cipher.PubKey) (rp RootPack, ok bool) {
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
			panic("missing root") // critical
		}
		if err := encoder.DeserializeRaw(value, &rp); err != nil {
			panic(err) // critical
		}
		ok = true
		return
	})
	return
}

func (d *driveDB) RangeFeed(pk cipher.PubKey,
	fn func(rp RootPack) (stop bool)) {

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
			if fn(rp) {
				break // stop
			}
		}
		return
	})
}

func (d *driveDB) RangeFeedReverse(pk cipher.PubKey,
	fn func(rp RootPack) (stop bool)) {

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
			if fn(rp) {
				break // stop
			}
		}
		return
	})
}

//
// Roots
//

func (d *driveDB) GetRoot(hash cipher.SHA256) (rp RootPack, ok bool) {
	d.view(func(t *bolt.Tx) (_ error) {
		var temp []byte
		if temp = t.Bucket(rootsBucket).Get(hash[:]); temp == nil {
			return
		}
		var err error
		if err = encoder.DeserializeRaw(temp, &rp); err != nil {
			return
		}
		ok = true
		return
	})
	return
}

func (d *driveDB) DelRootsBefore(pk cipher.PubKey, seq uint64) {
	d.update(func(t *bolt.Tx) (_ error) {
		f := t.Bucket(feedsBucket).Bucket(pk[:])
		if f == nil {
			return
		}
		r := t.Bucket(rootsBucket)
		err := f.ForEach(func(seqb, hashb []byte) error {
			if btou(seqb) < seq {
				return r.Delete(hashb)
			}
			return nil
		})
		if err != nil {
			panic(err) // critical
		}
		return
	})
}

func (d *driveDB) Stat() (s stat.Stat) {
	d.view(func(t *bolt.Tx) (_ error) {
		os := t.Bucket(objectsBucket).Stats()
		s.Objects = os.KeyN
		// s.Space = os.
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
