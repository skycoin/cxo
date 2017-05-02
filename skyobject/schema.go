package skyobject

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

const (
	TAG = "skyobject"

	SINGLE  = "-S" // Reference
	ARRAY   = "-A" // References
	DYNAMIC = "-D" // Dynamic
)

// ========================================================================== //
//                                                                            //
//                               registry                                     //
//                                                                            //
// ========================================================================== //

type Registry struct {
	db  *data.DB
	nnm map[reflect.Type]string // internal
	reg map[string]Reference    // shared
}

func newRegistery(db *data.DB) (r *Registry) {
	if db == nil {
		panic("nil db")
	}
	r = &Registry{
		db:  db,
		nnm: make(map[reflect.Type]string),
		reg: make(map[string]Reference),
	}
	return
}

// name restrictions
func (r *Registry) validateName(name string) {
	if name == "" {
		panicf("empty name")
	} else if strings.HasPrefix(name, "-") {
		panicf("'-'-prefixed names are for internal types")
	}
	return
}

// Register batch of types. It receive key-value pair where key is name of
// a type and value is the type. Order does not matter. For example:
//
//    type Group struct {
//    	Members References `skyobject:"schema=User"`
//    }
//
//    type User struct {
//    	Name string
//    }
//
//    type List struct {
//    	List  References `skyobject:"schema=User"`
//    	Group []Group
//    }
//
//    type Man struct {
//    	Name    string
//    	Owner   Group
//    	Friends List
//    }
//
//    cnt.Register(
//    	"Group", Group{},
//     	"User", User{},
//    	"List", List{},
//    	"Man", Man{},
//    )
//
// This approach was introduced to verify dependencies. The Group requires the
// User, the List requires the User and the Group etc. I.e. there are many
// complex recursive dependencies. It's possible to register any type without
// dependencies separate. For example
//
//    cnt.Register("User", User{})
//    cnt.Register(
//    	"Group", Group{},
//    	"List", List{},
//    	"Man", Man{},
//    )
//
func (r *Registry) Register(ni ...interface{}) {
	if len(ni)%2 != 0 {
		panicf("invalid arguments count: %d", len(ni))
	}
	type nameType struct {
		name string
		typ  reflect.Type
	}
	nts := make([]nameType, 0, len(ni)/2)
	// check arguments
	for i := 0; i < len(ni); i += 2 {
		name, ok := ni[i].(string)
		if !ok {
			panicf("invalid argument type %T (expected string)", ni[i])
		}
		r.validateName(name)
		typ := typeOf(ni[i+1])
		if _, ok := r.isSpecial(typ); ok {
			panicf("can't register special type: %s", typ.String())
		}
		if hn, ok := r.nnm[typ]; ok {
			if hn != name {
				panicf("repetative registration of %s"+
					" with different name: %s (prefious name: %s)",
					typ.String(), name, hn)
			}
		} else {
			r.nnm[typ] = name // hold the place
		}
		if _, ok := r.reg[name]; !ok {
			r.reg[name] = Reference{} // hold the place
		}
		nts = append(nts, nameType{name, typ})
	}
	for _, nt := range nts {
		sck := r.registerSchema(nt.name, nt.typ)
		// check existing type with the same name and its hash
		if hash := r.reg[nt.name]; hash != (Reference{}) && hash != sck {
			panicf("a different type already registered"+
				" with the name %s", nt.name)
		}
		r.reg[nt.name] = sck // register
	}
}

func (r *Registry) SchemaReference(i interface{}) (sck Reference) {
	var typ reflect.Type = typeOf(i)
	if name, ok := r.nnm[typ]; !ok {
		panic("no schema for: " + typ.String())
	} else if sck, ok = r.reg[name]; !ok {
		panic("no reference for: " + name)
	}
	return
}

func (r *Registry) SchemaByName(name string) (s *Schema, err error) {
	var sck Reference // schema reference
	var ok bool
	if sck, ok = r.reg[name]; !ok {
		err = ErrTypeNameNotFound
		return
	}
	s, err = r.SchemaByReference(sck)
	return
}

func (r *Registry) SchemaByReference(sr Reference) (s *Schema, err error) {
	var sd []byte
	var ok bool
	if sd, ok = r.db.Get(cipher.SHA256(sr)); !ok {
		err = &MissingSchema{sr}
		return
	}
	s = r.newSchema()
	if err = s.Decode(sd); err != nil {
		s = nil // gc
	}
	return
}

func (r *Registry) newSchema() (s *Schema) {
	s = new(Schema)
	s.sr = r
	return
}

func (r *Registry) isSpecial(typ reflect.Type) (s *Schema, ok bool) {
	switch typ {
	case reflect.TypeOf(Reference{}):
		s, ok = r.newSchema(), true
		s.kind, s.name = uint32(reflect.Ptr), []byte(SINGLE)
	case reflect.TypeOf(References{}):
		s, ok = r.newSchema(), true
		s.kind, s.name = uint32(reflect.Ptr), []byte(ARRAY)
	case reflect.TypeOf(Dynamic{}):
		s, ok = r.newSchema(), true
		s.kind, s.name = uint32(reflect.Ptr), []byte(DYNAMIC)
	}
	return
}

