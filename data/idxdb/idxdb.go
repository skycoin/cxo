package idxdb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

// common errors
var (
	ErrInvalidSize   = errors.New("invalid size of encoded obejct")
	ErrStopIteration = errors.New("stop iteration")
)

// An IterateObjectsFunc ...
type IterateObjectsFunc func(key cipher.SHA256, o *Object) (err error)

// An Objects represents
// bucket of objects
type Objects interface {
	// Inc increments RefsCount by given key.
	// The method never returns "not found" error.
	// The rc reply is new RefsCount
	Inc(key cipher.SHA256) (rc uint32, err error)
	// Get object by key. It returns (nil, nil)
	// if object has not found. Returned Object
	// will have previous AccessTime, but the
	// AccessTime will be updated in DB
	Get(key cipher.SHA256) (o *Object, err error)

	// MultiGet returns all existsing obejcts by
	// given keys. The method never returns
	// "not found" error. AccessTime of returned obejcts
	// will be previous, but the AccessTime will be
	// updated inside DB
	MultiGet(keys []cipher.SHA256) (os []*Object, err error)
	// MultiInc increments RefsCount of all existing
	// objects by given keys. The method never returns
	// "not found" error
	MultiInc(keys []cipher.SHA256) (err error)

	// Iterate all Obejcts
	Iterate(IterateObjectsFunc) (err error)

	// Dec decrements RefsCount by given key.
	// If the RefsCount turns zero, then this method
	// deletes the Object. The rc reply is new RefsCount.
	// The method never returns "not found" error
	Dec(key cipher.SHA256) (rc uint32, err error)
	// Set new obejct or overwrite existsing. If Obejct already
	// exists, then nothing chagned inside the Object except
	// RefsCount and AccessTime. If the Object doesn't exist
	// then its RefsCount set to 1 and CreateTime to now,
	// before saving
	Set(key cipher.SHA256, o *Object) (err error)

	// MultiSet performs Set for every given
	// key-objects pairs
	MultiSet(ko []KeyObject) (err error)
	// MultiDec decrements RefsCount of all obejcts
	// by given keys, removing objects for which
	//  the RefCount turns zero. The method ignores
	// obejcts has not found
	MulitDec(keys []cipher.SHA256) (err error)

	Amount() Amount // total objects
	Volume() Volume // total size of all objects
}

// A Feeds represetns bucket of feeds
type Feeds interface {
	// TODO
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
	// github.com/skycoin/skycoin/sr/cipher/encoder
	// but the Encode doesn't mess with reflection

	p = make([]byte, 32+8+len(cipher.SHA256{})*2+len(cipher.Sig{})+1)

	r.Object.EncodeTo(p)

	binary.LittleEndian.PutUint64(p[32:], r.Seq)

	copy(p[40:], r.Prev[:])
	copy(p[40+len(cipher.SHA256{}):], r.Hash[:])
	copy(p[40+len(cipher.SHA256{})*2:], r.Sig[:])

	if r.IsFull {
		p[len(p)-1] = 0x01 // the cipher/encoder uses 0x01 (not 0xff)
	}
	return
}

// Decode given encode Root to this one
func (r *Root) Decode(p []byte) (err error) {

	if len(p) != 41+len(cipher.SHA256{})*2+len(cipher.Sig{}) {
		return ErrInvalidSize
	}

	if err = r.Object.Decode(p[:32]); err != nil {
		return
	}

	r.Seq = binary.LittleEndian.Uint64(p[32:])

	copy(r.Prev[:], p[40:])
	copy(r.Hash[:], p[40+len(cipher.SHA256{}):])
	copy(r.Sig[:], p[40+len(cipher.SHA256{})*2:])

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
	Vol        Volume // size of the Object
	Subtree           // subtree info
	RefsCount  uint32 // refs to this Obejct (to this meta info)
	CreateTime int64  // created at, unix nano
	AccessTime int64  // last access time, unix nano
}

// Volume returns total volume
func (o *Object) Volume() (vol Volume) {
	return o.Subtree.Volume + o.Vol
}

// Amount returns total amount
func (o *Object) Amount() (amnt Amount) {
	return o.Subtree.Amount + 1
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

	p = make([]byte, 4+8+4+8+8)
	if err := o.EncodeTo(p); err != nil {
		panic(err)
	}
	return
}

// EncodeTo encodes the Object to given slice
func (o *Object) EncodeTo(p []byte) (err error) {

	if len(p) < 32 {
		return ErrInvalidSize
	}

	// Vol        4          |  0
	// Subtree    8 (4 + 4)  |  4, 8
	// RefsCount  4          | 12
	// CreateTime 8          | 16
	// AccessTime 8          | 24
	// ----------------------|-------
	//           32          |

	binary.LittleEndian.PutUint32(p, uint32(o.Vol))
	binary.LittleEndian.PutUint32(p[4:], uint32(o.Subtree.Volume))
	binary.LittleEndian.PutUint32(p[8:], uint32(o.Subtree.Amount))
	binary.LittleEndian.PutUint32(p[12:], o.RefsCount)
	binary.LittleEndian.PutUint64(p[16:], uint64(o.CreateTime))
	binary.LittleEndian.PutUint64(p[24:], uint64(o.AccessTime))
	return
}

// Decode given encoded Object to this one
func (o *Object) Decode(p []byte) (err error) {

	if len(p) != 32 {
		return ErrInvalidSize
	}

	o.Vol = Volume(binary.LittleEndian.Uint32(p))
	o.Subtree.Volume = Volume(binary.LittleEndian.Uint32(p[4:]))
	o.Subtree.Amount = Amount(binary.LittleEndian.Uint32(p[8:]))
	o.RefsCount = binary.LittleEndian.Uint32(p[12:])
	o.CreateTime = int64(binary.LittleEndian.Uint64(p[16:]))
	o.AccessTime = int64(binary.LittleEndian.Uint64(p[24:]))

	return
}

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

// An Amount represetns amount
// of all realted obejcts
type Amount uint32

// A Subtree represent brief information
// about subtree of an Object
type Subtree struct {
	Volume Volume // total volume of all elements in the Subtree
	Amount Amount // total amout fo elements of the Subtree
}