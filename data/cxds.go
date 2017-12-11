package data

import (
	"github.com/skycoin/skycoin/src/cipher"
)

// An IterateObjectsFunc sued to iterate over objects
// of the CXDS. The cal argumen is read onyl can must
// not be modified, the val can be used only inside the
// function.
type IterateObjectsFunc func(key cipher.SHA256, rc uint32, val []byte) error

// A CXDS is interface of CX data store. The CXDS is
// key-value store with references counters. There is
// data/cxds implementation that contains boltdb based
// and in-memory (golang map based) implementations of
// the CXDS. The CXDS returns ErrNotFound from this
// package if any value has not been found. The
// references counters is number of objects that point
// to an object. E.g. shema of the CXDS is
//
//     key -> {rc, val}
//
// The CXDS keeps elements with rc == 0. End used should
// track size of the DB and remove objects that doesn't
// used to free up space
type CXDS interface {

	// Get and change references counter (rc). If the
	// inc argument is zero then the rc will be leaved
	// as is. If value with given key doen't exist, then
	// the Get method returns (nil, 0, data.ErrNotFound).
	// Use negative inc argument to reduce the rc and
	// positive to increase it
	Get(key cipher.SHA256, inc int) (val []byte, rc uint32, err error)

	// Set and change references counter (rc). If the inc
	// argument is negative or zero, then the Set method
	// panics. Other words, the Set method used to create
	// and increase the rc (increase at least by one). E.g.
	// it's impossible to set vlaue with zero-rc
	Set(key cipher.SHA256, val []byte, inc int) (rc uint32, err error)

	// Inc increments or decrements (if given inc is negative)
	// references count for value with given key. If given
	// inc argument is zero, then the Inc method checks
	// presence of the value. E.g. if it returns ErrNotFound
	// then value doesn't exist. The Inc returns new rc
	Inc(key cipher.SHA256, inc int) (rc uint32, err error)

	// Iterate all keys in CXDS. The rc is refs count.
	// Use ErrStopIteration to stop an iteration.
	Iterate(iterateFunc IterateObjectsFunc) (err error)

	// Del removes obejct with given key unconditionally.
	// The Del method doesn't return an error if obejct
	// doesn't exist
	Del(key cipher.SHA256) (err error)

	//
	// Stat
	//

	// Amount of obejcts
	Amount() (all, used int)
	// Volume of objects
	Volume() (all, used int)

	//
	// Close
	//

	// Close the CXDS
	Close() (err error)
}
