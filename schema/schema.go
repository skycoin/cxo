package schema

import (
	"reflect"
	"bytes"
	"github.com/skycoin/cxo/encoder"
)

type Schema struct {
	StructName   string
	StructFields []encoder.ReflectionField
}

func ExtractSchema(data interface{}) Schema {
	st := reflect.TypeOf(data)
	sv := reflect.ValueOf(data)
	result := Schema{StructName:st.Name(), StructFields:[]encoder.ReflectionField{}}
	for i := 0; i < st.NumField(); i++ {
		result.StructFields = append(result.StructFields, getField(st.Field(i), sv.Field(i)))
	}
	return result
}

func (s *Schema) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("struct " + string(s.StructName) + "\n")
	for i := 0; i < len(s.StructFields); i++ {
		buffer.WriteString(s.StructFields[i].String())
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


