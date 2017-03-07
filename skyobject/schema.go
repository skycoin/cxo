package skyobject

import (
	"reflect"
	"strings"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// A schema represents schema-wire-type
type schema struct {
	Name   string
	Fields []encoder.StructField
}

func (c *Container) getSchema(i interface{}) (s schema) {
	c.getSchemaOf(reflect.TypeOf(i), reflect.ValueOf(i))
}

func (c *Container) getSchemaOf(typ reflect.Type,
	val reflect.Value) (s schema) {

	s.Name = typ.Name()
	nf := typ.NumField()
	s.Fields = make([]encoder.StructField, 0, nf)
	for i := 0; i < nf; i++ {
		ft := typ.Field(i)
		if ft.Tag.Get("enc") != "-" {
			s.Fields = append(s.Fields, c.getField(typ, ft))
		}
	}
	return
}

func (c *Container) getField(pt reflect.Type,
	ft reflect.StructField) (sf encoder.StructField) {

	var schemaTag string

	if isSHA256(ft.Type) {
		if fieldTypeName, ok := schemaTagValue(ft); ok {
			// if relative type name like Thread
			if strings.Contains(fieldTypeName, ".") {
				// make it absolute
				if pp := pt.PkgPath(); pp == "" {
					panic("invalid parent type: " + pt.Name())
				} else {
					fieldTypeName = pp + "." + fieldTypeName
				}
			}
			// lets look registry
			st, ok := c.registry[fieldTypeName]
			if !ok {
				panic("unregistered type: " + fieldTypeName)
			}
			// same of hex
			if fieldTypeName == pt.Name() {
				schemaTag = "same"
			} else {
				schemaTag = c.getSchemaKeyOf(st).Hex()
			}
		}
	}

	fieldType := strings.ToLower(fieldValue.Type().Name())
	sf.Name = ft.Name
	sf.Type = strings.ToLower(ft.Type.Name())
	sf.Kind = uint32(ft.Type.Kind())

	if schemaTag != "" {
		//
	} else {
		sf.Tag = ft.Tag
	}

}

func (c *Container) getSchemaKeyOf(typ reflect.Type) (sk cipher.SHA256) {
	var ok bool
	if sk, ok = c.schemas[typ]; !ok {
		sch := c.getSchemaOf(typ, reflect.New(typ))
		sk = c.db.AddAutoKey(encoder.Serialize(sch))
		c.schemas[typ] = sk
	}
	return
}

// is given type cipher.SHA256 or []cipher.SHA256
func isSHA256(typ reflect.Type) bool {
	return typ == reflect.TypeOf(cipher.SHA256{}) ||
		typ == reflect.TypeOf([]cipher.SHA256{})
}

func shemaTagValue(ft reflect.StructField) (sv string, ok bool) {
	if enc := ft.Tag.Get("enc"); strings.Contains(enc, "schema=") {
		for _, part := range strings.Split(enc, ",") {
			if strings.HasPrefix(part, "schema=") {
				ss := strings.Split(part, "=")
				if len(ss) != 2 {
					panic("invalid schema tag: ", part)
				}
				sv, ok = ss[1], true
				return
			}
		}
	}
	return
}
