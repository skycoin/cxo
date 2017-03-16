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
	htSingle  = typeName(reflect.TypeOf(cipher.SHA256{}))
	htArray   = typeName(reflect.TypeOf([]cipher.SHA256{}))
	htDynamic = typeName(reflect.TypeOf(Dynamic{}))
)

type schemaReg struct {
	reg map[string]cipher.SHA256
	db  *data.DB
}

type Schema struct {
	name []byte // string
	kind uint32
	elem struct { // for arrays and slices
		typeName string   // } for named types
		kind     uint32   // }
		schema   []Schema // (pointer) for unnamed types
	}
	length uint32  // for arrays only
	fields []Field // for structs only
	//
	reg *schemaReg `enc:"-"` // back reference
}

// is the type of the schema is named type
func (s *Schema) IsNamed() bool {
	return len(s.name) > 0
}

func (s *Schema) Kind() reflect.Kind {
	return reflect.Kind(s.kind)
}

func (s *Schema) Name() (n string) {
	if len(s.name) == 0 {
		n = reflect.Kind(s.kind).String()
	} else {
		n = string(s.name)
	}
	return
}

func (s *Schema) Elem() (sch *Schema, err error) {
	if len(s.elem) == 1 {
		sch = &s.elem[0]
	} else {
		if s.Kind() == reflect.Slice || s.Kind() == reflect.Array {
			err = ErrInvalidSchema
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

type Field struct {
	name     []byte   // field name (string)
	tag      []byte   // field tag
	typeName []byte   // for named types
	schema   []Schema // schema of the field (pointer) (for unnamed types)
	//
	reg *schemaReg `enc:"-"` // back reference
}

func (f *Field) Name() string {
	return string(f.name)
}

func (f *Field) Tag() reflect.StructTag {
	return reflect.StructTag(f.tag)
}

// is type of the field named
func (f *Field) IsNamed() bool {
	return len(f.typeName) > 0
}

func (f *Field) TypeName() string {
	if f.IsNamed() {
		return string(f.typeName)
	}
	if fs, err := f.Schema(); err == nil {
		return fs.Name()
	}
	return "<Invalid>"
}

func (f *Field) Schema() (sch *Schema, err error) {
	if f.IsNamed() { // field type is named
		typeName := f.TypeName()
		if sk, ok := f.reg.reg[typeName]; ok {
			if data, ok := f.reg.db.Get(sk); ok {
				err = encoder.DeserializeRaw(data, &sch)
			} else {
				err = ErrMissingInDB
			}
		} else {
			err = ErrUnregisteredSchema
		}
	} else if len(f.schema) == 1 {
		sch = &f.schema[0]
	} else {
		err = ErrInvalidSchema // missing encoded schema
	}
	return
}

func (s *schemaReg) getSchemaOfType(typ reflect.Type) (sch *Schema) {
	sch = new(Schema)
	sch.reg = s
	sch.name = []byte(typeName(typ))
	var name string = sch.Name()
	if sch.IsNamed() {
		// it's named type: make empty key to avoid recursive getSchemaOfType
		if _, ok := s.reg[name]; !ok {
			s.reg[name] = cipher.SHA256{} // temporary
		}
		// see: (*schemaReg).getField 'default' branch in switch
	}
	sch.kind = uint32(typ.Kind())
	switch typ.Kind() {
	case reflect.Bool,
		reflect.Int8, reflect.Uint8,
		reflect.Int16, reflect.Uint16,
		reflect.Int32, reflect.Uint32,
		reflect.Int64, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
	case reflect.Array:
		sch.length = uint32(typ.Len())
		sch.elem = []Schema{*s.getSchemaOfType(typ)}
	case reflect.Slice:
		sch.elem = []Schema{*s.getSchemaOfType(typ)}
	case reflect.Struct:
		var nf int = typ.NumField()
		sch.fields = make([]Field, 0, nf)
		for i := 0; i < nf; i++ {
			fl := typ.Field(i)
			if fl.Tag.Get("enc") == "-" || fl.Name == "_" || fl.PkgPath != "" {
				continue
			}
			sch.fields = append(sch.fields, s.getField(fl))
		}
	default:
		panic("invalid type: " + typ.String())
	}
	if sch.IsNamed() { // named type
		// place actual value to registery
		s.reg[name] = s.db.AddAutoKey(encoder.Serialize(sch))
	}
	return
}

func (s *schemaReg) getField(fl reflect.StructField) (sf Field) {
	sf.reg = s
	//
	sf.tag = []byte(fl.Tag)
	sf.name = []byte(fl.Name)
	sf.typeName = typeName(fl.Type)
	// schema=User for single (by tag)
	// schema=User for array (by tag)
	// Dynamic for synamic href (by type)
	switch sf.typeName {
	case htSingle:
		var tag string = fl.Tag.Get(TAG)
		if strings.Contains(tag, "schema=") {
			tagSchemaName(tag) // validate
		}
	case htArray:
		var tag string = fl.Tag.Get(TAG)
		if strings.Contains(tag, "schema=") {
			tagSchemaName(tag) // validate
		}
	case htDynamic:
		// do nothing
	case "":
		// unnamed type requires us to encode the schema inside the field
		if strings.Contains(fl.Tag.Get(TAG), "schema=") {
			panic("unexpected schema= tag")
		}
		sf.schema = []Schema{*s.getSchemaOfType(fl.Type)}
	default:
		// a named type (except references types)
		if strings.Contains(fl.Tag.Get(TAG), "schema=") {
			panic("unexpected schema= tag")
		}
		// be sure that the type is registered or register it
		if _, ok := s.reg[sf.typeName]; !ok {
			s.getSchemaOfType(fl.Type) // encode and register
		}
	}
	return
}

func typeName(typ reflect.Type) (n string) {
	if typ.PkgPath() == "" {
		n = typ.PkgPath() + "." + typ.Name()
	}
	return
}

// schema=User -> User
func tagSchemaName(tag string) (name string) {
	for _, part := range strings.Split(tag, ",") {
		if strings.HasPrefix(part, "schema=") {
			ss := strings.Split(part, "=")
			if len(ss) != 2 {
				panic("invalid schema tag: " + part)
			}
			if name = ss[1]; name == "" {
				panic("empty schema name: " + part)
			}
			break
		}
	}
	return
}

func isBasic(kind reflect.Kind) bool {
	switch kind {
	case reflect.Bool,
		reflect.Int8, reflect.Uint8,
		reflect.Int16, reflect.Uint16,
		reflect.Int32, reflect.Uint32,
		reflect.Int64, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		// reflect.String
		return true
	}
	return false
}

func basicSize(kind reflect.Kind) (n int) {
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
		panic("invalid kind")
	}
	return
}

func (s *Schema) String() (x string) {
	if s == nil {
		return "<Missing>"
	}
	if name := s.Name(); name != "" { // name for named types
		return name
	}
	switch s.Kind() {
	case reflect.Bool,
		reflect.Int8, reflect.Uint8,
		reflect.Int16, reflect.Uint16,
		reflect.Int32, reflect.Uint32,
		reflect.Int64, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		x = s.Name()
	case reflect.Array:
		x = fmt.Sprintf("[%d]%s", s.Len(), s.Elem().String())
	case reflect.Slice:
		x = "[]" + s.Elem().String()
	case reflect.Struct:
		x = "struct{"
		for i, sf := range s.fields {
			x += sf.String()
			if i < len(s.fields)-1 {
				x += ";"
			}
		}
		x += "}"
	default:
		x = "<Invalid>"
	}
	return
}

func (f *Field) String() string {
	return fmt.Sprintf("%s %s `%s`", f.Name(), f.Schema().String(), f.Tag())
}

func (s *Schema) Encode() []byte {
	return encoder.Serialize(s)
}

func (s *Schema) Decode(p []byte) (err error) {
	err = encoder.DeserializeRaw(p, s)
	return
}
