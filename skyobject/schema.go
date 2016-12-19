package skyobject

import (
	"reflect"
	"bytes"
	"strings"
	"github.com/skycoin/cxo/encoder"
)

type Schema struct {
	Name   string        `json:"name"`
	Fields []encoder.StructField `json:"fields"`
}

func ReadSchema(data interface{}) Schema {
	st := reflect.TypeOf(data)
	sv := reflect.ValueOf(data)
	result := &Schema{Name:st.Name(), Fields:[]encoder.StructField{}}
	for i := 0; i < st.NumField(); i++ {
		if (st.Field(i).Tag.Get("enc") != "-") {
			result.Fields = append(result.Fields, getField(st.Field(i), sv.Field(i)))
		}
	}
	return *result
}

func (s *Schema) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("struct " + string(s.Name) + "\n")
	for i := 0; i < len(s.Fields); i++ {
		buffer.WriteString(s.Fields[i].String())
	}
	return buffer.String()
}



func getField(field reflect.StructField, fieldValue reflect.Value) encoder.StructField {
	var fieldTag reflect.StructTag
	fieldType := strings.ToLower(fieldValue.Type().Name())
	return encoder.StructField{Name:field.Name, Type:fieldType, Tag:string(fieldTag), Kind:uint32(fieldValue.Type().Kind())}
}
