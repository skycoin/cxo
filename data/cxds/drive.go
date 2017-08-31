package cxds

import (
	"time"

	"github.com/boltdb/bolt"

	"github.com/skycoin/skycoin/src/cipher"
)

var objs = []byte("o") // obejcts bucket

type DriveCXDS struct {
	b *bolt.DB
}

// NewDriveCXDS opens existsing DB or creates
// new by given file name. Underlying database
// is boltdb (github.com/boltdb/bolt). E.g. the
// DriveCXDS stores data on disk. The DriveCXDS
// impelments CXDS interface of
// github.com/skycoin/cxo/data package
func NewDriveCXDS(fileName string) (ds *DriveCXDS, err error) {
	var b *bolt.DB
	b, err = bolt.Open(fileName, 0644, &bolt.Options{
		Timeout: time.Millisecond * 500,
	})
	if err != nil {
		return
	}
	err = b.Update(func(tx *bolt.Tx) (err error) {
		_, err = tx.CreateBucketIfNotExists(objs)
		return
	})
	if err != nil {
		b.Close()
		return
	}
	ds = &DriveCXDS{b}
	return
}

func (d *DriveCXDS) Get(key cipher.SHA256) (val []byte, rc uint32, err error) {
	err = d.b.View(func(tx *bolt.Tx) (_ error) {
		got := tx.Bucket(objs).Get(key[:])
		if len(got) == 0 {
			return ErrNotFound // pass through
		}
		rc = getRefsCount(got)
		val = make([]byte, len(got)-4)
		copy(val, got[4:])
		return
	})
	return
}

func (d *DriveCXDS) GetInc(key cipher.SHA256) (val []byte, rc uint32,
	err error) {

	err = d.b.Update(func(tx *bolt.Tx) (_ error) {
		got := tx.Bucket(objs).Get(key[:])
		if len(got) == 0 {
			return ErrNotFound // pass through
		}

		got = copy204(got) // this copying required

		rc = incRefsCount(got)
		val = got[4:]
		return
	})
	return
}

func (d *DriveCXDS) Set(key cipher.SHA256, val []byte) (rc uint32, err error) {
	if len(val) == 0 {
		err = ErrEmptyValue
		return
	}
	err = d.b.Update(func(tx *bolt.Tx) (_ error) {
		bk := tx.Bucket(objs)
		got := bk.Get(key[:])
		if len(got) == 0 {
			rc = 1
			return bk.Put(key[:], append(one, val...))
		}

		// TODO (kostyarin): take a look the issue
		// https://github.com/boltdb/bolt/issues/204
		got = copy204(got)

		rc = incRefsCount(got) // increment refs count
		return bk.Put(key[:], got)
	})
	return
}

func (d *DriveCXDS) Add(val []byte) (key cipher.SHA256, rc uint32, err error) {
	if len(val) == 0 {
		err = ErrEmptyValue
		return
	}
	key = getHash(val)
	rc, err = d.Set(key, val)
	return
}

func (d *DriveCXDS) Inc(key cipher.SHA256) (rc uint32, err error) {
	err = d.b.Update(func(tx *bolt.Tx) (_ error) {
		bk := tx.Bucket(objs)
		got := bk.Get(key[:])
		if len(got) == 0 {
			return ErrNotFound
		}

		// TODO (kostyarin): take a look the issue
		// https://github.com/boltdb/bolt/issues/204
		got = copy204(got)

		rc = incRefsCount(got)
		return bk.Put(key[:], got)
	})
	return
}

func (d *DriveCXDS) Dec(key cipher.SHA256) (rc uint32, err error) {
	err = d.b.Update(func(tx *bolt.Tx) (_ error) {
		bk := tx.Bucket(objs)
		got := bk.Get(key[:])
		if len(got) == 0 {
			return ErrNotFound
		}

		// TODO (kostyarin): take a look the issue
		// https://github.com/boltdb/bolt/issues/204
		got = copy204(got)

		rc = decRefsCount(got)
		if rc == 0 {
			return bk.Delete(key[:])
		}
		return bk.Put(key[:], got)
	})
	return
}

