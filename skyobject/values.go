package skyobject

import (
	"reflect"
)

type value struct {
	schema Schema
	data   []byte
}

func (v *value) Kind() reflect.Kind {
	sch := v.Schema()
	if sch.IsReference() && sch.ReferenceType() == ReferenceTypeSlice {
		return reflect.Slice
	}
	return sch.Kind()
}

func (v *value) Schema() Schema { return v.schema }
func (v *value) Data() []byte   { return v.data }

func (*value) Dereference() Value { return nil }

func (*value) Len() (ln int)                   { return }
func (*value) RangeIndex(RangeIndexFunc) error { return ErrInvalidType }
func (*value) Index(int) Value                 { return nil }

func (*value) FieldNum() (n int)                 { return }
func (*value) Fields() (fs []string)             { return }
func (*value) FieldByName(string) Value          { return nil }
func (*value) FieldByIndex(int) Value            { return nil }
func (*value) RangeFields(RangeFieldsFunc) error { return ErrInvalidType }

func (*value) Int() (_ int64)     { return }
func (*value) Uint() (_ uint64)   { return }
func (*value) Float() (_ float64) { return }
func (*value) String() (_ string) { return }
func (*value) Bytes() (_ []byte)  { return }
func (*value) Bool() (_ bool)     { return }

type intValue struct {
	value
	val int64
}

func (i *intValue) Int() int64 { return i.val }

type uintValue struct {
	value
	val uint64
}

func (u *uintValue) Uint() uint64 {
	return u.val
}

type floatValue struct {
	value
	val float64
}

func (f *floatValue) Float() float64 {
	return f.val
}

type stringValue struct {
	value
	val []byte
}

func (s *stringValue) String() string {
	return string(s.val)
}

func (s *stringValue) Bytes() []byte {
	return s.val
}

type bytesValue struct {
	value
	val []byte
}

func (b *bytesValue) Bytes() []byte {
	return b.val
}

type boolValue struct {
	value
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
	value
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
	value
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

// TODO: proper references

type ptrValue struct {
	value
	reference Value // dereference
}

func (p *ptrValue) Dereference() Value {
	return p.value
}
