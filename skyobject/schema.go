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
		st += fmt.Sprintf("  %s %s `%s` %d\n",
			sf.Name,
			sf.Type,
			sf.Tag,
			sf.Kind)
	}
	return
}

func getSchema(i interface{}) (s Schema) {
	typ, nf := reflect.Type(i), typ.NumField()
	s.Name = typ.Name()
	s.Fields = make([]encoder.StructField, 0, nf)
	for i := 0; i < nf; i++ {
		ft := typ.Field(i)
		if ft.Tag.Get("enc") != "-" {
			s.Fields = append(sch.Fields, getField(ft))
		}
	}
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
