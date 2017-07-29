package skyobject

import (
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// given val can be encoded reference; it guaraneed by Types map:
// we can't register a reference
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
	if err = p.setupReferencesOfGo(elem); err != nil {
		return
	}

	obj = reflect.Indirect(ptr).Interface()
	return
}

// setup references of a golang-value
func (p *Pack) setupReferencesOfGo(obj reflect.Value) (err error) {

	switch obj.Kind() {
	case reflect.Array:
		err = setupReferencesOfGoArray(obj)
	case reflect.Slice:
		err = setupReferencesOfGoSlice(obj)
	case reflect.Struct:
		err = setupReferencesOfGoStruct(obj)
	}

	return
}

// using (reflect.Value).Set
func (p *Pack) setupReferencesRenameMe(upper, val reflect.Value) (err error) {

	switch val.Type() {
	case singleRef:
		ref := val.Interface().(Reference)

		wn := new(walkNode)

		wn.pack = p
		wn.sch = nil // TODO (kostyarin): set up schema

		if upper.Kind() == reflect.Slice {
			//
		} else if upper.Kind() == reflect.Struct {
			//
		} else { // element of an array
			//
		}

		ref.walkNode = wn

		if p.flags&EntireTree != 0 {
			if err = p.unpackReferenceToGo(&ref); err != nil {
				return
			}
		}

		val.Set(relfect.ValueOf(ref))
	case sliceRef:
		refs := val.Interface().(References)
		// TODO (kostyarin): set up the reference
		val.Set(reflect.ValueOf(refs))
	case dynamicRef:
		dr := val.Interface().(Dynamic)
		// TODO (kostyarin): set up the reference
		val.Set(reflect.ValueOf(dr))
	}
	return
}

// an array can contain references only:
//   - array of Dynamic
//   - array of structs
func (p *Pack) setupReferencesOfGoArray(obj reflect.Value) (err error) {

	elemTyp := obj.Type().Elem()

	if elemTyp == dynamicRef {
		// TODO (kostyarin): setup dynamic reference
		return
	}

	if elemTyp.Kind() == reflect.Struct {
		err = p.setupReferencesOfGoArrayOrSliceOfStructs(obj)
	}

	return

}

// a slice can contain references only:
//   - slice of Dynamic
//   - slice of structs
func (p *Pack) setupReferencesOfGoSlice(obj reflect.Value) (err error) {

	elemTyp := obj.Type().Elem()

	if elemTyp == dynamicRef {
		for i := 0; i < obj.Len(); i++ {
			idx := obj.Index(i)

			var dr Dynamic

			dr = idx.Interface().(Dynamic)

			if dr.walkNode != nil {
				continue // already set up (skip circular references)
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

			idx.Set(dr)
		}
		return
	}

	if elemTyp.Kind() == reflect.Struct {
		err = p.setupReferencesOfGoArrayOrSliceOfStructs(obj)
	}

	return

}

func (p *Pack) setupReferencesOfGoArrayOrSliceOfDynamic(
	obj reflect.Value) (err error) {

	for i := 0; i < obj.Len(); i++ {
		idx := obj.Index(i)
		// TODO (kostyarin): implement
	}

	return
}

func (p *Pack) setupReferencesOfGoArrayOrSliceOfStructs(
	obj reflect.Value) (err error) {

	for i := 0; i < obj.Len(); i++ {
		idx := obj.Index(i)
		if err = p.setupReferencesOfGoStruct(idx); err != nil {
			return
		}
	}
	return
}

// a struct can contain references only:
//   - field of Dynamic
//   - field of array of Dynamic
//   - field of slice of Dynamic
//   - field of Reference
//   - field of References
//   - field of struct
func (p *Pack) setupReferencesOfGoStruct(ptr reflect.Value,
	typ reflect.Type) (err error) {

	// the struct can't contain pointers

	//

}

// the ref must have walkNode
func (p *Pack) unpackReferenceToGo(ref *Reference) (err error) {
	if ref.IsBlank() {
		return
	}
	var val []byte
	if val, err = p.get(ref.Hash); err != nil {
		return
	}

	return
}
