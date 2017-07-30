package skyobject

import (
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"
)

// knowsAboutFunc used to determine objects of a Root. If it returns
// an error the itteration breaks. Use ErrStopRange to stop.
// If it returns deeper = true, then current object will be
// inspected, otherwise skipped after call
type knowsAboutFunc func(cipher.SHA256) (deeper bool, err error)

// data.ViewObjects or data.UpdateObjects
type getter interface {
	Get(cipher.SHA256) []byte
}

// knowsAbout calls given knowsAboutFunc for every (not really) object of
// given Root. The method never returns "missing object" or "missing registry"
// errors. Use 'deeper' reply of knowsAboutFunc to control how deep an object
// should be inspected
func (c *Container) knowsAbout(r *Root, g getter, fn knowsAboutFunc) (err error) {

	// 1) registry

	var deeper bool
	if _, err = fn(cipher.SHA256(r.Reg)); err != nil {
		goto Return
	}

	var reg *Registry
	if reg = c.Registry(r.Reg); reg == nil {
		return // return nil (no "missing Registry" errors)
	}

	// 2) refs ([]Dynamic)

	var kn knowsAbout

	kn.fn = fn
	kn.g = g
	kn.reg = reg

	for _, dr := range r.Refs {
		if err = kn.Dynamic(dr); err != nil {
			break
		}
	}

Return:
	if err == ErrStopRange {
		err = nil
	}
	return
}

type knowsAbout struct {
	fn  knowsAboutFunc
	g   getter
	reg *Registry
}

func (k *knowsAbout) Dynamic(dr Dynamic) (err error) {

	if !dr.IsValid() {
		return fmt.Errorf("invalid dynamic %s", dr.Short())
	}

	if dr.Object == (cipher.SHA256{}) {
		return // represents nil
	}

	var deeper bool
	if deeper, err = k.fn(dr.Object); err != nil {
		return // some error
	}

	if deeper == false {
		return // don't inspect deeper
	}

	var sch Schema
	if sch, err = k.reg.SchemaByReference(dr.SchemaRef); err != nil {
		return
	}

	return k.Hash(sch, dr.Object)
}

func (k *knowsAbout) Hash(sch Schema, hash cipher.SHA256) (err error) {

	if !sch.HasReferences() {
		return // skip (no references)
	}

	var val []byte
	if val = k.g.Get(hash); val == nil {
		return // skip (not found)
	}

	return k.Data(sch, val)
}

func (k *knowsAbout) Data(sch Schema, val []byte) (err error) {

	if !sch.HasReferences() {
		return // skip (no references)
	}

	if sch.IsReference() {
		return k.DataRefsSwitch(sch, val)
	}

	switch sch.Kind() {
	case reflect.Array:
		return k.DataArray(sch, val)
	case reflect.Slice:
		return k.DataSlice(sch, val)
	case reflect.Struct:
		return k.DataStruct(sch, val)
	}

	return fmt.Errorf("schema is not reference, array, slice or struct but"+
		"HasReferenes() retruns true: %s", sch)
}

func (k *knowsAbout) DataArray(sch Schema, val []byte) (err error) {
	ln := sch.Len()  // length of the array
	el := sch.Elem() // schema of element
	if el == nil {
		err = fmt.Errorf("nil schema of element of array: %s", sch)
		return
	}
	return k.rangeArraySlice(el, ln, val)
}

func (k *knowsAbout) DataSlice(sch Schema, val []byte) (err error) {
	var ln int
	if ln, err = getLength(val); err != nil {
		return
	}
	el := sch.Elem() // schema of element
	if el == nil {
		err = fmt.Errorf("nil schema of element of slice: %s", sch)
		return
	}
	return k.rangeArraySlice(el, ln, val[4:])
}

