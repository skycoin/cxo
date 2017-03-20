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

var (
	singleRef  = typeName(reflect.TypeOf(Reference{}))
	arrayRef   = typeName(reflect.TypeOf(References{}))
	dynamicRef = typeName(reflect.TypeOf(Dynamic{}))
)

// ========================================================================== //
//                                                                            //
//                               registry                                     //
//                                                                            //
// ========================================================================== //

type Registry struct {
	db  *data.DB
	nmr map[string]string
	reg map[string]cipher.SHA256
}

func NewRegistery(db *data.DB) (r *Registry) {
	if db == nil {
		panic("nil db")
	}
	r = &Registry{
		db:  db,
		nmr: make(map[string]string),
		reg: make(map[string]cipher.SHA256),
	}
	return
}

func (r *Registry) Register(name string, i interface{}) {
	typ := reflect.Indirect(reflect.ValueOf(i)).Type()
	tn := typeName(typ)
	if tn == "" {
		panic("unnamed types are not allowed for registering")
	}
	if rn, ok := r.nmr[name]; ok {
		if rn != tn {
			panic("another type aready registered with given name")
		} else if rn == tn {
			return // the same name, the same type,  already registered
		}
	}
	if _, ok := r.reg[tn]; !ok {
		r.SaveSchema(i)
	}
	r.nmr[name] = tn
}

// SaveSchema of given type and get reference to the schema
func (r *Registry) SaveSchema(i interface{}) (ref Reference) {
	typ := reflect.Indirect(reflect.ValueOf(i)).Type()
	tn := typeName(typ)
	if tn == "" {
		panic("schema of unnamed and builtin types is not allowed for saving")
	}
	var ok bool
	var ch cipher.SHA256
	if ch, ok = r.reg[tn]; ok {
		ref = Reference(ch)
		return // already saved
	}
	s := r.getSchema(typ) // registers named types automatically
	switch s.Name() {
	case singleRef, arrayRef, dynamicRef:
		panic("reference types are not allowed to SaveShema")
	}
	// register the type even if it's not named
	ch = r.db.AddAutoKey(s.Encode())
	r.reg[s.Name()] = ch
	ref = Reference(ch)
	return
}

func (r *Registry) SchemaByTypeName(tn string) (s *Schema, err error) {
	var sr cipher.SHA256 // schema reference
	var ok bool
	if sr, ok = r.reg[tn]; !ok {
		err = ErrTypeNameNotFound
		return
	}
	s, err = r.SchemaByReference(Reference(sr))
	return
}

func (r *Registry) SchemaByReference(sr Reference) (s *Schema, err error) {
	var sd []byte
	var ok bool
	if sd, ok = r.db.Get(cipher.SHA256(sr)); !ok {
		err = &MissingSchema{Reference(sr)}
		return
	}
	s = new(Schema)
	err = s.Decode(r, sd)
	return
}

func (r *Registry) getSchema(typ reflect.Type) (s *Schema) {
	s = new(Schema)
	s.sr = r
	s.kind = uint32(typ.Kind())
	s.name = []byte(typeName(typ))
	if s.isNamed() {
		switch s.Name() {
		case singleRef, arrayRef, dynamicRef:
			s.kind = uint32(reflect.Ptr)
			return
		}
		// solve problem of recursion
		if _, ok := r.reg[s.Name()]; ok {
			return // already saved or holded
		} else {
			r.reg[s.Name()] = cipher.SHA256{} // hold using empty reference
		}
	}
	switch typ.Kind() {
	case reflect.Bool, reflect.Int8, reflect.Uint8,
		reflect.Int16, reflect.Uint16,
		reflect.Int32, reflect.Uint32, reflect.Float32,
		reflect.Int64, reflect.Uint64, reflect.Float64,
		reflect.String:
	case reflect.Slice:
		s.setElem(r.getSchema(typ.Elem()))
	case reflect.Array:
		s.length = uint32(typ.Len())
		s.setElem(r.getSchema(typ.Elem()))
	case reflect.Struct:
		var nf int = typ.NumField()
		if nf == 0 {
			break
		}
		s.fields = make([]Field, 0, nf)
		for i := 0; i < nf; i++ {
			fl := typ.Field(i)
			if fl.Tag.Get("enc") == "-" || fl.Name == "_" || fl.PkgPath != "" {
				continue
			}
			s.fields = append(s.fields, r.getField(fl))
		}
	default:
		panic("invlaid type: " + typ.String())
	}
	if s.isNamed() { // register named types anyway
		r.reg[s.Name()] = r.db.AddAutoKey(s.Encode())
	}
	return
}

