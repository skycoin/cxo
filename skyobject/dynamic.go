package skyobject

import (
	"errors"
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"
)

// A Dynamic represents dynamic reference that contains
// reference to schema and reference to object. Use
// SetSchemaRef and SetHash to chagne fields.
type Dynamic struct {
	Object    cipher.SHA256 // reference to object
	SchemaRef SchemaRef     // reference to Schema

	// internals
	dn *drNode `enc:"-"` // drNode of this Dynamic
	ch bool    `enc:"-"` // has been changed
}

func (d *Dynamic) isInitialized() bool {
	return d.dn != nil
}

type drNode struct {
	value interface{}
	sch   Schema
	pack  *Pack
}

// SetSchemaRef replaces SchemaRef field with given SchemaRef
func (d *Dynamic) SetSchemaRef(sr SchemaRef) {
	if d.SchemaRef == sr {
		return
	}
	d.SchemaRef = sr
	d.ch = true
	if d.dn != nil {
		d.dn.sch = nil // clear
	}
	return
}

// SetHash sets Object field of the Dynamic to given hash
func (d *Dynamic) SetHash(hash cipher.SHA256) {
	if d.Object == hash {
		return
	}
	d.Object = hash
	d.ch = true
	if d.dn != nil {
		d.dn.value = nil // clear
	}
	return
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

	if d.dn == nil {
		err = errors.New("can't get Schema: " +
			"the Dynamic is not attached to Pack")
		return
	}

	if d.dn.sch != nil {
		sch = d.dn.sch // already have
		return
	}

	if sch, err = d.dn.pack.reg.SchemaByReference(d.SchemaRef); err != nil {
		return
	}

	d.dn.sch = sch // keep
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
// the Value returns pointer (*User) anyway
func (d *Dynamic) Value() (obj interface{}, err error) {

	if !d.IsValid() {
		err = ErrInvalidDynamicReference
		return
	}

	if d.dn == nil {
		err = errors.New("can't get value: the Dynamic is not attached to Pack")
		return
	}

	if d.IsBlank() {
		return // this Dynamic represents nil interface{}
	}

	if d.dn.value != nil {
		obj = d.dn.value // already have
		return
	}

	if _, err = d.Schema(); err != nil {
		return // the call stores schema in the walkNode (d is not blank)
	}

	if d.Object == (cipher.SHA256{}) {
		// nil pointer of some type
		if obj, err = d.dn.pack.nilPtrOf(d.dn.sch.Name()); err != nil {
			return
		}
		d.dn.value = obj // keep
		return
	}

	// obtain object
	var val []byte
	if val, err = d.dn.pack.get(d.Object); err != nil {
		return
	}

	if obj, err = d.dn.pack.unpackToGo(d.dn.sch.Name(), val); err != nil {
		return
	}

	d.dn.value = obj // keep

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

	if d.dn == nil {
		return errors.New(
			"can't set value: the Dynamic is not attached to Pack")
	}

	if d.dn.pack.flags&ViewOnly != 0 {
		return ErrViewOnlyTree
	}

	var sch Schema
	if sch, obj, err = d.dn.pack.initialize(obj); err != nil {
		return fmt.Errorf("can't set value to Dynamic: %v", err)
	}

	d.dn.value = obj // keep

	if sr := sch.Reference(); sr != d.SchemaRef {
		d.SchemaRef = sr
		d.dn.sch = sch
		d.ch = true // changed
	}

	if obj == nil {
		if d.Object != (cipher.SHA256{}) {
			d.Object = cipher.SHA256{}
			d.ch = true // changed
		}
	} else if key, val := d.dn.pack.dsave(obj); key != d.Object {
		d.dn.pack.set(key, val) // save
		d.Object = key
		d.ch = true // changed
	}

	return
}

// Clear the Dynamic making it blank. It clears
// reference to object and reference to Schema too
func (d *Dynamic) Clear() (err error) {
	if d.SchemaRef == (SchemaRef{}) && d.Object == (cipher.SHA256{}) {
		return // already blank
	}
	d.SchemaRef, d.Object = SchemaRef{}, cipher.SHA256{}
	d.ch = true
	if d.dn != nil {
		d.dn.value = nil // clear value
		d.dn.sch = nil   // clear schema
	}
	return
}

// Copy returns copy of the Dynamic
// Underlying value does not copied.
// But schema does
func (d *Dynamic) Copy() (cp Dynamic) {
	cp.SchemaRef = d.SchemaRef
	cp.Object = d.Object
	cp.ch = true
	if d.dn != nil {
		cp.dn = &drNode{
			pack: d.dn.pack,
			sch:  d.dn.sch,
		}
	}
	return
}

func (d *Dynamic) Save() {
	if d.dn == nil {
		return
	}
	if d.dn.value == nil {
		return
	}

	key, val := d.dn.pack.dsave(d.dn.value)
	if key != d.Object {
		d.dn.pack.set(key, val) // save
		d.Object = key
		d.ch = true
	}

}
