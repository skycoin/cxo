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

func getSchema(i interface{}) (s Schema) {
	var (
		val reflect.Value
		typ reflect.Type
		nf  int
	)
	val = reflect.Indirect(reflect.ValueOf(i))
	typ = val.Type()
	nf = typ.NumField()
	s.Name = typ.Name()
	if nf == 0 {
		return
	}
	s.Fields = make([]encoder.StructField, 0, nf)
	for i := 0; i < nf; i++ {
		ft := typ.Field(i)
		fv := val.Field(i)
		if ft.Tag.Get("enc") == "-" || ft.Name == "_" || ft.PkgPath != "" {
			continue
		}
		// use fields of embeded struct as
		// field of the struct
		if fv.Kind() == reflect.Struct {
			es := getSchema(fv.Interface())
			for _, ef := range es.Fields {
				ef.Name = ft.Name + "." + ef.Name // EbededStructName.FieldName
				ef.Tag = string(ft.Tag)           // use tag of parent
				s.Fields = append(s.Fields, ef)
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
