package registry

import (
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"

	"github.com/DiSiqueira/GoTree"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// Tree used to print Root tree. The forceLoad
// argument used to  laod subtrees if they are
// not loaded. The Pack should have related
// Registry
func (r *Root) Tree(pack Pack) (tree string, err error) {

	var gt gotree.GTStructure

	gt.Name = fmt.Sprintf("(root) %s %s (reg: %s)",
		r.Hash.Hex()[:7], r.Short(), r.Reg.Short())

	var reg = pack.Registry()

	if reg == nil {
		err = errors.New("(*registry.Root).Tree: missing registry in Pack")
		return
	}

	if len(r.Refs) == 0 {

		gt.Items = []*gotree.GTStructure{
			&gotree.GTStructure{Name: "(empty)"},
		}

	} else {

		gt.Items = make([]*gotree.GTStructure, 0, len(r.Refs))

		for _, dr := range r.Refs {
			gt.Items = append(gt.Items, rootTreeDynamic(&dr, pack))
		}

	}

	tree = gotree.StringTree(&gt)
	return

}

func rootTreeDynamic(d *Dynamic, pack Pack) (it *gotree.GTStructure) {

	it = new(gotree.GTStructure)

	if d.IsValid() == false {
		it.Name = "*(dynamic) err: " + ErrInvalidDynamicReference.Error()
		return
	}

	if d.IsBlank() == true {
		it.Name = "*(dynamic) nil"
		return
	}

	var (
		sch Schema
		err error
	)

	if sch, err = pack.Registry().SchemaByReference(d.Schema); err != nil {
		it.Name = "*(dynamic) err: " + err.Error()
		return
	}

	if d.Hash == (cipher.SHA256{}) {
		it.Name = fmt.Sprintf("*(dynamic) nil (type %s)", sch.String())
		return
	}

	it.Name = "*(dynamic) " + d.Short()
	it.Items = []*gotree.GTStructure{
		rootTreeHash(pack, sch, d.Hash),
	}

	return
}

func rootTreeHash(
	pack Pack,
	sch Schema,
	hash cipher.SHA256,
) (
	it *gotree.GTStructure,
) {

	var (
		val []byte
		err error
	)

	it = new(gotree.GTStructure)

	if val, err = pack.Get(hash); err != nil {
		it.Name = "(err) " + err.Error()
		return
	}

	return rootTreeData(pack, sch, val)
}

func rootTreeData(
	pack Pack,
	sch Schema,
	val []byte,
) (
	it *gotree.GTStructure,
) {

	if sch.IsReference() == true {
		return rootTreeReferences(pack, sch, val)
	}

	switch sch.Kind() {
	case reflect.Bool:

		return rootTreeBool(sch, val)

	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		return rootTreeInt(sch, val)

	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:

		return rootTreeUint(sch, val)

	case reflect.Float32, reflect.Float64:

		return rootTreeFloat(sch, val)

	case reflect.String:

		return rootTreeString(sch, val)

	case reflect.Array, reflect.Slice:

		return rootTreeSlice(pack, sch, val)

	case reflect.Struct:

		return rootTreeStruct(pack, sch, val)

	default:

		it = new(gotree.GTStructure)
		it.Name = fmt.Sprintf("(err) invalid Kind <%s> of Schema %q",
			sch.Kind().String(),
			sch.String())

	}

	return
}

func rootTreeReferences(
	pack Pack, //             :
	sch Schema, //            :
	val []byte, //            :
) (
	it *gotree.GTStructure, // :
) {

	switch rt := sch.ReferenceType(); rt {

	case ReferenceTypeSingle:

		return rootTreeRef(pack, sch, val)

	case ReferenceTypeSlice:

		return rootTreeRefs(pack, sch, val)

	case ReferenceTypeDynamic:

		var (
			dr  Dynamic
			err error
		)

		it = new(gotree.GTStructure)

		if err = encoder.DeserializeRaw(val, &dr); err != nil {
			it.Name = "*(dynamic) err: " + err.Error()
			return
		}

		return rootTreeDynamic(&dr, pack)

	default:

		it = new(gotree.GTStructure)

		it.Name = fmt.Sprintf(
			"invalid schema (%s): reference with invalid type %d",
			sch.String(),
			rt,
		)
	}

	return
}

func rootTreeValue(sch Schema, val interface{}) (it *gotree.GTStructure) {

	var name string

	it = new(gotree.GTStructure)

	if name = sch.Name(); name != "" {
		it.Name = fmt.Sprintf("%v (type %s)", val, name)
		return
	}

	it.Name = fmt.Sprint(val)
	return

}

func rootTreeBool(sch Schema, val []byte) (it *gotree.GTStructure) {

	var (
		x   bool
		err error
	)

	if err = encoder.DeserializeRaw(val, &x); err != nil {
		it = new(gotree.GTStructure)
		it.Name = "(err) " + err.Error()
		return
	}

	return rootTreeValue(sch, x)
}

func rootTreeInt(sch Schema, val []byte) (it *gotree.GTStructure) {

	var (
		x   int64
		err error
	)

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
		it = new(gotree.GTStructure)
		it.Name = "(err) " + err.Error()
		return
	}

	return rootTreeValue(sch, x)
}

