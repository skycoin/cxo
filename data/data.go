package data

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"github.com/skycoin/skycoin/src/cipher"
)

// common errors
var (
	// ErrRootAlreadyExists oocurs when you try to save root object
	// that already exist in database. The error required for
	// networking to omit unnessesary work
	ErrRootAlreadyExists = errors.New("root already exists")
	// ErrNotFound occurs where any requested object doesn't exist in
	// database
	ErrNotFound = errors.New("not found")
	// ErrStopRange used by Range, RangeDelete and Reverse functions
	// to stop itterating. It's error never bubbles up
	ErrStopRange = errors.New("stop range")
)

// ViewObjects represents read-only bucket of objects
type ViewObjects interface {
	// Get obejct by key. It returns nil if requested object
	// doesn't exists. Retuned slice valid only inside current
	// transaction. To get long lived copy use GetCopy
	Get(key cipher.SHA256) (value []byte)
	// GetCopy similar to Get, but it returns long lived object
	GetCopy(key cipher.SHA256) (value []byte)
	// IsExist returns true if object with given hash presence in database
	IsExist(key cipher.SHA256) (ok bool)
	// Range over all objects. Use ErrStopRange to break itteration
	Range(func(key cipher.SHA256, value []byte) error) (err error)
}

// UpdateObjects represents read-write bucket of objects
type UpdateObjects interface {
	ViewObjects

	// Del deletes object by key. It never returns
	// "not found" error.
	Del(key cipher.SHA256) (err error)
	// Set key->value pair
	Set(key cipher.SHA256, value []byte) (err error)
	// Add value getting key
	Add(value []byte) (key cipher.SHA256, err error)

	// SetMap performs Set for each element of given map.
	// The method sorts given data by key increasing performance
	SetMap(map[cipher.SHA256][]byte) (err error)

	// RangeDel used for deleting. If given function returns del = true
	// then this object will be removed. Use ErrStopRange to break
	// the Range
	RangeDel(func(key cipher.SHA256, value []byte) (del bool, err error)) error
}

// ViewFeeds represents read-only bucket of feeds
type ViewFeeds interface {
	IsExist(pk cipher.PubKey) (ok bool) // presence check
	List() (list []cipher.PubKey)       // list of all

	// Range itterates all feeds. Use ErrStopRange to break
	// the Range
	Range(func(pk cipher.PubKey) error) (err error)

	// Roots of given feed. This method returns nil if
	// given feed doesn't exist. Use this method to access
	// root objects
	Roots(pk cipher.PubKey) ViewRoots
}

// UpdateFeeds represents bucket of feeds
type UpdateFeeds interface {

	//
	// inherited from ViewFeeds
	//

	IsExist(pk cipher.PubKey) (ok bool)
	List() (list []cipher.PubKey)
	Range(func(pk cipher.PubKey) error) (err error)

	//
	// change feed/roots
	//

	// Add an empty feed
	Add(pk cipher.PubKey) (err error)
	// Del deletes given feed and all roos.
	// It never returns "not found" error
	Del(pk cipher.PubKey) (err error)

	// RangeDel itterates all feeds. If given function returns del = true
	// then this feed will be deleted with all roots. Use ErrStopRange
	// to break the ReangeDel
	RangeDel(func(pk cipher.PubKey) (del bool, err error)) error

	// Roots of given feed. This method returns nil if
	// given feed doesn't exist
	Roots(pk cipher.PubKey) UpdateRoots
}

// ViewRoots represents read-only bucket of Roots
type ViewRoots interface {
	Feed() cipher.PubKey // feed of this Roots

	// Last returns last root of this feed.
	// It returns nil if feed is empty
	Last() (rp *RootPack)
	// Get a root object by seq number
	Get(seq uint64) (rp *RootPack)

	// Range itterates all root objects ordered
	// by seq from oldest to newest. Use ErrStopRange
	// to break the Range
	Range(func(rp *RootPack) (err error)) error
	// Revers is the same as Range in reversed order.
	// E.g. it itterates all root objects ordered by seq
	// from newest (latest) to oldest
	Reverse(fn func(rp *RootPack) (err error)) error
}

// UpdateRoots represents read-write bucket of Root obejcts
type UpdateRoots interface {
	ViewRoots

	// Add a root object. It returns ErrRootAlreadyExists
	// if root with the same seq number already exists
	Add(rp *RootPack) (err error)
	// Del deletes root object by seq number. It never
	// returns "not found" error
	Del(seq uint64) (err error)

	// RangeDelete used to delete Root obejcts.
	// If given function returns del = true, then
	// this root will b edeleted. Use ErrStopRange
	// to break the RangeDel
	RangeDel(fn func(rp *RootPack) (del bool, err error)) error

	// DelBefore deletes root objects of given feed
	// before given seq number (exclusive).
	// For example: for set of roots [0, 1, 2, 3],
	// the DelBefore(2) removes 0 and 1
	DelBefore(seq uint64) (err error)
}

// A Tv represents read-only transaction
type Tv interface {
	Objects() ViewObjects // access objects
	Feeds() ViewFeeds     // access feeds
}

// A Tu represents read-write transaction
type Tu interface {
	Objects() UpdateObjects // access objects
	Feeds() UpdateFeeds     // access feeds

}

// A DB is common database interface
type DB interface {
	View(func(t Tv) error) (err error)   // perform a read only transaction
	Update(func(t Tu) error) (err error) // perform a read-write transaction
	Stat() (s Stat)                      // statistic
	Close() (err error)                  // clsoe database
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

type keyValue struct {
	key cipher.SHA256
	val []byte
}

type keyValues []keyValue

// for sorting

func (k keyValues) Len() int { return len(k) }

func (k keyValues) Less(i, j int) bool {
	return bytes.Compare(k[i].key[:], k[j].key[:]) == -1
}

func (k keyValues) Swap(i, j int) {
	k[i], k[j] = k[j], k[i]
}

func sortMap(m map[cipher.SHA256][]byte) (slice keyValues) {
	if len(m) == 0 {
		return // nil
	}
	slice = make([]keyValue, 0, len(m))
	for k, v := range m {
		slice = append(slice, keyValue{k, v})
	}
	sort.Sort(slice)
	return
}
