package idxdb

import (
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
	return nil // TODO (kostyarin): implement
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
			os = nil
			return
		}
		os[i] = o // nil or Object
	}
	return
}

// TODO (kostyarin): use ordered get to speed up the MultiGet,
// because of B+-tree index
func (d *driveObjs) MultiInc(keys []cipher.SHA256) (err error) {
	for _, key := range keys {
		if _, err = d.Inc(key); err != nil {
			return
		}
	}
	return
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
	// already exists:
	//  - inc RefsCount
	//  - update access time
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

func (d *driveObjs) MulitDec(keys []cipher.SHA256) (err error) {
	for _, key := range keys {
		if _, err = d.Dec(key); err != nil {
			return
		}
	}
	return
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
