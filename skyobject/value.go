package skyobject

import (
	"errors"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// Value related errors
var (
	ErrInvalidSchemaOrData     = errors.New("invalid schema or data")
	ErrIndexOutOfRange         = errors.New("index out of range")
	ErrNoSuchField             = errors.New("no such field")
	ErrInvalidDynamicReference = errors.New("invalid dynamic reference")
	ErrInvalidSchema           = errors.New("invalid schema")
)

// A Value represents any value including references
type Value interface {
	// Kind of the Value. It returns reflect.Prt for
	// Reference and Dynamic, and refelct.Slice for
	// References
	Kind() reflect.Kind

	Schema() Schema // schema
	Data() []byte   // encoded value

	// references

	Dereference() Value // TOOD (kostyarin): dereference properly

	// arrays and slices (except []byte)

	Len() (ln int)                   // length
	RangeIndex(RangeIndexFunc) error // itterate
	Index(int) Value                 // value by index

	// structures

	FieldNum() (n int)                 // number of fields
	Fields() (fs []string)             // names of fields
	FieldByName(string) Value          // field by name
	FieldByIndex(int) Value            // field by index
	RangeFields(RangeFieldsFunc) error // itterate fields

	// scalar values

	Int() int64     // int8, 16, 32, 64
	Uint() uint64   // uint8, 16, 32, 64
	Float() float64 // flaot32, 64
	String() string // string
	Bytes() []byte  // []byte
	Bool() bool     // bool
}

// RangeIndexFunc used to itterate over array or slcie
type RangeIndexFunc func(i int, val *Value) error

// RangeFieldsFunc used to itterate over fields of a struct
type RangeFieldsFunc func(name string, val *Value) error

//
// utils
//

// SchemaSize returns size used by encoded data of given schema
func SchemaSize(s Schema, p []byte) (n int, err error) {

	if s.IsReference() {

		switch rt := sch.ReferenceType(); rt {
		case ReferenceTypeSingle:
			return len(cipher.SHA256{}) // legth of encoded Reference{}
		case ReferenceTypeSlice:
			n, err = refSize(&References{})
			return
		case ReferenceTypeDynamic:
			return 2 * len(cipher.SHA256{}) // length of encoded Dynamic{}
		}

		err = fmt.Errorf("[ERR] reference with invalid ReferenceType: %d", rt)
		return

	}

	switch s.Kind() {

	case reflect.Bool, reflect.Int8, reflect.Uint8:
		n = 1
	case reflect.Int16, reflect.Uint16:
		n = 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		n = 4
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		n = 8

	case reflect.String:
		if n, err = getLength(p); err != nil {
			return
		}
		n += 4 // encoded length (uint32)

	case reflect.Slice:
		if n, err = schemaSliceSize(s, p); err != nil {
			return
		}

	case reflect.Array:
		if n, err = schemaArraySize(s, p); err != nil {
			return
		}

	case reflect.Struct:
		if n, err = schemaStructSize(s, p); err != nil {
			return
		}

	default:
		err = ErrInvalidSchema
		return
	}

	if n > len(p) {
		err = ErrInvalidSchemaOrData
	}

	return
}

// refSize returns size used by encoded reference;
// ref argument must be pointer to Reference, References
// or Dynamic
func refSize(ref interface{}) (n int, err error) {
	n, err = encoder.DeserializeRawToValue(p, reflect.ValueOf(ref))
	return
}

// schemaArraySize returns size of slice; s argument must be
// kind of slice; the s must not be schema of a reference
func schemaSliceSize(s Schema, p []byte) (n int, err error) {
	var l int
	if l, err = getLength(p); err != nil {
		return
	}
	el := s.Elem()
	n, err = schemaArraySliceSize(el, l, 4, p)
	return
}

// schemaArraySize returns size of array; s argument must be
// kind of array; the s must not be schema of a reference
func schemaArraySize(s Schema, p []byte) (n int, err error) {
	l := s.Len()
	el := s.Elem()
	n, err = schemaArraySliceSize(el, l, 0, p)
	return
}

// schemaArraySliceSize itterates over encoded elements of array or slise
// to get size used by them; l is length of array or slice, shift is
// shift in p slice from which data begins, el is schema of element
func schemaArraySliceSize(el Schema, l, shift int, p []byte) (n int,
	err error) {

	n += shift

	if s := fixedSize(el.Kind()); s > 0 {
		n += l * s
	} else {
		var m int
		for i := 0; i < l; i++ {
			if n >= len(p) {
				err = ErrInvalidSchemaOrData
				return
			}
			if m, err = SchemaSize(el, p[n:]); err != nil {
				return
			}
			n += m
		}
	}
	return
}

// schemaStructSize returns size of structure; the s must be
// kind of struct; the s must not be schema of a reference
func schemaStructSize(s Schema, p []byte) (n int, err error) {
	var m int
	for _, sf := range s.Fields() {
		ss := sf.Schema()
		if n >= len(p) {
			err = ErrInvalidSchemaOrData
			return
		}
		if m, err = SchemaSize(ss, p[n:]); err != nil {
			return
		}
		n += m
	}
	return
}

// getLength of length prefixed values
// (like slice of string)
func getLength(p []byte) (l int, err error) {
	var u uint32
	err = encoder.DeserializeRaw(p, &u)
	l = int(u)
	return
}

// fixedSize returns -1 if given kind represents a
// varialbe size value (like array, slice or struct);
// in other cases it returns appropriate size
// (1, 2, 4 or 8)
func fixedSize(kind reflect.Kind) (n int) {
	switch kind {
	case reflect.Bool, reflect.Int8, reflect.Uint8:
		n = 1
	case reflect.Int16, reflect.Uint16:
		n = 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		n = 4
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		n = 8
	default:
		n = -1
	}
	return
}

//
// TORM (kostyarin): dead code below
//

// Dereference a reference
func (v *Value) Dereference() (val *Value, err error) {
	switch v.Schema().ReferenceType() {
	case ReferenceTypeSingle:
		// var ref Reference
		// if ref, err = v.Static(); err != nil {
		// 	return
		// }
		// if ref.IsBlank() {
		// 	val = &Value{nil, v.Schema().Elem(), v.root}
		// 	return // nil-value with schema
		// }
		// if data, ok := v.root.Get(ref); !ok {
		// 	err = &MissingObjectError{ref}
		// } else {
		// 	val = &Value{data, v.Schema().Elem(), v.root}
		// }
	case ReferenceTypeDynamic:
		// var dr Dynamic
		// if dr, err = v.Dynamic(); err != nil {
		// 	return
		// }
		// val, err = v.root.ValueByDynamic(dr)
	default:
		err = ErrInvalidType
	}
	return
}

// Len of array, slice or string
func (v *Value) Len() (l int, err error) {
	switch v.Kind() {
	case reflect.Array:
		l = v.schema.Len()
	case reflect.String, reflect.Slice:
		l, err = getLength(v.data)
	default:
		err = ErrInvalidType
	}
	return
}

// RangeIndex ranges over arrays and slices. Prefer to use this method
// if you want all elements (because it's faster)
func (v *Value) RangeIndex(rif RangeIndexFunc) (err error) {
	var (
		val   *Value
		el    Schema
		shift int
		ln    int
		s     int // fixed size
		m     int // variable size
	)
	switch v.Kind() {
	case reflect.Array:
		ln, el = v.Schema().Len(), v.Schema().Elem() // shift = 0
		if s = fixedSize(el.Kind()); s > 0 {
			goto fixedSize
		} else {
			goto variableSize
		}
	case reflect.Slice: // including References
		shift = 4 // encoded length
		if ln, err = v.Len(); err != nil {
			return
		}
		el = v.Schema().Elem()
		if v.Schema().IsReference() { // References
			el = &referenceSchema{ // fictive schema
				schema: schema{kind: reflect.Array},
				elem:   el,
				typ:    ReferenceTypeSingle,
			}
			s = len(Reference{})
			goto fixedSize
		}
		if s = fixedSize(el.Kind()); s > 0 {
			goto fixedSize
		} else {
			goto variableSize
		}
	default:
		return ErrInvalidType
	}
fixedSize:
	for i := 0; i < ln; i++ {
		if shift+s > len(v.data) {
			err = ErrInvalidSchemaOrData
			return
		}
		val = &Value{v.data[shift : shift+s], el, v.root}
		if err = rif(i, val); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return
		}
		shift += s
	}
	return
variableSize:
	for i := 0; i < ln; i++ {
		if shift >= len(v.data) {
			err = ErrInvalidSchemaOrData
			return
		}
		if m, err = SchemaSize(el, v.data[shift:]); err != nil {
			return
		}
		val = &Value{v.data[shift : shift+m], el, v.root}
		if err = rif(i, val); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return
		}
		shift += m
	}
	return
}

