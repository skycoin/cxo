package skyobject

import (
	"reflect"
)

type value struct{}

func (value) Kind() reflect.Kind { return reflect.Invalid }
func (value) Schema() Schema     { return nilSchema }

func (value) Dereference() Value { return nil }

func (value) Len() (ln int)                   { return }
func (value) RangeIndex(RangeIndexFunc) error { return ErrInvalidType }
func (value) Index(int) Value                 { return nil }

func (value) FieldNum() (n int)           { return }
func (value) Fields() (fs []string)       { return }
func (value) FieldByName(string) Value    { return nil }
func (value) FieldByIndex(int) Value      { return nil }
func (value) RangeFields(RangeFieldsFunc) { return ErrInvalidType }

func (value) Int() (int64, error)     { return 0, ErrInvalidType }
func (value) Uint() (uint64, error)   { return 0, ErrInvalidType }
func (value) Float() (float64, error) { return 0, ErrInvalidType }
func (value) String() (string, error) { return "", ErrInvalidType }
func (value) Bytes() ([]byte, error)  { return nil, ErrInvalidType }
func (value) Bool() (bool, error)     { return false, ErrInvalidType }

type valueSchema struct {
	value
	schema Schema
}

func (v *valueSchema) Kind() reflect.Kind {
	if v.schema.IsReference() && v.schema.Kind() != reflect.Slice {
		return reflect.Ptr // single or dynamic
	}
	return v.schema.Kind()
}

func (v *valueSchema) Schema() Schema {
	return v.schema
}

type intValue struct {
	valueSchema
	val int64
}

func (i *intValue) Int() int64 { return i.val }

type uintValue struct {
	valueSchema
	val uint64
}

func (u *uintValue) Uint() uint64 {
	return u.val
}

type floatValue struct {
	valueSchema
	val float64
}

func (f *floatValue) Float() float64 {
	return f.val
}

type stringValue struct {
	valueSchema
	val []byte
}

func (s *stringValue) String() string {
	return string(s.val)
}

func (s *stringValue) Bytes() []byte {
	return s.val
}

type bytesValue struct {
	valueSchema
	val []byte
}

func (b *bytesValue) Bytes() []byte {
	return b.val
}

type boolValue struct {
	valueSchema
	val bool
}

func (b *boolValue) Bool() bool {
	return b.val
}

type structField struct {
	name string
	val  Value
}

type structValue struct {
	valueSchema
	fields []structField
}

func (v *structValue) FieldNum() int {
	return len(v.fields)
}

func (v *structValue) Fields() (fs []string) {
	if len(v.fields) == 0 {
		return
	}
	fs = make([]stirng, 0, len(v.fields))
	for _, f := range v.fields {
		fs = append(fs, f.name)
	}
	return
}

func (v *structValue) FieldByName(name string) Value {
	for _, f := range v.fields {
		if f.name == name {
			return f.val
		}
	}
	return nil
}

func (v *structValue) FieldByIndex(i int) Value {
	if i >= 0 && i < len(v.fields) {
		return v.fields[i].val
	}
	return nil
}

func (v *structValue) RangeFields(rff RangeFieldsFunc) (err error) {
	for _, f := range v.fields {
		if err = rff(f.name, f.val); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return
		}
	}
	return
}

type sliceValue struct {
	valueSchema
	vals []Value
}

func (s *sliceValue) Len() int {
	return len(s.vals)
}

func (s *sliceValue) RangeIndex(rif RangeIndexFunc) (err error) {
	for i, v := range s.vals {
		if err = rif(i, v); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return
		}
	}
	return
}

func (s *sliceValue) Index(i int) Value {
	if i >= 0 && i < len(s.vals) {
		return s.vals[i]
	}
	return nil
}

type ptrValue struct {
	valueSchema
	value Value // dereference
}

func (p *ptrValue) Dereference() Value {
	return p.value
}
