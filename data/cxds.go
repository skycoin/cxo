package data

import (
	"github.com/skycoin/skycoin/src/cipher"
)

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
// The CXDS removes all elements with zero-rc
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
	// Given function must not mutate database. Use
	// ErrStopIteration to stop an iteration
	Iterate(func(key cipher.SHA256, rc uint32) error) error

	// Stat returns amount and volume of all obejcts.
	// The Stat is not live. It saved by the skyobject
	// packcage before closing and loaded next time. The
	// skyobject keeps live statistics. Keeping the stat
	// avoid walking through a whole CXDS on start
	Stat() (amount, volume uint32, err error)

	// SetStat saves current statistic inside the CXDS
	SetStat(amount, volume uint32) (err error)

	// Close the CXDS
	Close() (err error)
}
