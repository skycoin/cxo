package data

import (
	"errors"
	"fmt"
	"strings"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data/stat"
)

// special cases
var (
	// ErrRootAlreadyExists oocurs when you try to save root object
	// that already exist in database. The error required for
	// networking to omit unnessesary work
	ErrRootAlreadyExists = errors.New("root already exists")
	// ErrRootIsOld occurs when you try to save an old root
	// object. The database reject all old root objects
	// and collect only new roots. Thus, it's impossible
	// to save a root object older than first root of a
	// feed. For example if seq of first root in db is 55,
	// then database reject all roots with seq leser then 55.
	// This way, it's easy to set min seq threshold
	ErrRootIsOld = errors.New("root is older then newest one")
)

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

	//
	// Feeds
	//

	// DelFeed deletes feed and all related root objects.
	// The method doesn't remove related objects and schemas
	DelFeed(pk cipher.PubKey)
	// AddRoot to feed, rejecting all roots older then
	// oldest the feed has got
	AddRoot(pk cipher.PubKey, rp RootPack) (err error)
	// LastRoot returns last root of a feed
	LastRoot(pk cipher.PubKey) (rp RootPack, ok bool)
	// RangeFeed itterate root objects of tgiven feed
	// ordered by seq from oldest to newest
	RangeFeed(pk cipher.PubKey, fn func(rp RootPack) (stop bool))
	// RangeFeedReverse is same as RangeFeed, but order
	// is inversed
	RangeFeedReverse(pk cipher.PubKey, fn func(rp RootPack) (stop bool))

	//
	// Roots
	//

	// GetRoot by hash
	GetRoot(hash cipher.SHA256) (rp RootPack, ok bool)
	// DelRootsBefore deletes root objects of given feed
	// before given seq number (exclusive)
	DelRootsBefore(pk cipher.PubKey, seq uint64)

	//
	// Other methods
	//

	// Stat of the DB
	Stat() (s stat.Stat)
	// Close the DB
	Close() (err error)
}

// A RootPack represents encoded root object with signature,
// seq number, and next/prev/this hashes
type RootPack struct {
	Root []byte
	Sig  cipher.Sig
	Seq  uint64

	Hash cipher.SHA256 // skyobject.RootReference
	Prev cipher.SHA256 // skyobject.RootReference
	Next cipher.SHA256 // skyobject.RootReference
}

// A RootError represents error that can be returned by AddRoot method
type RootError struct {
	feed  cipher.PubKey // feed of root
	hash  cipher.SHA256 // hash of root
	seq   uint64        // seq of root
	descr string        // description
}

func shortHex(a string) string {
	return string([]byte(a)[:7])
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
