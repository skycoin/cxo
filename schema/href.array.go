package schema

import (
	"reflect"
	"github.com/skycoin/cxo/encoder"
)

type HArray []Href

func NewHArray() HArray {
	return HArray{}
}

//func HrefArrayEmpty(store *Store) HRef {
//		return HRef{context: HArray{Store:store}}
//}
//
//func HrefArray(store *Store, slice []interface{}) HRef {
//	if slice == nil {
//		return HRef{context: HArray{Store:store}}
//	}
//	var lst HArray =  HArray{Store:store, Items:slice}
//	return HRef{context: lst}
//}

//func (h HArray) Value() []HrefStatic {
//	return h.Items
//}
//
////Map is a Map implementation for a list
//func (h HArray) Map(f Morphism) HRef {
//	result := HArray{Store:h.Store}
//	itemsValue := reflect.ValueOf(h.Items)
//	newItems := []interface{}{}
//	for i := 0; i < itemsValue.Len(); i++ {
//		newItems = append(newItems, f(h.Store, itemsValue.Index(i).Interface()))
//	}
//	result.Items = newItems
//	return HRef{context: result}
//}

//func (h HArray) ToBinary(s *Store) [][]byte {
//	var result [][]byte = [][]byte{}
//	itemsValue := reflect.ValueOf(h)
//	for i := 0; i < itemsValue.Len(); i++ {
//		result = append(result, HrefToBinary(s, itemsValue.Index(i).Interface()).([]byte))
//	}
//	return result
//}

func (h HArray) Append(key Href) HArray {
	return append(h, key)
}

func (h HArray) ToObjects(s *Store, o interface{}) interface{} {
	resultType := reflect.TypeOf(o)
	slice := reflect.MakeSlice(reflect.SliceOf(resultType), 0, 0)
	for i := 0; i < len(h); i++ {
		data, _ := s.Get(h[i].Hash)
		ptr := reflect.New(resultType).Interface()
		sv := reflect.ValueOf(ptr).Elem()
		encoder.DeserializeRaw(data, ptr)
		slice = reflect.Append(slice, sv)
	}
	return slice.Interface()
}