// Index retusn value by index (for arrays and slices)
func (v *Value) Index(i int) (val *Value, err error) {
	if i < 0 {
		err = ErrIndexOutOfRange
		return
	}
	err = v.RangeIndex(func(j int, x *Value) error {
		if j == i {
			val = x
			return ErrStopRange
		}
		return nil
	})
	if err == nil && val == nil {
		err = ErrIndexOutOfRange
	}
	return
}

// FieldNum returns number of fileds of a struct
func (v *Value) FieldNum() int {
	return len(v.Schema().Fields())
}

// Fields returns all names of fields of the Value
func (v *Value) Fields() (fs []string) {
	ff := v.Schema().Fields()
	if len(ff) == 0 {
		return
	}
	fs = make([]string, 0, len(ff))
	for _, f := range ff {
		fs = append(fs, f.Name())
	}
	return
}

// FieldByName returns struct filed by field name
func (v *Value) FieldByName(name string) (val *Value, err error) {
	err = v.RangeFields(func(n string, x *Value) error {
		if n == name {
			val = x
			return ErrStopRange
		}
		return nil
	})
	if err == nil && val == nil {
		err = ErrNoSuchField
	}
	return
}

// FieldByIndex returns struct field by index
func (v *Value) FieldByIndex(i int) (val *Value, err error) {
	var j int
	err = v.RangeFields(func(_ string, x *Value) error {
		if j == i {
			val = x
			return ErrStopRange
		}
		j++
		return nil
	})
	if err == nil && val == nil {
		err = ErrIndexOutOfRange
	}
	return
}

// RangeFields iterate over all fields of the Value. Prefer this method
// if you want all fields (because it's faster)
func (v *Value) RangeFields(rff RangeFieldsFunc) (err error) {
	if v.Kind() != reflect.Struct {
		err = ErrInvalidType
		return
	}
	var (
		shift int
		val   *Value
		s     int
	)
	for _, f := range v.Schema().Fields() {
		if shift >= len(v.data) {
			err = ErrInvalidSchemaOrData
			return
		}
		if s, err = SchemaSize(f.Schema(), v.data[shift:]); err != nil {
			return
		}
		val = &Value{v.data[shift : shift+s], f.Schema(), v.root}
		if err = rff(f.Name(), val); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return
		}
		shift += s
	}
	return
}
