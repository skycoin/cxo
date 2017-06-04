package data

import (
	"errors"
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"
)

// ErrRootAlreadyExists oocurs when you try to save root object
// that already exist in database. The error required for
// networking to omit unnessesary work
var ErrRootAlreadyExists = errors.New("root already exists")

// A DB is common database interface
type DB interface {

	//
	// Objects and Schemas
	//

	// Del deletes by key
	Del(key cipher.SHA256)
	// Get by key
	Get(key cipher.SHA256) (value []byte, ok bool)
	// Set value using its precalculated key
	Set(key cipher.SHA256, value []byte)
	// Add value returning its key
	Add(value []byte) cipher.SHA256
	// IsExist check persistence of a value by key
	IsExist(key cipher.SHA256) (ok bool)
	// Range over all objects and schemas (read only)
	Range(func(key cipher.SHA256, value []byte) (stop bool))
	// RangeDelete used to delete objects
	RangeDelete(func(key cipher.SHA256) (del bool))

	//
	// Feeds
	//

	// AddFeed appends empty feed or does nothing if
	// given feed already exists in database
	AddFeed(pk cipher.PubKey)
	// HasFeed returns true if datbase contains given feed
	HasFeed(pk cipher.PubKey) (has bool)
	// Feeds returns list of feeds
	Feeds() []cipher.PubKey
	// DelFeed deletes feed and all related root objects.
	// The method doesn't remove related objects and schemas
	DelFeed(pk cipher.PubKey)
	// AddRoot to feed. AddRoot adds feed if it doesn't exist
	AddRoot(pk cipher.PubKey, rp *RootPack) (err error)
	// LastRoot returns last root of a feed
	LastRoot(pk cipher.PubKey) (rp *RootPack, ok bool)
	// RangeFeed itterate root objects of tgiven feed
	// ordered by seq from oldest to newest
	RangeFeed(pk cipher.PubKey, fn func(rp *RootPack) (stop bool))
	// RangeFeedReverse is same as RangeFeed, but order
	// is reversed
	RangeFeedReverse(pk cipher.PubKey, fn func(rp *RootPack) (stop bool))

	//
	// Roots
	//

	// GetRoot by hash
	GetRoot(hash cipher.SHA256) (rp *RootPack, ok bool)
	// DelRootsBefore deletes root objects of given feed
	// before given seq number (exclusive)
	DelRootsBefore(pk cipher.PubKey, seq uint64)

	//
	// Other methods
	//

	// Stat of the DB
	Stat() (s Stat)
	// Close the DB
	Close() (err error)
}

// A RootPack represents encoded root object with signature,
// seq number, and next/prev/this hashes
type RootPack struct {
	Root []byte

	// both Seq and Prev are encoded inside Root filed above
	// but we need them for database

	Seq  uint64        // seq number of this Root
	Prev cipher.SHA256 // previous Root or empty if seq == 0

	Hash cipher.SHA256 // hash of the Root filed
	Sig  cipher.Sig    // signature of the Hash field
}

// A RootError represents error that can be returned by AddRoot method
type RootError struct {
	feed  cipher.PubKey // feed of root
	hash  cipher.SHA256 // hash of root
	seq   uint64        // seq of root
	descr string        // description
}

// Error implements error interface
func (r *RootError) Error() string {
	return fmt.Sprintf("[%s:%s:%d] %s",
		shortHex(r.feed.Hex()),
		shortHex(r.hash.Hex()),
		r.seq,
		r.descr)
}

// Feed of erroneous Root
func (r *RootError) Feed() cipher.PubKey { return r.feed }

// Hash of erroneous Root
func (r *RootError) Hash() cipher.SHA256 { return r.hash }

// Seq of erroneous Root
func (r *RootError) Seq() uint64 { return r.seq }

func newRootError(pk cipher.PubKey, rp *RootPack, descr string) (r *RootError) {
	return &RootError{
		feed:  pk,
		hash:  rp.Hash,
		seq:   rp.Seq,
		descr: descr,
	}
}
