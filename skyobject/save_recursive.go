package skyobject

import (
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data/idxdb"
)

type saveRecursive struct {
	p     *Pack                      // related pack
	objs  idxdb.Objects              // save index
	saved map[cipher.SHA256]struct{} // saved obejct (to rollback on failure)
}

// setup references of a golang-value
func (p *saveRecursive) saveRecursive(
	obj reflect.Value,
) (
	amnt idxdb.Amount,
	vol idxdb.Volume,
	err error,
) {

	p.p.c.Debugln(VerbosePin, "saveRecursive", obj)

	if obj.Kind() == reflect.Ptr {
		obj = obj.Elem()
	}
	switch obj.Kind() {
	case reflect.Array, reflect.Slice:
		amnt, vol, err = p.saveRecursiveArrayOrSlice(obj)
	case reflect.Struct:
		amnt, vol, err = p.saveRecursiveStruct(obj)
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
func (p *saveRecursive) saveRecursiveDynamic(
	obj reflect.Value,
) (
	amnt idxdb.Amount,
	vol idxdb.Volume,
	err error,
) {

	p.p.c.Debugf(VerbosePin, "saveRecursiveDynamic %s", obj)

	dr := obj.Interface().(Dynamic)
	if !dr.IsValid() {
		// detailed error
		err = fmt.Errorf("invalid Dynamic reference %s", dr.Short())
		return
	}

	if dr.dn != nil && dr.dn.value != nil {

		// check out the dr.dn.value
		amnt, vol, err = p.saveRecursive(reflect.ValueOf(dr.dn.value))
		if err != nil {
			return
		}

		// save the value
		key, val := p.p.dsave(dr.dn.value)
		if _, err = p.p.c.DB().CXDS().Set(key, val); err != nil {
			return
		}
		p.saved[key] = struct{}{}

		io := new(idxdb.Object)
		io.Subtree.Amount = amnt
		io.Subtree.Volume = vol
		io.Vol = idxdb.Volume(len(val))

		if err = p.objs.Set(key, io); err != nil {
			return
		}

		amnt = io.Amount()
		vol = io.Volume()

	} else {
		// get from database

		if dr.Object == (cipher.SHA256{}) {
			return // empty obejct
		}

		var io *idxdb.Object
		if io, err = p.objs.Get(dr.Object); err != nil {
			return
		}
		amnt = io.Amount()
		vol = io.Volume()
	}

	obj.Set(reflect.ValueOf(dr)) // set it back
	return
}

//     sf  - field
//     val - field value, type of which is Reference
func (p *saveRecursive) saveRecursiveRef(
	sf reflect.StructField,
	val reflect.Value,
) (
	amnt idxdb.Amount,
	vol idxdb.Volume,
	err error,
) {

	p.p.c.Debugln(VerbosePin, "saveRecursiveRef", sf)

	ref := val.Interface().(Ref)

	if ref.rn != nil && ref.rn.value != nil {

		// check out the dr.dn.value
		amnt, vol, err = p.saveRecursive(reflect.ValueOf(ref.rn.value))
		if err != nil {
			return
		}

		// save the value
		key, val := p.p.dsave(ref.rn.value)
		if _, err = p.p.c.DB().CXDS().Set(key, val); err != nil {
			return
		}
		p.saved[key] = struct{}{}

		io := new(idxdb.Object)
		io.Subtree.Amount = amnt
		io.Subtree.Volume = vol
		io.Vol = idxdb.Volume(len(val))

		if err = p.objs.Set(key, io); err != nil {
			return
		}

		amnt = io.Amount()
		vol = io.Volume()

	} else {
		// get from database

		if ref.Hash == (cipher.SHA256{}) {
			return // empty obejct
		}

		var io *idxdb.Object
		if io, err = p.objs.Get(ref.Hash); err != nil {
			return
		}
		amnt = io.Amount()
		vol = io.Volume()
	}

	val.Set(reflect.ValueOf(ref)) // set it anyway
	return
}

func (p *saveRecursive) saveRecursiveRefs(
	sf reflect.StructField,
	val reflect.Value,
) (
	amnt idxdb.Amount,
	vol idxdb.Volume,
	err error,
) {

	p.p.c.Debugln(VerbosePin, "saveRecursiveRefs", sf, val)

	// total bullshit, digging this fucking Refs (fucking Merkle-tree)
	// I'll be damned, hell, hell, hell, hell is just a flowers

	refs := val.Interface().(Refs)

	if refs.Hash == (cipher.SHA256{}) {
		return
	}

	if refs.rn == nil {
		// get from database
		if refs.Hash == (cipher.SHA256{}) {
			return // empty refs
		}
		var io *idxdb.Object
		if io, err = p.objs.Get(refs.Hash); err != nil {
			return
		}
		amnt = io.Amount()
		vol = io.Volume()
	} else {
		// check out branches
		if refs.length == 0 {
			return // empty refs
		}
		amnt, vol, err = p.saveRecursiveRefsNode(&refs, refs.depth)
		if err != nil {
			return
		}
	}

	val.Set(reflect.ValueOf(refs)) // set it anyway
	return
}

func (p *saveRecursive) saveRecursiveRefsNode(
	rn *Refs,
	depth int,
) (
	amnt idxdb.Amount,
	vol idxdb.Volume,
	err error,
) {

	p.p.c.Debugln(VerbosePin, "saveRecursiveRefsNode", rn.Hash.Hex()[:7], depth)

	if rn.Hash == (cipher.SHA256{}) {
		return // empty branch
	}

	var pamnt idxdb.Amount
	var pvol idxdb.Volume

	if len(rn.leafs) == 0 && len(rn.branches) == 0 {

		// get from DB (not loaded)
		var io *idxdb.Object
		if io, err = p.objs.Get(rn.Hash); err != nil {
			return
		}
		amnt = io.Amount()
		vol = io.Volume()
		return

	}

	if rn.depth == 0 {

		for _, leaf := range rn.leafs {
			pamnt, pvol, err = p.saveRecursiveRefsElem(leaf)
			if err != nil {
				return
			}
			amnt += pamnt
			vol += pvol
		}

	} else {

		for _, br := range rn.branches {
			pamnt, pvol, err = p.saveRecursiveRefsNode(br, depth-1)
			if err != nil {
				return
			}
			amnt += pamnt
			vol += pvol
		}

	}

	// save the refs

	return
}

func (p *saveRecursive) saveRecursiveRefsElem(
	rn *RefsElem,
) (
	amnt idxdb.Amount,
	vol idxdb.Volume,
	err error,
) {

	p.p.c.Debugln(VerbosePin, "saveRecursiveRefsElem", rn.Hash.Hex()[:7])

	if rn.value == nil {

		// get from DB (not loaded)
		var io *idxdb.Object
		if io, err = p.objs.Get(rn.Hash); err != nil {
			return
		}
		amnt = io.Amount()
		vol = io.Volume()

		return

	} else {

		// loaded
		// check out the dr.dn.value
		amnt, vol, err = p.saveRecursive(reflect.ValueOf(rn.value))
		if err != nil {
			return
		}

		// save the value
		key, val := p.p.dsave(rn.value)
		if _, err = p.p.c.DB().CXDS().Set(key, val); err != nil {
			return
		}
		p.saved[key] = struct{}{}

		io := new(idxdb.Object)
		io.Subtree.Amount = amnt
		io.Subtree.Volume = vol
		io.Vol = idxdb.Volume(len(val))

		if err = p.objs.Set(key, io); err != nil {
			return
		}

		amnt = io.Amount()
		vol = io.Volume()
	}

	return
}

// an array or slice can contain references (we interest):
//   - array of Dynamic
//   - array of structs
func (p *saveRecursive) saveRecursiveArrayOrSlice(
	obj reflect.Value,
) (
	amnt idxdb.Amount,
	vol idxdb.Volume,
	err error,
) {

	p.p.c.Debugln(VerbosePin, "saveRecursiveArrayOrSlice", obj)

	var pamnt idxdb.Amount
	var pvol idxdb.Volume

	typ := obj.Type().Elem()
	if typ == typeOfDynamic {
		for i := 0; i < obj.Len(); i++ {
			idx := obj.Index(i)
			if pamnt, pvol, err = p.saveRecursiveDynamic(idx); err != nil {
				return
			}
			amnt += pamnt
			vol += pvol
		}
		return
	}

	if typ.Kind() == reflect.Struct {
		for i := 0; i < obj.Len(); i++ {
			idx := obj.Index(i)
			if pamnt, pvol, err = p.saveRecursiveStruct(idx); err != nil {
				return
			}
			amnt += pamnt
			vol += pvol
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
func (p *saveRecursive) saveRecursiveStruct(
	obj reflect.Value,
) (
	amnt idxdb.Amount,
	vol idxdb.Volume,
	err error,
) {

	p.p.c.Debugln(VerbosePin, "saveRecursiveStruct", obj)

	var pamnt idxdb.Amount
	var pvol idxdb.Volume

	typ := obj.Type()

	for i := 0; i < typ.NumField(); i++ {

		sf := typ.Field(i)

		if sf.Type == typeOfDynamic {

			pamnt, pvol, err = p.saveRecursiveDynamic(obj.Field(i))
			if err != nil {
				return
			}

			amnt += pamnt
			vol += pvol
			continue

		}

		if sf.Tag.Get("enc") == "-" || sf.PkgPath != "" || sf.Name == "_" {
			continue // skip unexported, unencoded and _-named fields
		}

		switch sf.Type {
		case typeOfRef:
			pamnt, pvol, err = p.saveRecursiveRef(sf, obj.Field(i))
		case typeOfRefs:
			pamnt, pvol, err = p.saveRecursiveRefs(sf, obj.Field(i))
		default:
			continue
		}

		if err != nil {
			return
		}

		amnt += pamnt
		vol += pvol

	}

	return
}
