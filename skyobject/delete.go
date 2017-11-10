package skyobject

import (
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

// decrementAll references of given *data.Root
// (do it before deleting the Root)
func (c *Container) decrementAll(ir *data.Root) (err error) {
	// ----
	// (0) get encoded Root by hash from CXDS
	// (1) decode the root turning it *skyobject.Root
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
	r.IsFull = true // but it doesn't make sense

	// (2), (3), (4), (5)
	err = c.decrement(r)

	// (6) or err
	return
}

/*
err = c.findRefs(r, func(key cipher.SHA256) (deepper bool, err error) {
	// if rc is zero then this value has been deleted
	// and we can't get it from database to explore deepper
	// shit! shit! shit!
	var rc uint32
	deepper = true
	_, err = c.db.CXDS().Dec(key)
	return
})
*/

// find all references including registry and r.Hash
func (c *Container) decrement(r *Root) (err error) {

	// root

	if r.Hash == (cipher.SHA256{}) {
		return ErrEmptyRootHash
	}

	var rc uint32
	if rc, err = c.db.CXDS().Dec(r.Hash); err != nil || rc > 0 {
		return // rc > 0: not removed, doesn't need to decrement all related
	}

	// ok, the Root has been removed, let's decrement related objects

	// registry

	if r.Reg == (RegistryRef{}) {
		return ErrEmptyRegistryRef
	}

	var val []byte
	if val, _, err = c.db.CXDS().DecGet(cipher.SHA256(r.Reg)); err != nil {
		return
	}

	var reg *Registry
	if reg, err = DecodeRegistry(val); err != nil {
		return
	}

	d := decRecur{reg, c}

	for _, dr := range r.Refs {
		if err = d.dynamic(dr); err != nil {
			return
		}
	}
	return
}

// decrement recursive
type decRecur struct {
	reg *Registry
	c   *Container
}

func (d *decRecur) dynamic(dr Dynamic) (err error) {
	if dr.Object == (cipher.SHA256{}) {
		return // ignore blank
	}
	if false == dr.IsValid() {
		return ErrInvalidDynamicReference
	}
	var rc uint32
	var val []byte
	if val, rc, err = d.c.db.CXDS().DecGet(dr.Object); err != nil || rc > 0 {
		return // rc > 0: not removed, doesn't need to decrement all related
	}
	var s Schema
	if s, err = d.reg.SchemaByReference(dr.SchemaRef); err != nil {
		return
	}
	return d.data(s, val)
}

func (d *decRecur) schemaKey(s Schema, key cipher.SHA256) (err error) {

	if key == (cipher.SHA256{}) {
		return
	}

	if false == s.HasReferences() {
		return // this object doesn't contain references
	}

	var val []byte
	var rc uint32
	if val, rc, err = d.c.db.CXDS().DecGet(key); err != nil || rc > 0 {
		return
	}

	return d.data(s, val)
}

func (d *decRecur) data(s Schema, val []byte) (err error) {

	if false == s.HasReferences() {
		return // this object doesn't have references to another objects
	}

	if s.IsReference() {
		return d.dataRefsSwitch(s, val)
	}

	switch s.Kind() {
	case reflect.Array:
		return d.dataArray(s, val)
	case reflect.Slice:
		return d.dataSlice(s, val)
	case reflect.Struct:
		return d.dataStruct(s, val)
	}

	return fmt.Errorf("[CXO BUG] schema is not reference, array, slice or "+
		"struct but (Schema).HasReferences() retruns true: %s", s)
}

func (d *decRecur) dataArray(s Schema, val []byte) (err error) {

	// length of the array, schema of element
	ln, el := s.Len(), s.Elem()
	if el == nil {
		return fmt.Errorf("[CXO BUG] nil schema of element of array: %s", s)
	}
	return d.rangeArraySlice(el, ln, val)
}

func (d *decRecur) dataSlice(s Schema, val []byte) (err error) {

	var ln int
	if ln, err = getLength(val); err != nil {
		return
	}
	el := s.Elem() // schema of element
	if el == nil {
		return fmt.Errorf("nil schema of element of slice: %s", s)
	}
	return d.rangeArraySlice(el, ln, val[4:])
}

func (d *decRecur) dataStruct(s Schema, val []byte) (err error) {

	var shift, z int
	for i, fl := range s.Fields() {
		if shift > len(val) {
			return fmt.Errorf("unexpected end of encoded struct <%s>, "+
				"field number: %d, field name: %q, schema of field: %s",
				s.String(), i, fl.Name(), fl.Schema())
		}
		if z, err = fl.Schema().Size(val[shift:]); err != nil {
			return
		}
		if err = d.data(fl.Schema(), val[shift:shift+z]); err != nil {
			return
		}
		shift += z
	}
	return
}

func (d *decRecur) rangeArraySlice(el Schema, ln int, val []byte) (err error) {

	var shift, m int
	for i := 0; i < ln; i++ {
		if shift > len(val) {
			return fmt.Errorf("unexpected end of encoded  array or slice "+
				"of <%s>, length: %d, index: %d", el, ln, i)
		}
		if m, err = el.Size(val[shift:]); err != nil {
			return
		}
		if err = d.data(el, val[shift:shift+m]); err != nil {
			return
		}
		shift += m
	}
	return
}

func (d *decRecur) dataRefsSwitch(s Schema, val []byte) error {
	switch rt := s.ReferenceType(); rt {
	case ReferenceTypeSingle:
		return d.dataRef(s, val)
	case ReferenceTypeSlice:
		return d.dataRefs(s, val)
	case ReferenceTypeDynamic:
		return d.dataDynamic(val)
	default:
		return fmt.Errorf("[CXO BUG] reference with invalid ReferenceType: %d",
			rt)
	}
}

func (d *decRecur) dataRef(s Schema, val []byte) (err error) {

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

	return d.schemaKey(el, ref.Hash)
}

func (d *decRecur) dataRefs(s Schema, val []byte) (err error) {

	var refs Refs
	if err = encoder.DeserializeRaw(val, &refs); err != nil {
		return
	}
	if refs.IsBlank() {
		return
	}

	el := s.Elem()
	if el == nil {
		return fmt.Errorf("[CXO BUG] schema of Refs [%s] without element: %s",
			refs.Short(), s)
	}

	var rc uint32
	if val, rc, err = d.c.db.CXDS().DecGet(refs.Hash); err != nil || rc > 0 {
		return
	}

	var ers encodedRefs
	if err = encoder.DeserializeRaw(val, &ers); err != nil {
		return
	}

	return d.refsNode(el, int(ers.Depth), ers.Nested)
}

func (d *decRecur) refsNode(el Schema, depth int,
	keys []cipher.SHA256) (err error) {

	for _, key := range keys {
		if key == (cipher.SHA256{}) {
			continue
		}

		if depth == 0 { // the leaf
			if err = d.schemaKey(el, key); err != nil {
				return
			}
			continue
		}

		var val []byte
		var rc uint32
		if val, rc, err = d.c.db.CXDS().DecGet(key); err != nil || rc > 0 {
			return
		}

		var ern encodedRefsNode
		if err = encoder.DeserializeRaw(val, &ern); err != nil {
			return
		}
		if err = d.refsNode(el, depth-1, ern.Nested); err != nil {
			return
		}
	}

	return
}

func (d *decRecur) dataDynamic(val []byte) (err error) {
	var dr Dynamic
	if err = encoder.DeserializeRaw(val, &dr); err != nil {
		return
	}
	return d.dynamic(dr)

}