func (k *knowsAbout) DataStruct(sch Schema, val []byte) (err error) {

	var shift, s int

	for i, fl := range sch.Fields() {

		if shift >= len(val) {
			err = fmt.Errorf("unexpected end of encoded struct <%s>, "+
				"field number: %d, field name: %q, schema of field: %s",
				i,
				fl.Name(),
				fl.Schema())
			return
		}

		if s, err = SchemaSize(sch, val[shift:]); err != nil {
			return
		}

		if err = k.Data(fl.Schema(), val[shift:shift+s]); err != nil {
			return
		}

		shift += s

	}

	return
}

func (k *knowsAbout) rangeArraySlice(el Schema, ln int,
	val []byte) (err error) {

	var shift, m int
	for i := 0; i < ln; i++ {
		if shift >= len(val) {
			err = fmt.Errorf("unexpected end of encoded  array or slice "+
				"of <%s>, length: %d, index: %d", el, ln, i)
			return
		}
		if m, err = SchemaSize(el, val[shift:]); err != nil {
			return
		}
		if err = k.Data(el, val[shift:shift+m]); err != nil {
			return
		}
		shift += m
	}
	return
}

func (k *knowsAbout) DataRefsSwitch(sch Schema, val []byte) error {

	switch rt := sch.ReferenceType(); rt {
	case ReferenceTypeSingle:
		return k.DataReference(sch, val)
	case ReferenceTypeSlice:
		return k.DataReferences(sch, val)
	case ReferenceTypeDynamic:
		return k.DataDynamic(val)
	}

	return fmt.Errorf("[ERR] reference with invalid ReferenceType: %d", rt)

}

func (k *knowsAbout) DataReference(sch Schema, val []byte) (err error) {

	var ref Reference
	if err = encoder.DeserializeRaw(val, &ref); err != nil {
		return
	}

	el := sch.Elem()
	if el == nil {
		err = fmt.Errorf("[ERR] schema of Reference [%s] without element: %s",
			ref.Short(),
			sch)
		return
	}

	if ref.IsBlank() {
		return
	}

	var deeper bool
	if deeper, err = k.fn(ref.Hash); err != nil {
		return
	}

	if deeper == false {
		return
	}

	return k.Hash(el, ref.Hash)

}

func (k *knowsAbout) DataReferences(sch Schema, val []byte) (err error) {

	var refs References
	if err = encoder.DeserializeRaw(val, &refs); err != nil {
		return
	}

	if refs.IsBlank() {
		return
	}

	for _, rn := range refs.Nodes {
		if err = k.RefsNode(refs.Depth, sch, rn); err != nil {
			return
		}
	}

	return

}

func (k *knowsAbout) RefsNode(depth uint32, sch Schema,
	rn RefsNode) (err error) {

	var deeper bool

	if depth == 0 {
		// the leaf
		for _, hash := range rn.Hashes {

			if hash == (cipher.SHA256{}) {
				continue
			}

			if deeper, err = k.fn(hash); err != nil {
				return
			}
			if deeper == false {
				return
			}

			if err = k.Hash(sch, hash); err != nil {
				return
			}
		}
		return
	}

	depth-- // go deepper

	for _, hash := range rn.Hashes {

		if hash == (cipher.SHA256{}) {
			continue
		}

		if deeper, err = k.fn(hash); err != nil {
			return
		}
		if deeper == false {
			return
		}

		if err = k.HashRefsNode(depth, sch, hash); err != nil {
			return
		}
	}

	return

}

func (k *knowsAbout) HashRefsNode(depth uint32, sch Schema,
	hash cipher.SHA256) (err error) {

	// hash never be empty

	val, ok := k.g.get(hash)
	if !ok {
		return
	}

	var rn RefsNode
	if err = encoder.DeserializeRaw(val, &rn); err != nil {
		return
	}

	return k.RefsNode(depth, sch, rn)
}

func (k *knowsAbout) DataDynamic(val []byte) (err error) {

	var dr Dynamic
	if err = encoder.DeserializeRaw(val, &dr); err != nil {
		return
	}

	return k.Dynamic(dr)

}
