package schema

import (
	"testing"
)

type TestHrefStruct struct {
	Field1 int32
	Field2 []byte
}

type TestHref1 struct{
	Field1 uint32
}

type TestHref2 struct{
	Field1 uint32
	Field2 HrefStatic `hrefType:"TestHref1"`
}

func Test_Href_1(T *testing.T) {
	var t TestHrefStruct
	t.Field1 = 255
	t.Field2 = []byte("TEST1")
}
