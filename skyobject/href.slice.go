package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"reflect"
	"github.com/skycoin/cxo/encoder"
	"fmt"
)

var _sliceSchemaKey cipher.SHA256 = cipher.SumSHA256(encoder.Serialize(ReadSchema(HashSlice{})))

type HashSlice Href

func NewSlice(sliceType interface{}, items ...interface{}) HashSlice {
	objSchema := ReadSchema(sliceType)
	return HashSlice{value:items, rtype:encoder.Serialize(objSchema)}
}

func (h *HashSlice) SetData(data []byte) {
	h.rdata = data
}

func (h *HashSlice) save(c ISkyObjects) Href {
	v := h.value.([]interface{})
	items := InterfaceSlice(v[0])
	keys := []cipher.SHA256{}
	fmt.Println("Lokking object referencies for slice ")
	for _, v := range items {
		obj := NewObject(v)
		item := obj.save(c)
		keys = append(keys, item.Ref)
	}
	data := encoder.Serialize(keys)
	h.Ref = c.SaveObject(h.Type(), data)
	return Href(*h)
}

func (h *HashSlice) Type() cipher.SHA256 {
	return _sliceSchemaKey
}

func (h *HashSlice) References(c ISkyObjects) RefInfoMap {
	result := RefInfoMap{}
	items := []cipher.SHA256{}
	encoder.DeserializeRaw(h.rdata, &items)
	for _, k := range items {
		ref := HashLink{}
		ref.SetData(k[:])
		return ref.References(c)
	}
	return result
}

func (h *HashSlice) String(c ISkyObjects) string {
	return ""
}

func InterfaceSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return []interface{}{}
	}
	ret := make([]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}
	return ret
}
