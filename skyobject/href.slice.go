package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/sqdron/squad/encoder"
	"reflect"
)

var _sliceSchemaKey cipher.SHA256 = cipher.SumSHA256(encoder.Serialize(*ReadSchema(HashSlice{})))

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
	itemType := c.SaveData(h.rtype)
	for _, v := range items {
		itemHash := &href{Type:itemType, Data:encoder.Serialize(v)}
		item := c.SaveObject(itemHash)
		keys = append(keys, item)
	}
	data := encoder.Serialize(keys)
	h.Ref = c.SaveObject(&href{Type:h.Type(), Data:data})
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
		mergeRefs(result, ref.References(c))
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
