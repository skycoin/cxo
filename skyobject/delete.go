package skyobject

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data/idxdb"
)

var ErrEmptyRootHash = errors.New("empty hash of Root")

// decrementAll references of given *idxdb.Root
// (do it before deleting the Root)
func (c *Container) decrementAll(ir *idxdb.Root) (err error) {
	// ----
	// (0) get encoded Root by hash from CXDS
	// (1) decode the root turning it *skyobejct.Root
	// (2) get registry (and decrement it)
	// (3) range over Refs decrementing them
	// (4) if a Ref of the Refs deleted decode it and
	//     decrement its branches
	// (5) and so on
	// (6) Profit!
	// ----

	var val []byte

	// (0)
	if val, _, err = c.db.CXDS().Get(ir.Hash); err != nil {
		return
	}

	// (1)
	var r *Root
	if r, err = decodeRoot(val); err != nil {
		return
	}
	r.Hash = ir.Hash
	r.Sig = ir.Sig
	r.IsFull = true // but it doesn't make sence

	// (2), (3), (4), (5)
	err = c.findRefs(r, func(key cipher.SHA256) (deepper bool, err error) {
		deepper = true
		_, err = c.db.CXDS().Dec(key)
		return
	})

	// (6) or err
	return
}

// find all references including regsitry and r.Hash
func (c *Container) findRefs(r *Root, fr findRefsFunc) (err error) {

	// root

	if r.Hash == (cipher.SHA256{}) {
		return ErrEmptyRootHash
	}

	var deepper bool
	if deepper, err = fr(r.Hash); err != nil || deepper == false {
		if err == ErrStopIteration {
			err = nil
		}
		return
	}

	// registry

	if r.Reg == (RegistryRef{}) {
		return ErrEmptyRegsitryRef
	}

	if _, err = fr(cipher.SHA256(r.Reg)); err != nil {
		if err == ErrStopIteration {
			err = nil
		}
		return
	}

	var reg *Registry
	if reg, err = c.Registry(r.Reg); err != nil {
		return
	}

	f := findRefs{reg, c}

	for _, dr := range r.Refs {
		if err = f.dynamic(dr, fr); err != nil {
			if err == ErrStopIteration {
				err = nil
			}
			return
		}
	}
	return
}

// A findRefsFunc used by findRefs. Error reply used
// to stop finding (use ErrStopIteration) and to
// terminate the finding returning any other error.
// Use the deepper reply to ectract and explore current
// object. A findRefsFunc never called with empty hash
type findRefsFunc func(key cipher.SHA256) (deepper bool, err error)

// find refs of an element
type findRefs struct {
	reg *Registry
	c   *Container
}

func (f *findRefs) dynamic(dr Dynamic, fr findRefsFunc) (err error) {
	if dr.Object == (cipher.SHA256{}) {
		return // ignore blank
	}
	if false == dr.IsValid() {
		return ErrInvalidDynamicReference
	}
	var deepper bool
	if deepper, err = fr(dr.Object); err != nil || deepper == false {
		return
	}
	var s Schema
	if s, err = f.reg.SchemaByReference(dr.SchemaRef); err != nil {
		return
	}
	return f.schemaKey(s, dr.Object, fr)
}

func (f *findRefs) schemaKey(s Schema, key cipher.SHA256,
	fr findRefsFunc) (err error) {

	if key == (cipher.SHA256{}) {
		return
	}

	if false == s.HasReferences() {
		return // this obejct doesn't contain references
	}

	var val []byte
	if val, _, err = f.c.db.CXDS().Get(key); err != nil {
		return
	}

	return f.data(s, val, fr)
}

func (f *findRefs) data(s Schema, val []byte, fr findRefsFunc) (err error) {

	if false == s.HasReferences() {
		return // no references in this obejct
	}

	if s.IsReference() {
		return f.dataRefsSwitch(s, val, fr)
	}

	switch s.Kind() {
	case reflect.Array:
		return f.dataArray(s, val, fr)
	case reflect.Slice:
		return f.dataSlice(s, val, fr)
	case reflect.Struct:
		return f.dataStruct(s, val, fr)
	}

	return fmt.Errorf("[CXO BUG] schema is not reference, array, slice or "+
		"struct but (Schema).HasReferenes() retruns true: %s", s)
}

func (f *findRefs) dataArray(s Schema, val []byte,
	fr findRefsFunc) (err error) {

	// length of the array, schema of element
	ln, el := s.Len(), s.Elem()
	if el == nil {
		return fmt.Errorf("[CXO BUG] nil schema of element of array: %s", s)
	}
	return f.rangeArraySlice(el, ln, val, fr)
}

