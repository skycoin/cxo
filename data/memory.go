package data

import (
	"bytes"
	"encoding/binary"
	"sort"
	"sync"
	"time"

	"github.com/boltdb/bolt"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// buckets:
//  - objects hash -> []byte (including schemas)
//  - roots   hash -> encoded RootPack
//  - feeds   pubkey -> { seq -> hash of root }
type memoryDB struct {
	mx      sync.Mutex
	objects map[cipher.SHA256][]byte
	roots   map[cipher.SHA256][]byte
	feeds   map[cipher.PubKey]map[uint64]cipher.SHA256
}

// NewMemoryDB creates new database in memory
func NewMemoryDB() (db DB) {
	db = &memoryDB{
		objects: make(map[cipher.SHA256][]byte),
		roots:   make(map[cipher.SHA256][]byte),
		feeds:   make(map[cipher.PubKey]map[uint64]cipher.SHA256),
	}
	return
}

func (d *memoryDB) Del(key cipher.SHA256) {
	d.mx.Lock()
	defer d.mx.Unlock()

	delete(d.objects, key)
}

func (d *memoryDB) Get(key cipher.SHA256) (value []byte, ok bool) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	value, ok = d.objects[key] // unsafe
	return
}

func (d *memoryDB) Set(key cipher.SHA256, value []byte) {
	d.mx.Lock()
	defer d.mx.Unlock()

	d.objects[key] = value
}

func (d *memoryDB) Add(value []byte) (key cipher.SHA256) {
	key = cipher.SumSHA256(value)
	d.Set(key, value)
	return
}

func (d *memoryDB) IsExist(key cipher.SHA256) (ok bool) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	_, ok = d.objects[key]
	return
}

func (d *memoryDB) Len() (ln int) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	ln = len(d.objects)
	return
}

func (d *memoryDB) Range(fn func(key cipher.SHA256, value []byte) (stop bool)) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	for key, value := range d.objects {
		if fn(key, value) {
			return
		}
	}
	return
}

//
// Feeds
//

func (d *memoryDB) DelFeed(pk cipher.PubKey) {
	d.mx.Lock()
	defer d.mx.Unlock()

	for _, hash := range d.feeds[pk] {
		delete(d.roots, hash)
	}
	delete(d.feeds, pk)
}

func (d *memoryDB) AddRoot(pk cipher.PubKey, rp RootPack) (err error) {
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
	//////
	//////
	//////
	///////////
	////
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
	//////////////////
	////////////
	//////////
	////////////
	return
}

func (d *memoryDB) LastRoot(pk cipher.PubKey) (rp RootPack, ok bool) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	roots := d.feeds[pk]
	if len(roots) == 0 {
		return // not found
	}
	var max uint64 = 0
	for seq := range roots {
		if max < seq {
			max = seq
		}
	}
	data, exists := d.roots[roots[max]]
	if !exists {
		panic("broken db: misisng root") // critical
	}
	if err := encoder.DeserializeRaw(data, &rp); err != nil {
		panic(err) // critical
	}
	ok = true // found
	return
}

func (d *memoryDB) RangeFeed(pk cipher.PubKey,
	fn func(rp RootPack) (stop bool)) {

	d.mx.RLock()
	defer d.mx.RUnlock()

	roots := d.feeds
	if len(roots) == 0 {
		return // empty feed
	}

	var o order = make(order, 0, len(roots))
	for seq := range roots {
		o = append(o, seq)
	}
	sort.Sort(o)

	var rp RootPack

	for _, seq := range o {
		hash := roots[seq]
		data, ok := d.roots[hash]
		if !ok {
			panic("broken database: misisng root") // critical
		}
		if err := encoder.DeserializeRaw(data, &rp); err != nil {
			panic(err) // critical
		}
		if fn(rp) {
			return // break
		}
	}
}

func (d *memoryDB) RangeFeedReverse(pk cipher.PubKey,
	fn func(rp RootPack) (stop bool)) {

	d.mx.RLock()
	defer d.mx.RUnlock()

	roots := d.feeds
	if len(roots) == 0 {
		return // empty feed
	}

	var o order = make(order, 0, len(roots))
	for seq := range roots {
		o = append(o, seq)
	}
	sort.Reverse(o)

	var rp RootPack

	for _, seq := range o {
		data, ok := d.roots[roots[seq]]
		if !ok {
			panic("broken database: misisng root") // critical
		}
		if err := encoder.DeserializeRaw(data, &rp); err != nil {
			panic(err) // critical
		}
		if fn(rp) {
			return // break
		}
	}
}

func (d *memoryDB) FeedsLen() (ln int) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	ln = len(d.feeds)
	return
}

func (d *memoryDB) FeedLen(pk cipher.PubKey) (ln int) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	ln = len(d.feeds[pk])
	return
}

//
// Roots
//

func (d *memoryDB) GetRoot(hash cipher.SHA256) (rp RootPack, ok bool) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	var data []byte
	if data, ok = d.roots[hash]; !ok {
		return
	}
	if err := encoder.DeserializeRaw(data, &rp); err != nil {
		panic(err) // critical
	}
	return
}

func (d *memoryDB) RootsLen() (ln int) {
	d.mx.RLock()
	defer d.mx.RUnlock()
	ln = len(d.roots)
	return
}

func (d *memoryDB) DelRootsBefore(pk cipher.PubKey, seq uint64) {
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

func (d *memoryDB) Stat() (s Stat) {
	//
	return
}

func (d *memoryDB) Close() (_ error) {
	return
}

type order []uint64

func (o order) Len() int           { return len(o) }
func (o order) Less(i, j int) bool { return o[i] < o[j] }
func (o order) Swap(i, j int)      { o[i], o[j] = o[j], o[i] }
