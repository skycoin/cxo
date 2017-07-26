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
	IsNil() bool

	Kind() reflect.Kind
	Schema() Schema

	Dereference() (val *Value)

	Len() (l int)
	RangeIndex(rif RangeIndexFunc)
	Index(i int) (val *Value)

	FieldNum() int
	Fields() (fs []string)
	FieldByName(name string) (val *Value)
	FieldByIndex(i int) (val *Value)
	RangeFields(rff RangeFieldsFunc)

	Int() (i int64)
	Uint() (u uint64)
	Float() (f float64)
	String() (s string)
	Bytes() (p []byte)
	Bool() (b bool)
}

// value represtns any value
type value struct {
	data   []byte // encoded object
	schema Schema // schema of the Value
}

// IsNil retusn true if this Value represents nil
func (v *Value) IsNil() bool {
	return v.data == nil
}

// Kind is Kind of Schema of the Value. But it returns relfect.Prt
// if the value keeps Reference of Dynamic (it returns relfect.Slice
// if value is References)
func (v *Value) Kind() reflect.Kind {
	if v.schema.IsReference() && v.schema.Kind() != reflect.Slice {
		return reflect.Ptr // single or dynamic
	}
	return v.schema.Kind()
}

// Data of the Value (encoded object)
func (v *Value) Data() []byte {
	return v.data
}

// Schema of the Value
func (v *Value) Schema() Schema {
	return v.schema
}

// Reference

// Static retursn Reference to obejct
func (v *Value) Static() (ref Reference, err error) {
	err = encoder.DeserializeRaw(v.data, &ref)
	return
}

// Dynamic returns Dynamic reference to an obejct
func (v *Value) Dynamic() (dr Dynamic, err error) {
	err = encoder.DeserializeRaw(v.data, &dr)
	return
}

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

// RangeIndexFunc used to itterate over array or slcie
type RangeIndexFunc func(i int, val *Value) error

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

// RangeFieldsFunc used to itterate over fields of a struct
type RangeFieldsFunc func(name string, val *Value) error

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

// obtain

// Int returns decoded int8, int16, int32, and int64
func (v *Value) Int() (i int64, err error) {
	switch v.Kind() {
	case reflect.Int8:
		var t int8
		err = encoder.DeserializeRaw(v.data, &t)
		i = int64(t)
	case reflect.Int16:
		var t int16
		err = encoder.DeserializeRaw(v.data, &t)
		i = int64(t)
	case reflect.Int32:
		var t int32
		err = encoder.DeserializeRaw(v.data, &t)
		i = int64(t)
	case reflect.Int64:
		err = encoder.DeserializeRaw(v.data, &i)
	default:
		err = ErrInvalidType
	}
	return
}

// Uint return decoded uint8, uint16, uint32, and uint64
func (v *Value) Uint() (u uint64, err error) {
	switch v.Kind() {
	case reflect.Uint8:
		var t uint8
		err = encoder.DeserializeRaw(v.data, &t)
		u = uint64(t)
	case reflect.Uint16:
		var t uint16
		err = encoder.DeserializeRaw(v.data, &t)
		u = uint64(t)
	case reflect.Uint32:
		var t uint32
		err = encoder.DeserializeRaw(v.data, &t)
		u = uint64(t)
	case reflect.Uint64:
		err = encoder.DeserializeRaw(v.data, &u)
	default:
		err = ErrInvalidType
	}
	return
}

// Float returns decoded float32 or float64
func (v *Value) Float() (f float64, err error) {
	switch v.Kind() {
	case reflect.Float32:
		var t float32
		err = encoder.DeserializeRaw(v.data, &t)
		f = float64(t)
	case reflect.Float64:
		err = encoder.DeserializeRaw(v.data, &f)
	default:
		err = ErrInvalidType
	}
	return
}

// String returns decoded string
func (v *Value) String() (s string, err error) {
	if v.Kind() == reflect.String {
		err = encoder.DeserializeRaw(v.data, &s)
	} else {
		err = ErrInvalidType
	}
	return
}

// Bytes returns decoded []byte
func (v *Value) Bytes() (p []byte, err error) {
	sch := v.Schema()
	if sch.Kind() == reflect.Slice && sch.Elem().Kind() == reflect.Uint8 {
		err = encoder.DeserializeRaw(v.data, &p)
	} else if sch.Kind() == reflect.String {
		var s string
		if err = encoder.DeserializeRaw(v.data, &s); err != nil {
			return
		}
		p = []byte(s)
	} else {
		err = ErrInvalidType
	}
	return
}

// Bool retursn decoded bool
func (v *Value) Bool() (b bool, err error) {
	if v.Kind() != reflect.Bool {
		err = ErrInvalidType
		return
	}
	err = encoder.DeserializeRaw(v.data, &b)
	return
}

//
// schema size
//

// SchemaSize returns size that holds encoded data of the schema
func SchemaSize(s Schema, p []byte) (n int, err error) {
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

func schemaSliceSize(s Schema, p []byte) (n int, err error) {
	if s.IsReference() {
		if n, err = getLength(p); err == nil {
			n *= len(Reference{})
			n += 4 // length prefix
		}
		return
	}
	var l int
	if l, err = getLength(p); err != nil {
		return
	}
	n += 4
	el := s.Elem()
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

func schemaArraySize(s Schema, p []byte) (n int, err error) {
	if s.IsReference() { // Reference
		n = len(cipher.SHA256{})
		return
	}
	l := s.Len()
	el := s.Elem()
	if s := fixedSize(el.Kind()); s > 0 {
		n = l * s
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

func schemaStructSize(s Schema, p []byte) (n int, err error) {
	if s.IsReference() { // Dynamic
		n = 2 * len(cipher.SHA256{})
		return
	}
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

func getLength(p []byte) (l int, err error) {
	var u uint32
	err = encoder.DeserializeRaw(p, &u)
	l = int(u)
	return
}

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
