package skyobject

import (
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
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
// and increments seq number. The Touch implicitly called
// inside Inject, InjectMany and Replace methods
func (r *Root) Touch() (sig cipher.Sig, p []byte) {
	r.Lock()
	defer r.Unlock()
	return r.touch()
}

func (r *Root) touch() (sig cipher.Sig, p []byte) {
	r.mustHaveSecretKey()
	r.time = time.Now().UnixNano()
	r.seq++
	sig, p = r.encode() // to update signature
	r.cnt.addRoot(r)    // updated
	return
}

// Seq returns seq number of the Root
func (r *Root) Seq() uint64 {
	r.RLock()
	defer r.RUnlock()
	return r.seq
}

func (r *Root) SetSeq(seq uint64) {
	r.Lock()
	defer r.Unlock()
	r.seq = seq
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
// (has all related schemas and objects).
// The IsFull always retruns false for freshly
// created root objects
func (r *Root) IsFull() bool {
	r.RLock()
	defer r.RUnlock()
	if r.full {
		return true
	}
	if !r.HasRegistry() {
		return false
	}
	if r.sig == (cipher.Sig{}) { // fresh
		return false
	}
	var want int
	err := r.wantFunc(func(Reference) (_ error) {
		want++
		return
	})
	if err != nil {
		return false // can't determine
	}
	if want == 0 {
		r.full = true
		return true
	}
	return false
}

// Encode the Root and get its Signature
func (r *Root) Encode() (b []byte, sig cipher.Sig) {
	r.RLock()
	defer r.RUnlock()
	sig, b = r.encode()
	return
}

func (r *Root) encode() (sig cipher.Sig, b []byte) {
	var x encodedRoot
	x.Refs = r.refs
	x.Reg = r.reg
	x.Time = r.time
	x.Seq = r.seq
	x.Pub = r.pub
	b = encoder.Serialize(x) // b
	if r.sec != (cipher.SecKey{}) {
		// sign if need
		hash := cipher.SumSHA256(b)
		sig = cipher.SignHash(hash, r.sec) // sig
		r.sig = sig
	} else {
		// or use existing signature
		sig = r.sig
	}
	return
}

// Sign the Root. The Sign implicitly called inside
// Encode, Inject, InjectMany and Replace methods
func (r *Root) Sign() (sig cipher.Sig) {
	r.Lock()
	defer r.Unlock()
	sig, _ = r.encode()
	return
}

// HasRegistry returns false if Registry of the Root
// doesn't exist in related Container
func (r *Root) HasRegistry() bool {
	reg, _ := r.cnt.Registry(r.reg) // container never changes (no lock/unlock)
	return reg != nil
}

// Get is short hand to Get of related Container
func (r *Root) Get(ref Reference) ([]byte, bool) {
	return r.cnt.Get(ref)
}

// Save is sort hand for (*Container).Save()
func (r *Root) Save(i interface{}) Reference {
	return r.cnt.Save(i)
}

// SaveArray is sort hand for (*Container).SaveArray()
func (r *Root) SaveArray(i ...interface{}) References {
	return r.cnt.SaveArray(i...)
}

// DB returns database of related Container
func (r *Root) DB() *data.DB {
	return r.cnt.DB()
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

func (r *Root) mustHaveSecretKey() {
	if r.sec == (cipher.SecKey{}) {
		panic("unable to modify received root")
	}
}

// Inject an object to the Root updating the seq,
// timestamp and signature of the Root
func (r *Root) Inject(i interface{}) (inj Dynamic, sig cipher.Sig, p []byte) {
	inj = r.Dynamic(i)
	r.Lock()
	defer r.Unlock()
	r.refs = append(r.refs, inj)
	sig, p = r.touch()
	return
}

// InjectMany objects to the Root updating the seq,
// timestamp and signature of the Root
func (r *Root) InjectMany(i ...interface{}) (injs []Dynamic,
	sig cipher.Sig, p []byte) {

	injs = make([]Dynamic, 0, len(i))
	for _, e := range i {
		injs = append(injs, r.Dynamic(e))
	}
	r.Lock()
	defer r.Unlock()
	r.refs = append(r.refs, injs...)
	sig, p = r.touch()
	return
}

// Refs returns references of the Root
func (r *Root) Refs() []Dynamic {
	r.RLock()
	defer r.RUnlock()
	return r.refs
}

// Replace all references of the Root with given references.
// All Dynamic objects must be created by the Root (or, at
// least, by a Root that uses the same Registry). The Replace
// implicitly updates seq, timestamp and signature of the Root.
// The method returns list of previous references of the Root
func (r *Root) Replace(refs []Dynamic) (prev []Dynamic, sig cipher.Sig,
	p []byte) {

	r.Lock()
	defer r.Unlock()
	prev, r.refs = r.refs, refs
	sig, p = r.touch()
	return
}

// ValueByDynamic returns value by Dynamic reference. Returned
// value will be dereferenced. The value can be a value, nil-value with
// non-nil schema (if dr.Object is blank), or nil-value with nil-schema
// (if given dr.Object and dr.Schema are blank). Registry by which the
// dr created must be registry of the Root. One of returned errors can
// be *MissingObjectError if object the dr refers to doesn't exist
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

// ValueByStatic return value by Reference and schema name
func (r *Root) ValueByStatic(schemaName string, ref Reference) (val *Value,
	err error) {

	var s Schema
	if s, err = r.SchemaByName(schemaName); err != nil {
		return
	}
	if ref.IsBlank() {
		val = &Value{nil, s, r}
		return // nil-value with schema
	}
	if data, ok := r.Get(ref); !ok {
		err = &MissingObjectError{ref}
	} else {
		val = &Value{data, s, r}
	}
	return
}

func (r *Root) Values() (vals []*Value, err error) {
	r.RLock()
	defer r.RUnlock()
	if len(r.refs) == 0 {
		return
	}
	vals = make([]*Value, 0, len(r.refs))
	var val *Value
	for _, dr := range r.refs {
		if val, err = r.ValueByDynamic(dr); err != nil {
			vals = nil
			return // the error
		}
		vals = append(vals, val)
	}
	return
}

// SchemaByReference returns Schema by name or
// (1) missing registry error, or (2) missing schema error
func (r *Root) SchemaByName(name string) (s Schema, err error) {
	var reg *Registry
	if reg, err = r.Registry(); err != nil {
		return
	}
	s, err = reg.SchemaByName(name)
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

// A WantFunc represents function for (*Root).WantFunc method
// Impossible to manipulate Root object using the
// function because of locks
type WantFunc func(Reference) error

// WantFunc ranges over the Root calling given WantFunc
// every time an object missing in database. An error
// returned by the WantFunc stops itteration. It the error
// is ErrStopRange then WantFunc returns nil. Otherwise
// it returns the error
func (r *Root) WantFunc(wf WantFunc) (err error) {
	r.RLock()
	defer r.RUnlock()
	err = r.wantFunc(wf)
	return
}

func (r *Root) wantFunc(wf WantFunc) (err error) {
	var val *Value
	for _, dr := range r.refs {
		if val, err = r.ValueByDynamic(dr); err != nil {
			if mo, ok := err.(*MissingObjectError); ok {
				if err = wf(mo.Reference()); err != nil {
					break // range loop
				}
				continue // range loop
			} // else
			return // the error
		}
		if err = wantValue(val); err != nil {
			if mo, ok := err.(*MissingObjectError); ok {
				if err = wf(mo.Reference()); err != nil {
					break // range loop
				}
				continue // range loop
			} // else
			return // the error
		}
	}
	if err == ErrStopRange {
		err = nil
	}
	return
}

func wantValue(v *Value) (err error) {
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		err = v.RangeIndex(func(_ int, d *Value) error {
			return wantValue(d)
		})
	case reflect.Struct:
		err = v.RangeFields(func(name string, d *Value) error {
			return wantValue(d)
		})
	case reflect.Ptr:
		var d *Value
		if d, err = v.Dereference(); err != nil {
			return
		}
		err = wantValue(d)
	}
	return
}

// A GotFunc represents function for (*Root).GotFunc method
// Impossible to manipulate Root object using the
// function because of locks
type GotFunc func(Reference) error

// GotFunc ranges over the Root calling given GotFunc
// every time it has got a Reference that exists in
// database
func (r *Root) GotFunc(gf GotFunc) (err error) {
	r.RLock()
	defer r.RUnlock()
	var val *Value
	for _, dr := range r.refs {
		if val, err = r.ValueByDynamic(dr); err != nil {
			if _, ok := err.(*MissingObjectError); ok {
				continue // range loop
			} // else
			return // the error
		}
		if err = gotValue(val, gf); err != nil {
			if _, ok := err.(*MissingObjectError); ok {
				continue // range loop
			} // else
			break // range loop
		}
	}
	if err == ErrStopRange {
		err = nil
	}
	return
}

// GotOfFunc is like GotFunc but for particular
// Dynamic reference of the Root
func (r *Root) GotOfFunc(dr Dynamic, gf GotFunc) (err error) {
	r.RLock()
	defer r.RUnlock()
	var val *Value
	if val, err = r.ValueByDynamic(dr); err != nil {
		if _, ok := err.(*MissingObjectError); ok {
			err = nil // never return MissingObjectError
		} // else
		return // the error
	} else if err = gotValue(val, gf); err != nil {
		if _, ok := err.(*MissingObjectError); ok {
			err = nil // never return MissingObjectError
		}
	}
	if err == ErrStopRange {
		err = nil
	}
	return
}

func gotValue(v *Value, gf GotFunc) (err error) {
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		err = v.RangeIndex(func(_ int, d *Value) error {
			return gotValue(d, gf)
		})
	case reflect.Struct:
		err = v.RangeFields(func(name string, d *Value) error {
			return gotValue(d, gf)
		})
	case reflect.Ptr:
		var d *Value
		if d, err = v.Dereference(); err != nil {
			return
		}
		if d.IsNil() {
			return
		}
		switch v.Schema().ReferenceType() {
		case ReferenceTypeSlice:
			var ref Reference      //
			copy(ref[:], v.Data()) // v.Static()
			err = gf(ref)
		case ReferenceTypeDynamic:
			var dr Dynamic
			if dr, err = v.Dynamic(); err != nil {
				return // never happens
			}
			err = gf(dr.Object)
		}
		if err != nil {
			err = gotValue(d, gf)
		}
	}
	return
}

// encoding

type encodedRoot struct {
	Refs []Dynamic
	Reg  RegistryReference
	Time int64
	Seq  uint64
	Pub  cipher.PubKey
	// don't send secret key, because it's a secret
	// don't send signature, because signaure is signature of encoded root
}
