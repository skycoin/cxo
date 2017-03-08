package skyobject

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/skycoin/skycoin/src/cipher/encoder"
)

type Schema struct {
	Name   string
	Fields []encoder.StructField
}

func (s *Schema) String() (st string) {
	st = s.Name + "\n"
	for _, sf := range s.Fields {
		st += fmt.Sprintf("  %s %s `%s` %s\n",
			sf.Name,
			sf.Type,
			sf.Tag,
			kindString(sf.Kind))
	}
	return
}

func getSchema(i interface{}) (s *Schema) {
	var (
		typ reflect.Type
		nf  int
	)
	s = new(Schema)
	typ = reflect.TypeOf(i)
	nf = typ.NumField()
	s.Name = typ.Name()
	if nf == 0 {
		return
	}
	s.Fields = make([]encoder.StructField, 0, nf)
	for i := 0; i < nf; i++ {
		ft := typ.Field(i)
		if ft.Tag.Get("enc") != "-" {
			s.Fields = append(s.Fields, getField(ft))
		}
	}
	return
}

func getField(ft reflect.StructField) (sf encoder.StructField) {
	sf.Name = ft.Name
	sf.Type = typeName(ft.Type)
	sf.Tag = string(ft.Tag)
	sf.Kind = uint32(ft.Type.Kind())
	return
}

func typeName(typ reflect.Type) string {
	return strings.ToLower(typ.Name())
}

var kinds = [...]string{
	"Invalid",
	"Bool",
	"Int",
	"Int8",
	"Int16",
	"Int32",
	"Int64",
	"Uint",
	"Uint8",
	"Uint16",
	"Uint32",
	"Uint64",
	"Uintptr",
	"Float32",
	"Float64",
	"Complex64",
	"Complex128",
	"Array",
	"Chan",
	"Func",
	"Interface",
	"Map",
	"Ptr",
	"Slice",
	"String",
	"Struct",
	"UnsafePointer",
}

func kindString(k uint32) string {
	if k >= 0 && int(k) < len(kinds) {
		return "<" + kinds[k] + ">"
	}
	return fmt.Sprintf("<kind %d>", k)
}
