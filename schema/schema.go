package schema

import (
	"reflect"
	"bytes"
	"github.com/skycoin/cxo/encoder"
)

//type TypeDefinition struct {
//	FieldName []byte
//	FieldType []byte
//	FieldTag  []byte
//}

type StructSchema struct {
	StructName   []byte
	StructFields []encoder.ReflectionField
}

func ExtractSchema(data interface{}) StructSchema {
	st := reflect.TypeOf(data)
	sv := reflect.ValueOf(data)
	result := StructSchema{StructName:[]byte(st.Name()), StructFields:[]encoder.ReflectionField{}}
	for i := 0; i < st.NumField(); i++ {
		result.StructFields = append(result.StructFields, getField(st.Field(i), sv.Field(i)))
	}
	return result
}

func (s *StructSchema) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("struct " + string(s.StructName) + "\n")
	for i := 0; i < len(s.StructFields); i++ {
		buffer.WriteString(s.StructFields[i].String())
	}
	return buffer.String()
}

//func (s *TypeDefinition) string() string {
//	return fmt.Sprintln(string(s.FieldName), string(s.FieldType), string(s.FieldTag))
//}

func getField(field reflect.StructField, fieldValue reflect.Value) encoder.ReflectionField {
	//fmt.Println("fieldValue.Type()", fieldValue.Type())
	//return NameTypePair{FieldName:[]byte(field.Name), FieldType:[]byte(fieldValue.Kind().String()), FieldTag:[]byte(field.Tag)}
	return encoder.ReflectionField{Name:field.Name, Type:fieldValue.Type().String(), Tag:string(field.Tag)}
}
