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
	Object    cipher.SHA256   // reference to object
	SchemaRef SchemaReference // reference to Schema

	// internals

	walkNode *walkNode `enc:"-"` // WalkNode of this Dynamic
}

// IsBlank returns true if the reference is blank
func (d *Dynamic) IsBlank() bool {
	return d.SchemaRef.IsBlank() && d.Object.IsBlank()
}

// IsValid returns true if the Dynamic is blank, full
// or hash reference to shcema only. A Dynamic is invalid if
// it has reference to object but deosn't have reference to
// shcema
func (d *Dynamic) IsValid() bool {
	if d.SchemaRef.IsBlank() {
		return d.Object.IsBlank()
	}
	return true
}

// Short string
func (d *Dynamic) Short() string {
	return fmt.Sprintf("{%s:%s}", d.SchemaRef.Short(), d.Object.Hex()[:7])
}

// String implements fmt.Stringer interface
func (d *Dynamic) String() string {
	return "{" + d.SchemaRef.String() + ", " + d.Object.String() + "}"
}

// Eq returns true if given reference is the same as this one
func (d *Dynamic) Eq(x *Dynamic) bool {
	return d.SchemaRef == x.SchemaRef && d.Object.Eq(&x.Object)
}

// Schema of the Dynamic. It returns nil if the Dynamic is blank
func (d *Dynamic) Schema() (sch Schema, err error) {

	wn := d.walkNode

	if wn == nil {
		// TODO (kostyarin): make the error global
		err = errors.New(
			"can't get Schema of Dynamic reference detached from Pack")
		return
	}

	if wn.sch == nil {
		// obtain schema first
		wn.sch, err = wn.pack.reg.SchemaByReference(d.SchemaRef)
		if err != nil {
			return
		}
	}
	sch = wn.sch
	return
}

