package data

import (
	"github.com/skycoin/cxo/data/cxds"
	"github.com/skycoin/cxo/data/idxdb"
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
