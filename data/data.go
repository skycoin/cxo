package data

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

// common errors
var (
	ErrNotFound       = errors.New("not found")
	ErrStopIteration  = errors.New("stop iteration")
	ErrFeedIsNotEmpty = errors.New("can't remove feed: feed is not empty")
	ErrNoSuchFeed     = errors.New("no such feed")

	ErrInvalidSize = errors.New("invalid size of encoded data")
)

// A DB represents joiner of IdxDB and CXDS
type DB struct {
	cxds  CXDS
	idxdb IdxDB
}

// IdxDB of the DB
func (d *DB) IdxDB() IdxDB {
	return d.idxdb
}

// CXDS of the DB
func (d *DB) CXDS() CXDS {
	return d.cxds
}

// Clsoe the DB and all underlying
func (d *DB) Close() (err error) {
	if err = d.cxds.Close(); err != nil {
		d.idxdb.Close() // drop error
	} else {
		err = d.idxdb.Close() // use this error
	}
	return
}

// NewDB creates DB by given CXDS and IdxDB.
// The arguments must not be nil
func NewDB(cxds CXDS, idxdb IdxDB) *DB {
	if cxds == nil {
		panic("missing CXDS")
	}
	if idxdb == nil {
		panic("missing IdxDB")
	}
	return &DB{cxds, idxdb}
}

// A Root represetns meta information
// of a saved skyobject.Root
type Root struct {
	CreateTime int64 // created at, unix nano
	AccessTime int64 // last access time, unix nano

	Seq  uint64        // seq number of this Root
	Prev cipher.SHA256 // previous Root or empty if seq == 0

	Hash cipher.SHA256 // hash of the Root
	Sig  cipher.Sig    // signature of the Root
}

// Validate the Root
func (r *Root) Validate() (err error) {
	if r.Seq == 0 {
		if r.Prev != (cipher.SHA256{}) {
			return errors.New("(idxdb.Root.Validate) unexpected Prev hash")
		}
	} else if r.Prev == (cipher.SHA256{}) {
		return errors.New("(idxdb.Root.Validate) missing Prev hash")
	}

	if r.Hash == (cipher.SHA256{}) {
		return errors.New("(idxdb.Root.Validate) empty Hash")
	}

	if r.Sig == (cipher.Sig{}) {
		return errors.New("(idxdb.Root.Validate) empty Sig")
	}
	return
}

// Encode the Root
func (r *Root) Encode() (p []byte) {

	// the method genertes bytes equal to genrated by
	// github.com/skycoin/skycoin/src/cipher/encoder
	// but the Encode doesn't mess with reflection

	p = make([]byte, 8+8+8+len(cipher.SHA256{})*2+len(cipher.Sig{}))

	binary.LittleEndian.PutUint64(p, uint64(r.CreateTime))
	binary.LittleEndian.PutUint64(p[8:], uint64(r.AccessTime))

	binary.LittleEndian.PutUint64(p[8+8:], r.Seq)

	copy(p[8+8+8:], r.Prev[:])
	copy(p[8+8+8+len(cipher.SHA256{}):], r.Hash[:])
	copy(p[8+8+8+len(cipher.SHA256{})*2:], r.Sig[:])

	return
}

// Decode given encoded Root to this one
func (r *Root) Decode(p []byte) (err error) {

	if len(p) != 8+8+8+len(cipher.SHA256{})*2+len(cipher.Sig{}) {
		return ErrInvalidSize
	}

	r.CreateTime = int64(binary.LittleEndian.Uint64(p))
	r.AccessTime = int64(binary.LittleEndian.Uint64(p[8:]))

	r.Seq = binary.LittleEndian.Uint64(p[8+8:])

	copy(r.Prev[:], p[8+8+8:])
	copy(r.Hash[:], p[8+8+8+len(cipher.SHA256{}):])
	copy(r.Sig[:], p[8+8+8+len(cipher.SHA256{})*2:])
	return
}

// UpdateAccessTime updates access time
func (r *Root) UpdateAccessTime() {
	r.AccessTime = time.Now().UnixNano()
}

/*
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
*/
