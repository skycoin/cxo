package skyobject

import (
	"bytes"
	"fmt"
	"reflect"

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
		fmt.Fprintf(w, "%s %s `%s` <%s>",
			sf.Name,
			sf.Type,
			sf.Tag,
			reflect.Kind(sf.Kind).String())
		if i != len(s.Fields)-1 {
			w.WriteString("; ")
		}
	}
	w.WriteByte('}')
	return w.String()
}

func getSchema(i interface{}) Schema {
	return getSchemaOfType(
		reflect.Indirect(
			reflect.ValueOf(i),
		).Type(),
	)
}

func getSchemaOfType(typ reflect.Type) (s Schema) {
	var nf int = typ.NumField()
	s.Name = typ.Name()
	if nf == 0 {
		return
	}
	s.Fields = make([]encoder.StructField, 0, nf)
	for i := 0; i < nf; i++ {
		ft := typ.Field(i)
		if ft.Tag.Get("enc") == "-" || ft.Name == "_" || ft.PkgPath != "" {
			continue
		}
		if ft.Type.Kind() == reflect.Struct {
			nt := getSchemaOfType(ft.Type)
			for _, nf := range nt.Fields {
				nf.Name = ft.Name + "." + nf.Name
				s.Fields = append(s.Fields, nf)
			}
		} else {
			s.Fields = append(s.Fields, getField(ft))
		}
	}
	return
}

func getField(ft reflect.StructField) (sf encoder.StructField) {
	sf.Name = ft.Name
	sf.Type = ft.Type.Name()
	sf.Tag = string(ft.Tag)
	sf.Kind = uint32(ft.Type.Kind())
	return
}
