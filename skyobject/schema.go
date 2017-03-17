package skyobject

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

const TAG = "skyobject"

// href types
var (
	singleRef  = typeName(reflect.TypeOf(Reference{}))
	arrayRef   = typeName(reflect.TypeOf(References{}))
	dynamicRef = typeName(reflect.TypeOf(Dynamic{}))
)

type schemaReg struct {
	db  *data.DB                 // store schemas
	nmr map[string]string        // registered name -> type name
	reg map[string]cipher.SHA256 // type name -> schema key
}

func newSchemaReg(db *data.DB) *schemaReg {
	if db == nil {
		panic("misisng db")
	}
	return &schemaReg{
		db:  db,
		nmr: make(map[string]string),
		reg: make(map[string]cipher.SHA256),
	}
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
	sv = new(Schema)
	err = sv.Decode(data)
	sv.sr = s
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
	schema []Schema // pointer simulation: single element or empty slice
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
	sho.schemaHead = s.schemaHead
	if isFlat(s.Kind()) || s.IsNamed() {
		return // we don't need the schema
	}
	sho.schema = []Schema{*s} // non-flat unnamed type
	return
}

//
// fields
//

// A field represents field of a struct with name, schema and tag
type Field struct {
	name []byte // name of the field
	tag  []byte // tag of the feild
	shortSchema
}

// Name returns field name
func (f *Field) Name() string {
	return string(f.name)
}

// Tag returns tag of the field
func (f *Field) Tag() reflect.StructTag {
	return reflect.StructTag(f.tag)
}

// Schema returns schema of the field or error if any
func (f *Field) Schema() (sv *Schema, err error) {
	sv, err = f.Schema()
	return
}

// IsReference reports that the field contains references
func (f *Field) IsReference() bool {
	switch f.TypeName() {
	case singleRef, arrayRef, dynamicRef:
	}
	return false
}

// SchemaOfReference returns schema of type, to which the field refer to.
// The method returns error if the field is not a reference
func (f *Field) SchemaOfReference() (sv *Schema, err error) {
	var tag, name string = f.Tag().Get(TAG), ""
	if name, err = schemaNameFromTag(tag); err != nil {
		return
	}
	sv, err = f.sr.schemaByRegisteredName(name)
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
	sv.sr = s // back reference to the schemaReg
	if isFlat(typ.Kind()) {
		return // we're done (don't need a schema for flat types)
	}
	if name != "" { // named type
		if name == singleRef || name == dynamicRef || name == arrayRef {
			return // we don't register special types
		}
		if _, ok := s.reg[name]; !ok {
			s.reg[name] = cipher.SHA256{} // temporary (hold the place)
		} else { // already registered (or holded)
			return // only name and kind
		}
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
		s.reg[name] = s.db.AddAutoKey(sv.Encode())
	}
	return
}

func (s *schemaReg) getField(fl reflect.StructField) (sf Field) {
	sf.name = []byte(fl.Name)
	sf.tag = []byte(fl.Tag)
	sf.shortSchema = *s.getSchemaOfType(fl.Type).toShort()
	var tag string = fl.Tag.Get(TAG)
	if strings.Contains(tag, "schema=") {
		if sf.TypeName() == singleRef {
			if _, err := schemaNameFromTag(tag); err != nil { // validate
				panic(err)
			}
		} else if sf.TypeName() == arrayRef {
			if _, err := schemaNameFromTag(tag); err != nil { // validate
				panic(err)
			}
		} else {
			panic("unexpected schema tag: " + tag)
		}
	}
	return
}

//
// string
//

