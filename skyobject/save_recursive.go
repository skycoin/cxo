package skyobject

import (
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
)

type saveRecursive struct {
	p     *Pack                 // related pack
	saved map[cipher.SHA256]int // saved obejct (to rollback on failure)
}

func (p *saveRecursive) inc(key cipher.SHA256) (err error) {
	if _, err = p.p.c.db.CXDS().Inc(key); err != nil {
		return
	}
	p.saved[key]++
	return
}

func (p *saveRecursive) save(key cipher.SHA256) (err error) {
	val, ok := p.p.unsaved[key]
	if !ok {
		return
		panic("missing cached value: " + key.Hex())
	}
	if _, err = p.p.c.db.CXDS().Set(key, val); err != nil {
		return
	}
	p.saved[key]++
	return
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

	p.p.c.Debugf(VerbosePin, "saveRecursiveDynamic %v", obj)

	dr := obj.Interface().(Dynamic)
	defer obj.Set(reflect.ValueOf(dr))

	if false == dr.IsValid() {
		// TODO (kostyarin): detailed error
		return fmt.Errorf("invalid Dynamic reference %s", dr.Short())
	}

	if dr.Object == (cipher.SHA256{}) {
		dr.ch = false
		return // blank object (nothing to save)
	}

	if dr.ch == false {
		// if the dr has not been changed then
		// we increment it and return, otherwise
		// we have to go deepper to save or increment
		// related (nested) obejcts
		return p.inc(dr.Object)
	}

	dr.ch = false // set to saved

	if dr.dn != nil && dr.dn.value != nil {
		// value is fresh or has been changed, we have to go deepper
		// to explore this value about references
		err = p.saveRecursive(reflect.ValueOf(dr.dn.value))
		if err != nil {
			return
		}
		return p.save(dr.Object)
	}

	// hash is not blank, so if the hash is not blank and value is nil
	// then this hash represents alreay saved value and we need to
	// increment refs count only
	return p.inc(dr.Object)
}

//     sf  - field
//     val - field value, type of which is Reference
func (p *saveRecursive) saveRecursiveRef(sf reflect.StructField,
	val reflect.Value) (err error) {

	p.p.c.Debugln(VerbosePin, "saveRecursiveRef", sf)

	ref := val.Interface().(Ref)
	defer val.Set(reflect.ValueOf(ref))

	if ref.Hash == (cipher.SHA256{}) {
		ref.ch = false
		return
	}

	if false == ref.ch {
		return p.inc(ref.Hash)
	}

	ref.ch = false

	if ref.rn != nil && ref.rn.value != nil {
		err = p.saveRecursive(reflect.ValueOf(ref.rn.value))
		if err != nil {
			return
		}
		// save the value
		return p.save(ref.Hash)
	}

	return p.inc(ref.Hash)
}

func (p *saveRecursive) saveRecursiveRefs(sf reflect.StructField,
	val reflect.Value) (err error) {

	p.p.c.Debugln(VerbosePin, "saveRecursiveRefs", sf, val)

	// total bullshit, digging this fucking Refs (fucking Merkle-tree)
	// I'll be damned, hell, hell, hell, hell is just a flowers

	refs := val.Interface().(Refs)
	defer val.Set(reflect.ValueOf(refs))

	if refs.Hash == (cipher.SHA256{}) {
		refs.ch = false
		return
	}

	if false == refs.ch {
		return p.inc(refs.Hash)
	}

	refs.ch = false

	if refs.rn != nil {
		// check out leafs/branches
		if refs.depth == 0 {
			for _, leaf := range refs.leafs {
				if err = p.saveRecursiveRefsElem(leaf); err != nil {
					return
				}
			}
		} else {
			for _, br := range refs.branches {
				if err = p.saveRecursiveRefsNode(br, refs.depth-1); err != nil {
					return
				}
			}
		}

		// save the refs
		return p.save(refs.Hash)
	}

	return p.inc(refs.Hash)
}

func (p *saveRecursive) saveRecursiveRefsNode(rn *Refs, depth int) (err error) {

	p.p.c.Debugln(VerbosePin, "saveRecursiveRefsNode", rn.Hash.Hex()[:7], depth)

	if rn.Hash == (cipher.SHA256{}) {
		rn.ch = false
		return
	}

	if false == rn.ch {
		return p.inc(rn.Hash)
	}

	rn.ch = false

	if rn.isLoaded() {
		if depth == 0 {
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
		return p.save(rn.Hash)
	}

	return p.inc(rn.Hash)
}

func (p *saveRecursive) saveRecursiveRefsElem(rn *RefsElem) (err error) {

	p.p.c.Debugln(VerbosePin, "saveRecursiveRefsElem", rn.Hash.Hex()[:7])

	if rn.Hash == (cipher.SHA256{}) {
		rn.ch = false
		return
	}

	if false == rn.ch {
		return p.inc(rn.Hash)
	}

	rn.ch = false

	if rn.value != nil {
		if err = p.saveRecursive(reflect.ValueOf(rn.value)); err != nil {
			return
		}
		return p.save(rn.Hash)
	}

	return p.inc(rn.Hash)
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
