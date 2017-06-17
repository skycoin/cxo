package skyobject

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

var (
	// ErrInvalidType occurs when you tries to register any unacceptable type
	ErrInvalidType = errors.New("invalid type")
)

var (
	singleRef  = typeOf(Reference{})
	sliceRef   = typeOf(References{})
	dynamicRef = typeOf(Dynamic{})
)

// A ReferenceType represents type of a reference
type ReferenceType int

// possible reference
const (
	ReferenceTypeNone    ReferenceType = iota
	ReferenceTypeSingle                // Reference (cipher.SHA256)
	ReferenceTypeSlice                 // References ([]Reference)
	ReferenceTypeDynamic               // Dynamic (struct{Object, Schema Ref.})
)

// A Schema represents schema of a CX object
type Schema interface {
	// Reference of the Schema. The reference is valid after call of Done of
	// Registry by which the Schema created
	Reference() SchemaReference

	IsReference() bool            // is the schema is reference
	ReferenceType() ReferenceType // type of the reference if it is a reference

	IsNil() bool // is the schema a nil

	Kind() reflect.Kind // Kind of the Schema
	Name() string       // Name of the Schema if named
	Len() int           // Length if array
	Fields() []Field    // Fields if struct
	// Elem if array, slice or pointer (reference). The Elem returns nil
	// for other types and if it's Dynamic reference (because schema of
	// element is not specified by schema)
	Elem() (s Schema)

	RawName() []byte    // raw name if named
	IsRegistered() bool // is registered or not

	Encode() (b []byte) // encode the schema

	fmt.Stringer // String() string
}

var nilSchema Schema = &schema{kind: reflect.Invalid}

// schema core

type schema struct {
	ref SchemaReference

	kind reflect.Kind
	name []byte
}

func (s *schema) IsNil() bool {
	return s.kind == reflect.Invalid
}

func (s *schema) IsReference() bool {
	return false
}

func (s *schema) ReferenceType() ReferenceType {
	return ReferenceTypeNone // not a reference
}

func (s *schema) Reference() SchemaReference {
	if s.ref == (SchemaReference{}) {
		s.ref = SchemaReference(cipher.SumSHA256(s.Encode()))
	}
	return s.ref
}

func (s *schema) Kind() reflect.Kind {
	return s.kind
}

func (s *schema) Name() string {
	return string(s.name)
}

func (s *schema) RawName() []byte {
	return s.name
}

func (s *schema) IsRegistered() bool {
	return s.kind == reflect.Struct && len(s.name) > 0
}

func (s *schema) Len() int {
	return 0
}

func (s *schema) Fields() []Field {
	return nil
}

func (s *schema) Elem() Schema {
	return nil
}

func (s *schema) encodedSchema() (x encodedSchema) {
	x.Kind = uint32(s.kind)
	x.Name = s.name
	return
}

func (s *schema) Encode() (b []byte) {
	b = encoder.Serialize(s.encodedSchema())
	return
}

func (s *schema) String() string {
	if s == nil {
		return "<missing>"
	}
	if len(s.name) > 0 {
		return s.Name()
	}
	return s.kind.String()
}

// reference

type referenceSchema struct {
	schema

	typ  ReferenceType
	elem Schema
}

func (r *referenceSchema) IsReference() bool {
	return true
}

func (r *referenceSchema) ReferenceType() ReferenceType {
	return r.typ
}

func (r *referenceSchema) Elem() Schema {
	return r.elem
}

func (r *referenceSchema) encodedSchema() (x encodedSchema) {
	x.Kind = uint32(r.kind)
	x.RefTyp = uint32(r.typ)
	// the schema of the Elem is registered allways
	if r.typ != ReferenceTypeDynamic {
		x.Elem = (&schema{
			SchemaReference{},
			r.elem.Kind(),
			r.elem.RawName(),
		}).Encode()
	}
	return
}

func (r *referenceSchema) Encode() (b []byte) {
	b = encoder.Serialize(r.encodedSchema())
	return
}

func (r *referenceSchema) String() string {
	if r == nil {
		return "<missing>"
	}
	switch r.typ {
	case ReferenceTypeSingle:
		return fmt.Sprintf("*%s", r.Elem().String())
	case ReferenceTypeSlice:
		return fmt.Sprintf("[]*%s", r.Elem().String())
	case ReferenceTypeDynamic:
		return "*<dynamic>"
	}
	return "<invalid>"
}

