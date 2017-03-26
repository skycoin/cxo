package skyobject

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher/encoder"
)

//
// TODO: DRY
//

var (
	// ErrInvalidType occurs when you call some mehtod like (*Value).Int()
	// when the value is not integer, etc
	ErrInvalidType = errors.New("invalid type of value")
	// ErrNoSuchField relates to (*Value).FieldByName()
	ErrNoSuchField = errors.New("no such field")
	// ErrInvalidSchemaOrData occurs any time encoding of a data failed
	ErrInvalidSchemaOrData = errors.New("invalid schema or data")
	// ErrIndexOutOfRange relates to (*Value).Index()
	ErrIndexOutOfRange = errors.New("index out of range")

	// ErrStopRange is used to stop range function
	ErrStopRange = errors.New("stop range")
)

// MissingObject occurs when a schema is not in db
type MissingSchema struct {
	key Reference
}

// Key return key of missing schema
func (m *MissingSchema) Key() Reference {
	return m.key
}

// Error implements error interface
func (m *MissingSchema) Error() string {
	return "missing schema: " + m.key.String()
}

// MissingObject occurs when an object is not in db
type MissingObject struct {
	key        Reference
	schemaName string
}

// Key returns kye of missing object
func (m *MissingObject) Key() Reference {
	return m.key
}

// Error implements error interface
func (m *MissingObject) Error() string {
	return fmt.Sprintf("missing object %q: %s", m.schemaName, m.key.String())
}

// ========================================================================== //
//                                                                            //
//                                data value                                  //
//                                                                            //
// ========================================================================== //

// A Value represent any value from tree of objects
type Value struct {
	r  *Root // back reference
	s  *Schema
	od []byte
}

// Kind returns reflect.Kind of value:
//  - reflect.Ptr for references
//  - reflect.Invalid for nil-values
//  ...and appropriate kinds for other values
// Use (*Value).Schema().Kind() to get appropriate schema of value if the
// value is nil. The schema can be nil-schema for nil-dynamic-references.
// But, be careful, the (*Value).Schema().Kind() returns relfect.Ptr
// instead of reflect.Slice for array of references
func (x *Value) Kind() reflect.Kind {
	var kind reflect.Kind = x.s.Kind()
	if kind == reflect.Ptr && x.s.Name() == arrayRef {
		return reflect.Slice // treat a slice of references as slice
	}
	if x.od == nil { // nil
		return reflect.Invalid
	}
	return kind
}

// ========================================================================== //
//                                references                                  //
// ========================================================================== //

// Dereference returns value by reference. A value is reference if
// its kind is reflect.Ptr
func (x *Value) Dereference() (v *Value, err error) {
	if x.Kind() != reflect.Ptr {
		err = ErrInvalidType
		return
	}
	switch x.s.Name() {
	case dynamicRef:
		var dr Dynamic
		if err = encoder.DeserializeRaw(x.od, &dr); err != nil {
			return
		}
		if !dr.IsValid() {
			err = ErrInvalidReference
			return
		}
		if dr.IsBlank() {
			v = nilValue(x.r, nil) // no value nor schema
			return
		}
		var (
			s Schema

			sd, od []byte
			ok     bool
		)
		if sd, ok = x.r.cnt.get(dr.Schema); !ok {
			err = &MissingSchema{dr.Schema}
			return
		}
		if err = s.Decode(x.r.reg, sd); err != nil {
			return
		}
		if od, ok = x.r.cnt.get(dr.Object); !ok {
			err = &MissingObject{key: dr.Object, schemaName: s.Name()}
			return
		}
		v = &Value{x.r, &s, od}
	case singleRef:
		var el *Schema
		if el, err = x.s.Elem(); err != nil {
			return
		}
		var ref Reference
		if err = encoder.DeserializeRaw(x.od, &ref); err != nil {
			return
		}
		if ref.IsBlank() {
			v = nilValue(x.r, el) // with appropriate schema
			return
		}
		var (
			od []byte
			ok bool
		)
		if od, ok = x.r.cnt.get(ref); !ok {
			err = &MissingObject{key: ref, schemaName: x.s.Name()}
			return
		}
		v = &Value{x.r, el, od}
	default:
		err = ErrInvalidType
	}
	return
}

// ========================================================================== //
//                              scalar values                                 //
// ========================================================================== //

// Bool returns bolean of the value or an error
func (x *Value) Bool() (b bool, err error) {
	if x.Kind() != reflect.Bool {
		err = ErrInvalidType
		return
	}
	err = encoder.DeserializeRaw(x.od, &b)
	return
}

// Int returns int64 of the value if type of underlying value is
// one of int8, int16, int32 or int64
func (x *Value) Int() (i int64, err error) {
	switch x.Kind() {
	case reflect.Int8:
		var t int8
		err = encoder.DeserializeRaw(x.od, &t)
		i = int64(t)
	case reflect.Int16:
		var t int16
		err = encoder.DeserializeRaw(x.od, &t)
		i = int64(t)
	case reflect.Int32:
		var t int32
		err = encoder.DeserializeRaw(x.od, &t)
		i = int64(t)
	case reflect.Int64:
		err = encoder.DeserializeRaw(x.od, &i)
	default:
		err = ErrInvalidType
	}
	return
}