func (r *Registry) getField(fl reflect.StructField) (sf Field) {
	sf.schema = *r.getSchema(fl.Type)
	sf.name = []byte(fl.Name)
	sf.tag = []byte(fl.Tag)
	if sf.isReference() && sf.TypeName() != dynamicRef {
		sn := mustGetSchemaOfTag(fl.Tag.Get(TAG))
		tn, ok := r.nmr[sn]
		if !ok {
			panic("schema name is not registered: " + sn)
		}
		sf.ref = []byte(tn) // use actual type name instead of registered
	}
	return
}

// ========================================================================== //
//                                                                            //
//                                 schema                                     //
//                                                                            //
// ========================================================================== //

type Schema struct {
	kind   uint32   // reflect.Kind (relfect.Ptr for references)
	name   []byte   // type name for named types
	elem   []Schema // for arrays and slices
	length uint32   // for arrays
	fields []Field  // for structs

	sr *Registry `enc:"-"`
}

// IsNil reports that the schema is an empty schema.
func (s *Schema) IsNil() bool {
	return s.kind == uint32(reflect.Invalid)
}

// Name returns name for named types
func (s *Schema) Name() string {
	return string(s.name)
}

// Kind returns relfect.Kind of the type (that is relfect.Ptr for references)
func (s *Schema) Kind() reflect.Kind {
	return reflect.Kind(s.kind)
}

// Elem returns schema of element
func (s *Schema) Elem() (es *Schema, err error) {
	if len(s.elem) != 1 {
		err = ErrInvalidSchema
		return
	}
	es = &s.elem[0]
	if es.isSaved() {
		err = es.load()
	}
	return
}

func (s *Schema) Len() int {
	return int(s.length)
}

func (s *Schema) Fields() []Field {
	return s.fields
}

func (s *Schema) setElem(el *Schema) {
	s.elem = []Schema{*el}
}

// the type is named
func (s *Schema) isNamed() bool {
	return len(s.name) > 0
}

// TODO: check if a schema already loaded

// if a type
// - is not reference
// - is not flat
// - is named
func (s *Schema) isSaved() (yep bool) {
	if kind := s.Kind(); !isFlat(kind) && kind != reflect.Ptr {
		yep = s.isNamed()
	}
	return
}

func (s *Schema) load() (err error) {
	var x *Schema
	x, err = s.sr.SchemaByTypeName(s.Name())
	*s = *x
	return
}

func (s *Schema) String() string {
	if s == nil {
		return "<Missing>"
	}
	if s.isNamed() {
		return s.Name()
	}
	switch s.Kind() {
	case reflect.Array:
		el, _ := s.Elem()
		return fmt.Sprintf("[%d]%s", s.Len(), el.String())
	case reflect.Slice:
		el, _ := s.Elem()
		return "[]" + el.String()
	case reflect.Struct:
		x := "struct{"
		for i, sf := range s.Fields() {
			x += sf.String()
			if i < len(s.Fields())-1 {
				x += ";"
			}
		}
		x += "}"
		return x
	}
	return s.Kind().String()
}

// ========================================================================== //
//                                                                            //
//                                  field                                     //
//                                                                            //
// ========================================================================== //

type Field struct {
	schema Schema
	name   []byte
	tag    []byte
	ref    []byte // name of type the field referes to (for references)
}

func (f *Field) Kind() reflect.Kind { // prevent loading the schema
	return f.schema.Kind()
}

func (f *Field) TypeName() string { // prevent loading the schema
	return f.schema.Name()
}

func (f *Field) Name() string {
	return string(f.name)
}

