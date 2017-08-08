package skyobject

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
)

// A Dynamic represents dynamic reference that contains
// reference to schema and reference to object
type Dynamic struct {
	Object    cipher.SHA256 // reference to object
	SchemaRef SchemaRef     // reference to Schema

	// internals
	wn *walkNode `enc:"-"` // walkNode of this Dynamic
}

// IsBlank returns true if both SchemaRef
// and Object fields are blank
func (d *Dynamic) IsBlank() bool {
	return d.SchemaRef.IsBlank() && d.Object == (cipher.SHA256{})
}

// IsValid returns true if the Dynamic is blank, full
// or has reference to shcema only. A Dynamic is invalid if
// it has reference to object but deosn't have reference to
// shcema
func (d *Dynamic) IsValid() bool {
	if d.SchemaRef.IsBlank() {
		return d.Object == (cipher.SHA256{})
	}
	return true
}

// Short string
func (d *Dynamic) Short() string {
	return fmt.Sprintf("{%s:%s}", d.SchemaRef.Short(), d.Object.Hex()[:7])
}

// String implements fmt.Stringer interface
func (d *Dynamic) String() string {
	return "{" + d.SchemaRef.String() + ", " + d.Object.Hex() + "}"
}

// Eq returns true if given reference is the same as this one
func (d *Dynamic) Eq(x *Dynamic) bool {
	return d.SchemaRef == x.SchemaRef && d.Object == x.Object
}

// Schema of the Dynamic. It returns nil if the Dynamic is blank
func (d *Dynamic) Schema() (sch Schema, err error) {

	if d.IsBlank() {
		return // nil, nil
	}

	if !d.IsValid() {
		err = ErrInvalidDynamicReference
		return
	}

	if d.wn == nil {
		err = errors.New("can't get Schema: " +
			"the Dynamic is not attached to Pack")
		return
	}

	if d.wn.sch != nil {
		sch = d.wn.sch // already have
		return
	}

	if sch, err = r.wn.pack.reg.SchemaByReference(d.SchemaRef); err != nil {
		return
	}

	wn.sch = sch // keep
	return
}

// Value of the Dynamic. It's pointer to golang struct.
// It can be nil if the Dynamic is blank. It can be nil pointer
// of some type if the Dynamic has SchemaRef but reference to
// object is blank. For example
//
//     valueInterface, err := dr.Value()
//     if err != nil {
//         // handle the err
//     }
//     if valueInterface == nil {
//         // this case possible only if the dr is blank
//     }
//     // for example the dr represents *User
//     usr := valueInterface.(*User)
//     // work with the usr
//
// Even if you pass non-pointer (User{}) to the SetValue,
// the Value returns pointer (*User) anyway.
func (d *Dynamic) Value() (obj interface{}, err error) {

	if !d.IsValid() {
		err = ErrInvalidDynamicReference
		return
	}

	if d.IsBlank() {
		return // this Dynamic represents nil interface{}
	}

	if d.wn == nil {
		err = errors.New("can't get value: the Dynamic is not attached to Pack")
		return
	}

	if d.wn.value != nil {
		obj = wn.value // already have
		return
	}

	if _, err = d.Schema(); err != nil {
		return // the call stores schema in the walkNode (d is not blank)
	}

	if d.Object == (cipher.SHA256{}) {
		var ptr reflect.Value
		if ptr, err = d.wn.pack.newOf(d.wn.sch.Name()); err != nil {
			return
		}
		d.wn.value = ptr.Interface() // keep
		obj = d.wn.value             // nil pointer of some type
		return
	}

	// obtain object
	var val []byte
	if val, err = wn.pack.get(d.Object); err != nil {
		return
	}

	if obj, err = wn.pack.unpackToGo(wn.sch.Name(), val); err != nil {
		return
	}

	wn.value = obj // keep

	// TODO (kostyarin): track changes

	return
}

// SetValue replacing the Dynamic with new, that points
// to given value. Pass nil to make the Dynamic blank.
// If related pack created with ViewOnly flag, then this
// method returns error
func (d *Dynamic) SetValue(obj interface{}) (err error) {

	if obj == nil {
		d.Clear()
		return
	}

	if d.wn == nil {
		return errors.New(
			"can't set value: the Dynamic is not attached to Pack")
	}

	if d.wn.pack.flags&ViewOnly != 0 {
		return ErrViewOnlyTree
	}

	var sch Schema
	if sch, obj, err = r.wn.pack.initialize(obj); err != nil {
		return fmt.Errorf("can't set value to Dynamic: %v", err)
	}

	d.wn.value = obj // keep

	var changed bool

	if sr := sch.Reference(); sr != d.SchemaRef {
		d.SchemaRef, changed = sr, true
	}

	if obj == nil {
		if d.Object != (cipher.SHA256{}) {
			d.Object, changed = cipher.SHA256{}, true
		}
	} else if key, val := d.wn.pack.dsave(obj); key != d.Object {
		d.wn.pack.set(key, val) // save
		d.Object, changed = key, true
	}

	if changed {
		d.wn.unsave()
	}

	// TODO (kostyarin): track changes

	return
}

// Clear the Dynamic making it blank. It clears
// reference to object and refernece to Schema too
func (d *Dynamic) Clear() {
	if d.SchemaRef == (SchemaRef{}) && d.Object == (cipher.SHA256{}) {
		return // already blank
	}
	d.SchemaRef, d.Object = SchemaRef{}, cipher.SHA256{}
	if d.wn != nil {
		d.wn.value = nil // clear value
		d.wn.sch = nil   // clear schema
		wn.unsave()
	}

	// TODO (kostyarin): track changes
}

// Copy returns copy of the Dynamic
// Underlying value does not copied.
// But schema does
func (d *Dynamic) Copy() (cp Dynamic) {
	cp.SchemaRef = d.SchemaRef
	cp.Object = d.Object
	if d.wn != nil {
		cp.wn = &walkNode{
			pack: wn.pack,
			sch:  wn.sch,
		}
	}
	return
}
