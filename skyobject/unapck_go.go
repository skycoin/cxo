package skyobject

import (
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// create reflect.Value that represetns pointer of regsitered type
func (p *Pack) newOf(schemaName string) (ptr reflect.Value, err error) {

	p.c.Debugf(VerbosePin, "(*Pack).newOf %q", schemaName)

	var typ reflect.Type
	var ok bool

	if typ, ok = p.types.Direct[schemaName]; !ok {
		err = fmt.Errorf("missing reflect.Type of %q schema in Types.Direct",
			schemaName)
		return
	}
	ptr = reflect.New(typ)
	return
}

// given val can't be encoded reference; it guaraneed by
// Types map: we can't register a reference
func (p *Pack) unpackToGo(schemaName string,
	val []byte) (obj interface{}, err error) {

	p.c.Debugf(VerbosePin, "(*Pack).unpackToGo %d-bytes of %q", len(val),
		schemaName)

	var ptr reflect.Value
	if ptr, err = p.newOf(schemaName); err != nil {
		return
	}
	if _, err = encoder.DeserializeRawToValue(val, ptr); err != nil {
		return
	}
	elem := ptr.Elem()
	if err = p.setupToGo(elem); err != nil {
		return
	}
	obj = ptr.Interface()
	return
}

// setup references of a golang-value
func (p *Pack) setupToGo(obj reflect.Value) (err error) {

	p.c.Debugln(VerbosePin, "(*Pack).setupToGo", obj)

	if obj.Kind() == reflect.Ptr {
		obj = obj.Elem()
	}
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

	p.c.Debugf(VerbosePin, "(*Pack).setupDynamicToGo %s of %s", idx, obj)

	dr := idx.Interface().(Dynamic)
	if !dr.IsValid() {
		// detailed error
		err = fmt.Errorf("invalid Dynamic reference %s, in %s",
			dr.Short(), obj.Type().String())
		return
	}
	if dr.walkNode == nil {
		dr.walkNode = new(walkNode)
	}
	dr.walkNode.pack = p
	if p.flags&EntireTree != 0 && !dr.IsBlank() {
		if _, err = dr.Schema(); err != nil { // setup schema
			return
		}
		if _, err = dr.Value(); err != nil { // setup value
			return
		}
	}
	idx.Set(reflect.ValueOf(dr))
	return
}

//     sf  - field
//     val - field value, type of which is Reference
func (p *Pack) setupRefToGo(sf reflect.StructField,
	val reflect.Value) (err error) {

	p.c.Debugln(VerbosePin, "(*Pack).setupRefToGo", sf)

	var name string
	if name, err = TagSchemaName(sf.Tag); err != nil {
		return
	}
	var sch Schema
	if sch, err = p.reg.SchemaByName(name); err != nil {
		return
	}
	ref := val.Interface().(Ref)
	wn := ref.walkNode
	if wn == nil {
		wn = new(walkNode)
	}
	wn.sch = sch
	wn.pack = p
	ref.walkNode = wn
	if p.flags&EntireTree != 0 && !ref.IsBlank() {
		if _, err = ref.Value(); err != nil { // setup value
			return
		}
	}
	val.Set(reflect.ValueOf(ref))
	return
}

func (p *Pack) setupRefsToGo(sf reflect.StructField,
	val reflect.Value) (err error) {

	p.c.Debugln(VerbosePin, "(*Pack).setupRefsToGo", sf, val)

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
	val.Set(reflect.ValueOf(rr).Elem())
	return
}

// an array or slice can contain references (we interest):
//   - array of Dynamic
//   - array of structs
func (p *Pack) setupArrayOrSliceToGo(obj reflect.Value) (err error) {

	p.c.Debugln(VerbosePin, "(*Pack).setupArrayOrSliceToGo", obj)

	typ := obj.Type().Elem()
	if typ == typeOfDynamic {
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

	p.c.Debugln(VerbosePin, "(*Pack).setupStructToGo", obj)

	typ := obj.Type()
	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		if sf.Type == typeOfDynamic {
			if err = p.setupDynamicToGo(obj, obj.Field(i)); err != nil {
				return
			}
			continue
		}
		if sf.Tag.Get("enc") == "-" || sf.PkgPath != "" || sf.Name == "_" {
			continue // skip unexported, unencoded and _-named fields
		}
		switch sf.Type {
		case typeOfRef:
			err = p.setupRefToGo(sf, obj.Field(i))
		case typeOfRefs:
			err = p.setupRefsToGo(sf, obj.Field(i))
		default:
			continue
		}
		if err != nil {
			return
		}
	}
	return
}
