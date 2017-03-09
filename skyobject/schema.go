package skyobject

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/skycoin/skycoin/src/cipher/encoder"
)

type Schema struct {
	Name   string
	Fields []encoder.StructField
}

func (s *Schema) String() string {
	w := new(bytes.Buffer)
	w.WriteString(s.Name)
	w.WriteString(" {")
	for i, sf := range s.Fields {
		fmt.Fprintf(w, "%s %s `%s` %s",
			sf.Name,
			sf.Type,
			sf.Tag,
			kindString(sf.Kind))
		if i != len(s.Fields)-1 {
			w.WriteString("; ")
		}
	}
	w.WriteByte('}')
	return w.String()
}

func getSchema(i interface{}) (s Schema) {
	var (
		typ reflect.Type
		nf  int
	)
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
	"invalid",
	"bool",
	"int",
	"int8",
	"int16",
	"int32",
	"int64",
	"uint",
	"uint8",
	"uint16",
	"uint32",
	"uint64",
	"uintptr",
	"float32",
	"float64",
	"complex64",
	"complex128",
	"array",
	"chan",
	"func",
	"interface",
	"map",
	"ptr",
	"slice",
	"string",
	"struct",
	"unsafePointer",
}

func kindString(k uint32) string {
	if k >= 0 && int(k) < len(kinds) {
		return "<" + kinds[k] + ">"
	}
	return fmt.Sprintf("<kind %d>", k)
}
