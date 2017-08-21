package data

import (
	"bytes"
	"encoding/binary"
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
	// ErrStopIteration used by Range, RangeDelete and Reverse functions
	// to stop itterating. It's error never bubbles up
	ErrStopIteration = errors.New("stop range")
)

// ViewObjects represents read-only bucket of objects
type ViewObjects interface {
	// Get obejct by key. It returns nil if requested object
	// doesn't exists
	Get(key cipher.SHA256) (value []byte)
	// MultiGet requests many objects returning slice
	// or objects, and nils for objects not found
	MultiGet(keys []cipher.SHA256) (values [][]byte)
	// Ascend over all objects. Use ErrStopIteration to break iteration.
	// The value argument can be used only inside transaction
	Ascend(func(key cipher.SHA256, value []byte) error) (err error)

	// GetObject returns encoded value with meta information
	GetObject(key cipher.SHA256) (obj *Object)
	// MultiGetObjects returns all requested object. Place
	// of not found objects will be holded by nil. E.g. length
	// of result will be the same as length of keys, even if
	// no objects found
	MultiGetObjects(keys []cipher.SHA256) (objects []*Object)
	// Ascend over all objects. Use ErrStopIteration to break iteration.
	// Value of obj argument can be used inside transaction
	AscendObjects(func(key cipher.SHA256, obj *Object) error) (err error)

	Amount() uint32 // total amount of all objects
	Volume() Volume // total volume of all obejcts
}

// UpdateObjects represents read-write bucket of objects
type UpdateObjects interface {
	ViewObjects

	// Set key->value pair
	Set(key cipher.SHA256, obj *Object) (err error)
	// Add object and get its key back
	Add(obj *Object) (key cipher.SHA256, err error)
	// Del delete by key. This method used by tests
	Del(key cipher.SHA256) (err error)

	// MultiAdd appends given objects, But it doesn't
	// returns keys
	MultiAdd(objects []*Object) (err error)

	// AscendDel used for deleting. If given function returns del = true
	// then this object will be removed. Use ErrStopIteration to break
	// the iteration. The AscendDel is low level method and you should not
	// use it. Otherwise, it can break some things of skyobject packge
	//
	// Note: it can delete objects when given function
	// returns. And it can delete them after all
	AscendDel(func(key cipher.SHA256, value []byte) (del bool, err error)) error
}

// ViewFeeds represents read-only bucket of feeds
type ViewFeeds interface {
	List() (list []cipher.PubKey) // list of all

	// Ascend itterates all feeds. Use ErrStopIteration to break
	// the iteration
	Ascend(func(pk cipher.PubKey) error) (err error)

	// Roots of given feed. This method returns nil if
	// given feed doesn't exist. Use this method to access
	// root objects
	Roots(pk cipher.PubKey) ViewRoots

	// IsExist returns true if given feed exists in database
	IsExist(pk cipher.PubKey) bool

	// Stat of the Feed
	Stat(pk cipher.PubKey) FeedStat
}

// UpdateFeeds represents bucket of feeds
type UpdateFeeds interface {

	//
	// inherited from ViewFeeds
	//

	List() (list []cipher.PubKey)
	Ascend(func(pk cipher.PubKey) error) (err error)
	IsExist(pk cipher.PubKey) bool
	Stat(pk cipher.PubKey) FeedStat

	//
	// change feed/roots
	//

	// Add an empty feed
	Add(pk cipher.PubKey) (err error)
	// Del deletes given feed and all roos.
	// It never returns "not found" error
	Del(pk cipher.PubKey) (err error)

	// AscendDel itterates all feeds. If given function returns del = true
	// then this feed will be deleted with all roots. Use ErrStopIteration
	// to break the iteration
	//
	// Note: it can delete objects when given function
	// returns. And it can delete them after all
	AscendDel(func(pk cipher.PubKey) (del bool, err error)) error

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

	// Ascend itterates all root objects ordered
	// by seq from oldest to newest. Use ErrStopIteration
	// to break the iteration
	Ascend(func(rp *RootPack) (err error)) error
	// Descend is the same as Ascend in reversed order.
	// E.g. it itterates all root objects ordered by seq
	// from newest (latest) to oldest
	Descend(fn func(rp *RootPack) (err error)) error
}

// UpdateRoots represents read-write bucket of Root obejcts
type UpdateRoots interface {
	ViewRoots

	// Add a root object. It returns ErrRootAlreadyExists
	// if root with the same seq number already exists.
	// The method doesn't modify rp. And if rp.IsFull is
	// true then it will be saved as full
	Add(rp *RootPack) (err error)
	// Update a Root. The method allows to update fields:
	// Volume, Amount and IsFull
	Update(rp *RootPack) (err error)
	// Del deletes root object by seq number. It never
	// returns "not found" error
	Del(seq uint64) (err error)

	// AscendDel used to delete Root obejcts.
	// If given function returns del = true, then
	// this root will be deleted. Use ErrStopIteration
	// to break the iteration
	//
	// Note: it can delete root objects when given function
	// returns. And it can delete them after all
	AscendDel(fn func(rp *RootPack) (del bool, err error)) error

	// DelBefore deletes root objects of given feed
	// before given seq number (exclusive).
	// For example: for set of roots [0, 1, 2, 3],
	// the DelBefore(2) removes 0 and 1
	DelBefore(seq uint64) (err error)
}

