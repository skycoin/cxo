package skyobject

import (
	"github.com/skycoin/cxo/encoder"
	"github.com/skycoin/skycoin/src/cipher"
	"reflect"
)

//var _arraySchemaKey cipher.SHA256 = cipher.SumSHA256(encoder.Serialize(*ReadSchema(HashSlice{})))

type HashArray Href

func NewArray(items ...interface{}) HashArray {
	h := HashArray{value: items}
	h.rtype = encoder.Serialize(ReadSchema(HashArray{}))
	return h
}

func (h *HashArray) SetData(tp []byte, data []byte) {
	h.rdata = data
}

func (h *HashArray) save(c ISkyObjects) Href {
	typeKey := c.SaveData(_schemaType, h.rtype)
	v := h.value.([]interface{})
	items := InterfaceSlice(v[0])
	keys := []cipher.SHA256{}

	for _, v := range items {
		obj := NewObject(v)
		r := obj.save(c)
		keys = append(keys, r.Ref)
	}

	h.rdata = encoder.Serialize(keys)
	h.Ref = c.SaveData(typeKey, h.rdata)
	return Href(*h)
}

func (h *HashArray) Type() cipher.SHA256 {
	return cipher.SumSHA256(h.rtype)
}

func (h *HashArray) References(c ISkyObjects) RefInfoMap {
	result := RefInfoMap{}
	items := []cipher.SHA256{}
	encoder.DeserializeRaw(h.rdata, &items)
	for _, k := range items {
		ref := Href{Ref: k}
		mergeRefs(result, ref.References(c))
	}
	return result
}

func (h *HashArray) String(c ISkyObjects) string {
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