func (f *findRefs) dataSlice(s Schema, val []byte,
	fr findRefsFunc) (err error) {

	var ln int
	if ln, err = getLength(val); err != nil {
		return
	}
	el := s.Elem() // schema of element
	if el == nil {
		return fmt.Errorf("nil schema of element of slice: %s", s)
	}
	return f.rangeArraySlice(el, ln, val[4:], fr)
}

func (f *findRefs) dataStruct(s Schema, val []byte,
	fr findRefsFunc) (err error) {

	var shift, z int
	for i, fl := range s.Fields() {
		if shift > len(val) {
			return fmt.Errorf("unexpected end of encoded struct <%s>, "+
				"field number: %d, field name: %q, schema of field: %s",
				i, fl.Name(), fl.Schema())
		}
		if z, err = fl.Schema().Size(val[shift:]); err != nil {
			return
		}
		if err = f.data(fl.Schema(), val[shift:shift+z], fr); err != nil {
			return
		}
		shift += z
	}
	return
}

func (f *findRefs) rangeArraySlice(el Schema, ln int, val []byte,
	fr findRefsFunc) (err error) {

	var shift, m int
	for i := 0; i < ln; i++ {
		if shift > len(val) {
			return fmt.Errorf("unexpected end of encoded  array or slice "+
				"of <%s>, length: %d, index: %d", el, ln, i)
		}
		if m, err = el.Size(val[shift:]); err != nil {
			return
		}
		if err = f.data(el, val[shift:shift+m], fr); err != nil {
			return
		}
		shift += m
	}
	return
}

func (f *findRefs) dataRefsSwitch(s Schema, val []byte, fr findRefsFunc) error {
	switch rt := s.ReferenceType(); rt {
	case ReferenceTypeSingle:
		return f.dataRef(s, val, fr)
	case ReferenceTypeSlice:
		return f.dataRefs(s, val, fr)
	case ReferenceTypeDynamic:
		return f.dataDynamic(val, fr)
	default:
		return fmt.Errorf("[CXO BUG] reference with invalid ReferenceType: %d",
			rt)
	}
}

func (f *findRefs) dataRef(s Schema, val []byte, fr findRefsFunc) (err error) {

	var ref Ref
	if err = encoder.DeserializeRaw(val, &ref); err != nil {
		return
	}

	el := s.Elem()
	if el == nil {
		return fmt.Errorf("[CXO BUG] schema of Ref [%s] without element: %s",
			ref.Short(), s)
	}

	if ref.IsBlank() {
		return
	}

	var deepper bool
	if deepper, err = fr(ref.Hash); err != nil || deepper == false {
		return
	}

	return f.schemaKey(el, ref.Hash, fr)
}

func (f *findRefs) dataRefs(s Schema, val []byte, fr findRefsFunc) (err error) {

	var refs Refs
	if err = encoder.DeserializeRaw(val, &refs); err != nil {
		return
	}
	if refs.IsBlank() {
		return
	}

	var deepper bool
	if deepper, err = fr(refs.Hash); err != nil || deepper == false {
		return
	}

	el := s.Elem()
	if el == nil {
		return fmt.Errorf("[CXO BUG] schema of Refs [%s] without element: %s",
			refs.Short(), s)
	}

	if val, _, err = f.c.db.CXDS().Get(refs.Hash); err != nil {
		return
	}

	var ers encodedRefs
	if err = encoder.DeserializeRaw(val, &ers); err != nil {
		return
	}

	return f.refsNode(el, int(ers.Depth), ers.Nested, fr)
}

func (f *findRefs) refsNode(el Schema, depth int, keys []cipher.SHA256,
	fr findRefsFunc) (err error) {

	for _, key := range keys {
		if key == (cipher.SHA256{}) {
			continue
		}

		var deepper bool
		if deepper, err = fr(key); err != nil || deepper == false {
			return
		}

		if depth == 0 { // the leaf
			if err = f.schemaKey(el, key, fr); err != nil {
				return
			}
			continue
		}

		var val []byte
		if val, _, err = f.c.db.CXDS().Get(key); err != nil {
			return
		}

		var ern encodedRefsNode
		if err = encoder.DeserializeRaw(val, &ern); err != nil {
			return
		}
		if err = f.refsNode(el, depth-1, ern.Nested, fr); err != nil {
			return
		}
	}

	return
}

func (f *findRefs) dataDynamic(val []byte, fr findRefsFunc) (err error) {
	var dr Dynamic
	if err = encoder.DeserializeRaw(val, &dr); err != nil {
		return
	}
	return f.dynamic(dr, fr)

}
