package encoder

import (
	"testing"
	"bytes"
	"fmt"
)

type TestSchemaStruct struct {
	Field1 int32
	Field2 []byte
}

type TestSchemaStruct1 struct {
	Field1 int32
}

type TestSchemaStruct2 struct {
	Field1 int32
	Field2 []byte
}

func Test_Schema_1(T *testing.T) {
	var t TestSchemaStruct
	t.Field1 = 255
	t.Field2 = []byte("TEST1")
	schema := ExtractSchema(t)
	if bytes.Compare(schema.StructName, []byte("TestSchemaStruct")) != 0 {
		T.Fatal("Struct name is not equal")
	}

	if bytes.Compare(schema.StructFields[0].FieldName, []byte("Field1")) != 0 {
		T.Fatal("Field name is not equal")
	}
}

func Test_Encode_With_Schema_1(T *testing.T) {
	var t TestSchemaStruct2
	t.Field1 = 255
	t.Field2 = []byte("TEST1")
	//schema := ExtractSchema(t)
	res := Serialize(t)
	//schemeData := encoder.Serialize(schema)
	fmt.Println(res)
	//fmt.Println(schemeData)

	var tt TestSchemaStruct
	DeserializeRaw(res, &tt)
	fmt.Println(tt)
}
