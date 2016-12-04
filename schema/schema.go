package schema

import (
	"reflect"
	"bytes"
	"github.com/skycoin/cxo/encoder"
)

type Schema struct {
	Name   string `json:"name"`
	Fields []encoder.ReflectionField `json:"fields"`
}

func ExtractSchema(data interface{}) Schema {
	st := reflect.TypeOf(data)
	sv := reflect.ValueOf(data)
	result := Schema{Name:st.Name(), Fields:[]encoder.ReflectionField{}}
	for i := 0; i < st.NumField(); i++ {
		result.Fields = append(result.Fields, getField(st.Field(i), sv.Field(i)))
	}
	return result
}

func (s *Schema) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("struct " + string(s.Name) + "\n")
	for i := 0; i < len(s.Fields); i++ {
		buffer.WriteString(s.Fields[i].String())
	}
	return buffer.String()
}

func getField(field reflect.StructField, fieldValue reflect.Value) encoder.ReflectionField {
	fieldType := ""
	var fieldTag reflect.StructTag

	fieldType = fieldValue.Kind().String()
	switch fieldValue.Kind() {
	case reflect.Struct:
		switch field.Type {
		case reflect.TypeOf(Href{}):
			fieldTag = `href:"object"`
		case reflect.TypeOf(HArray{}):
			fieldTag = `href:"array"`
		}
	}

	return encoder.ReflectionField{Name:field.Name, Type:fieldType, Tag:string(fieldTag)}
}


