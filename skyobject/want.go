package skyobject

import (
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

type viewObjects interface {
	IsExist(cipher.SHA256) bool
	Get(cipher.SHA256) []byte
}

// A WantFunc is a function that calleld
// for every wanted object. It never called
// for wanted Registry. Use ErrStopRange to
// terminate. The WantFunc can be called
// many times for a Reference
type WantFunc func(ref Reference) error

type unpackWant struct {
	objs viewObjects

	reg *Registry
	wf  WantFunc

	// keep state alive
}

func (u *unpackWant) get(ref Reference) []byte {
	return u.objs.Get(cipher.SHA256(ref))
}

func (u *unpackWant) isExist(ref Reference) bool {
	return u.objs.IsExist(cipher.SHA256(ref))
}

func (u *unpackWant) unpackDynamic(dr Dynamic) (err error) {
	if !dr.IsValid() {
		err = ErrInvalidDynamicReference
		return
	}
	if dr.Object.IsBlank() {
		return // nil, nil
	}
	var sch Schema
	if sch, err = u.reg.SchemaByReference(dr.Schema); err != nil {
		return
	}
	data := u.get(dr.Object)
	if data == nil {
		err = u.wf(dr.Object)
		return
	}
	return u.unpack(sch, data)
}

func (u *unpackWant) unpackReference(el Schema, ref Reference) (err error) {
	if ref.IsBlank() {
		return // nil
	}
	data = u.get(ref)
	if data == nil {
		err = wf(ref)
		return //  [err]
	}
	if !el.HasReferences() {
		return // nil
	}
	return u.unpack(el, data)
}

func (u *unpackWant) unpackEncodedReference(el Schema,
	data []byte) (err error) {

	var ref Reference
	if err = encoder.DeserializeRaw(data, &ref); err != nil {
		return // err
	}
	return u.unpackReference(el, ref)
}

func (u *unpackWant) unpackEncodedReferences(el Schema,
	data []byte) (err error) {

	var refs References
	if err = encoder.DeserializeRaw(da, &refs); err != nil {
		return // err
	}
	for _, ref := range refs {
		if err = u.unpackReference(el, ref); err != nil {
			return
		}
	}
	return
}

func (u *unpackWant) unpackEncodedDynamic(data []byte) (err error) {
	var dr Dynamic
	if err = encoder.DeserializeRaw(data, &dr); err != nil {
		return
	}
	return u.unpackDynamic(dr)
}

func (u *unpackWant) unpack(sch Schema, data []byte) (val Value, err error) {

	if !sch.HasReferences() {
		return // nil, nil
	}

	if sch.IsReference() {
		switch sch.ReferenceType() {
		case ReferenceTypeSingle:
			return u.unpackEncodedReference(sch.Elem(), data)
		case ReferenceTypeSlice:
			return u.unpackEncodedReferences(sch.Elem(), data)
		case ReferenceTypeDynamic:
			return u.unpackEncodedDynamic(data)
		}
		return
	}

	// arrays, slices and structs that has references
	switch sch.Kind() {
	case reflect.Array:
		//
	case reflect.Slice:
		//
	case reflect.Struct:
		//
	default:
		err = ErrInvalidSchema
	}
	return
}
