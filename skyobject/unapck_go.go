package skyobject

import (
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// given val can't be encoded reference; it guaraneed by
// Types map: we can't register a reference
func (p *Pack) unpackToGo(schemaName string, val []byte) (obj interface{},
	err error) {

	var typ reflect.Type
	var ok bool

	if typ, ok = p.types.Direct[schemaName]; !ok {
		err = fmt.Errorf("missing reflect.Type of %q schema in Types.Direct",
			schemaName)
		return
	}

	ptr := reflect.New(typ)
	if _, err = encoder.DeserializeRawToValue(val, ptr); err != nil {
		return
	}

	elem := ptr.Elem()
	if err = p.setupToGo(elem); err != nil {
		return
	}

	obj = reflect.Indirect(ptr).Interface()
	return
}

// setup references of a golang-value
func (p *Pack) setupToGo(obj reflect.Value) (err error) {
	switch obj.Kind() {
	case reflect.Array, reflect.Slice:
		err = p.setupArrayOrSliceToGo(obj)
	case reflect.Struct:
		err = p.setupStructToGo(obj)
	}
	return
}

// obj is parent object, idx is reflect.Value that is Dynamic;
// for examples:
//
//     - struct {  A Dynamic }
//        - the struct is obj
//        - value of the A field is idx
//
//     - []Dynamic
//       - the slice is obj
//       - a value of the slice is idx
//
//     - [3]Dynamic
//       - the array is obj
//       - a value of the array is idx
//
func (p *Pack) setupDynamicToGo(obj, idx reflect.Value) (err error) {

	dr := idx.Interface().(Dynamic)
	if dr.walkNode != nil {
		return // already set up (skip circular references)
	}

	if !dr.IsValid() {
		// detailed error
		err = fmt.Errorf("invalid Dynamic reference %s, in %s",
			dr.Short(), obj.Type().String())
		return
	}

	wn := new(walkNode)

	wn.place = idx
	wn.pack = p

	dr.walkNode = wn

	if p.flags&EntireTree != 0 {
		if _, err = dr.Schema(); err != nil { // setup schema
			return
		}
		if _, err = dr.Value(); err != nil { // setup value
			return
		}
	}
	wn.set(dr)
	return
}

//     sf  - field
//     val - field value, type of which is Reference
func (p *Pack) setupRefToGo(sf reflect.StructField,
	val reflect.Value) (err error) {

	var name string
	if name, err = TagSchemaName(sf.Tag); err != nil {
		return
	}

	var sch Schema
	if sch, err = p.reg.SchemaByName(name); err != nil {
		return
	}

	ref := val.Interface().(Ref)
	if ref.walkNode != nil {
		return // already set up (skip circular references)
	}

	wn := new(walkNode)

	wn.sch = sch
	wn.place = val
	wn.pack = p

	ref.walkNode = wn

	if p.flags&EntireTree != 0 {
		if _, err = ref.Value(); err != nil { // setup value
			return
		}
	}

	wn.set(ref)
	return
}

func (p *Pack) setupRefsToGo(sf reflect.StructField,
	val reflect.Value) (err error) {

	var name string
	if name, err = TagSchemaName(sf.Tag); err != nil {
		return
	}

	var sch Schema
	if sch, err = p.reg.SchemaByName(name); err != nil {
		return
	}

	refs := val.Interface().(Refs)
	if refs.wn != nil {
		return // already set up (skip circular references)
	}

	var rr *Refs
	if rr, err = p.getRefs(sch, refs.Hash, val); err != nil {
		return
	}
	rr.wn.set(rr)
	return
}

// an array or slice can contain references (we interest):
//   - array of Dynamic
//   - array of structs
func (p *Pack) setupArrayOrSliceToGo(obj reflect.Value) (err error) {

	typ := obj.Type().Elem()

	if typ == dynamicRef {
		for i := 0; i < obj.Len(); i++ {
			idx := obj.Index(i)
			if err = p.setupDynamicToGo(obj, idx); err != nil {
				return
			}
		}
		return
	}

	if typ.Kind() == reflect.Struct {
		for i := 0; i < obj.Len(); i++ {
			idx := obj.Index(i)
			if err = p.setupStructToGo(idx); err != nil {
				return
			}
		}
	}

	return
}

// a struct can contain references only:
//   - field of Dynamic
//   - field of array of Dynamic
//   - field of slice of Dynamic
//   - field of Ref
//   - field of Refs
//   - field of struct
func (p *Pack) setupStructToGo(obj reflect.Value) (err error) {

	typ := obj.Type()

	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		if sf.Tag.Get("enc") == "-" || sf.PkgPath != "" || sf.Name == "_" {
			continue // skip unexported, unencoded and _-named fields
		}
		switch sf.Type {
		case singleRef:
			err = p.setupRefToGo(sf, obj.Field(i))
		case sliceRef:
			err = p.setupRefsToGo(sf, obj.Field(i))
		case dynamicRef:
			err = p.setupDynamicToGo(obj, obj.Field(i))
		default:
			continue
		}
		if err != nil {
			return
		}
	}

	return
}