// Value of the Dynamic. It's golang value if related Pack
// created with GoTypes flag. Other cases are not implemented
// yet
func (d *Dynamic) Value() (obj interface{}, err error) {

	if !d.IsValid() {
		err = fmt.Errorf("invalid Dynamic %s", d.Short())
		return
	}

	if d.IsBlank() {
		return // nil, nil
	}

	if d.Object == (cipher.SHA256{}) {
		return // this Dynamic represents nil
	}

	wn := d.walkNode

	if wn == nil {
		// TODO (kostyarin): make the error global
		err = errors.New(
			"can't get value of Dynamic reference detached from Pack")
		return
	}

	if wn.value != nil {
		// already have
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

	if wn.pack.flags&GoTypes != 0 {

		if obj, err = wn.pack.unpackToGo(wn.sch.Name(), val); err != nil {
			return
		}

	} else {

		// TODO (kostyarin): other unpacking methods

		panic("not implemented yet")
	}

	wn.value = obj // keep

	return
}

// SetValue of the Dynamic. You can pass another
// Dynamic reference, golang value. Other cases are
// not implemented yet. Pass nil to make the Dynamic
// blank
func (d *Dynamic) SetValue(obj interface{}) (err error) {

	if obj == nil {
		d.clear()
		return
	}

	wn := d.walkNode

	if wn == nil {
		// TODO (kostyarin): make the error global
		err = errors.New(
			"can't set value to Dynamic reference detached from Pack")
		return
	}

	// is the obj Dynamic or *Dynamic
	var odr *Dynamic
	if x, ok := obj.(Dynamic); ok {
		odr = &x
	} else if x, ok := obj.(*Dynamic); ok {
		odr = x
	}

	if odr != nil {
		return d.setDynamic(odr)
	}

	if wn.pack.flags&GoTypes != 0 {

		typ := typeOf(obj)

		var sch Schema

		if name, ok := p.types.Inverse[typ]; !ok {
			// detailed error
			err = fmt.Errorf(`can't set Dynamic value:
    given object not found in Types map
    reflect.Type of the object: %s`,
				typ.String())
			return
		} else if sch, err = p.reg.SchemaByName(name); err != nil {
			// dtailed error
			err = fmt.Errorf(`wrong Types of Pack:
    schema name found in Types, but schema by the name not found in Registry
    error:                      %s
    registry reference:         %s
    schema name:                %s
    reflect.Type of the obejct: %s`,
				err,
				p.reg.Reference().Short(),
				name,
				typ.String())
			return
		}

		// else everything is ok

		// setup references of given obj

		if err = wn.pack.setupReferencesOfGo(obj); err != nil {
			return
		}

		// save the object getting hash back
		key, _ := p.save(obj)

		// setup

		wn.sch = sch
		wn.value = obj

		if key != d.Object || d.SchemaRef != sch.Reference() {
			wn.unsaved = true
		}

		d.SchemaRef = sch.Reference()
		d.Object = key

	} else {
		// TODO (kostyarin): other packing methods
		panic("not implemented yet")
	}

	wn.set(d)

	return
}

func (d *Dynamic) setDynamic(odr *Dynamic) (err error) {

	if odr.IsBlank() {
		d.clear()
		return
	}

	wn := d.walkNode // valid

	if !odr.IsValid() {
		err = fmt.Errorf("argument is not valid Dynamic: %s", odr.Short())
		return
	}

	if d.Eq(odr) {
		return // equal Dynamic references
	}

	var sch Schema
	var value interface{}

	// schema

	if d.SchemaRef != odr.SchemaRef {
		if odr.walkNode != nil && odr.walkNode.sch != nil {
			sch = odr.walkNode.sch
		} else {
			sch, err = wn.pack.reg.SchemaByReference(odr.SchemaRef)
			if err != nil {
				return
			}
		}
	}

	// value

	if odr.Object != (cipher.SHA256{}) {
		// not blank
		if odr.walkNode != nil && odr.walkNode.value != nil {
			value = odr.walkNode.value
		} else {
			if value, err = wn.pack.unpack(sch, odr.Object); err != nil {
				return
			}
		}
	}

	// else: value cleared below

	// setup

	wn.sch = schema
	wn.value = value // set or clear value
	wn.unsaved = true

	d.SchemaRef = odr.SchemaRef
	d.Object = odr.Object

	wn.set(d) // set to place

	return
}

func (d *Dynamic) clear() {
	d.SchemaRef = SchemaReference{}
	d.Object = cipher.SHA256{}
	if wn := d.walkNode; wn != nil {
		wn.value = nil
		wn.sch = nil
		wn.set(d)
	}
}

// Copy returns detached copy of the Dynamic
//
// TODO (kostyarin): explain detailed
func (d *Dynamic) Copy() (cp Dynamic) {
	cp.SchemaRef = d.SchemaRef
	cp.Object = d.Object
	if wn := d.walkNode; wn != nil {
		cp.walkNode = &walkNode{
			pack:  wn.pack,
			sch:   wn.sch,
			value: wn.value,
		}
	}
}

// Detach the Dynamic from its place
func (d *Dynamic) Detach() {
	if wn := d.walkNode; wn != nil {
		wn.place = nil
	}
}

// Attach used to attach the Dynamic to a place in
// slice or array. It's impossible to attach the Dynamic
// to struct field. The Detach and Attach methods are
// useful if the Dynamic is member of slice or array.
// The first argument is array or slice to place to.
// The second is index. The obj must to be array or
// slice or pointer to array or slice. The method
// panics if the Dynamic (receiver) hasn't been created
// by Pack. It also panics if given 'ary' argumen is not
// addressable or index out of range. You should not
// care about this method if you are not using
// arrays or slices of the Dynamic. The Attach method
// can be used without preceding Detach
//
// TODO (kostyarin): add usage example
func (d *Dynamic) Attach(ary interface{}, i int) {

	if i < 0 {
		err := fmt.Errorf("can't attach Dynamic: index below zero %d", i)
		panic(err)
	}

	wn := d.walkNode

	if wn == nil {
		err := errors.New(
			"can't attach Dynamic: missing internal reference to Pack")
		panic(err)
	}

	val := reflect.ValueOf(ary)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Array, reflect.Slice:
	default:
		err := fmt.Errorf("can't attach Dynamic to %T", ary)
		panic(err)
	}

	if i >= val.Len() {
		err := fmt.Errorf("can't attach Dynamic: index out of range %d of %d",
			i, val.Len())
		panic(err)
	}

	idx := val.Index(i)

	if !idx.CanSet() {
		err := fmt.Errorf(
			"can't attach Dynamic: provided %s is not addressable <%T>",
			val.Kind().String(), ary)
		panic(err)
	}

	this := reflect.ValueOf(d)
	this = reflect.Indirect(this)

	idx.Set(this)

}
