package skyobject

import (
	"errors"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

var (
	ErrStopRange = errors.New("stop range")
)

// A Root represents ...
type Root struct {
	sync.RWMutex

	refs []Dynamic         // list of objects (rw)
	reg  RegistryReference // reference to registry of the root (ro)
	time int64             // timestamp (rw)
	seq  uint64            // seq number (rw)

	pub cipher.PubKey // public key (ro)
	sec cipher.SecKey // secret key (ro)

	sig cipher.Sig // signature (rw)

	full bool       // is the root full (rw)
	cnt  *Container // back reference (ro)
}

// Registry returns related registry of "missing registry" error
func (r *Root) Registry() (reg *Registry, err error) {
	reg, err = r.cnt.Registry(r.reg) // container never changed (no lock/unlock)
	return
}

func (r *Root) RegistryReference() RegistryReference {
	return r.reg
}

// Tocuh updates timestapt of the Root (setting it to now)
// and increments seq number. Touch implicitly called inside
// Inject method
func (r *Root) Touch() {
	r.Lock()
	defer r.Unlock()
	r.touch()
}

func (r *Root) touch() {
	r.time = time.Now().UnixNano()
	r.seq++
}

// Seq returns seq number of the Root
func (r *Root) Seq() uint64 {
	r.RLock()
	defer r.RUnlock()
	return r.seq
}

// Time returns unix nano timestamp of the Root
func (r *Root) Time() int64 {
	r.RLock()
	defer r.RUnlock()
	return r.time
}

// Pub returns puclic key (feed) of the Root
func (r *Root) Pub() cipher.PubKey {
	return r.pub // unable to change
}

// Sig returns signature of the Root
func (r *Root) Sig() cipher.Sig {
	r.RLock()
	defer r.RUnlock()
	return r.sig
}

// IsFull reports true if the Root is full
// (has all related schemas and objects)
func (r *Root) IsFull() bool {
	r.RLock()
	defer r.RUnlock()
	return r.full
}

// Encode the Root and get its Signature
func (r *Root) Encode() (sig cipher.Sig, b []byte) {
	r.RLock()
	defer r.RUnlock()
	sig, b = r.encode()
	return
}

func (r *Root) encode() (sig cipher.Sig, b []byte) {
	var x rootEncoding
	x.Refs = r.refs
	x.Reg = r.reg
	x.Time = r.time
	x.Seq = r.seq
	x.Pub = r.pub
	b = encoder.Serialize(x) // b
	hash := cipher.SumSHA256(b)
	sig = cipher.SignHash(hash, r.sec) // sig
	r.sig = sig
	return
}

// Sign the Root. Sign implicitly called inside Encode and Inject methods
func (r *Root) Sign() (sig cipher.Sig) {
	r.Lock()
	defer r.Unlock()
	sig, _ = r.encode()
	return
}

func (r *Root) HasRegistry() bool {
	reg, _ := r.cnt.Registry(r.reg) // container never changes (no lock/unlock)
	return reg != nil
}

// Get is short hand to Get of related Container
func (r *Root) Get(ref Reference) ([]byte, bool) {
	return r.cnt.Get(ref)
}

func (r *Root) Save(i interface{}) Reference {
	return r.cnt.Save(i)
}

func (r *Root) SaveArray(i ...interface{}) References {
	return r.cnt.SaveArray(i...)
}

// Dynamic created Dynamic objct using Registry related to
// the Root. If Regsitry doesn't exists or type was not
// registered then the method panics
func (r *Root) Dynamic(i interface{}) (dr Dynamic) {
	reg, err := r.Registry()
	if err != nil {
		panic(err)
	}
	s, err := reg.SchemaByInterface(i)
	if err != nil {
		panic(err)
	}
	dr.Schema = s.Reference()
	dr.Object = r.cnt.Save(i)
	return
}

func (r *Root) Inject(i interface{}) (inj Dynamic) {
	inj = r.Dynamic(i)
	r.Lock()
	defer r.Unlock()
	r.refs = append(r.refs, inj)
	r.touch()
	r.encode() // to update signature
	return
}

// ValueByDynamic returns value by Dynamic reference. Returned
// value will be dereferenced. The value can be a value, nil-value with
// non-nil schema (if dr.Object is blank), or nil-value with nil-schema
// (if given dr.Object and dr.Schema are blank). Registry by which the
// dr created must be registry of the Root. One of returned errors can
// be *MissingObjectError if object the dr refers to doesn't exists
// in database
func (r *Root) ValueByDynamic(dr Dynamic) (val *Value, err error) {
	if !dr.IsValid() {
		err = ErrInvalidDynamicReference
		return
	}
	if dr.IsBlank() {
		val = &Value{nil, nilSchema, r}
		return // nil-value with nil-schema
	}
	var el Schema
	if el, err = r.SchemaByReference(dr.Schema); err != nil {
		return
	}
	if dr.Object.IsBlank() {
		val = &Value{nil, el, r} // nil value with non-nil schema
		return
	}
	if data, ok := r.Get(dr.Object); !ok {
		err = &MissingObjectError{dr.Object}
	} else {
		val = &Value{data, el, r}
	}
	return
}

// SchemaByReference returns Schema by reference or
// (1) missing registry error, or (2) missing schema error
func (r *Root) SchemaByReference(sr SchemaReference) (s Schema, err error) {
	var reg *Registry
	if reg, err = r.Registry(); err != nil {
		return
	}
	s, err = reg.SchemaByReference(sr)
	return
}

type WantFunc func(Reference) error

func (r *Root) WantFunc(wf WantFunc) (err error) {
	r.RLock()
	defer r.RUnlock()
	// TODO
	return
}

type GotFunc func(Reference) error

func (r *Root) GotFunc(gf GotFunc) (err error) {
	r.RLock()
	defer r.RUnlock()
	// TODO
	return
}

// internal
func (r *Root) setFull() {
	r.Lock()
	defer r.Unlock()
	r.full = true
}

// encoding

type rootEncoding struct {
	Refs []Dynamic
	Reg  RegistryReference
	Time int64
	Seq  uint64
	Pub  cipher.PubKey
	// don't send secret key, because it's a secret
	// don't send signature, because signaure is
	// signature of encoded root
}
