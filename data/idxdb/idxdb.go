package idxdb

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

// common errors
var (
	ErrInvalidSize    = errors.New("invalid size of encoded obejct")
	ErrStopIteration  = errors.New("stop iteration")
	ErrFeedIsNotEmpty = errors.New("can't remove feed: feed is not empty")
	ErrNoSuchFeed     = errors.New("no such feed")
	ErrNotFound       = errors.New("not found")
)

// An IterateObjectsFunc ...
type IterateObjectsFunc func(key cipher.SHA256, o *Object) (err error)

// An Objects represents
// bucket of objects
type Objects interface {
	// Inc increments RefsCount by given key.
	// The method never returns "not found" error.
	// The rc reply is new RefsCount, if the rc is
	// zero, then object hasn't been found
	Inc(key cipher.SHA256) (rc uint32, err error)
	// Get object by key. Returned Object
	// will have previous AccessTime, but the
	// AccessTime will be updated in DB. If
	// object not found, then it returns ErrNotFound
	Get(key cipher.SHA256) (o *Object, err error)

	// MultiGet returns all existsing obejcts by
	// given keys. The method never returns
	// "not found" error. AccessTime of returned obejcts
	// will be previous, but the AccessTime will be
	// updated inside DB
	MultiGet(keys []cipher.SHA256) (os []*Object, err error)
	// MultiInc increments RefsCount of all existing
	// objects by given keys. It returns new refs. counts.
	// If a count is zero, then this object doesn't
	// exist
	MultiInc(keys []cipher.SHA256) (rcs []uint32, err error)

	// Iterate all Obejcts
	Iterate(IterateObjectsFunc) (err error)

	// Dec decrements RefsCount by given key.
	// If the RefsCount turns zero, then this method
	// deletes the Object or the Object has not been
	// found. The rc reply is new RefsCount.
	// The method never returns "not found" error
	Dec(key cipher.SHA256) (rc uint32, err error)
	// Set new obejct or overwrite existsing. The
	// method changes given object. If object already exist
	// the stored object will be loaded to given.
	// RefsCount, AccessTime and CreateTie fields will
	// be set to appropriate values (even if the obejct
	// does not exist)
	Set(key cipher.SHA256, o *Object) (err error)

	// MultiSet performs Set for every given
	// key-objects pairs
	MultiSet(ko []KeyObject) (err error)
	// MultiDec decrements RefsCount of all obejcts
	// by given keys, removing objects for which
	// the RefCount turns zero. The method ignores
	// obejcts has not found. It returns resulting
	// references count. A ref. count is zero for
	// objects not found or deleted during the call
	MulitDec(keys []cipher.SHA256) (rcs []uint32, err error)

	Amount() int // total objects
}

// An IterateFeedsFunc represetns ...
type IterateFeedsFunc func(cipher.PubKey) error

// A Feeds represetns bucket of feeds
type Feeds interface {
	// Add feed. Adding a feed twice or
	// mote times does nothing
	Add(cipher.PubKey) error
	// Del feed if its empty. It's impossible to
	// delete non-empty feed. This restriction required
	// for related obejcts. We need to decrement refs count
	// of all related obejcts. Del never returns 'not found'
	// error
	Del(cipher.PubKey) error

	Iterate(IterateFeedsFunc) error // iterate all feeds
	HasFeed(cipher.PubKey) bool     // presence check

	// Roots of feed. You'll got ErrNoSuchFeed
	// if given feed doesn't exist
	Roots(cipher.PubKey) (Roots, error)
}

// An IterateRootsFunc represents ...
type IterateRootsFunc func(*Root) error

// A Roots represents bucket of Root objects
type Roots interface {
	Ascend(IterateRootsFunc) error  // iterate ascending oreder
	Descend(IterateRootsFunc) error // iterate descending order

	// Set or update Root. Method modifies orginal Root
	// setting AccessTime and CreateTime to appropriate
	// values. If Root already exists, then it's possible
	// to update only IsFull field (AccessTime will be
	// updated)
	Set(*Root) error

	// Inc refs count of Root by seq number
	Inc(seq uint64) error
	// Dec refs count of Root by seq numeber
	Dec(seq uint64) error

	// Del Root by seq number
	Del(uint64) error

	// Get Root by seq
	Get(uint64) (*Root, error)
}

// A Tx represetns ACID-transaction
type Tx interface {
	Objects() Objects
	Feeds() Feeds
}