func rootTreeUint(sch Schema, val []byte) (it *gotree.GTStructure) {

	var (
		x   uint64
		err error
	)

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
		it = new(gotree.GTStructure)
		it.Name = "(err) " + err.Error()
		return
	}

	return rootTreeValue(sch, x)
}

func rootTreeFloat(sch Schema, val []byte) (it *gotree.GTStructure) {

	var (
		x   float64
		err error
	)

	switch sch.Kind() {
	case reflect.Float32:

		var y float32
		err = encoder.DeserializeRaw(val, &y)
		x = float64(y)

	case reflect.Float64:

		err = encoder.DeserializeRaw(val, &x)

	}

	if err != nil {
		it = new(gotree.GTStructure)
		it.Name = "(err) " + err.Error()
		return
	}

	return rootTreeValue(sch, x)
}

func rootTreeString(
	sch Schema,
	val []byte,
) (
	it *gotree.GTStructure,
) {

	var (
		x   string
		err error
	)

	if err = encoder.DeserializeRaw(val, &x); err != nil {
		it = new(gotree.GTStructure)
		it.Name = "(err) " + err.Error()
		return
	}

	return rootTreeValue(sch, x)
}

// slice or array
func rootTreeSlice(pack Pack, sch Schema, val []byte) (it *gotree.GTStructure) {

	var (
		el  Schema
		err error
	)

	it = new(gotree.GTStructure)

	if el = sch.Elem(); el == nil {
		it.Name = fmt.Sprintf("(err) invalid schema %q: nil-element",
			sch.String())
		return
	}

	// special case for []byte
	if sch.Kind() == reflect.Slice && el.Kind() == reflect.Uint8 {

		var x []byte
		if err = encoder.DeserializeRaw(val, &x); err != nil {
			it.Name = "(err) " + err.Error()
			return
		}

		it.Name = "([]byte) " + hex.EncodeToString(x)
		return
	}

	var (
		ln    int // length
		shift int // shift
	)

	if sch.Kind() == reflect.Array {

		ln = sch.Len()

		if name := sch.Name(); name != "" {
			it.Name = fmt.Sprintf("[%d]%s (%s)", ln, el.String(), name)
		} else {
			it.Name = fmt.Sprintf("[%d]%s", ln, el.String())
		}

	} else {

		// reflect.Slice
		if name := sch.Name(); name != "" {
			it.Name = fmt.Sprintf("[]%s (%s)", el.String(), name)
		} else {
			it.Name = fmt.Sprintf("[]%s", el.String())
		}

		if ln, err = getLength(val); err != nil {
			it.Name += " (err) " + err.Error()
			return
		}

		it.Name += fmt.Sprintf(" (length %d)", ln)
		shift = 4

	}

	var m, s, k int

	if s = fixedSize(el.Kind()); s < 0 {

		for k = 0; k < ln; k++ {

			if shift+s > len(val) {
				it.Items = append(it.Items, &gotree.GTStructure{
					Name: fmt.Sprintf(
						"(err) unexpected end of %s at %d element",
						sch.Kind().String(), k),
				})
				return
			}

			it.Items = append(
				it.Items,
				rootTreeData(pack, el, val[shift:shift+s]),
			)
			shift += s

		}

	} else {

		for k = 0; k < ln; k++ {

			if shift > len(val) {
				it.Items = append(it.Items, &gotree.GTStructure{
					Name: fmt.Sprintf(
						"(err) unexpected end of %s at %d element",
						sch.Kind().String(), k),
				})
				return
			}

			if m, err = el.Size(val[shift:]); err != nil {
				return
			}

			it.Items = append(
				it.Items,
				rootTreeData(pack, el, val[shift:shift+m]),
			)
			shift += m

		}

	}

	return
}

