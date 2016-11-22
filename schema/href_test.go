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
	Field2 Href `hrefType:"TestHref1"`
}

func Test_Href_1(T *testing.T) {
	var t TestHrefStruct
	t.Field1 = 255
	t.Field2 = []byte("TEST1")
}


type TestHrefArray1 struct{
	Field1 uint32
	Field2 HArray `href:"TestHref1"`
}
//
//
//func Test_Href_Array(T *testing.T) {
//	store:= NewStore()
//
//	t1:= TestHref1{Field1:25}
//	key1, _ := store.Save(t1)
//
//	t2:= TestHref1{Field1:77}
//	key2, _ := store.Save(t2)
//	h := HrefArray(store, []interface{}{HrefStatic{Hash:key1},HrefStatic{Hash:key2}})
//
//	fmt.Println("Hashes", h.Value())
//
//	res:= h.Map(HrefToBinary).Value()
//fmt.Println("res", res)
//	//resBytes := res.([]interface{})[0].([]byte)
//	//var r TestHref1
//	//encoder.DeserializeRaw(resBytes, &r)
//	//if (r.Field1 != 25){
//	//	T.Fatal("Field value is not equal")
//	//}
//}
//
//func Test_Href_Array_2(T *testing.T) {
//	store:= NewStore()
//
//	t1:= TestHref1{Field1:25}
//	key1, _ := store.Save(t1)
//
//	t2:= TestHref1{Field1:77}
//	key2, _ := store.Save(t2)
//
//	h := HrefArray(store, []interface{}{HrefStatic{Hash:key1},HrefStatic{Hash:key2}})
//
//	var BytesToObject Morphism = func(source *Store, item interface{}) interface{} {
//		bytes := item.([]byte)
//		var r TestHref1
//		encoder.DeserializeRaw(bytes, &r)
//		return r
//	}
//
//	res:= h.Map(HrefToBinary).Map(BytesToObject).Value()
//
//	if (res.([]interface{})[1].(TestHref1).Field1 != 77){
//		T.Fatal("Field value is not equal")
//	}
//	//resBytes := res.([]interface{})[0]
//}
