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
	htDynamic = typeName(reflect.TypeOf(Dynamic{}))
	// htArray is unnamed
)

type schemaReg struct {
	db  *data.DB                 // store schemas
	nmr map[string]string        // registered name -> type name
	reg map[string]cipher.SHA256 // type name -> schema key
}

// TODO: performance of the solution
func (s *schemaReg) Register(name string, typ interface{}) {
	sch := s.getSchema(typ)
	if !sch.IsNamed() {
		panic("unnamed type registering")
	}
	s.nmr[name] = sch.Name()
	// registered name -> type name -> schema key -> schema data -> schema
}

func (s *schemaReg) schemaByRegisteredName(name string) (sv *Schema,
	err error) {

	var ex bool
	if name, ex = s.nmr[name]; !ex {
		err = ErrUnregisteredSchema
		return
	}
	sv, err = s.schemaByName(name)
	return
}

// by typme name (not by registered name)
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

// TODO: register

//
// schema head
//

type schemaHead struct {
	kind     uint32 // used to filter flat types
	typeName []byte // for named types
	//
	sr *schemaReg `enc:"-"` // back reference
}

func (s *schemaHead) Kind() reflect.Kind {
	return reflect.Kind(s.kind)
}

// is the type nemed
func (s *schemaHead) IsNamed() bool {
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
}

func (s *shortSchema) Schema() (sv *Schema, err error) {
	if isFlat(s.Kind()) { // create schema
		// no elemnts, length, and fields for flat types
		sv = &Schema{
			schemaHead: s.schemaHead,
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

// if a type is flat or named we don't need to keep entire schema
func (s *Schema) toShort() (sho *shortSchema) {
	sho = new(shortSchema)
	sho.kind = s.kind
	sho.typeName = s.typeName
	if isFlat(s.Kind()) || s.IsNamed() {
		return // we don't need the schema
	}
	sho.schema = []Schema{*s} // non-flat unnamed type
	return
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
// field.IsNamed() -> is the type of the field named?

// name of the field type
func (f *Field) TypeName() string {
	return f.shortSchema.Name()
}

//
// get
//

func (s *schemaReg) getSchema(i interface{}) (sv *Schema) {
	sv = s.getSchemaOfType(
		reflect.Indirect(reflect.ValueOf(i)).Type(),
	)
	return
}

func (s *schemaReg) getSchemaOfType(typ reflect.Type) (sv *Schema) {
	sv = new(Schema)
	name := typeName(typ)
	sv.typeName = []byte(name)
	sv.kind = uint32(typ.Kind())
	if name != "" { // named type
		if name == htSingle || name == htDynamic { // htArray is unnamed
			return // we don't register special types
		}
		if _, ok := s.reg[name]; !ok {
			s.reg[name] = cipher.SHA256{} // temporary (hold the place)
		} else { // already registered (or holded)
			return // only name and kind
		}
	}
	if isFlat(typ.Kind()) {
		return // we're done
	}
	switch typ.Kind() {
	case reflect.Array:
		// it never be cipher.SHA256
		sv.length = uint32(typ.Len())
		sv.elem = *s.getSchemaOfType(typ.Elem()).toShort()
	case reflect.Slice:
		// it can be a []cipher.SHA256 (because it's unnamed type)
		sv.elem = *s.getSchemaOfType(typ.Elem()).toShort()
	case reflect.Struct:
		// it never be Dynamic
		var nf int = typ.NumField()
		sv.fields = make([]Field, 0, nf)
		for i := 0; i < nf; i++ {
			fl := typ.Field(i)
			if fl.Tag.Get("enc") == "-" || fl.Name == "_" || fl.PkgPath != "" {
				continue
			}
			sv.fields = append(sv.fields, s.getField(fl))
		}
	default:
		panic("invalid type: " + typ.String())
	}
	// save named type to registery and db
	if name != "" { // named type
		if name == htSingle || name == htDynamic { // htArray is unnamed
			return // we don't register special types
		}
		s.reg[name] = s.db.AddAutoKey(encoder.Serialize(sv))
	}
	return
}

func (s *schemaReg) getField(fl reflect.StructField) (sf Field) {
	sf.name = []byte(fl.Name)
	sf.tag = []byte(fl.Tag)
	sf.shortSchema = *s.getSchemaOfType(fl.Type).toShort()
	var tag string = fl.Tag.Get(TAG)
	if strings.Contains(tag, "schema=") {
		if sf.TypeName() == htSingle {
			schemaNameFromTag(tag) // validate the tag
		} else if fl.Type == reflect.TypeOf([]cipher.SHA256{}) {
			schemaNameFromTag(tag) // validate the tag
		} else {
			panic("unexpected schema= in tag: " + tag)
		}
	}
	return
}

//
// string
//

func (s *Schema) String() (x string) {
	// TODO
	return
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

func schemaNameFromTag(tag string) (name string) {
	for _, part := range strings.Split(tag, ",") {
		if strings.HasPrefix(part, "schema=") {
			ss := strings.Split(part, "=")
			if len(ss) != 2 {
				panic("invalid schema tag: " + part)
			}
			if name = ss[1]; name != "" {
				panic("empty schema name in tag")
			}
			break
		}
	}
	return
}
