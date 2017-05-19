package data

import (
	"fmt"
	"strings"

	"github.com/skycoin/skycoin/src/cipher"
)

type Stat struct {
	Total  int `json:"total"`  // total objects and schemas
	Memory int `json:"memory"` // all objects and roots
	Roots  int `json:"roots"`  // count of all root object
	Feeds  int `json:"feeds"`  // count of all non-empty feeds
}

// String implemets fmt.Stringer interface and returns
// human readable string
func (s Stat) String() string {
	return fmt.Sprintf("{total: %d, memory: %s, roots: %d, feeds: %d}",
		s.Total,
		HumanMemory(s.Memory),
		s.Roots,
		s.Feeds)
}

// HumanMemory returns human readable memory string
func HumanMemory(bytes int) string {
	var fb float64 = float64(bytes)
	var ms string = "B"
	for _, m := range []string{"KiB", "MiB", "GiB"} {
		if fb > 1024.0 {
			fb = fb / 1024.0
			ms = m
			continue
		}
		break
	}
	if ms == "B" {
		return fmt.Sprintf("%.0fB", fb)
	}
	// 2.00 => 2
	// 2.10 => 2.1
	// 2.53 => 2.53
	return strings.TrimRight(
		strings.TrimRight(fmt.Sprintf("%.2f", fb), "0"),
		".") + ms
}

// A DB is common database interface
type DB interface {

	//
	// Objects and Schemas
	//

	// Del deletes vlaue by key
	Del(key cipher.SHA256)
	// Find value by key and value
	Find(filter func(key cipher.SHA256, value []byte) bool) []byte
	// ForEach key-value pair (read-only)
	ForEach(f func(k cipher.SHA256, v []byte))
	// Get by key (nil is not found)
	Get(key cipher.SHA256) (data []byte, ok bool)
	// Get all key-value pairs
	GetAll() map[cipher.SHA256][]byte
	// GetSlice of values by slice of keys
	GetSlice(keys []cipher.SHA256) [][]byte
	// IsExist an object with provided key
	IsExist(k cipher.SHA256) bool
	// Len of the database (all objects and schemas)
	Len() (ln int)
	// Set value with provided key overwriting exist
	Set(key cipher.SHA256, value []byte)

	//
	// Root objects
	//

	// DelFeed deletes entire feed with all root objects.
	// The method doesn't remove related objects and schemas
	DelFeed(pk cipher.PubKey)
	// AddRoot to pk-feed
	AddRoot(pk cipher.PubKey, rp RootPack) (err error)
	// LastRoot returns last root of a feed
	LastRoot(pk cipher.PubKey) (rp RootPack, ok bool)
	// ForEachRoot of a feed by seq order (read only)
	ForEachRoot(pk cipher.PubKey,
		fn func(hash cipher.SHA256, rp RootPack) (stop bool))
	// Feeds returns all feeds that has at least
	// one root object
	Feeds() []cipher.PubKey
	// GetRoot by hash
	GetRoot(hash cipher.SHA256) (rp RootPack, ok bool)
	// DelRoot deletes root object by hash
	DelRoot(hash cipher.SHA256)

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
