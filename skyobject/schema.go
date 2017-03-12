package skyobject

import (
	"bytes"
	"fmt"
	"reflect"

	//"github.com/skycoin/skycoin/src/cipher/encoder"
	"github.com/logrusorgru/skycoin/src/cipher/encoder"
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
		typ reflect.Type = reflect.Indirect(reflect.ValueOf(i)).Type()
		nf  int          = typ.NumField()
	)
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
		s.Fields = append(s.Fields, getField(ft))
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