type ViewMisc interface {
	// Get obejct by key. The value is nil
	// if obejct not found. The value can be used
	// only inside current trnsaction
	Get(key []byte) (value []byte)
	// GetCopy returns long-lived value
	GetCopy(key []byte) (value []byte)

	// Ascend itterates all misc-objects ordered
	// by key from. Use ErrStopIteration to break the
	// iteration
	Ascend(func(key, value []byte) (err error)) error

	// TODO (kostyarin): add
	//
	// // Descend is the same as Ascend in reversed order
	// Descend(fn func(key, value []byte) (err error)) error
}

type UpdateMisc interface {
	ViewMisc

	// Set key-value pair
	Set(key, value []byte) error
	// Del by key. The Del never returns "not found" errors
	Del(key []byte) error

	// AscendDel itterates all misc-objects ordered
	// by key from. Use ErrStopIteration to break the
	// iteration. If given function returns del=true for
	// an object, then this object will be deleted.
	//
	// Note: it can delete objects when given function
	// returns. And it can delete them after all
	AscendDel(func(key, value []byte) (del bool, err error)) error

	// TODO (kostyarin): DescendDel
}

// A Tv represents read-only transaction
type Tv interface {
	Misc() ViewMisc // bucket for end-user needs

	Objects() ViewObjects // access objects
	Feeds() ViewFeeds     // access feeds
}

// A Tu represents read-write transaction
type Tu interface {
	Misc() UpdateMisc // bucket for end-user needs

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
// seq number, prev/this hashes and some necessary metadata,
// such as "is full", "space"
type RootPack struct {
	Root []byte

	// both Seq and Prev are encoded inside Root filed above
	// but we need them for database

	Seq  uint64        // seq number of this Root
	Prev cipher.SHA256 // previous Root or empty if seq == 0

	Hash cipher.SHA256 // hash of the Root filed
	Sig  cipher.Sig    // signature of the Hash field

	// machine local fields

	// IsFull is true if the Root has all related objects in
	// this DB. The field is temporary added and likely to
	// be moved outside the RootPack
	IsFull bool

	// Volume fit by this Root and all nested objects
	// including registry (wihtout place fit by keys)
	Volume Volume

	// Amount of all nested objects of this Root
	// including registry. E.g. for an empty root
	// the field will be 1 (registry onyl)
	Amount uint32
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

type skeys []cipher.SHA256

// for sorting

func (k skeys) Len() int {
	return len(k)
}

func (k skeys) Less(i, j int) bool {
	return bytes.Compare(k[i][:], k[j][:]) == -1
}

func (k skeys) Swap(i, j int) {
	k[i], k[j] = k[j], k[i]
}

func sortKeys(keys []cipher.SHA256) []cipher.SHA256 {
	s := skeys(keys)
	sort.Sort(s)
	return []cipher.SHA256(s)
}

type keyObj struct {
	key cipher.SHA256
	obj *Object
}

type keyObjs []keyObj

// for sorting

func (k keyObjs) Len() int { return len(k) }

func (k keyObjs) Less(i, j int) bool {
	return bytes.Compare(k[i].key[:], k[j].key[:]) == -1
}

func (k keyObjs) Swap(i, j int) {
	k[i], k[j] = k[j], k[i]
}

func sortObjs(objs []*Object) (slice keyObjs) {
	if len(objs) == 0 {
		return // nil
	}
	slice = make(keyObjs, 0, len(objs))
	for _, o := range objs {
		k := cipher.SumSHA256(o.Value)
		slice = append(slice, keyObj{k, o})
	}
	sort.Sort(slice)
	return
}

func canUpdateRoot(er, rp *RootPack) (err error) {
	if er.Seq != rp.Seq {
		panic("canUpdateRoot called with Roots with different Seq")
	}
	if er.Prev != rp.Prev {
		return errors.New("can't update Root, different Prev hashes")
	}
	if er.Hash != rp.Hash {
		return errors.New("can't update Root, different hashes")
	}
	if er.Sig != rp.Sig {
		return errors.New("can't update Root, different signatures")
	}
	if bytes.Compare(er.Root, rp.Root) != 0 {
		return errors.New("can't update Root, different encoded values")
	}
	return
}

// An Object represents encoded object
// with brief information about subtree
// of the object. The Object is representation
// of every object (object or registry)
// inside DB
type Object struct {
	Subtree        // brief info about subtree of the Object
	Value   []byte // encoded object in person
}

// Volue of the Object and its subtree
func (o *Object) Volume() Volume {
	return Volume(len(o.Value)) + o.Subtree.Volume + 4
}

// Amount is amount of subtree + 1, lol.
// E.g. this method returns total amount of
// objects of this Object including the Object
func (o *Object) Amount() uint32 {
	return o.Subtree.Amount + 1
}

// Encode is fast way to encode the Object.
// The Object can be encoded using cipher/encoder,
// this method returns the same result
func (o *Object) Encode() (val []byte) {
	// amount and volume are uint32
	val = make([]byte, 4+4, 4+4+len(o.Value))
	binary.LittleEndian.PutUint32(val, o.Subtree.Amount)
	binary.LittleEndian.PutUint32(val[4:], uint32(o.Subtree.Volume))
	val = append(val, o.Value...)
	return
}

// DecodeObject from raw encoded Object
func DecodeObject(val []byte) (o *Object) {
	if len(val) <= 8 {
		panic(fmt.Errorf("invalid object stored, length %d", len(val)))
	}
	o = new(Object)
	o.Subtree.Amount = binary.LittleEndian.Uint32(val)
	o.Subtree.Volume = Volume(binary.LittleEndian.Uint32(val[4:]))
	o.Value = val[8:]
	return
}

// A Subtree contains brief information
// about subtree of an object
type Subtree struct {
	Amount uint32 // total amount of all elements in subtree
	Volume Volume // total volume of all elements in subtree
}
