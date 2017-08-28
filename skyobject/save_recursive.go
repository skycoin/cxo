package skyobject

import (
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
)

type saveRecursive struct {
	p     *Pack                      // related pack
	saved map[cipher.SHA256]struct{} // saved obejct (to rollback on failure)
}

// setup references of a golang-value
func (p *saveRecursive) saveRecursive(obj reflect.Value) (err error) {

	p.p.c.Debugln(VerbosePin, "saveRecursive", obj)

	if obj.Kind() == reflect.Ptr {
		obj = obj.Elem()
	}
	switch obj.Kind() {
	case reflect.Array, reflect.Slice:
		err = p.saveRecursiveArrayOrSlice(obj)
	case reflect.Struct:
		err = p.saveRecursiveStruct(obj)
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
func (p *saveRecursive) saveRecursiveDynamic(obj reflect.Value) (err error) {

	p.p.c.Debugf(VerbosePin, "saveRecursiveDynamic %s", obj)

	dr := obj.Interface().(Dynamic)
	if !dr.IsValid() {
		// detailed error
		err = fmt.Errorf("invalid Dynamic reference %s", dr.Short())
		return
	}

	if dr.dn != nil && dr.dn.value != nil {
		// check out the dr.dn.value
		err = p.saveRecursive(reflect.ValueOf(dr.dn.value))
		if err != nil {
			return
		}
		// save the value
		key, val := p.p.dsave(dr.dn.value)
		if _, err = p.p.c.DB().CXDS().Set(key, val); err != nil {
			return
		}
		p.saved[key] = struct{}{}
	}

	obj.Set(reflect.ValueOf(dr)) // set it back
	return
}

//     sf  - field
//     val - field value, type of which is Reference
func (p *saveRecursive) saveRecursiveRef(sf reflect.StructField,
	val reflect.Value) (err error) {

	p.p.c.Debugln(VerbosePin, "saveRecursiveRef", sf)

	ref := val.Interface().(Ref)

	if ref.rn != nil && ref.rn.value != nil {
		// check out the dr.dn.value
		err = p.saveRecursive(reflect.ValueOf(ref.rn.value))
		if err != nil {
			return
		}
		// save the value
		key, val := p.p.dsave(ref.rn.value)
		if _, err = p.p.c.DB().CXDS().Set(key, val); err != nil {
			return
		}
		p.saved[key] = struct{}{}
	}

	val.Set(reflect.ValueOf(ref)) // set it anyway
	return
}

func (p *saveRecursive) saveRecursiveRefs(sf reflect.StructField,
	val reflect.Value) (err error) {

	p.p.c.Debugln(VerbosePin, "saveRecursiveRefs", sf, val)

	// total bullshit, digging this fucking Refs (fucking Merkle-tree)
	// I'll be damned, hell, hell, hell, hell is just a flowers

	refs := val.Interface().(Refs)

	if refs.Hash == (cipher.SHA256{}) {
		return
	}

	if refs.rn != nil {
		// check out branches
		if refs.length == 0 {
			return // empty refs
		}
		if err = p.saveRecursiveRefsNode(&refs, refs.depth); err != nil {
			return
		}
	}

	val.Set(reflect.ValueOf(refs))
	return
}

func (p *saveRecursive) saveRecursiveRefsNode(rn *Refs, depth int) (err error) {

	p.p.c.Debugln(VerbosePin, "saveRecursiveRefsNode", rn.Hash.Hex()[:7], depth)

	if rn.Hash == (cipher.SHA256{}) {
		return // empty branch
	}

	if len(rn.leafs) == 0 && len(rn.branches) == 0 {
		return
	}

	if rn.depth == 0 {
		for _, leaf := range rn.leafs {
			if err = p.saveRecursiveRefsElem(leaf); err != nil {
				return
			}
		}
	} else {
		for _, br := range rn.branches {
			if err = p.saveRecursiveRefsNode(br, depth-1); err != nil {
				return
			}
		}
	}
	return
}

func (p *saveRecursive) saveRecursiveRefsElem(rn *RefsElem) (err error) {

	p.p.c.Debugln(VerbosePin, "saveRecursiveRefsElem", rn.Hash.Hex()[:7])

	if rn.value != nil {
		// loaded
		// check out the dr.dn.value
		if err = p.saveRecursive(reflect.ValueOf(rn.value)); err != nil {
			return
		}
		// save the value
		key, val := p.p.dsave(rn.value)
		if _, err = p.p.c.DB().CXDS().Set(key, val); err != nil {
			return
		}
		p.saved[key] = struct{}{}
	}

	return
}

// an array or slice can contain references (we interest):
//   - array of Dynamic
//   - array of structs
func (p *saveRecursive) saveRecursiveArrayOrSlice(obj reflect.Value) (
	err error) {

	p.p.c.Debugln(VerbosePin, "saveRecursiveArrayOrSlice", obj)

	typ := obj.Type().Elem()
	if typ == typeOfDynamic {
		for i := 0; i < obj.Len(); i++ {
			idx := obj.Index(i)
			if err = p.saveRecursiveDynamic(idx); err != nil {
				return
			}
		}
		return
	}

	if typ.Kind() == reflect.Struct {
		for i := 0; i < obj.Len(); i++ {
			idx := obj.Index(i)
			if err = p.saveRecursiveStruct(idx); err != nil {
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
func (p *saveRecursive) saveRecursiveStruct(obj reflect.Value) (err error) {

	p.p.c.Debugln(VerbosePin, "saveRecursiveStruct", obj)

	typ := obj.Type()
	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		if sf.Type == typeOfDynamic {
			if err = p.saveRecursiveDynamic(obj.Field(i)); err != nil {
				return
			}
			continue
		}
		if sf.Tag.Get("enc") == "-" || sf.PkgPath != "" || sf.Name == "_" {
			continue // skip unexported, unencoded and _-named fields
		}
		switch sf.Type {
		case typeOfRef:
			err = p.saveRecursiveRef(sf, obj.Field(i))
		case typeOfRefs:
			err = p.saveRecursiveRefs(sf, obj.Field(i))
		default:
			continue
		}
		if err != nil {
			return
		}
	}
	return
}