// Uint returns int64 of the value if type of underlying value is
// one of uint8, uint16, uint32 or uint64
func (x *Value) Uint() (u uint64, err error) {
	switch x.Kind() {
	case reflect.Uint8:
		var t uint8
		err = encoder.DeserializeRaw(x.od, &t)
		u = uint64(t)
	case reflect.Uint16:
		var t uint16
		err = encoder.DeserializeRaw(x.od, &t)
		u = uint64(t)
	case reflect.Uint32:
		var t uint32
		err = encoder.DeserializeRaw(x.od, &t)
		u = uint64(t)
	case reflect.Uint64:
		err = encoder.DeserializeRaw(x.od, &u)
	default:
		err = ErrInvalidType
	}
	return
}

// String returns string if underlying value is string
func (x *Value) String() (s string, err error) {
	if x.Kind() == reflect.String {
		err = encoder.DeserializeRaw(x.od, &s)
	} else {
		err = ErrInvalidType
	}
	return
}

// Bytes returns []byte of underlying value if the value is []byte or string
func (x *Value) Bytes() (p []byte, err error) {
	if kind := x.Kind(); kind == reflect.Slice {
		var el *Schema
		if el, err = x.s.Elem(); err != nil {
			return // invalid schema
		}
		if el.Kind() == reflect.Uint8 {
			err = encoder.DeserializeRaw(x.od, &p)
		} else {
			err = ErrInvalidType
		}
	} else if kind == reflect.String {
		var s string
		if err = encoder.DeserializeRaw(x.od, &s); err != nil {
			return
		}
		p = []byte(s)
	} else {
		err = ErrInvalidType
	}
	return
}

// Float returns float64 if type of underlying encoded value is float32 or
// float64
func (x *Value) Float() (f float64, err error) {
	switch x.Kind() {
	case reflect.Float32:
		var t float32
		err = encoder.DeserializeRaw(x.od, &t)
		f = float64(t)
	case reflect.Float64:
		err = encoder.DeserializeRaw(x.od, &f)
	default:
		err = ErrInvalidType
	}
	return
}

// ========================================================================== //
//                                structures                                  //
// ========================================================================== //

// Fields returns list of name of all fields. It returns empty slice for
// structs without fields and for non-struct values
func (x *Value) Fields() (fs []string) {
	if len(x.s.Fields()) == 0 {
		return
	}
	fs = make([]string, 0, len(x.s.Fields()))
	for _, sf := range x.s.Fields() {
		fs = append(fs, sf.Name())
	}
	return
}

// FieldByName returns value of the field by given name. It returns
// ErrInvalidType if type of the value is not a struct. It returns
// ErrNoSuchField if field with given name doesn't exist
func (x *Value) FieldByName(name string) (v *Value, err error) {
	if x.Kind() != reflect.Struct {
		err = ErrInvalidType
		return
	}
	var shift int
	for _, sf := range x.s.Fields() {
		var fs *Schema
		if fs, err = sf.Schema(); err != nil {
			return
		}
		if shift >= len(x.od) {
			err = ErrInvalidSchemaOrData
			return
		}
		var n int
		if n, err = fs.Size(x.od[shift:]); err != nil {
			return
		}
		if sf.Name() != name {
			shift += n
			continue
		}
		if shift+n > len(x.od) {
			err = ErrInvalidSchemaOrData
			return
		}
		v = &Value{x.r, fs, x.od[shift : shift+n]}
		return
	}
	err = ErrNoSuchField
	return
}

// RangeFields ...
func (x *Value) RangeFields(fn func(string, *Value) error) (err error) {
	if x.Kind() != reflect.Struct {
		err = ErrInvalidType
		return
	}
	var shift int
	for _, sf := range x.s.Fields() {
		var fs *Schema
		if fs, err = sf.Schema(); err != nil {
			return
		}
		if shift >= len(x.od) {
			err = ErrInvalidSchemaOrData
			return
		}
		var n int
		if n, err = fs.Size(x.od[shift:]); err != nil {
			return
		}
		if shift+n > len(x.od) {
			err = ErrInvalidSchemaOrData
			return
		}
		//
		err = fn(sf.Name(), &Value{x.r, fs, x.od[shift : shift+n]})
		if err != nil {
			if err == ErrStopRange { // special error
				err = nil
			}
			return
		}
		//
		shift += n
	}
	return
}

// ========================================================================== //
//                            slices and arrays                               //
// ========================================================================== //

// Len returns length of array, slice or string. It returns ErrInvalidType
// for another types
func (x *Value) Len() (l int, err error) {
	switch x.Kind() {
	case reflect.Array:
		l = x.s.Len()
	case reflect.String, reflect.Slice:
		l, err = getLength(x.od)
	default:
		err = ErrInvalidType
	}
	return
}