// slice

type sliceSchema struct {
	schema
	elem Schema
}

func (s *sliceSchema) Elem() Schema {
	return s.elem
}

func (s *sliceSchema) encodedSchema() (x encodedSchema) {
	x = s.schema.encodedSchema()
	if el := s.elem; el.IsRegistered() {
		x.Elem = (&schema{SchemaReference{}, el.Kind(), el.RawName()}).Encode()
	} else {
		x.Elem = s.elem.Encode()
	}
	return
}

func (s *sliceSchema) Encode() (b []byte) {
	b = encoder.Serialize(s.encodedSchema())
	return
}

func (s *sliceSchema) String() string {
	if s == nil {
		return "<missing>"
	}
	if len(s.name) > 0 {
		return s.Name()
	}
	return "[]" + s.elem.String()
}

// array

type arraySchema struct {
	sliceSchema
	length int
}

func (a *arraySchema) Len() int {
	return a.length
}

func (a *arraySchema) encodedSchema() (x encodedSchema) {
	x = a.sliceSchema.encodedSchema()
	x.Len = uint32(a.length)
	return
}

func (a *arraySchema) Encode() (b []byte) {
	b = encoder.Serialize(a.encodedSchema())
	return
}

func (a *arraySchema) String() string {
	if a == nil {
		return "<missing>"
	}
	if len(a.name) > 0 {
		return a.Name()
	}
	return fmt.Sprintf("[%d]%s", a.length, a.elem.String())
}

// struct

type structSchema struct {
	schema
	fields []Field
}

func (s *structSchema) Fields() []Field {
	return s.fields
}

func (s *structSchema) encodedSchema() (x encodedSchema) {
	x = s.schema.encodedSchema()
	if len(s.fields) == 0 {
		return
	}
	x.Fields = make([][]byte, 0, len(s.fields))
	for _, f := range s.fields {
		x.Fields = append(x.Fields, f.Encode())
	}
	return
}

func (s *structSchema) Encode() (b []byte) {
	b = encoder.Serialize(s.encodedSchema())
	return
}

//
// TODO:
//  (1) rid out of simpleField
//  (2) merge coreField and field
//  (3) make Field (interface) to be *Field (struct)
//
// Because simpleField can't be created from encodedField.
// And (Field).Kind() never used to be a big advantag of
// permormance and memory
//
// And, creating Schema from simpleField adds some memory
// and GC pressure
//

// field

// A Field represetns struct field
type Field interface {
	Schema() Schema     // Schema of the Field
	Kind() reflect.Kind // kind of the Field (short hand)

	Name() string    // Name of the Filed
	RawName() []byte // raw name of the Filed

	Tag() reflect.StructTag // Tag of the Filed
	RawTag() []byte         // raw tag of the Field

	Encode() (b []byte) // Encode field

	fmt.Stringer // String() string
}

// core

type field struct {
	name   []byte
	tag    []byte
	schema Schema
}

func (f *field) Name() string {
	return string(f.name)
}

func (f *field) RawName() []byte {
	return f.name
}

func (f *field) Tag() reflect.StructTag {
	return reflect.StructTag(f.tag)
}

func (f *field) RawTag() []byte {
	return f.tag
}

func (f *field) Schema() Schema {
	return f.schema
}

func (f *field) Kind() reflect.Kind {
	return f.schema.Kind()
}

func (f *field) encodedField() (x encodedField) {
	x.Name = f.name
	x.Tag = f.tag
	x.Schema = f.schema.Encode()
	return
}

func (f *field) Encode() (b []byte) {
	b = encoder.Serialize(f.encodedField())
	return
}

func (f *field) String() string {
	return fmt.Sprintf("%s %s `%s`", f.Name(), f.Schema().String(), f.Tag())
}

// encoded

type encodedSchema struct {
	RefTyp uint32
	Kind   uint32
	Name   []byte
	Len    uint32
	Fields [][]byte
	Elem   []byte // encoded schema
}

type encodedField struct {
	Name   []byte
	Tag    []byte
	Kind   uint32
	Schema []byte
}
