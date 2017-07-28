package skyobject

import (
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// unpack to Value

func (p *Pack) unpackToValue(sch Schema, val []byte) (obj Value, err error) {

	if sch.IsReference() {
		return p.unpackRefereneToValue(sch, val)
	}

	switch sch.Kind() {
	case reflect.Bool:
		return newBoolValue(sch, val)
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return newIntValue(sch, val)
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return newUintValue(sch, val)
	case reflect.Float32, reflect.Float64:
		return newFloatValue(sch, val)
	case reflect.String:
		return newStringValue(sch, val)
	case reflect.Array, reflect.Slice:
		return p.newSliceValue(sch, val)
	case reflect.Struct:
		return newStructValue(sch, val)
	default:
		err = fmt.Errorf("invalid Kind <%s> of Schema %q",
			sch.Kind().String(),
			sch.String())
	}

	return

}

func newBoolValue(sch Schema, val []byte) (obj Value, err error) {
	var x bool
	if err = encoder.DeserializeRaw(val, &x); err != nil {
		return
	}
	obj = &boolValue{value{sch, val}, x}
	return
}

func newIntValue(sch Schema, val []byte) (obj Value, err error) {
	var x int64
	switch sch.Kind() {
	case reflect.Int8:
		var y int8
		if err = encoder.DeserializeRaw(val, &y); err != nil {
			return
		}
		x = int64(y)
	case reflect.Int16:
		var y int16
		if err = encoder.DeserializeRaw(val, &y); err != nil {
			return
		}
		x = int64(y)
	case reflect.Int32:
		var y int32
		if err = encoder.DeserializeRaw(val, &y); err != nil {
			return
		}
		x = int64(y)
	case reflect.Int64:
		if err = encoder.DeserializeRaw(val, &x); err != nil {
			return
		}
	default:
		panic("invalid schema argument for newIntValue(), kind is not Int*:" +
			sch.String())
	}
	obj = &intValue{value{sch, val}, x}
	return
}

func newUintValue(sch Schema, val []byte) (obj Value, err error) {
	var x uint64
	switch sch.Kind() {
	case reflect.Uint8:
		var y uint8
		if err = encoder.DeserializeRaw(val, &y); err != nil {
			return
		}
		x = uint64(y)
	case reflect.Uint16:
		var y uint16
		if err = encoder.DeserializeRaw(val, &y); err != nil {
			return
		}
		x = uint64(y)
	case reflect.Uint32:
		var y uint32
		if err = encoder.DeserializeRaw(val, &y); err != nil {
			return
		}
		x = uint64(y)
	case reflect.Uint64:
		if err = encoder.DeserializeRaw(val, &x); err != nil {
			return
		}
	default:
		panic("invalid schema argument for newUintValue(), kind is not Uint*:" +
			sch.String())
	}
	obj = &uintValue{value{sch, val}, x}
	return
}

func newFloatValue(sch Schema, val []byte) (obj Value, err error) {
	var x float64
	switch sch.Kind() {
	case reflect.Float32:
		var y float32
		if err = encoder.DeserializeRaw(val, &y); err != nil {
			return
		}
		x = float64(y)
	case reflect.Float64:
		if err = encoder.DeserializeRaw(val, &x); err != nil {
			return
		}
	default:
		panic(
			"invalid schema argument for newFloatValue(), kind is not Float*:" +
				sch.String())
	}
	obj = &floatValue{value{sch, val}, x}
	return
}

func newStringValue(sch Schema, val []byte) (obj Value, err error) {
	var x string
	if err = encoder.DeserializeRaw(val, &x); err != nil {
		return
	}
	obj = &stringValue{value{sch, val}, x}
	return
}

func (p *Pack) newSliceValue(sch Schema, val []byte) (obj Value, err error) {

	el := sch.Elem()
	if el == nil {
		err = fmt.Errorf("invalid schema %q: nil-element", sch.String())
		return
	}

	if sch.Kind() == reflect.Slice && el.Kind() == reflect.Uint8 {
		// special case for []byte

		var x []byte
		if err = encoder.DeserializeRaw(val, &x); err != nil {
			return
		}
		obj = bytesValue{value{sch, val}, x}
		return

	}

	var ln int    // length
	var shift int // shift

	if sch.Kind() == reflect.Array {
		ln = sch.Len()
	} else {
		// reflect.Slice
		if ln, err = getLength(val); err != nil {
			return
		}
		shift = 4
	}

	var iobj Value
	var sv sliceValue

	sv.value = value{sch, val}
	sv.vals = make([]Value, 0, ln)

	if s := fixedSize(el.Kind()); s < 0 {
		for i := 0; i < ln; i++ {
			if shift+s > len(val) {
				err = unexpectedEndOfArraySliceError(sch, el, i, ln)
				return
			}
			if iobj, err = p.unpackToValue(el, val[shift:shift+s]); err != nil {
				return
			}
			sv.vals = append(sv.value, iobj)
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
			if iobj, err = p.unpackToValue(el, val[shift:shift+m]); err != nil {
				return
			}
			sv.vals = append(sv.value, iobj)
			shift += m
		}
	}

	obj = &sv

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

func (p *Pack) newStructValue(sch Schema, val []byte) (obj Value, err error) {

	var (
		shift int
		s     int
		fobj  Value       // Value of a field
		sv    structValue // the obj
	)

	sv.value = value{sch, val}
	sv.fields = make([]structField, 0, len(sch.Fields()))

	for _, f := range sch.Fields() {

		if shift >= len(val) {
			// detailed error
			err = fmt.Errorf("unexpected end of encoded struct '%s' "+
				"at field '%s', schema of field: '%s'",
				sch.String(),
				f.Name(),
				f.Schema().String())
			return
		}

		if s, err = SchemaSize(f.Schema(), val[shift:]); err != nil {
			return
		}

		fobj, err = p.unpackToValue(f.Schema(), val, val[shift:shift+s])
		if err != nil {
			return
		}

		sv.fields = append(sv.fields, structField{
			name: f.Name(),
			val:  fobj,
		})

		shift += s

	}

	obj = &sv

	return
}

func (p *Pack) unpackRefereneToValue(sch Schema, val []byte) (obj Value,
	err error) {

	// TODO (kostyarin): implement

	return
}
