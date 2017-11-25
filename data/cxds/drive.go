package cxds

import (
	"encoding/binary"
	"fmt"
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

// NewDriveCXDS opens existing CXDS-database
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

func incr(
	o *bolt.Bucket, // : objects
	key []byte, //     : key[:]
	val []byte, //     : value without leading rc (4 bytes)
	rc uint32, //      : existing rc
	inc int, //        : change the rc
) (
	nrc uint32, //     : new rc
	err error, //      : an error
) {

	switch {

	case inc == 0:

		// all done (no changes)
		nrc = rc

	case inc < 0:

		inc = -inc // change its sign

		if uinc := uint32(inc); uinc >= rc {

			// delete value (rc <= 0), nrc = 0
			err = o.Delete(key[:])

		} else {

			// reduce (rc > 0), keep value
			var repl = make([]byte, 4, 4+len(val))
			nrc = rc - uinc // reduced
			setRefsCount(repl, nrc)
			repl = append(repl, val...)
			err = o.Put(key[:], repl)

		}

	case inc > 0:

		// increase the rc
		nrc = rc + uint32(inc)
		var repl = make([]byte, 4, 4+len(val))
		setRefsCount(repl, nrc)
		repl = append(repl, val...)
		err = o.Put(key[:], repl)

	}

	return
}

// Get value by key changing or
// leaving as is references counter
func (d *driveCXDS) Get(
	key cipher.SHA256, // :
	inc int, //           :
) (
	val []byte, //        :
	rc uint32, //         :
	err error, //         :
) {

	var tx = func(tx *bolt.Tx) (err error) {

		var (
			o   = tx.Bucket(objs)
			got = o.Get(key[:])
		)

		if len(got) == 0 {
			return data.ErrNotFound // pass through
		}

		rc = getRefsCount(got)
		val = make([]byte, len(got)-4)
		copy(val, got[4:])

		rc, err = incr(o, key[:], val, rc, inc)
		return
	}

	if inc == 0 {
		err = d.b.View(tx) // lookup only
	} else {
		err = d.b.Update(tx) // some changes
	}

	return
}

func panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

// Set value and its references counter
func (d *driveCXDS) Set(
	key cipher.SHA256,
	val []byte,
	inc int,
) (
	rc uint32,
	err error,
) {

	if inc <= 0 {
		panicf("invalid inc argument in CXDS.Set: %d", inc)
	}

	if len(val) == 0 {
		err = ErrEmptyValue
		return
	}

	err = d.b.Update(func(tx *bolt.Tx) (err error) {

		var (
			o   = tx.Bucket(objs)
			got = o.Get(key[:])
		)

		if len(got) == 0 {
			rc, err = incr(o, key[:], val, 0, 1)
			return
		}

		rc, err = incr(o, key[:], got[4:], getRefsCount(got), inc)
		return
	})

	return
}

// Inc changes references counter
func (d *driveCXDS) Inc(
	key cipher.SHA256,
	inc int,
) (
	rc uint32,
	err error,
) {

	var tx = func(tx *bolt.Tx) (_ error) {

		var (
			o   = tx.Bucket(objs)
			got = o.Get(key[:])
		)

		if len(got) == 0 {
			return data.ErrNotFound
		}

		rc = getRefsCount(got)

		if inc == 0 {
			return // done
		}

		rc, err = incr(o, key[:], got[4:], rc, inc)
		return
	}

	if inc == 0 {
		err = d.b.View(tx) // lookup only
	} else {
		err = d.b.Update(tx) // changes required
	}

	return
}

// Iterate all keys
func (d *driveCXDS) Iterate(iterateFunc func(cipher.SHA256,
	uint32) error) (err error) {

	err = d.b.View(func(tx *bolt.Tx) (err error) {

		var (
			key cipher.SHA256
			c   = tx.Bucket(objs).Cursor()
		)

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

func copySlice(in []byte) (got []byte) {
	got = make([]byte, len(in))
	copy(got, in)
	return
}
