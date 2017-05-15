package skyobject

import (
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

var (
	ErrStopRange        = errors.New("stop range")
	ErrMissingSecretKey = errors.New("missing secret key")
)

// A Root represents ...
type Root struct {
	sync.RWMutex

	refs []Dynamic         // list of objects (rw)
	reg  RegistryReference // reference to registry of the root (ro)
	time int64             // timestamp (rw)
	seq  uint64            // seq number (rw) atomic

	pub cipher.PubKey // public key (ro)
	sec cipher.SecKey // secret key (ro)

	sig cipher.Sig // signature (rw)

	full bool       // is the root full (rw)
	cnt  *Container // back reference (ro)

	rsh *Registry // short hand for registry
}

func (r *Root) dup() (d *Root) {
	d = new(Root)
	if len(r.refs) > 0 {
		d.refs = make([]Dynamic, len(r.refs))
		copy(d.refs, r.refs)
	}
	d.reg = r.reg
	d.time = r.time
	d.seq = r.Seq()
	d.pub = r.pub
	d.sec = r.sec
	d.sig = r.sig
	d.full = r.full
	d.cnt = r.cnt
	d.rsh = r.rsh
	return
}

// Registry returns related registry of "missing registry" error
func (r *Root) Registry() (reg *Registry, err error) {
	r.RLock()
	defer r.RUnlock()
	reg, err = r.registry()
	return
}

func (r *Root) registry() (reg *Registry, err error) {
	if r.rsh != nil {
		reg = r.rsh // use short hand instead of accesing map
		return
	}
	if reg, err = r.cnt.Registry(r.reg); err == nil {
		r.rsh = reg
	}
	return
}

func (r *Root) RegistryReference() RegistryReference {
	return r.reg
}

// Touch updates timestamp of the Root (setting it to now)
// and increments seq number. The Touch implicitly called
// inside Inject, InjectMany and Replace methods
func (r *Root) Touch() (RootPack, error) {
	r.Lock()
	defer r.Unlock()
	return r.touch()
}

func (r *Root) touch() (rp RootPack, err error) {
	if r.sec == (cipher.SecKey{}) {
		err = ErrMissingSecretKey
		return
	}
	r.time = time.Now().UnixNano()
	atomic.AddUint64(&r.seq, 1)
	rp = r.encode()        // to update signature
	err = r.cnt.addRoot(r) // updated
	return
}

// Seq returns seq number of the Root
func (r *Root) Seq() uint64 {
	return atomic.LoadUint64(&r.seq)
}

func (r *Root) SetSeq(seq uint64) {
	atomic.StoreUint64(&r.seq, seq)
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
	var want bool = false
	err := r.wantFunc(func(Reference) error {
		want = true
		return ErrStopRange
	})
	if err != nil {
		return false // can't determine
	}
	if want {
		return false
	}
	r.full = true
	return true
}

// Encode the Root and get its Signature
func (r *Root) Encode() (rp RootPack) {
	r.RLock()
	defer r.RUnlock()
	rp = r.encode()
	return
}

// A RootPack represents encoded root object with signature
type RootPack struct {
	Root []byte
	Sig  cipher.Sig
}

func (r *Root) encode() (rp RootPack) {
	var x encodedRoot
	x.Refs = r.refs
	x.Reg = r.reg
	x.Time = r.time
	x.Seq = r.Seq()
	x.Pub = r.pub
	rp.Root = encoder.Serialize(x) // b
	if r.sec != (cipher.SecKey{}) {
		// sign if need
		hash := cipher.SumSHA256(rp.Root)
		r.sig = cipher.SignHash(hash, r.sec) // sig
	}
	rp.Sig = r.sig
	return
}

// Sign the Root. The Sign implicitly called inside
// Encode, Inject, InjectMany and Replace methods
func (r *Root) Sign() (sig cipher.Sig) {
	r.Lock()
	defer r.Unlock()
	sig = r.encode().Sig
	return
}

// HasRegistry returns false if Registry of the Root
// doesn't exist in related Container
func (r *Root) HasRegistry() bool {
	reg, _ := r.Registry()
	return reg != nil
}

// Get is short hand to Get of related Container
func (r *Root) Get(ref Reference) ([]byte, bool) {
	return r.cnt.Get(ref)
}

// DB returns database of related Container
func (r *Root) DB() *data.DB {
	return r.cnt.DB()
}

// Save is short hand for (*Container).Save()
func (r *Root) Save(i interface{}) Reference {
	return r.cnt.save(i)
}

// SaveArray is short hand for (*Container).SaveArray()
func (r *Root) SaveArray(i ...interface{}) References {
	return r.cnt.saveArray(i...)
}

// Dynamic creates Dynamic object using Registry related to
// the Root. If Registry doesn't exists or type was not
// registered then the method returns error
func (r *Root) Dynamic(schemaName string, i interface{}) (dr Dynamic,
	err error) {

	var reg *Registry
	if reg, err = r.Registry(); err != nil {
		return
	}
	var s Schema
	if s, err = reg.SchemaByName(schemaName); err != nil {
		return
	}
	dr.Schema = s.Reference()
	dr.Object = r.Save(i)
	return
}

// MustDynamic panics if registry missing or schema not registered
func (r *Root) MustDynamic(schemaName string, i interface{}) (dr Dynamic) {
	var err error
	if dr, err = r.Dynamic(schemaName, i); err != nil {
		panic(err)
	}
	return
}

// Inject an object to the Root updating the seq,
// timestamp and signature of the Root
func (r *Root) Inject(schemaName string, i interface{}) (inj Dynamic,
	rp RootPack, err error) {

	if inj, err = r.Dynamic(schemaName, i); err != nil {
		return
	}
	r.Lock()
	defer r.Unlock()
	r.refs = append(r.refs, inj)
	rp, err = r.touch()
	return
}

// InjectMany objects to the Root updating the seq,
// timestamp and signature of the Root
func (r *Root) InjectMany(schemaName string, i ...interface{}) (injs []Dynamic,
	rp RootPack, err error) {

	injs = make([]Dynamic, 0, len(i))
	var inj Dynamic
	for _, e := range i {
		if inj, err = r.Dynamic(schemaName, e); err != nil {
			injs = nil
			return
		}
		injs = append(injs, inj)
	}
	r.Lock()
	defer r.Unlock()
	r.refs = append(r.refs, injs...)
	rp, err = r.touch()
	return
}

// Refs returns references of the Root
func (r *Root) Refs() (refs []Dynamic) {
	r.RLock()
	defer r.RUnlock()
	if len(r.refs) > 0 {
		refs = make([]Dynamic, len(r.refs))
		copy(refs, r.refs)
	}
	return
}

// Replace all references of the Root with given references.
// All Dynamic objects must be created by the Root (or, at
// least, by a Root that uses the same Registry). The Replace
// implicitly updates seq, timestamp and signature of the Root.
// The method returns list of previous references of the Root
func (r *Root) Replace(refs []Dynamic) (prev []Dynamic, rp RootPack,
	err error) {

	r.Lock()
	defer r.Unlock()
	prev, r.refs = r.refs, refs
	rp, err = r.touch()
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

// SchemaReferenceByName returns re
func (r *Root) SchemaReferenceByName(name string) (sr SchemaReference,
	err error) {

	var reg *Registry
	if reg, err = r.Registry(); err != nil {
		return
	}
	sr, err = reg.SchemaReferenceByName(name)
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
	if r.full {
		return // the Root is full
	}
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
