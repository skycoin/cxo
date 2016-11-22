package schema

import (
	"testing"
	"fmt"
)

type TestHrefStruct struct {
	Field1 int32
	Field2 []byte
}

type TestHref1 struct {
	Field1 uint32
}

type TestHref2 struct {
	Field1 uint32
	Field2 Href `hrefType:"TestHref1"`
}

func Test_Href_1(T *testing.T) {
	var t TestHrefStruct
	t.Field1 = 255
	t.Field2 = []byte("TEST1")
}

type TestHrefDynamic struct {
	Field1 uint32
	Field2 HDynamic
}

//
func Test_Href_Array(T *testing.T) {
	store := NewStore()

	t1 := TestHref1{Field1:25}
	key1, _ := store.Save(t1)

	t2 := TestHref1{Field1:77}
	key2, _ := store.Save(t2)
	h := HArray{Href{Hash:key1}, Href{Hash:key2}}

	fmt.Println("Hashes", h)
}

func Test_Href_Dynamic(T *testing.T) {
	store := NewStore()

	t1 := TestHref1{Field1:25}
	key1, _ := store.Save(t1)

	t2 := TestHrefDynamic{Field1:77}
	t2.Field2 = HrefDynamic(key1, t1)
	key2, _ := store.Save(t2)

	var res TestHrefDynamic
	store.Load(key2, &res)

	fmt.Println(res.Field2.GetSchema())
	//
	if (string(res.Field2.GetSchema().StructName) != "TestHref1") {
		T.Fatal("Schema type is not equal")
	}
}