func (r *Registry) registerSchema(name string,
	typ reflect.Type) (sck Reference) {

	var s *Schema = r.newSchema()
	s.kind = uint32(typ.Kind())
	if typ.Kind() != reflect.Struct {
		panic("non-struct type can't be registered: " + typ.String())
	}
	s.name = []byte(name)
	r.getFields(s, typ)
	sck = Reference(r.db.AddAutoKey(s.Encode()))
	// don't register
	return
}

func (r *Registry) getSchema(typ reflect.Type) (s *Schema) {
	// special types
	var ok bool
	if s, ok = r.isSpecial(typ); ok {
		return
	}
	s = r.newSchema()
	s.kind = uint32(typ.Kind())
	if typ.PkgPath() != "" { // non-builtin named type
		if typ.Kind() == reflect.Struct { // must be registered
			name, ok := r.nnm[typ]
			if !ok {
				panicf("type required but not registered: %s", typ.String())
			}
			s.name = []byte(name)
			return // we done
		}
		s.name = []byte(typ.Name()) // a named non-struct type (don't register)
	} // else -> fuck the name
	switch typ.Kind() {
	case reflect.Bool, reflect.Int8, reflect.Uint8,
		reflect.Int16, reflect.Uint16,
		reflect.Int32, reflect.Uint32, reflect.Float32,
		reflect.Int64, reflect.Uint64, reflect.Float64,
		reflect.String:
		// do nothing for flat types
	case reflect.Slice:
		// get schema of element of the slice
		s.setElem(r.getSchema(typ.Elem()))
	case reflect.Array:
		// get length and schema of element of the array
		s.length = uint32(typ.Len())
		s.setElem(r.getSchema(typ.Elem()))
	case reflect.Struct:
		r.getFields(s, typ)
	default:
		panic("invlaid type: " + typ.String())
	}
	return
}

func (r *Registry) getFields(s *Schema, typ reflect.Type) {
	var nf int = typ.NumField()
	if nf == 0 {
		return
	}
	s.fields = make([]Field, 0, nf)
	for i := 0; i < nf; i++ {
		fl := typ.Field(i)
		if fl.Tag.Get("enc") == "-" || fl.Name == "_" || fl.PkgPath != "" {
			continue
		}
		s.fields = append(s.fields, r.getField(fl))
	}
}

func (r *Registry) getField(fl reflect.StructField) (sf Field) {
	sf.schema = *r.getSchema(fl.Type)
	sf.name = []byte(fl.Name)
	sf.tag = []byte(fl.Tag)
	if sf.isReference() && sf.schema.Name() != DYNAMIC {
		sn := mustGetSchemaOfTag(fl.Tag.Get(TAG))
		if _, ok := r.reg[sn]; !ok {
			panic("unregistered schema: " + sn)
		}
		sf.ref = []byte(sn)
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

	sr *Registry `enc:"-"` // back reference
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
	if es.isRegistered() {
		if err = es.load(); err != nil {
			es = nil // gc
		}
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

// is registered or should be registered
func (s *Schema) isRegistered() bool {
	return s.Kind() == reflect.Struct && s.isNamed()
}

// load a registered shema by name
func (s *Schema) load() (err error) {
	if len(s.fields) > 0 {
		return // already loaded
	}
	var x *Schema
	if x, err = s.sr.SchemaByName(s.Name()); err == nil {
		*s = *x
	}
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

// A Field represents struct field
type Field struct {
	schema Schema
	name   []byte
	tag    []byte
	ref    []byte // name of type the field referes to (for references)
}

// Kind of the field
func (f *Field) Kind() reflect.Kind { // prevent loading the schema
	return f.schema.Kind()
}

// TypeName if the type is named
func (f *Field) TypeName() string { // prevent loading the schema
	return f.schema.Name()
}

// Name of the field
func (f *Field) Name() string {
	return string(f.name)
}

// Schema of the field
func (f *Field) Schema() (fs *Schema, err error) {
	fs = &f.schema
	if fs.isRegistered() {
		if err = fs.load(); err != nil {
			fs = nil // gc
		}
		return
	}
	if len(f.ref) > 0 { // reference (SINGLE or ARRAY)
		// elem of SINLGE or ARRAY will be schema of type the reference
		// points to
		var el *Schema
		el, err = f.schema.sr.SchemaByName(f.tagSchemaName())
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
	s = sr.newSchema()
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

func (s *Schema) Decode(p []byte) (err error) {
	var es encodingSchema
	if err = encoder.DeserializeRaw(p, &es); err != nil {
		return
	}
	*s = *es.Schema(s.sr)
	return
}

// ========================================================================== //
//                                                                            //
//                                helpers                                     //
//                                                                            //
// ========================================================================== //

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

func typeOf(i interface{}) reflect.Type {
	return reflect.Indirect(reflect.ValueOf(i)).Type()
}

func panicf(format string, args ...interface{}) {
	panic(fmt.Errorf(format, args...))
}
