package skyobject

import (
	"errors"
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"
)

// A Dynamic represents dynamic reference that contains
// reference to schema and reference to object. You
// should not change fields of the Dynamic manually
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

	if sch, err = d.wn.pack.reg.SchemaByReference(d.SchemaRef); err != nil {
		return
	}

	d.wn.sch = sch // keep
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

	if d.wn == nil {
		err = errors.New("can't get value: the Dynamic is not attached to Pack")
		return
	}

	if d.IsBlank() {
		d.trackChanges() // if AutoTrackChangses flag set
		return           // this Dynamic represents nil interface{}
	}

	if d.wn.value != nil {
		obj = d.wn.value // already have
		return
	}

	if _, err = d.Schema(); err != nil {
		return // the call stores schema in the walkNode (d is not blank)
	}

	if d.Object == (cipher.SHA256{}) {
		// nil pointer of some type
		if obj, err = d.wn.pack.nilPtrOf(d.wn.sch.Name()); err != nil {
			return
		}
		d.wn.value = obj // keep
		d.trackChanges() // if AutoTrackChanges flag set
		return
	}

	// obtain object
	var val []byte
	if val, err = d.wn.pack.get(d.Object); err != nil {
		return
	}

	if obj, err = d.wn.pack.unpackToGo(d.wn.sch.Name(), val); err != nil {
		return
	}

	d.wn.value = obj // keep

	d.trackChanges() // if AutoTrackChanges flag set
	return
}

func (d *Dynamic) trackChanges() {
	if f := d.wn.pack.flags; f&AutoTrackChanges != 0 && f&ViewOnly == 0 {
		d.wn.pack.Push(d) // track
	}
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
	if sch, obj, err = d.wn.pack.initialize(obj); err != nil {
		return fmt.Errorf("can't set value to Dynamic: %v", err)
	}

	d.wn.value = obj // keep

	var changed bool

	if sr := sch.Reference(); sr != d.SchemaRef {
		d.SchemaRef, d.wn.sch, changed = sr, sch, true
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
		if err = d.wn.unsave(); err != nil {
			return
		}
	}

	d.trackChanges()
	return
}

// Clear the Dynamic making it blank. It clears
// reference to object and refernece to Schema too
func (d *Dynamic) Clear() (err error) {
	if d.SchemaRef == (SchemaRef{}) && d.Object == (cipher.SHA256{}) {
		return // already blank
	}
	d.SchemaRef, d.Object = SchemaRef{}, cipher.SHA256{}
	if d.wn != nil {
		d.wn.value = nil // clear value
		d.wn.sch = nil   // clear schema
		if err = d.wn.unsave(); err != nil {
			return
		}
	}
	return
}

// Copy returns copy of the Dynamic
// Underlying value does not copied.
// But schema does
func (d *Dynamic) Copy() (cp Dynamic) {
	cp.SchemaRef = d.SchemaRef
	cp.Object = d.Object
	if d.wn != nil {
		cp.wn = &walkNode{
			pack: d.wn.pack,
			sch:  d.wn.sch,
		}
	}
	return
}

func (d *Dynamic) commit() (err error) {

	if d.wn == nil {
		panic("commit not initialized Ref")
	}

	var changed bool
	var obj interface{} = d.wn.value

	if obj == nil {
		if d.Object != (cipher.SHA256{}) {
			d.Object, changed = cipher.SHA256{}, true
		}
	} else if key, val := d.wn.pack.dsave(obj); key != d.Object {
		d.wn.pack.set(key, val) // save
		d.Object, changed = key, true
	}

	if changed {
		err = d.wn.unsave()
	}
	return
}
