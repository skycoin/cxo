package schema

import (
	"reflect"
	"fmt"
	"bytes"
)

type NameTypePair struct {
	FieldName []byte
	FieldType []byte
	FieldTag  []byte
}

type StructSchema struct {
	StructName   []byte
	StructFields []NameTypePair
}

func ExtractSchema(data interface{}) StructSchema {
	st := reflect.TypeOf(data)
	sv := reflect.ValueOf(data)
	result := StructSchema{StructName:[]byte(st.Name()), StructFields:[]NameTypePair{}}
	for i := 0; i < st.NumField(); i++ {
		result.StructFields = append(result.StructFields, extractField(st.Field(i), sv.Field(i)))
	}
	return result
}

func (s *StructSchema) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("struct " + string(s.StructName) + "\n")
	for i := 0; i < len(s.StructFields); i++ {
		buffer.WriteString(s.StructFields[i].string())
	}
	return buffer.String()
}

func (s *NameTypePair) string() string {
	return fmt.Sprintln(string(s.FieldName), string(s.FieldType), string(s.FieldTag))
}

func extractField(field reflect.StructField, fieldValue reflect.Value) NameTypePair {
	return NameTypePair{FieldName:[]byte(field.Name), FieldType:[]byte(fieldValue.Kind().String()), FieldTag:[]byte(field.Tag)}
}
