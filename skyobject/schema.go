package skyobject

import (
	"reflect"
	"strings"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

const TAG = "skyobject"

// href types
var (
	htSingle  = typeName(reflect.TypeOf(cipher.SHA256{}))
	htArray   = typeName(reflect.TypeOf([]cipher.SHA256{}))
	htDynamic = typeName(reflect.TypeOf(Dynamic{}))
)

type schemaReg struct {
	db  *data.DB                 // store schemas
	reg map[string]cipher.SHA256 // type name -> schema key
}

func (s *schemaReg) schemaByName(name string) (sv *Schema, err error) {
	sk, ok := s.reg[name]
	if !ok {
		err = ErrUnregisteredSchema
		return
	}
	sv, err = s.schemaByKey(sk)
	return
}

func (s *schemaReg) schemaByKey(sk cipher.SHA256) (sv *Schema, err error) {
	data, ok := s.db.Get(sk)
	if !ok {
		err = ErrMissingInDB
	}
	err = encoder.DeserializeRaw(data, &sv)
	return
}

//
// schema head
//

type schemaHead struct {
	kind     uint32 // used to filter flat types
	typeName []byte // for named types
}

func (s *schemaHead) Kind() reflect.Kind {
	return reflect.Kind(s.kind)
}

// is the type nemed
func (s *schemaHead) IsNamed() {
	return len(s.typeName) > 0
}

// returns name of the type (even if unnamed)
func (s *schemaHead) Name() (n string) {
	if s.IsNamed() {
		n = string(s.typeName)
	} else {
		n = s.Kind().String() // reflect.Kind
	}
	return
}

//
// short schema
//

// if a type is Basic, string or named then schema is empty;
// for non-basic (and non-string) named types the typeName is used to
// obtain the schema from registery
type shortSchema struct {
	schemaHead
	schema []Schema // pointer (single element) of empty slice
	//
	sr *schemaReg `enc:"-"` // back reference
}

func (s *shortSchema) Schema() (sv *Schema, err error) {
	if isFlat(s.Kind()) { // create schema
		// no elemnts, length, and fields for flat types
		sv = &Schema{
			schemaHead: s.schemaHead,
			sr:         s.sr, // not necessary
		}
	} else if s.IsNamed() { // get from db
		sv, err = s.sr.schemaByName(s.Name())
	} else if len(s.schema) == 1 { // get from slice
		sv = &s.schema[0]
	} else {
		err = ErrInvalidSchema // missing schema in the slice
	}
	return
}

//
// schema
//

type Schema struct {
	schemaHead
	elem   shortSchema
	length uint32
	fields []Field
}

func (s *Schema) Elem() (sv *Schema, err error) {
	sv, err = s.elem.Schema()
	return
}

func (s *Schema) Len() int {
	return int(s.length)
}

func (s *Schema) Fields() []Field {
	return s.fields
}

//
// fields
//

type Field struct {
	name []byte // name of the field
	tag  []byte // tag of the feild
	shortSchema
}

func (f *Field) Name() string {
	return string(f.name)
}

func (f *Field) Tag() reflect.StructTag {
	return reflect.StructTag(f.tag)
}

func (f *Field) Schema() (sv *Schema, err error) {
	sv, err = f.Schema()
	return
}

// field.Kind() -> relfect.Kind of the field type
// field.IsNamed() -> is the type of the field named

// name of the field type
func (f *Field) TypeName() string {
	return f.shortSchema.Name()
}

//
// helpers
//

// get name full name of named type
func typeName(typ reflect.Type) (n string) {
	if pp := typ.PkgPath(); pp != "" {
		n = pp + "." + typ.Name()
	}
	return
}

// flat type with fixed length
func isBasic(kind reflect.Kind) bool {
	switch kind {
	case reflect.Bool,
		reflect.Int8, reflect.Uint8,
		reflect.Int16, reflect.Uint16,
		reflect.Int32, reflect.Uint32,
		reflect.Int64, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// basic or string (can't has references)
func isFlat(kind reflect.Kind) bool {
	return isBasic(kind) || kind == reflect.String
}
