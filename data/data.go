package data

import (
	"errors"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data/cxds"
	"github.com/skycoin/cxo/data/idxdb"
)

// common errors
var (
	ErrObjectRemovedFromCXDS = errors.New("obejct removed from CXDS")
)

// A DB represents joiner of
// IdxDB and CXDS
type DB struct {
	idxdb idxdb.IdxDB
	cxds  cxds.CXDS
}

// IdxDB of the DB
func (d *DB) IdxDB() idxdb.IdxDB {
	return d.idxdb
}

// CXDS of the DB
func (d *DB) CXDS() cxds.CXDS {
	return d.cxds
}

// NewMemoryDB returns DB in memory
func NewMemoryDB() (db *DB) {
	db = new(DB)
	db.cxds = cxds.NewMemoryCXDS()
	db.idxdb = idxdb.NewMemeoryDB()
	return
}

// NewDriveDB returns DB on drive. The prefix argument
// is path to two files: <prefix>.cxds and <prefix>.index
func NewDriveDB(prefix string) (db *DB, err error) {
	db = new(DB)
	if db.cxds, err = cxds.NewDriveCXDS(prefix + ".cxds"); err != nil {
		return
	}
	if db.idxdb, err = idxdb.NewDriveIdxDB(prefix + ".index"); err != nil {
		db.cxds.Close()
		return
	}
	return
}

// Get Object by key
func (d *DB) Get(key cipher.SHA256) (o *Object, err error) {
	var io *idxdb.Object
	err = d.idxdb.Tx(func(tx idxdb.Tx) (err error) {
		io, err = tx.Objects().Get(key)
		return
	})
	if err != nil {
		return
	}
	var val []byte
	var rc uint32
	if val, rc, err = d.cxds.Get(key); err != nil {
		return
	}
	if rc == 0 {
		err = ErrObjectRemovedFromCXDS // alert
		return
	}
	o = new(Object)
	o.Value = val
	o.Rc = rc
	o.Object = io
	return
}

// GetMeta information of object
func (d *DB) GetMeta(key cipher.SHA256) (meta *idxdb.Object, err error) {
	err = d.idxdb.Tx(func(tx idxdb.Tx) (err error) {
		meta, err = tx.Objects().Get(key)
		return
	})
	return
}

// Set object
func (d *DB) Set(key cipher.SHA256, o *Object) error {
	return d.idxdb.Tx(func(tx idxdb.Tx) (err error) {
		if err = tx.Objects().Set(key, o.Object); err != nil {
			return
		}
		if o.Object.RefsCount == 1 {
			// new object we need to push to CXDS
			_, err = d.cxds.Set(key, o.Value)
		}
		return // rollback on error
	})
}

// Dec decrements RefsCount of an obejct. It's like delete,
// if RefsCount turn zero, then obejct will be deleted
func (d *DB) Dec(key cipher.SHA256) error {
	return d.idxdb.Tx(func(tx idxdb.Tx) (err error) {
		var rc uint32
		if rc, err = tx.Objects().Dec(key); err != nil {
			return
		}
		if rc == 0 { // deleted
			_, err = d.cxds.Dec(key)
		}
		return // rollback on error
	})
}

// An Object represents encoded value with meta information
type Object struct {
	Value         []byte // encoded obejct
	Rc            uint32 // refs count of CXDS
	*idxdb.Object        // meta information
}
