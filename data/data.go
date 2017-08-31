package data

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data/cxds"
	"github.com/skycoin/cxo/data/idxdb"
)

// A CXDS is interface of CX data store that is client
// for CX data server or any stub package. The CXDS is
// key-value store with references count
type CXDS interface {
	// Get value by key. Result is value and references count
	Get(key cipher.SHA256) (val []byte, rc uint32, err error)
	// GetInc is the same as Inc+Get
	GetInc(key cipher.SHA256) (val []byte, rc uint32, err error)
	// Set key-value pair. If value already exists the Set
	// increments references count
	Set(key cipher.SHA256, val []byte) (rc uint32, err error)
	// Add value, calculating hash internally. If
	// value already exists the Add increments references
	// count
	Add(val []byte) (key cipher.SHA256, rc uint32, err error)
	// Inc increments references count
	Inc(key cipher.SHA256) (rc uint32, err error)
	// Dec decrements references count. The rc is zero if
	// value has been deleted by the Dec
	Dec(key cipher.SHA256) (rc uint32, err error)
	// DecGet is the same as Dec but it returns value even if
	// it has been deleted by call
	DecGet(key cipher.SHA256) (val []byte, rc uint32, err error)

	// batch operation

	// MultiGet returns values by keys
	MultiGet(keys []cipher.SHA256) (vals [][]byte, err error)
	// MultiAdd append given values
	MultiAdd(vals [][]byte) (err error)

	// MultiInc increments
	MultiInc(keys []cipher.SHA256) (err error)
	// MultiDec decrements
	MultiDec(keys []cipher.SHA256) (err error)

	// Clsoe the CXDS
	Close() (err error)
}

// A DB represents joiner of
// IdxDB and CXDS
type DB struct {
	cxds  CXDS
	idxdb idxdb.IdxDB
}

// IdxDB of the DB
func (d *DB) IdxDB() idxdb.IdxDB {
	return d.idxdb
}

// CXDS of the DB
func (d *DB) CXDS() CXDS {
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

func (d *DB) Close() (err error) {
	if err = d.cxds.Close(); err != nil {
		d.idxdb.Close() // drop error
	} else {
		err = d.idxdb.Close() // use this error
	}
	return
}

// NewDB creates DB by given CXDS and IdxDB. The arguments
// must not be nil
func NewDB(cxds CXDS, idxdb idxdb.IdxDB) *DB {
	return &DB{cxds, idxdb}
}
