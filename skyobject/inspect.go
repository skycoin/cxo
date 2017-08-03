package skyobject

import (
	"encoding/hex"
	"fmt"
	"reflect"

	"github.com/disiqueira/gotree" // alpha

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func (c *Container) Inspect(r *Root) string {
	ins := inspector{
		c: c,
		r: r,
	}
	return ins.Inspect()
}

type inspector struct {
	c   *Container
	r   *Root
	reg *Registry

	gt gotree.GTStructure
}

func (i *inspector) Inspect() (s string) {

	i.gt.Name = "(root) " + i.r.Short()
	defer func() {
		s = gotree.StringTree(i.gt)
	}()

	if i.r.Reg == (RegistryRef{}) {
		i.rootError("invalid Root: empty registry reference")
		return
	}

	if i.reg = i.c.Registry(i.r.Reg); i.reg == nil {
		i.rootError(fmt.Sprintf("missing Registry [s%s] in container",
			i.r.Reg.Short()))
		return
	}

	if len(i.r.Refs) == 0 {
		i.rootError("(empty)") // not an error
		return
	}

	for _, dr := range i.r.Refs {
		i.gt.Items = append(i.gt.Items, i.Dynamic(dr))
	}

	return

}

func (i *inspector) rootError(err string) {
	i.gt.Items = []gotree.GTStructure{{
		Name: err,
	}}
}

func (i *inspector) Dynamic(dr Dynamic) (it gotree.GTStructure) {
	if dr.IsBlank() {
		it.Name = "(dynamic) nil"
		return
	}
	if !dr.IsValid() {
		it.Name = "(err) invalid Dynamic" + dr.Short()
		return
	}
	if dr.Object == (cipher.SHA256{}) {
		it.Name = "(dynamic) nil of " + dr.SchemaRef.Short()
		return
	}
	sch, err := i.reg.SchemaByReference(dr.SchemaRef)
	if err != nil {
		it.Name = "(err) " + err.Error()
		return
	}
	val := i.c.Get(dr.Object)
	if val == nil {
		it.Name = "(dynamic) " + dr.Short()
		it.Items = []gotree.GTStructure{{
			Name: "(err) missing object: " + dr.Object.Hex()[:7],
		}}
		return
	}
	it.Name = "(dynamic) " + dr.Short()
	it.Items = []gotree.GTStructure{
		i.Data(sch, val),
	}
	return
}

func (i *inspector) Data(sch Schema, val []byte) (it gotree.GTStructure) {
	if sch.IsReference() {
		return i.refSwitch(sch, val)
	}
	switch sch.Kind() {
	case reflect.Bool:
		return i.Bool(sch, val)
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return i.Int(sch, val)
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return i.Uint(sch, val)
	case reflect.Float32, reflect.Float64:
		return i.Float(sch, val)
	case reflect.String:
		return i.String(sch, val)
	case reflect.Array, reflect.Slice:
		return i.Slice(sch, val)
	case reflect.Struct:
		return i.Struct(sch, val)
	default:
		it.Name = fmt.Sprintf("(err) invalid Kind <%s> of Schema %q",
			sch.Kind().String(),
			sch.String())
	}
	return
}

func (i *inspector) refSwitch(sch Schema, val []byte) (it gotree.GTStructure) {
	switch rt := sch.ReferenceType(); rt {
	case ReferenceTypeSingle:
		return i.Ref(sch, val)
	case ReferenceTypeSlice:
		return i.Refs(sch, val)
	case ReferenceTypeDynamic:
		var dr Dynamic
		if err := encoder.DeserializeRaw(val, &dr); err != nil {
			it.Name = "(err) " + err.Error()
			return
		}
		return i.Dynamic(dr)
	default:
		it.Name = fmt.Sprintf(
			"invalid schema (%s): reference with invalid type %d",
			sch.String(),
			rt)
		return
	}
}

// unpack to Value

func (*inspector) Bool(sch Schema, val []byte) (it gotree.GTStructure) {
	var x bool
	if err := encoder.DeserializeRaw(val, &x); err != nil {
		it.Name = "(err) " + err.Error()
		return
	}
	it.Name = fmt.Sprint(x)
	return
}

func (*inspector) Int(sch Schema, val []byte) (it gotree.GTStructure) {
	var x int64
	var err error
	switch sch.Kind() {
	case reflect.Int8:
		var y int8
		err = encoder.DeserializeRaw(val, &y)
		x = int64(y)
	case reflect.Int16:
		var y int16
		err = encoder.DeserializeRaw(val, &y)
		x = int64(y)
	case reflect.Int32:
		var y int32
		err = encoder.DeserializeRaw(val, &y)
		x = int64(y)
	case reflect.Int64:
		err = encoder.DeserializeRaw(val, &x)
	}
	if err != nil {
		it.Name = "(err) " + err.Error()
		return
	}
	it.Name = fmt.Sprint(x)
	return
}

func (*inspector) Uint(sch Schema, val []byte) (it gotree.GTStructure) {
	var x uint64
	var err error
	switch sch.Kind() {
	case reflect.Uint8:
		var y uint8
		err = encoder.DeserializeRaw(val, &y)
		x = uint64(y)
	case reflect.Uint16:
		var y uint16
		err = encoder.DeserializeRaw(val, &y)
		x = uint64(y)
	case reflect.Uint32:
		var y uint32
		err = encoder.DeserializeRaw(val, &y)
		x = uint64(y)
	case reflect.Uint64:
		err = encoder.DeserializeRaw(val, &x)
	}
	if err != nil {
		it.Name = "(err) " + err.Error()
		return
	}
	it.Name = fmt.Sprint(x)
	return
}

func (*inspector) Float(sch Schema, val []byte) (it gotree.GTStructure) {
	var x float64
	var err error
	switch sch.Kind() {
	case reflect.Float32:
		var y float32
		err = encoder.DeserializeRaw(val, &y)
		x = float64(y)
	case reflect.Float64:
		err = encoder.DeserializeRaw(val, &x)
	}
	if err != nil {
		it.Name = "(err) " + err.Error()
		return
	}
	it.Name = fmt.Sprint(x)
	return
}

func (*inspector) String(sch Schema, val []byte) (it gotree.GTStructure) {
	var x string
	if err := encoder.DeserializeRaw(val, &x); err != nil {
		it.Name = "(err) " + err.Error()
		return
	}
	it.Name = fmt.Sprintf("%q", x)
	return
}

// slcie or array
func (p *inspector) Slice(sch Schema, val []byte) (it gotree.GTStructure) {
	el := sch.Elem()
	if el == nil {
		it.Name = fmt.Sprintf("(err) invalid schema %q: nil-element",
			sch.String())
		return
	}
	if sch.Kind() == reflect.Slice && el.Kind() == reflect.Uint8 {
		// special case for []byte
		var x []byte
		if err := encoder.DeserializeRaw(val, &x); err != nil {
			it.Name = "(err) " + err.Error()
			return
		}
		it.Name = "([]byte) " + hex.EncodeToString(x)
		return
	}
	var ln int    // length
	var shift int // shift
	var err error
	if sch.Kind() == reflect.Array {
		ln = sch.Len()
	} else { // reflect.Slice
		if ln, err = getLength(val); err != nil {
			return
		}
		shift = 4
	}

	if sch.Name() != "" {
		it.Name = sch.Name()
	} else {
		if sch.Kind() == reflect.Array {
			it.Name = fmt.Sprintf("[%d]", ln)
		} else {
			it.Name = "[]"
		}
		if el.Name() != "" {
			it.Name += el.Kind().String()
		} else {
			it.Name += el.Name()
		}
	}

	defer func() {
		if err != nil {
			it.Name = "(err) " + err.Error()
			it.Items = nil
		}
	}()

	var m int
	if s := fixedSize(el.Kind()); s < 0 {
		for i := 0; i < ln; i++ {
			if shift+s > len(val) {
				err = unexpectedEndOfArraySliceError(sch, el, i, ln)
				return
			}
			it.Items = append(it.Items, p.Data(el, val[shift:shift+s]))
			shift += s
		}
	} else {
		for i := 0; i < ln; i++ {
			if shift >= len(val) {
				err = unexpectedEndOfArraySliceError(sch, el, i, ln)
				return
			}
			if m, err = SchemaSize(el, val[shift:]); err != nil {
				return
			}
			it.Items = append(it.Items, p.Data(el, val[shift:shift+m]))
			shift += m
		}
	}
	return
}

func unexpectedEndOfArraySliceError(sch, el Schema, i, ln int) (err error) {
	// detailed error
	var kindOf string
	if sch.Kind() == reflect.Array {
		kindOf = "array"
	} else {
		kindOf = "slice"
	}
	err = fmt.Errorf("unexpected end of encoded %s at index %d, "+
		"schema: '%s', element: '%s', length %d",
		kindOf,
		i,
		sch.String(),
		el.Kind().String(),
		ln)
	return
}

func (p *inspector) Struct(sch Schema, val []byte) (it gotree.GTStructure) {
	var shift int
	var s int
	var err error
	for _, f := range sch.Fields() {
		if shift >= len(val) {
			// detailed error
			it.Name = fmt.Sprintf(
				"(err) unexpected end of encoded struct '%s' "+
					"at field '%s', schema of field: '%s'",
				sch.String(),
				f.Name(),
				f.Schema().String())
			return
		}
		if s, err = SchemaSize(f.Schema(), val[shift:]); err != nil {
			it.Name = "(err) " + err.Error()
			return
		}
		fit := p.Data(f.Schema(), val[shift:shift+s])
		fit.Name = f.Name() + ": " + fit.Name // field name
		it.Items = append(it.Items, fit)
		shift += s
	}
	return
}

func (p *inspector) Ref(sch Schema, val []byte) (it gotree.GTStructure) {
	var ref Ref
	if err := encoder.DeserializeRaw(val, &ref); err != nil {
		it.Name = "(err) " + err.Error()
		return
	}
	return p.RefHash(sch, ref.Hash)
}

func (i *inspector) RefHash(sch Schema,
	hash cipher.SHA256) (it gotree.GTStructure) {

	if hash == (cipher.SHA256{}) {
		it.Name = "(ref) nil"
		return
	}
	it.Name = "(ref) " + hash.Hex()[:7]
	val := i.c.Get(hash)
	if val == nil {
		it.Items = []gotree.GTStructure{{
			Name: "(err) missing object " + hash.Hex()[:7],
		}}
	}
	it.Items = []gotree.GTStructure{i.Data(sch.Elem(), val)}
	return
}

func (i *inspector) Refs(sch Schema, val []byte) (it gotree.GTStructure) {
	var refs Refs
	if err := encoder.DeserializeRaw(val, &refs); err != nil {
		it.Name = "(err) " + err.Error()
		return
	}

	if refs.IsBlank() {
		it.Name = "(refs) nil"
		return
	}

	val = i.c.Get(refs.Hash)
	if val == nil {
		it.Name = "(err) missing obejct " + refs.Short()
		return
	}

	var er encodedRefs
	if err := encoder.DeserializeRaw(val, &er); err != nil {
		it.Name = "(err) " + err.Error()
		return
	}
	if er.Length == 0 {
		it.Name = "(refs) empty"
		return
	}
	it.Items = i.refsNode(sch, er.Depth, er)
	return
}

func (i *inspector) refsNode(sch Schema, depth uint32,
	er encodedRefs) (its []gotree.GTStructure) {

	if er.Depth == 0 {
		for _, h := range er.Nested {
			its = append(its, i.RefHash(sch, h))
		}
		return
	}
	for _, h := range er.Nested {
		its = append(its, i.refsHashNode(sch, depth-1, h)...)
	}
	return
}

func (i *inspector) refsHashNode(sch Schema, depth uint32,
	hash cipher.SHA256) (its []gotree.GTStructure) {

	val := i.c.Get(hash)
	if val == nil {
		its = append(its, gotree.GTStructure{
			Name: "missing object " + hash.Hex()[:7],
		})
		return
	}

	var er encodedRefs
	if err := encoder.DeserializeRaw(val, &er); err != nil {
		its = append(its, gotree.GTStructure{
			Name: "(err) " + err.Error(),
		})
		return
	}

	return i.refsNode(sch, depth, er)
}
