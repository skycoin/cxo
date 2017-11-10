package cxds

import (
	"encoding/binary"
	"os"
	"time"

	"github.com/boltdb/bolt"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

var (
	objs    = []byte("o")       // objects bucket
	meta    = []byte("m")       // meta information
	version = []byte("version") // field with version in meta bucket
)

type driveCXDS struct {
	b *bolt.DB
}

// NewDriveCXDS opens existsing CXDS-database
// or creates new by given file name. Underlying
// database is boltdb (github.com/boltdb/bolt).
// E.g. this stores data on disk
func NewDriveCXDS(fileName string) (ds data.CXDS, err error) {

	var created bool // true if the file does not exist

	_, err = os.Stat(fileName)
	created = os.IsNotExist(err)

	var b *bolt.DB
	b, err = bolt.Open(fileName, 0644, &bolt.Options{
		Timeout: time.Millisecond * 500,
	})

	if err != nil {
		return
	}

	err = b.Update(func(tx *bolt.Tx) (err error) {

		// first of all, take a look the meta bucket
		var info = tx.Bucket(meta)

		if info == nil {

			// if the file has not been created, then
			// this DB file seems outdated (version 0)
			if created == false {
				return ErrMissingMetaInfo // report
			}

			// create the bucket and put meta information
			if info, err = tx.CreateBucket(meta); err != nil {
				return
			}

			// put version
			if err = info.Put(version, versionBytes()); err != nil {
				return
			}

		} else {

			// check out the version

			var vb []byte
			if vb = info.Get(version); len(vb) == 0 {
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

		_, err = tx.CreateBucketIfNotExists(objs)
		return

	})

	if err != nil {
		b.Close() // finialize
		return
	}

	ds = &driveCXDS{b} // wrap
	return
}

// Get value by key
func (d *driveCXDS) Get(key cipher.SHA256) (val []byte, rc uint32, err error) {
	err = d.b.View(func(tx *bolt.Tx) (_ error) {
		got := tx.Bucket(objs).Get(key[:])
		if len(got) == 0 {
			return data.ErrNotFound // pass through
		}
		rc = getRefsCount(got)
		val = make([]byte, len(got)-4)
		copy(val, got[4:])
		return
	})
	return
}

// GetInc is Get + Inc
func (d *driveCXDS) GetInc(key cipher.SHA256) (val []byte, rc uint32,
	err error) {

	err = d.b.Update(func(tx *bolt.Tx) (_ error) {
		got := tx.Bucket(objs).Get(key[:])
		if len(got) == 0 {
			return data.ErrNotFound // pass through
		}

		got = copy204(got) // this copying required

		rc = incRefsCount(got)
		val = got[4:]
		return
	})
	return
}

// Set of Inc if exists
func (d *driveCXDS) Set(key cipher.SHA256, val []byte) (rc uint32, err error) {
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

// Add is like Set, but it calculates key inside
func (d *driveCXDS) Add(val []byte) (key cipher.SHA256, rc uint32, err error) {
	if len(val) == 0 {
		err = ErrEmptyValue
		return
	}
	key = getHash(val)
	rc, err = d.Set(key, val)
	return
}

// Inc increments references counter
func (d *driveCXDS) Inc(key cipher.SHA256) (rc uint32, err error) {
	err = d.b.Update(func(tx *bolt.Tx) (_ error) {
		bk := tx.Bucket(objs)
		got := bk.Get(key[:])
		if len(got) == 0 {
			return data.ErrNotFound
		}

		// TODO (kostyarin): take a look the issue
		// https://github.com/boltdb/bolt/issues/204
		got = copy204(got)

		rc = incRefsCount(got)
		return bk.Put(key[:], got)
	})
	return
}

// Dec decrements referecnes counter and removes vlaue if it turns 0
func (d *driveCXDS) Dec(key cipher.SHA256) (rc uint32, err error) {
	err = d.b.Update(func(tx *bolt.Tx) (_ error) {
		bk := tx.Bucket(objs)
		got := bk.Get(key[:])
		if len(got) == 0 {
			return data.ErrNotFound
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

// DecGet is Get + Dec
func (d *driveCXDS) DecGet(key cipher.SHA256) (val []byte, rc uint32,
	err error) {

	err = d.b.Update(func(tx *bolt.Tx) (_ error) {
		bk := tx.Bucket(objs)
		got := bk.Get(key[:])
		if len(got) == 0 {
			return data.ErrNotFound
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

// MultiGet returns many values by list of keys
func (d *driveCXDS) MultiGet(keys []cipher.SHA256) (vals [][]byte, err error) {
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
				return data.ErrNotFound
			}
		}
		return
	})
	return
}

// MultiAdd appends all given values like the Add
func (d *driveCXDS) MultiAdd(vals [][]byte) (err error) {
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

// MultiInc increments references counter for all values by given keys
func (d *driveCXDS) MultiInc(keys []cipher.SHA256) (err error) {
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
				return data.ErrNotFound
			}
		}
		return
	})
	return
}

// MultiDec decrements all values by given keys
func (d *driveCXDS) MultiDec(keys []cipher.SHA256) (err error) {
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
				return data.ErrNotFound
			}
		}
		return
	})
	return
}

// Iterate all keys
func (d *driveCXDS) Iterate(iterateFunc func(cipher.SHA256,
	uint32) error) (err error) {

	err = d.b.View(func(tx *bolt.Tx) (err error) {
		var key cipher.SHA256

		bk := tx.Bucket(objs)
		c := bk.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			copy(key[:], k)
			if err = iterateFunc(key, getRefsCount(v)); err != nil {
				if err == data.ErrStopIteration {
					err = nil
				}
				return
			}
		}
		return
	})
	return
}

// Close DB
func (d *driveCXDS) Close() (err error) {
	return d.b.Close()
}

// https://github.com/boltdb/bolt/issues/204
func copy204(in []byte) (got []byte) {
	got = make([]byte, len(in))
	copy(got, in)
	return
}
