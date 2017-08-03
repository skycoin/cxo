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

	wn := d.walkNode

	if wn == nil {
		// TODO (kostyarin): make the error global
		err = errors.New(
			"can't get Schema of Dynamic reference detached from Pack")
		return
	}

	if wn.sch == nil { // obtain schema first
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

	if obj, err = wn.pack.unpackToGo(wn.sch.Name(), val); err != nil {
		return
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

			wn.set(d)

			// clear, but don't celar schema reference
			if prevs != sch.Reference() || d.Object != (cipher.SHA256{}) {
				wn.unsave()
			}
			return
		}
		val = reflect.Indirect(val) // indirect
	} else {
		// not a pointer (unaddressable value);
		// make it adressable to be able to setupToGo
		val, obj = makeAddressable(typ, val)
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

	wn.set(d)

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
		wn.set(d)

		if d.Object != prevo || d.SchemaRef != prevs {
			wn.unsave()
		}
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
			pack: wn.pack,
			sch:  wn.sch,
		}
	}
	return
}

// Detach the Dynamic from its place
func (d *Dynamic) Detach() {
	if wn := d.walkNode; wn != nil {
		wn.place = reflect.Value{} // make it invalid
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