func (d *DriveCXDS) DecGet(key cipher.SHA256) (val []byte, rc uint32,
	err error) {

	err = d.b.Update(func(tx *bolt.Tx) (_ error) {
		bk := tx.Bucket(objs)
		got := bk.Get(key[:])
		if len(got) == 0 {
			return ErrNotFound
		}

		got = copy204(got) // copying requied
		val = got[4:]

		rc = decRefsCount(got)
		if rc == 0 {
			return bk.Delete(key[:])
		}
		return bk.Put(key[:], got)
	})
	return
}

// TODO (kostyarin): ordered get to speed up get, because of B+-tree index
func (d *DriveCXDS) MultiGet(keys []cipher.SHA256) (vals [][]byte, err error) {
	if len(keys) == 0 {
		return
	}
	vals = make([][]byte, len(keys))
	err = d.b.View(func(tx *bolt.Tx) (_ error) {
		bk := tx.Bucket(objs)
		for i, k := range keys {
			if got := bk.Get(k[:]); len(got) > 0 {
				val := make([]byte, len(got)-4)
				copy(val, got[4:])
				vals[i] = val
			} else {
				return ErrNotFound
			}
		}
		return
	})
	return
}

// TODO (kostyarin): ordered add to speed up insert, because of B+-tree index
func (d *DriveCXDS) MultiAdd(vals [][]byte) (err error) {
	if len(vals) == 0 {
		return
	}
	err = d.b.Update(func(tx *bolt.Tx) (err error) {
		bk := tx.Bucket(objs)
		for _, val := range vals {
			if len(val) == 0 {
				return ErrEmptyValue
			}
			key := getHash(val)

			got := bk.Get(key[:])
			if len(got) == 0 {
				if err = bk.Put(key[:], append(one, val...)); err != nil {
					return
				}
				continue
			}

			// TODO (kostyarin): take a look the issue
			// https://github.com/boltdb/bolt/issues/204
			got = copy204(got)

			incRefsCount(got) // increment refs count
			if err = bk.Put(key[:], got); err != nil {
				return
			}
		}
		return
	})
	return
}

// TODO (kostyarin): ordered get to speed up get, because of B+-tree index
func (d *DriveCXDS) MultiInc(keys []cipher.SHA256) (err error) {
	if len(keys) == 0 {
		return
	}
	err = d.b.Update(func(tx *bolt.Tx) (err error) {
		bk := tx.Bucket(objs)
		for _, k := range keys {
			if got := bk.Get(k[:]); len(got) > 0 {

				// TODO (kostyarin): take a look the issue
				// https://github.com/boltdb/bolt/issues/204
				got = copy204(got)

				incRefsCount(got)
				if err = bk.Put(k[:], got); err != nil {
					return
				}
			} else {
				return ErrNotFound
			}
		}
		return
	})
	return
}

// TODO (kostyarin): ordered add to speed up insert, because of B+-tree index
func (d *DriveCXDS) MultiDec(keys []cipher.SHA256) (err error) {
	if len(keys) == 0 {
		return
	}
	err = d.b.Update(func(tx *bolt.Tx) (err error) {
		bk := tx.Bucket(objs)
		for _, k := range keys {
			if got := bk.Get(k[:]); len(got) > 0 {

				// TODO (kostyarin): take a look the issue
				// https://github.com/boltdb/bolt/issues/204
				got = copy204(got)

				if rc := decRefsCount(got); rc == 0 {
					if err = bk.Delete(k[:]); err != nil {
						return
					}
					continue
				}
				if err = bk.Put(k[:], got); err != nil {
					return
				}
			} else {
				return ErrNotFound
			}
		}
		return
	})
	return
}

func (d *DriveCXDS) Close() (err error) {
	return d.b.Close()
}

// https://github.com/boltdb/bolt/issues/204
func copy204(in []byte) (got []byte) {
	got = make([]byte, len(in))
	copy(got, in)
	return
}