func (s *Schema) String() (x string) {
	if s == nil {
		x = "<Missing>"
	} else {
		if s.IsNamed() {
			x = s.Name()
		} else {
			switch kind := s.Kind(); kind {
			case reflect.Array:
				elem, _ := s.Elem()
				x = fmt.Sprintf("[%d]%s", s.Len(), elem.String())
			case reflect.Slice:
				elem, _ := s.Elem()
				x = "[]" + elem.String()
			case reflect.Struct:
				x += "struct {"
				for i, sf := range s.Fields() {
					sv, _ := sf.Schema()
					x += fmt.Sprintf("%s %s `%s`",
						sf.Name(),
						sv.String(),
						sf.Tag())
					if i < len(s.fields)-1 {
						x += ";"
					}
				}
				x += "}"
			default:
				x = kind.String()
			}
		}
	}
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

func schemaNameFromTag(tag string) (name string, err error) {
	for _, part := range strings.Split(tag, ",") {
		if strings.HasPrefix(part, "schema=") {
			ss := strings.Split(part, "=")
			if len(ss) != 2 {
				err = ErrInvalidTag
				return
			}
			name = ss[1]
			break
		}
	}
	if name == "" {
		err = ErrMissingSchemaTag
	}
	return
}

//
// serialize and deserialze (TODO: simplify, DRY)
//

func (s *schemaHead) encode() []byte {
	var x struct {
		Kind     uint32
		TypeName []byte
	}
	x.Kind = s.kind
	x.TypeName = s.typeName
	return encoder.Serialize(x)
}

func (s *shortSchema) encode() []byte {
	var x struct {
		Head   []byte
		Schema []byte
	}
	x.Head = s.schemaHead.encode()
	if len(s.schema) == 1 {
		x.Schema = s.schema[0].encode()
	}
	return encoder.Serialize(x)
}

func (f *Field) encode() []byte {
	var x struct {
		Name  []byte
		Tag   []byte
		Short []byte
	}
	x.Name = f.name
	x.Tag = f.tag
	x.Short = f.shortSchema.encode()
	return encoder.Serialize(x)
}

func (s *Schema) Encode() []byte {
	var x struct {
		Head   []byte
		Elem   []byte
		Len    uint32
		Fields [][]byte
	}
	x.Head = s.schemaHead.encode()
	x.Elem = s.elem.encode()
	x.Len = s.length
	for _, sf := range s.fields {
		x.Fields = append(x.Fields, sf.encode())
	}
	return encoder.Serialize(x)
}

func (s *schemaHead) decode(p []byte) (err error) {
	var x struct {
		Kind     uint32
		TypeName []byte
	}
	if err = encoder.DeserializeRaw(p, &x); err != nil {
		return
	}
	s.kind = x.Kind
	s.typeName = x.TypeName
	return
}

func (s *shortSchema) decode(p []byte) (err error) {
	var x struct {
		Head   []byte
		Schema []byte
	}
	if err = encoder.DeserializeRaw(p, &x); err != nil {
		return
	}
	if len(s.schema) > 0 {
		var ns Schema
		if err = ns.Decode(x.Schema); err != nil {
			return
		}
		s.schema = []Schema{ns}
	}
	err = s.schemaHead.decode(x.Head)
	return
}

func (f *Field) decode(p []byte) (err error) {
	var x struct {
		Name  []byte
		Tag   []byte
		Short []byte
	}
	if err = encoder.DeserializeRaw(p, &x); err != nil {
		return
	}
	f.name = x.Name
	f.tag = x.Tag
	err = f.shortSchema.decode(x.Short)
	return
}

func (s *Schema) Decode(p []byte) (err error) {
	var x struct {
		Head   []byte
		Elem   []byte
		Len    uint32
		Fields [][]byte
	}
	if err = encoder.DeserializeRaw(p, &x); err != nil {
		return
	}
	if err = s.schemaHead.decode(x.Head); err != nil {
		return
	}
	if err = s.elem.decode(x.Elem); err != nil {
		return
	}
	s.length = x.Len
	s.fields = nil // clear
	for _, fv := range x.Fields {
		var f Field
		if err = f.decode(fv); err != nil {
			return
		}
		s.fields = append(s.fields, f)
	}
	return
}
