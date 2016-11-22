package schema

import (
	"testing"
	"bytes"
	"fmt"
	"github.com/skycoin/cxo/encoder"
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
	Field2 HrefStatic `href:"TestSchemaStruct1"`
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
	var t TestSchemaStruct
	t.Field1 = 255
	t.Field2 = []byte("TEST1")
	schema := ExtractSchema(t)
	fmt.Println(schema.String())
	res := encoder.Serialize(t)
	schemeData := encoder.Serialize(schema)

	fmt.Println(res)
	fmt.Println(schemeData)
}
//
//func Test_Schema_2(T *testing.T) {
//	var t1 TestSchemaStruct1
//	t1.Field1 = 255
//	t2 := TestSchemaStruct2{}
//	t2.Field1 = 111
//	t2.Field2 = CreateStaticHref(t2)
//	schema := ExtractSchema(t2)
//	if bytes.Compare(schema.StructName, []byte("TestSchemaStruct2")) != 0 {
//		T.Fatal("Struct name is not equal")
//	}
//	fmt.Println(schema.String())
//}