func (f *Field) Schema() (fs *Schema, err error) {
	fs = &f.schema
	if fs.isSaved() {
		err = fs.load()
	}
	if len(f.ref) > 0 { // reference (singleRef or arrayRef)
		// elem of singleRef or arrayRef will be schema of type the reference
		// points to
		var el *Schema
		el, err = f.schema.sr.SchemaByTypeName(f.tagSchemaName())
		if err != nil {
			fs = nil
			return
		}
		f.schema.setElem(el)
	}
	return
}

func (f *Field) Tag() reflect.StructTag {
	return reflect.StructTag(f.tag)
}

func (f *Field) tagSchemaName() string {
	return string(f.ref)
}

func (f *Field) isReference() bool {
	return f.Kind() == reflect.Ptr
}

func (f *Field) String() string {
	return fmt.Sprintf("%s %s `%s`", f.Name(), f.TypeName(), f.Tag())
}

// ========================================================================== //
//                                                                            //
//                            encode / decode                                 //
//                                                                            //
// ========================================================================== //

type encodingSchema struct {
	Kind   uint32
	Name   []byte
	Elem   []encodingSchema
	Length uint32
	Fields []encodingField
}

func (e *encodingSchema) Schema(sr *Registry) (s *Schema) {
	s = new(Schema)
	s.sr = sr
	s.kind = e.Kind
	s.name = e.Name
	s.length = e.Length
	s.elem = nil // reset
	if len(e.Elem) == 1 {
		s.elem = []Schema{*e.Elem[0].Schema(sr)}
	}
	s.fields = nil // reset
	for _, ef := range e.Fields {
		s.fields = append(s.fields, ef.Field(sr))
	}
	return
}

func newEncodingSchema(s *Schema) (e encodingSchema) {
	e.Kind = s.kind
	e.Name = s.name
	e.Length = s.length
	if len(s.elem) == 1 {
		e.Elem = []encodingSchema{newEncodingSchema(&s.elem[0])}
	}
	for _, sf := range s.fields {
		e.Fields = append(e.Fields, newEncodingField(&sf))
	}
	return
}

type encodingField struct {
	Schema encodingSchema
	Name   []byte
	Tag    []byte
	Ref    []byte
}

func newEncodingField(sf *Field) (e encodingField) {
	e.Schema = newEncodingSchema(&sf.schema)
	e.Name = sf.name
	e.Tag = sf.tag
	e.Ref = sf.ref
	return
}

func (e *encodingField) Field(sr *Registry) (sf Field) {
	sf.name = e.Name
	sf.tag = e.Tag
	sf.ref = e.Ref
	sf.schema = *e.Schema.Schema(sr)
	return
}

func (s *Schema) Encode() []byte {
	return encoder.Serialize(newEncodingSchema(s))
}

func (s *Schema) Decode(sr *Registry, p []byte) (err error) {
	var es encodingSchema
	if err = encoder.DeserializeRaw(p, &es); err != nil {
		return
	}
	*s = *es.Schema(sr)
	return
}

// ========================================================================== //
//                                                                            //
//                                helpers                                     //
//                                                                            //
// ========================================================================== //

func typeName(typ reflect.Type) (s string) {
	if typ.PkgPath() != "" {
		s = typ.PkgPath() + "." + typ.Name()
	}
	return
}

func isFlat(kind reflect.Kind) (yep bool) {
	switch kind {
	case reflect.Bool, reflect.Int8, reflect.Uint8,
		reflect.Int16, reflect.Uint16,
		reflect.Int32, reflect.Uint32, reflect.Float32,
		reflect.Int64, reflect.Uint64, reflect.Float64,
		reflect.String:
		yep = true
	}
	return
}

func mustGetSchemaOfTag(tag string) string {
	for _, part := range strings.Split(tag, ",") {
		if !strings.HasPrefix(part, "schema=") {
			continue
		}
		ss := strings.Split(part, "=")
		if len(ss) != 2 {
			panic("invalid schema tag: " + part)
		}
		if ss[1] == "" {
			panic("empty tag schema name: " + part)
		}
		return ss[1]
	}
	panic("invalid tag: " + tag)
	return ""
}