// An IdxDB repesents API of the index-DB
type IdxDB interface {
	Tx(func(Tx) error) error // lookup transaction
	Close() error            // close the DB

	// TODO: stat
}

// A Root represetns meta information
// of Root obejct
type Root struct {
	Object // root is object

	Seq  uint64        // seq number of this Root
	Prev cipher.SHA256 // previous Root or empty if seq == 0

	Hash cipher.SHA256 // hash of the Root filed
	Sig  cipher.Sig    // signature of the Hash field

	IsFull bool // is the Root full
}

// Encode the Root
func (r *Root) Encode() (p []byte) {

	// the method genertes bytes equal to genrated by
	// github.com/skycoin/skycoin/src/cipher/encoder
	// but the Encode doesn't mess with reflection

	p = make([]byte, 20+8+len(cipher.SHA256{})*2+len(cipher.Sig{})+1)

	r.Object.EncodeTo(p)

	binary.LittleEndian.PutUint64(p[20:], r.Seq)

	copy(p[28:], r.Prev[:])
	copy(p[28+len(cipher.SHA256{}):], r.Hash[:])
	copy(p[28+len(cipher.SHA256{})*2:], r.Sig[:])

	if r.IsFull {
		p[len(p)-1] = 0x01 // the cipher/encoder uses 0x01 (not 0xff)
	}
	return
}

// Decode given encode Root to this one
func (r *Root) Decode(p []byte) (err error) {

	if len(p) != 20+8+len(cipher.SHA256{})*2+len(cipher.Sig{})+1 {
		return ErrInvalidSize
	}

	r.Object.Decode(p[:20])

	r.Seq = binary.LittleEndian.Uint64(p[20:])

	copy(r.Prev[:], p[28:])
	copy(r.Hash[:], p[28+len(cipher.SHA256{}):])
	copy(r.Sig[:], p[28+len(cipher.SHA256{})*2:])

	r.IsFull = (p[len(p)-1] != 0)

	return
}

// A KeyObject represents
// Object with its key
type KeyObject struct {
	Key    cipher.SHA256
	Object *Object
}

// An Object represents meta information of an obejct
type Object struct {
	RefsCount  uint32 // refs to this Obejct (to this meta info)
	CreateTime int64  // created at, unix nano
	AccessTime int64  // last access time, unix nano
}

// UpdateAccessTime updates access time
func (o *Object) UpdateAccessTime() {
	o.AccessTime = time.Now().UnixNano()
}

// Encode the Object
func (o *Object) Encode() (p []byte) {

	// the Encode generates equal []byte if we are using
	// github.com/skycoin/skycoin/src/cipher/encoder,
	// but the Encode doesn't mess with reflection,
	// thus it's faster

	p = make([]byte, 4+8+8)
	o.EncodeTo(p)
	return
}

// EncodeTo encodes the Object to given slice
func (o *Object) EncodeTo(p []byte) (err error) {

	if len(p) < 20 {
		return ErrInvalidSize
	}

	// RefsCount  4          |  0
	// CreateTime 8          |  4
	// AccessTime 8          | 12
	// ----------------------|-------
	//           20          |

	binary.LittleEndian.PutUint32(p, o.RefsCount)
	binary.LittleEndian.PutUint64(p[4:], uint64(o.CreateTime))
	binary.LittleEndian.PutUint64(p[12:], uint64(o.AccessTime))
	return
}

// Decode given encoded Object to this one
func (o *Object) Decode(p []byte) (err error) {

	if len(p) != 20 {
		return ErrInvalidSize
	}

	o.RefsCount = binary.LittleEndian.Uint32(p)
	o.CreateTime = int64(binary.LittleEndian.Uint64(p[4:]))
	o.AccessTime = int64(binary.LittleEndian.Uint64(p[12:]))

	return
}

/*

// A Volume represents size
// of an object in bytes
type Volume uint32

var units = [...]string{
	"B", "kB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB",
}

// String implements fmt.String interface
// and returns human-readable string
// represents the Volume
func (v Volume) String() (s string) {

	fv := float64(v)

	var i int
	for ; fv >= 1024.0; i++ {
		fv /= 1024.0
	}

	s = fmt.Sprintf("%.2f", fv)
	s = strings.TrimRight(s, "0") // remove trailing zeroes (x.10, x.00)
	s = strings.TrimRight(s, ".") // remove trailing dot (x.)
	s += units[i]

	return
}

*/
