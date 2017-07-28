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
		// special case for referecnes
	}

	switch sch.Kind() {
	case reflect.Bool:
		obj, err = newBoolValue(sch, val)
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		obj, err = newIntValue(sch, val)
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		obj, err = newUintValue(sch, val)
	case reflect.Float32, reflect.Float64:
		obj, err = newFloatValue(sch, val)
	case reflect.String:
		obj, err = newStringValue(sch, val)
	case reflect.Array, reflect.Slice:
		obj, err = nweSliceValue(sch, val)
	case reflect.Struct:
		//
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

func newSliceValue(sch Schema, val []byte) (obj Value, err error) {

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

	return
}
