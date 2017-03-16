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
	name   []byte // string
	kind   uint32
	elem   []Schema // for arrays and slices (pointer)
	length uint32   // for arrays only
	fields []Field  // for structs only
	//
	reg *schemaReg `enc:"-"` // back reference
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

func (s *Schema) Elem() (sch *Schema) {
	if len(s.elem) == 1 {
		sch = &s.elem[0]
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
	name   []byte   // field name (string)
	tag    []byte   // field tag
	schema []Schema // schema of the field (pointer)
}

func (f *Field) Name() string {
	return string(f.name)
}

func (f *Field) Tag() reflect.StructTag {
	return reflect.StructTag(f.tag)
}

func (f *Field) Schema() (sch *Schema) {
	if len(f.schema) == 1 {
		sch = &f.schema[0]
	}
	return
}

func (s *schemaReg) getSchemaOfType(typ reflect.Type) (sch *Schema) {
	sch = new(Schema)
	sch.reg = s
	sch.name = []byte(typeName(typ))
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
	return
}

func (s *schemaReg) getField(fl reflect.StructField) (sf Field) {
	sf.tag = []byte(fl.Tag)
	sf.name = []byte(fl.Name)
	sf.schema = []Schema{*s.getSchemaOfType(fl.Type)}
	if strings.Contains(fl.Tag.Get(TAG), "href") {
		switch sf.schema[0].Name() {
		case htSingle, htArray, htDynamic:
		default:
			panic("unexpected href tag")
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

// TORM
func isBasic(kind reflect.Kind) bool {
	switch kind {
	case reflect.Bool,
		reflect.Int8, reflect.Uint8,
		reflect.Int16, reflect.Uint16,
		reflect.Int32, reflect.Uint32,
		reflect.Int64, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return true
	}
	return false
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
