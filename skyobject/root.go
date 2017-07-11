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
	// ErrStopRange is special service error used
	// to stop WantFunc, etc itteratings
	ErrStopRange = errors.New("stop range")
	// ErrMissingSecretKey occurs when you want to
	// modify a root that hassn't been created by
	// NewRoot or NewRootReg methods of a Container
	ErrMissingSecretKey = errors.New("missing secret key")
)

// A RootTemplate used to create new Root object
type RootTemplate struct {
	Reg  RegistryReference // registry of schemas of this Root
	Refs []Dynamic         // branches
	Pub  cipher.PubKey     // feed

	// prepare to save
	objects map[cipher.SHA256][]byte
}

// A Root represents root object of a feed
type Root struct {
	RootTemplate // user provided fields

	Time time.Time // timestamp or time.Time{}
	Seq  uint64    // seq number

	Sig cipher.Sig // signature

	Hash RootReference // hash of this Root
	Prev RootReference // hash of previous Root

	// service fields

	full bool       // is this Root full
	cnt  *Container // back reference (ro)
	rsh  *Registry  // short hand for registry
}

// Registry returns related Registry. It can returns nil
func (r *Root) Registry() (reg *Registry) {
	if r.rsh != nil {
		reg = r.rsh // use short hand instead of accesing map
		return
	}
	if reg, _ = r.cnt.Registry(r.reg); reg != nil {
		r.rsh = reg // keep short hand
	}
	return
}

// IsFull reports true if the Root is full
// (has all related schemas and objects)
func (r *Root) IsFull() bool {
	if r.full {
		return true
	}
	if r.Registry() == nil {
		return false
	}
	want := false
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

// Dynamic creates Dynamic object using Registry related to
// the Root. If Registry doesn't exists or type has not been
// registered then the method returns error
func (r *Root) Dynamic(schemaName string, i interface{}) (dr Dynamic,
	err error) {

	var reg *Registry
	if reg = r.Registry(); reg == nil {
		err = &MissingRegistryError{r.Reg}
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

// MustDynamic panics if registry missing or schema does not registered
func (r *Root) MustDynamic(schemaName string, i interface{}) (dr Dynamic) {
	var err error
	if dr, err = r.Dynamic(schemaName, i); err != nil {
		panic(err)
	}
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

// Values returns root vlaues of the root. It can returns
// errors if related Registry, Schemas or Objects are misisng
func (r *Root) Values() (vals []*Value, err error) {
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

// SchemaByName returns Schema by name or
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
type WantFunc func(Reference) error

// WantFunc ranges over the Root calling given WantFunc
// every time an object missing in database. An error
// returned by the WantFunc stops itteration. It the error
// is ErrStopRange then WantFunc returns nil. Otherwise
// it returns the error
func (r *Root) WantFunc(wf WantFunc) (err error) {
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
	if v.IsNil() {
		return
	}
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
	var val *Value
	for _, dr := range r.refs {
		if val, err = r.ValueByDynamic(dr); err != nil {
			if _, ok := err.(*MissingObjectError); ok {
				err = nil
				continue // range loop
			} // else
			return // the error
		}
		// if we got ValueByDynamic then we have got
		// the object
		if err = gf(dr.Object); err != nil {
			break // range loop
		}
		// go deepper
		if err = gotValue(val, gf); err != nil {
			if _, ok := err.(*MissingObjectError); ok {
				err = nil
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

// RefsFunc used to determine references used by a Root.
// It returns skip to skip a brach by reference
type RefsFunc func(Reference) (skip bool, err error)

// RefsFunc used to determine possible references
// of a Root. If the Root is full, then given
// function will be called for every object
// (ecxept skipped). Given function can be
// called for objects that is not present
// in database yet
func (r *Root) RefsFunc(rf RefsFunc) (err error) {
	var val *Value
	var skip bool
	for _, dr := range r.refs {
		if dr.Object.IsBlank() {
			continue
		}
		if skip, err = rf(dr.Object); err != nil {
			break // range loop
		}
		if skip {
			continue // next value
		}
		if val, err = r.ValueByDynamic(dr); err != nil {
			if _, ok := err.(*MissingObjectError); ok {
				err = nil
				continue // range loop
			} // else
			return // the error
		}
		// go deepper
		if err = refsValue(val, rf); err != nil {
			if _, ok := err.(*MissingObjectError); ok {
				err = nil
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

func refsValue(v *Value, rf RefsFunc) (err error) {
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		err = v.RangeIndex(func(_ int, d *Value) error {
			return refsValue(d, rf)
		})
	case reflect.Struct:
		err = v.RangeFields(func(name string, d *Value) error {
			return refsValue(d, rf)
		})
	case reflect.Ptr:
		switch v.Schema().ReferenceType() {
		case ReferenceTypeSingle:
			var ref Reference
			if ref, err = v.Static(); err != nil {
				return
			}
			if ref.IsBlank() {
				return // do nothing for empty references
			}
			var skip bool
			if skip, err = rf(ref); err != nil || skip {
				return
			}
			if data, ok := v.root.Get(ref); ok {
				var val *Value
				val = &Value{data, v.Schema().Elem(), v.root}
				return refsValue(val, rf)
			}
			return // nil
		case ReferenceTypeDynamic:
			var dr Dynamic
			if dr, err = v.Dynamic(); err != nil {
				return
			}
			if dr.Object.IsBlank() {
				return // do nothing for empty references
			}
			var skip bool
			if skip, err = rf(dr.Object); err != nil || skip {
				return
			}
			var val *Value
			if val, err = v.root.ValueByDynamic(dr); err != nil {
				return
			}
			return refsValue(val, rf)
		default:
			err = ErrInvalidType
		}
	}
	return
}