// Index returns value by index. It returns ErrInvalidType if type of the
// value is not array, slice or slice of references (that treated as slice)
func (x *Value) Index(idx int) (v *Value, err error) {
	if idx < 0 {
		err = ErrIndexOutOfRange
		return
	}
	// (*value).Kind returns reflect.Slice for slice of references too,
	// but (*value).s.Kind() returns actual kind of its schema
	switch x.Kind() {
	case reflect.Array:
		var (
			ln int = x.s.Len()
			el *Schema

			shift int
		)
		if idx >= ln {
			err = ErrIndexOutOfRange
			return
		}
		if el, err = x.s.Elem(); err != nil {
			return
		}
		if s := fixedSize(el.Kind()); s > 0 {
			shift = idx * s
			if shift+s > len(x.od) {
				err = ErrInvalidSchemaOrData
				return
			}
			v = &Value{x.r, el, x.od[shift : shift+s]}
		} else {
			var m int
			for ; idx >= 0; idx-- {
				if shift >= len(x.od) {
					err = ErrInvalidSchemaOrData
					return
				}
				if m, err = el.Size(x.od[shift:]); err != nil {
					return
				}
				if idx == 0 {
					if shift+m > len(x.od) {
						err = ErrInvalidSchemaOrData
						return
					}
					v = &Value{x.r, el, x.od[shift : shift+m]}
					break
				}
				shift += m
			}
		}
	case reflect.Slice:
		// any slice or slice of references too;
		// a slice of references must have element that set by field;
		var (
			ln int
			el *Schema

			shift int = 4 // length prefix
		)
		if ln, err = getLength(x.od); err != nil {
			return
		}
		if idx >= ln {
			err = ErrIndexOutOfRange
			return
		}
		if el, err = x.s.Elem(); err != nil {
			return
		}
		if s := fixedSize(el.Kind()); s > 0 {
			shift = idx * s
			if shift+s > len(x.od) {
				err = ErrInvalidSchemaOrData
				return
			}
			v = &Value{x.r, el, x.od[shift : shift+s]}
		} else {
			var m int
			for ; idx >= 0; idx-- {
				if shift >= len(x.od) {
					err = ErrInvalidSchemaOrData
					return
				}
				// real kind
				if x.s.Kind() == reflect.Ptr {
					m = len(Reference{})
				} else {
					if m, err = el.Size(x.od[shift:]); err != nil {
						return
					}
				}
				if idx == 0 {
					if shift+m > len(x.od) {
						err = ErrInvalidSchemaOrData
						return
					}
					// real kind
					if x.s.Kind() == reflect.Ptr {
						el = &Schema{
							kind: uint32(reflect.Ptr),
							name: []byte(singleRef),
							elem: []Schema{*el},
						}
					}
					v = &Value{x.r, el, x.od[shift : shift+m]}
					break
				}
				shift += m
			}
		}
	default:
		err = ErrInvalidType
	}
	return
}

// Schema returns schema of the value. It can be a nil-schema if value got from
// blank dynamic reference. Check it out using (*Schema).IsNil()
func (x *Value) Schema() *Schema {
	return x.s
}

//
// schema size
//

func (s *Schema) Size(p []byte) (n int, err error) {
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
		n += 4
	case reflect.Slice:
		var l int
		if l, err = getLength(p); err != nil {
			return
		}
		n += 4
		var el *Schema
		if el, err = s.Elem(); err != nil {
			return
		}
		if s := fixedSize(el.Kind()); s > 0 {
			n += l * s
		} else {
			var m int
			for i := 0; i < l; i++ {
				if n >= len(p) {
					err = ErrInvalidSchemaOrData
					return
				}
				if m, err = el.Size(p[n:]); err != nil {
					return
				}
				n += m
			}
		}
	case reflect.Array:
		var l int = s.Len()
		var el *Schema
		if el, err = s.Elem(); err != nil {
			return
		}
		if s := fixedSize(el.Kind()); s > 0 {
			n = l * s
		} else {
			var m int
			for i := 0; i < l; i++ {
				if n >= len(p) {
					err = ErrInvalidSchemaOrData
					return
				}
				if m, err = el.Size(p[n:]); err != nil {
					return
				}
				n += m
			}
		}
	case reflect.Struct:
		var m int
		for _, sf := range s.Fields() {
			var ss *Schema
			if ss, err = sf.Schema(); err != nil {
				return
			}
			if n >= len(p) {
				err = ErrInvalidSchemaOrData
				return
			}
			if m, err = ss.Size(p[n:]); err != nil {
				return
			}
			n += m
		}
	case reflect.Ptr:
		switch s.Name() {
		case singleRef:
			n = len(Reference{})
		case arrayRef:
			if n, err = getLength(p); err == nil {
				n *= len(Reference{})
				n += 4 // length prefix
			}
		case dynamicRef:
			n = 2 * len(Reference{})
		default:
			err = ErrInvalidSchema
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

//
// helpers
//

func nilValue(r *Root, s *Schema) *Value {
	if s == nil {
		s = &Schema{}
	}
	return &Value{r, s, nil}
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
