package data

import (
	"github.com/skycoin/skycoin/src/cipher"
)

// A CXDS is interface of CX data store. The CXDS is
// key-value store with references count. There is
// data/cxds implementation that contains boltdb based
// and in-memeory (golang map based) implementations of
// the CXDS. The CXDS returns ErrNotFound from this
// package if any value has not been found
type CXDS interface {
	// Get value by key. Result is value and references count
	Get(key cipher.SHA256) (val []byte, rc uint32, err error)
	// GetInc is the same as Inc+Get
	GetInc(key cipher.SHA256) (val []byte, rc uint32, err error)
	// Set key-value pair. If value already exists the Set
	// increments references count. Otherwise the references count
	// will be set to 1
	Set(key cipher.SHA256, val []byte) (rc uint32, err error)
	// Add value, calculating hash internally. E.g. it clculates
	// hash and perfroms Set
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

	// MultiGet returns values by keys. It stops on first error.
	// If a value doesn't exist it returns ErrNotFound
	MultiGet(keys []cipher.SHA256) (vals [][]byte, err error)
	// MultiAdd append given values calulating hashes internally
	MultiAdd(vals [][]byte) (err error)

	// MultiInc increments all by keys.
	MultiInc(keys []cipher.SHA256) (err error)
	// MultiDec decrements
	MultiDec(keys []cipher.SHA256) (err error)

	// Iterate all keys in CXDS. The rc is refs count.
	// Given function must not mutate database. Use
	// ErrStopIteration to stop an iteration
	Iterate(func(key cipher.SHA256, rc uint32) error) error

	// Clsoe the CXDS
	Close() (err error)
}
