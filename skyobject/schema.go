package skyobject

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

type Schema struct {
	Name   string   //
	Kind   uint32   // reflect.Kind
	Elem   []Schema // schema of element of array or slice (pointer)
	Len    uint32   // for arrays only
	Fields []Field  // for structs only
}

type Field struct {
	Name   string            // name of the field
	Schema Schema            // schema of the field
	Tag    reflect.StructTag // tag
}

func getField(ft reflect.StructField) (sf Field) {
	sf.Name = ft.Name
	sf.Schema = getSchemaOfType(ft.Type)
	if tag := skyobjectTag(ft.Tag); strings.Contains(tag, "href") {
		if !(sf.Schema.Name == refTypeName ||
			sf.Schema.Name == refsTypeName ||
			sf.Schema.Name == dynamicTypeName) {
			panic("unexpected references tag :" + tag)
		}
	}
	sf.Tag = ft.Tag
	return
}

func getSchemaOfType(typ reflect.Type) (s Schema) {
	s.Name = typ.Name()
	s.Kind = uint32(typ.Kind())
	switch typ.Kind() {
	case reflect.Bool,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
	case reflect.Slice:
		s.Elem = []Schema{getSchemaOfType(typ.Elem())}
	case reflect.Array:
		s.Elem = []Schema{getSchemaOfType(typ.Elem())}
		s.Len = uint32(typ.Len())
	case reflect.Struct:
		var nf int = typ.NumField()
		if nf == 0 {
			break
		}
		s.Fields = make([]Field, 0, nf)
		var ft reflect.StructField
		for i := 0; i < nf; i++ {
			ft = typ.Field(i)
			if ft.Tag.Get("enc") == "-" || ft.Name == "_" || ft.PkgPath != "" {
				continue
			}
			s.Fields = append(s.Fields, getField(ft))
		}
	default:
		panic("invalid type: " + typ.Kind().String())
	}
	return
}

func getSchema(i interface{}) Schema {
	return getSchemaOfType(
		reflect.Indirect(
			reflect.ValueOf(i),
		).Type(),
	)
}

func (s *Schema) Size(p []byte) (sz int) {
	switch kind := reflect.Kind(s.Kind); kind {
	case reflect.Bool, reflect.Int8, reflect.Uint8:
		sz = 1
	case reflect.Int16, reflect.Uint16:
		sz = 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		sz = 4
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		sz = 8
	case reflect.Slice, reflect.String:
		var (
			ln  uint32
			err error
		)
		defer func() {
			if recover() != nil {
				sz = -1
			}
		}()
		encoder.DeserializeAtomic(p, &ln) // can panic
		sz = 4 + int(ln)
	case reflect.Array:
		if len(s.Elem) != 1 {
			sz = -1
			break
		}
		sz = int(s.Len) * s.Elem[0].Size(p)
	case reflect.Struct:
		for _, sf := range s.Fields {
			if x := sf.Schema.Size(p[sz:]); x < 0 {
				return -1
			} else {
				sz += x
			}
		}
	default:
		sz = -1
	}
	return
}
