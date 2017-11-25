package data

import (
	"github.com/skycoin/skycoin/src/cipher"
)

// An IterateFeedsFunc represents function for
// iterating over all feeds IdxDB contains
type IterateFeedsFunc func(cipher.PubKey) error

// A Feeds represents bucket of feeds
type Feeds interface {
	// Add feed. Adding a feed twice or
	// mote times does nothing
	Add(cipher.PubKey) error
	// Del feed if its empty. It's impossible to
	// delete non-empty feed. This restriction required
	// for related objects. We need to decrement refs count
	// of all related objects. Del never returns 'not found'
	// error
	Del(cipher.PubKey) error

	// Iterate all feeds. Use ErrStopRange to break
	// it iteration. The Iterate passes any error
	// returned from given function through. Except
	// ErrStopIteration that turns nil. It's possible
	// to mutate the IdxDB inside the Iterate
	Iterate(IterateFeedsFunc) error
	// Has returns true if the IdxDB contains
	// feed with given public key
	Has(cipher.PubKey) bool

	// Heads of feed. It returns ErrNoSuchFeed
	// if given feed doesn't exist
	Heads(cipher.PubKey) (hs Heads, err error)

	// Len returns number of feeds
	Len() int
}

// A Heads represens all heads of a feed
type Heads interface {
	// Roots of head with given nonce
	Roots(nonce uint64) (rs Roots, err error)
	// List all heads
	List() (heads []uint64, err error)
	// Len returns number of heads
	Len() int
}

// An IterateRootsFunc represents function for
// iterating over all Root objects of a feed
type IterateRootsFunc func(*Root) error

// A Roots represents bucket of Root objects.
// All Root objects ordered by seq number
// from small to big
type Roots interface {
	// Ascend iterates all Root object ascending order.
	// Use ErrStopIteration to stop iteration. Any error
	// (except the ErrStopIteration) returned by given
	// IterateRootsFunc will be passed through
	Ascend(IterateRootsFunc) error
	// Descend is the same as the Ascend, but it iterates
	// decending order. From lates Root objects to
	// oldes
	Descend(IterateRootsFunc) error

	// Set or update Root. Method modifies given Root
	// setting AccessTime and CreateTime to appropriate
	// values
	Set(*Root) error

	// Del Root by seq number
	Del(uint64) error

	// Get Root by seq number
	Get(uint64) (*Root, error)

	// Has the Roots Root with given seq?
	Has(uint64) bool

	// Len returns amount of saved Root
	// objects of this feed
	Len() int
}

// An IdxDB repesents database that contains
// meta information: feeds meta information
// about Root objects. There is data/idxdb
// package that implements the IdxDB. The
// IdxDB returns and uses errors ErrNotFound,
// ErrNoSuchFeed, ErrStopIteration and
// ErrFeedIsNotEmpty from this package
type IdxDB interface {
	Tx(func(Feeds) error) error // transaction
	Close() error               // close the IdxDB

	// TODO: stat
}
