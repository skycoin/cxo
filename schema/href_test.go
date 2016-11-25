package schema

import (
	"testing"
	"fmt"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/cxo/encoder"
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

type TestHrefDynamic struct {
	Field1 uint32
	//Field2 HDynamic
}

type TestHrefArray struct {
	Field1 uint32
	Field2 HArray
	Field3 uint32
	Field4 uint32
}

type TestHrefAll struct {
	Field1 Href     `type:"TestHrefStruct"`
	Field2 HArray   `type:"TestHrefStruct"`
	//Field3 HDynamic `type:"TestHrefStruct"`
	Field4 uint32
}

type Test1 struct {
	F1 int32
	F2 []int32
	F3 int32
}

func Test_Href_1(T *testing.T) {
	var t TestHrefStruct
	t.Field1 = 255
	t.Field2 = []byte("TEST1")
}

func Test_Href_Array(T *testing.T) {
	store := NewStore()

	t1 := TestHref1{Field1:25}
	h1, _ := store.Save(t1)

	t2 := TestHref1{Field1:77}
	h2, _ := store.Save(t2)
	h := HArray{h1, h2}

	fmt.Println("Hashes", h)
}
//
//func Test_Href_Dynamic(T *testing.T) {
//	store := NewStore()
//
//	t1 := TestHref1{Field1:25}
//	key1, _ := store.Save(t1)
//
//	//t2 := TestHrefDynamic{Field1:77}
//	//t2.Field2 = HrefDynamic(key1, t1)
//	//key2, _ := store.Save(t2)
//
//	var res TestHrefDynamic
//	store.Load(key2, &res)
//
//	fmt.Println(res.Field2.GetSchema())
//	//
//	if (string(res.Field2.GetSchema().StructName) != "TestHref1") {
//		T.Fatal("Schema type is not equal")
//	}
//}

func Test_Get_Property_Value_Href(T *testing.T) {
	t1 := TestHref1{Field1:77}
	data1 := encoder.Serialize(t1)
	var t TestHref2
	t.Field1 = 32
	t.Field2 = Href{Hash:cipher.SumSHA256(data1)}

	schema := ExtractSchema(t)
	data := encoder.Serialize(t)

	//schemeData := encoder.Serialize(schema)
	fmt.Println("data", data)

	var f uint32
	encoder.DeserializeField(data, schema.StructFields, "Field1", &f)
	fmt.Println(f)

	var f2 Href
	encoder.DeserializeField(data, schema.StructFields, "Field2", &f2)
	fmt.Println("f2.Hash", f2.Hash)

	if (f2.Hash != t.Field2.Hash) {
		T.Fatal("Hash is not equal")
	}
}
//
//func Test_Get_Property_Value_Href_Dynamic(T *testing.T) {
//	t1 := TestHref1{Field1:55}
//	data1 := Serialize(t1)
//	var t TestHrefDynamic
//	t.Field1 = 32
//	t.Field2 = HDynamic{Href:Href{Hash:cipher.SumSHA256(data1)}, Schema: Serialize(ExtractSchema(t1))}
//
//	schema := ExtractSchema(t)
//	data := Serialize(t)
//
//	var f uint32
//	DeserializeField(data, schema, "Field1", &f)
//	if (f != t.Field1) {
//		T.Fatal("Fields value is not equal")
//	}
//
//	var f2 HDynamic
//	DeserializeField(data, schema, "Field2", &f2)
//
//	if (f2.Hash != t.Field2.Hash) {
//		T.Fatal("Hash is not equal")
//	}
//}

func Test_Get_Property_Value_Href_Array(T *testing.T) {
	t1 := TestHref1{Field1:55}
	data1 := encoder.Serialize(t1)
	t2 := TestHref1{Field1:55}
	data2 := encoder.Serialize(t2)

	var t TestHrefArray
	t.Field1 = 32
	t.Field2 = HArray{Href{Hash:cipher.SumSHA256(data1)}, Href{Hash:cipher.SumSHA256(data2)}}
	t.Field3 = 77
	t.Field4 = 99
	schema := ExtractSchema(t)
	data := encoder.Serialize(t)
	fmt.Println("data", data)

	var f2 HArray
	encoder.DeserializeField(data, schema.StructFields, "Field2", &f2)

	fmt.Println(f2)
	if (f2[1].Hash != t.Field2[1].Hash) {
		T.Fatal("Hash is not equal")
	}
}

func Test_Get_Property_Value_Href_All(T *testing.T) {
	store := NewStore()

	t1 := TestHrefStruct{Field1:1, Field2:[]byte("TEST1")}
	h1, _ := store.Save(t1)

	t2 := TestHrefStruct{Field1:3, Field2:[]byte("TEST2")}
	h2, _ := store.Save(t2)

	t3 := TestHrefStruct{Field1:5, Field2:[]byte("TEST3")}
	h3, _ := store.Save(t3)
	//
	//t4 := TestHrefStruct{Field1:7, Field2:[]byte("TEST4")}
	//key4, _ := store.Save(t4)
	//
	//th1 := TestHref2{Field1:7, Field2:Href{Hash:key4}}
	//keyTh1, _ := store.Save(th1)

	a := TestHrefAll{Field1:h1, Field2:HArray{h2, h3}, Field4: 88}
	h, _ := store.Save(a)
	sch := ExtractSchema(a)

	res, _ := store.Get(h.Hash)

	var f2 HArray
	encoder.DeserializeField(res, sch.StructFields, "Field2", &f2)
	if (f2[1].Hash != a.Field2[1].Hash) {
		T.Fatal("Hash is not equal")
	}

	var f4 uint32
	encoder.DeserializeField(res, sch.StructFields, "Field4", &f4)
	if (f4 != a.Field4) {
		T.Fatal("Fields are not equal")
	}
}

func Test_Get_Property_Value_Href_All_One_Missing(T *testing.T) {
	store := NewStore()

	t1 := TestHrefStruct{Field1:1, Field2:[]byte("TEST1")}
	h1, _ := store.Save(t1)

	t2 := TestHrefStruct{Field1:3, Field2:[]byte("TEST2")}
	h2, _ := store.Save(t2)

	t3 := TestHrefStruct{Field1:5, Field2:[]byte("TEST3")}
	h3, _ := store.Save(t3)

	t4 := TestHrefStruct{Field1:8, Field2:[]byte("TEST5")}
	d4 := encoder.Serialize(t4)
	h4 := cipher.SumSHA256(d4)
	s4 := ExtractSchema(t4)
	a := TestHrefAll{Field1:h1, Field2:HArray{h2, h3, Href{Hash:h4, Type:encoder.Serialize(s4)}}, Field4: 88}
	root, _ := store.Save(a)

	info := HrefInfo{}
	root.Expand(store, &info)

	if (len(info.has) != 4) {
		T.Fatal("Count of objects in DB are not equal")
	}
	if (len(info.no) != 1) {
		T.Fatal("Count of missing in DB are not equal")
	}

}
