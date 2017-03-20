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
	ErrStopTraversing      = errors.New("stop traversing")
	ErrInvalidType         = errors.New("invalid type of value")
	ErrNoSuchField         = errors.New("no such field")
	ErrInvalidSchemaOrData = errors.New("invalid schema or data")
	ErrIndexOutOfRange     = errors.New("index out of range")
)

type MissingSchema struct {
	key Reference
}

func (m *MissingSchema) Key() Reference {
	return m.key
}

func (m *MissingSchema) Error() string {
	return "missing schema: " + m.key.String()
}

type MissingObject struct {
	key        Reference
	schemaName string
}

func (m *MissingObject) Key() Reference {
	return m.key
}

func (m *MissingObject) Error() string {
	return fmt.Sprintf("missing object %q: %s", m.schemaName, m.key.String())
}

type Value interface {
	// Kind returns reflect.Kind of value:
	//  - reflect.Ptr for references
	//  - reflect.Invalid for nil-values
	//  ...and appropriate kinds for other values
	Kind() reflect.Kind

	// Dereference returns value by reference. A value is reference if
	// its kind is reflect.Ptr
	Dereference() (v Value, err error)

	// scalar values
	Bool() (b bool, err error)     // reflect.Bool
	Int() (i int64, err error)     // reflect.Int(8|16|32|64)
	Uint() (u uint64, err error)   // reflect.Uint(8|16|32|64)
	String() (s string, err error) // reflect.String
	Bytes() (p []byte, err error)  // reflect.Slice of bytes or reflect.String
	Float() (f float64, err error) // reflect.Ptr

	// TODO: fast range over fields

	// structs
	Fields() []string // names of fields
	FieldByName(name string) (v Value, err error)

	// slices, arrays and slice of references
	Len() (l int, err error)
	Index(idx int) (v Value, err error)

	Schema() *Schema
}

//
// data value
//

type value struct {
	r  *Root // back reference
	s  *Schema
	od []byte
}

func (x *value) Kind() reflect.Kind {
	var kind reflect.Kind = x.s.Kind()
	if kind == reflect.Ptr && x.s.Name() == arrayRef {
		return reflect.Slice // treat a slice of references as slice
	}
	return kind
}

// If
func (x *value) Dereference() (v Value, err error) {
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
		if dr.Schema == (Reference{}) {
			v = nilValue(x.r)
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
		v = &value{x.r, &s, od}
	case singleRef:
		var el *Schema
		if el, err = x.s.Elem(); err != nil {
			return
		}
		var ref Reference
		if err = encoder.DeserializeRaw(x.od, &ref); err != nil {
			return
		}
		if ref == (Reference{}) {
			v = nilValue(x.r)
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
		v = &value{x.r, el, od}
	default:
		err = ErrInvalidType
	}
	return
}

func (x *value) Bool() (b bool, err error) {
	if x.Kind() != reflect.Bool {
		err = ErrInvalidType
		return
	}
	err = encoder.DeserializeRaw(x.od, &b)
	return
}

func (x *value) Int() (i int64, err error) {
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

func (x *value) Uint() (u uint64, err error) {
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
func (x *value) String() (s string, err error) {
	if x.Kind() == reflect.String {
		err = encoder.DeserializeRaw(x.od, &s)
	} else {
		err = ErrInvalidType
	}
	return
}

// Bytes returns []byte of underlying value if the value is []byte or string
func (x *value) Bytes() (p []byte, err error) {
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
func (x *value) Float() (f float64, err error) {
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

// Fields returns list of name of all fields. It returns empty slice for
// structs without fields and for non-struct values
func (x *value) Fields() (fs []string) {
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
func (x *value) FieldByName(name string) (v Value, err error) {
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
		v = &value{x.r, fs, x.od[shift : shift+n]}
		return
	}
	err = ErrNoSuchField
	return
}

// Len returns length of array, slice or string. It returns ErrInvalidType
// for another types
func (x *value) Len() (l int, err error) {
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
func (x *value) Index(idx int) (v Value, err error) {
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
			v = &value{x.r, el, x.od[shift : shift+s]}
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
					v = &value{x.r, el, x.od[shift : shift+m]}
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
			v = &value{x.r, el, x.od[shift : shift+s]}
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
					v = &value{x.r, el, x.od[shift : shift+m]}
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

func (x *value) Schema() *Schema {
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

func nilValue(r *Root) Value {
	return &value{r, &Schema{kind: uint32(reflect.Invalid)}, nil}
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