func rootTreeStruct(
	pack Pack,
	sch Schema,
	val []byte,
) (
	it *gotree.GTStructure,
) {

	var (
		shift int
		s     int
		err   error
	)

	it = new(gotree.GTStructure)
	it.Name = sch.String()

	for _, f := range sch.Fields() {

		if shift > len(val) {

			// detailed error
			it.Name = fmt.Sprintf(
				"(err) unexpected end of encoded struct '%s' "+
					"at field '%s', schema of field: '%s'",
				sch.String(),
				f.Name(),
				f.Schema().String())
			return

		}

		if s, err = f.Schema().Size(val[shift:]); err != nil {
			it.Name = "(err) " + err.Error()
			return
		}

		fit := rootTreeData(pack, f.Schema(), val[shift:shift+s])
		fit.Name = f.Name() + ": " + fit.Name // field name
		it.Items = append(it.Items, fit)
		shift += s

	}

	return
}

func rootTreeRef(pack Pack, sch Schema, val []byte) (it *gotree.GTStructure) {

	var (
		ref Ref
		el  Schema
		err error
	)

	it = new(gotree.GTStructure)

	if el = sch.Elem(); el == nil {

		it.Name = fmt.Sprintf(
			"*(<ref>) err: missing schema of element: %s ",
			sch,
		)

		return
	}

	if err = encoder.DeserializeRaw(val, &ref); err != nil {
		it.Name = fmt.Sprintf("*(%s) err: %s", el.String(), err.Error())
		return
	}

	if ref.Hash == (cipher.SHA256{}) {
		it.Name = fmt.Sprintf("*(%s) nil", el.String())
		return
	}

	it.Name = fmt.Sprintf("*(%s) %s", el.String(), ref.Short())
	it.Items = []*gotree.GTStructure{
		rootTreeHash(pack, el, ref.Hash),
	}

	return
}

func rootTreeRefs(pack Pack, sch Schema, val []byte) (it *gotree.GTStructure) {

	var (
		refs Refs
		el   Schema
		err  error
	)

	it = new(gotree.GTStructure)

	if el = sch.Elem(); el == nil {
		it.Name = "[]*(<refs>) err: missing schema of element"
		return
	}

	if err = encoder.DeserializeRaw(val, &refs); err != nil {
		it.Name = fmt.Sprintf("[]*(-----) %s err: %s", el.String(), err.Error())
		return
	}

	// initialize first

	var ln int
	if ln, err = refs.Len(pack); err != nil {
		it.Name = fmt.Sprintf("[]*(%s) %s err: %s", el.String(), refs.Short(),
			err.Error())
		return
	}

	// can be blank

	if ln == 0 {
		it.Name = fmt.Sprintf("[]*(%s) %s nil", el.String(), refs.Short())
		return
	}

	it.Name = fmt.Sprintf("[]*(%s) %s length: %d", el.String(), refs.Short(),
		ln)

	it.Items = make([]*gotree.GTStructure, 0, ln)

	err = refs.Walk(pack, el, func(
		hash cipher.SHA256,
		depth int,
	) (
		deepper bool,
		err error,
	) {

		if depth != 0 {
			return true, nil
		}

		it.Items = append(it.Items, rootTreeHash(pack, el, hash))
		return true, nil
	})

	if err != nil {
		it.Items = append(it.Items, &gotree.GTStructure{
			Name: "err: " + err.Error(),
		})
	}

	return
}
