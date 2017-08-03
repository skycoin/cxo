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
	walkNode *walkNode `enc:"-"` // WalkNode of this Dynamic
}

// IsBlank returns true if the reference is blank
func (d *Dynamic) IsBlank() bool {
	return d.SchemaRef.IsBlank() && d.Object == (cipher.SHA256{})
}

// IsValid returns true if the Dynamic is blank, full
// or hash reference to shcema only. A Dynamic is invalid if
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
	wn := d.walkNode
	if wn == nil {
		err = errors.New("can't get Schema of detached Dynamic")
		return
	}
	if wn.sch == nil { // obtain schema first
		if sch, err = wn.pack.reg.SchemaByReference(d.SchemaRef); err != nil {
			return
		}
	}
	wn.sch = sch
	return
}

// Value of the Dynamic. It's pointer to golang struct.
// It can be nil if the dynamic is blank or references to
// object is blank
func (d *Dynamic) Value() (obj interface{}, err error) {
	if !d.IsValid() {
		err = fmt.Errorf("invalid Dynamic %s", d.Short())
		return
	}
	if d.Object == (cipher.SHA256{}) {
		return // this Dynamic represents nil
	}
	wn := d.walkNode
	if wn == nil {
		err = errors.New("can't get value of detached Dynamic")
		return
	}
	if wn.value != nil { // already have
		obj = wn.value
		return
	}
	// schema
	if _, err = d.Schema(); err != nil {
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
	return
}

// SetValue of the Dynamic. Pass nil to make the Dynamic
// blank. The obj must be pointer to registered golang
// struct
func (d *Dynamic) SetValue(obj interface{}) (err error) {
	if obj == nil {
		d.clear()
		return
	}
	wn := d.walkNode
	if wn == nil {
		err = errors.New("can't set value to detached Dynamic")
		return
	}
	typ := typeOf(obj)
	var sch Schema
	if name, ok := wn.pack.types.Inverse[typ]; !ok {
		// detailed error
		err = fmt.Errorf(`can't set Dynamic value:
    given object not found in Types map
    reflect.Type of the object: %s`,
			typ.String())
		return
	} else if sch, err = wn.pack.reg.SchemaByName(name); err != nil {
		// dtailed error
		err = fmt.Errorf(`wrong Types of Pack:
    schema name found in Types, but schema by the name not found in Registry
    error:                      %s
    registry reference:         %s
    schema name:                %s
    reflect.Type of the obejct: %s`,
			err,
			wn.pack.reg.Reference().Short(),
			name,
			typ.String())
		return
	}
	// else everything is ok
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		// obj represents nil of typ
		if val.IsNil() {
			prevs := d.SchemaRef
			wn.sch = sch
			wn.value = nil
			d.SchemaRef = sch.Reference()
			d.Object = cipher.SHA256{}
			// clear, but don't celar schema reference
			if prevs != sch.Reference() || d.Object != (cipher.SHA256{}) {
				wn.unsave()
			}
			return
		}
	} else { // not a pointer
		err = errors.New("can't accept non-pointer value")
		return
	}
	// setup references of given obj
	if err = wn.pack.setupToGo(val); err != nil {
		return
	}
	// save the object getting hash back
	key, _ := wn.pack.save(obj)
	// setup
	prevs, prevo := d.SchemaRef, d.Object
	wn.sch = sch
	wn.value = obj
	d.SchemaRef = sch.Reference()
	d.Object = key
	if d.Object != prevo || d.SchemaRef != prevs {
		wn.unsave()
	}
	return
}

func (d *Dynamic) clear() {
	prevs, prevo := d.SchemaRef, d.Object
	d.SchemaRef = SchemaRef{}
	d.Object = cipher.SHA256{}
	if wn := d.walkNode; wn != nil {
		wn.value = nil
		wn.sch = nil
		if d.Object != prevo || d.SchemaRef != prevs {
			wn.unsave()
		}
	}
}

// Copy returns copy of the Dynamic
// Underlying value not copied
func (d *Dynamic) Copy() (cp Dynamic) {
	cp.SchemaRef = d.SchemaRef
	cp.Object = d.Object
	if wn := d.walkNode; wn != nil {
		cp.walkNode = &walkNode{
			pack: wn.pack,
			sch:  wn.sch,
		}
	}
	return
}
