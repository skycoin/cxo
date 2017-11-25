package data

import (
	"github.com/skycoin/skycoin/src/cipher"
)

// A CXDS is interface of CX data store. The CXDS is
// key-value store with references count. There is
// data/cxds implementation that contains boltdb based
// and in-memory (golang map based) implementations of
// the CXDS. The CXDS returns ErrNotFound from this
// package if any value has not been found
type CXDS interface {

	// Get and increment. Use negative inc to decrement and zero
	// to keep untouched
	Get(key cipher.SHA256, inc int) (val []byte, rc uint32, err error)
	// Set and increment. Use negative inc to decrement or zero to
	// keep untouched. E.g. Set(smthng, 0) can be a bad choose,
	// because the Set doesn't appends one to references counter
	Set(key cipher.SHA256, inc int) (rc uint32, err error)
	// Inc increments or decrements (if given inc is negative)
	// references count for value with given key
	Inc(key cipher.SHA256, inc int) (err error)

	// Iterate all keys in CXDS. The rc is refs count.
	// Given function must not mutate database. Use
	// ErrStopIteration to stop an iteration
	Iterate(func(key cipher.SHA256, rc uint32) error) error

	// Close the CXDS
	Close() (err error)
}
