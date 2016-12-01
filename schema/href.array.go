package schema

import (
	"reflect"
)

type HArray struct {
	Items []HKey
	Type  HKey
}

func newArray(schemaKey HKey, hrefList ...HKey) HArray {
	return HArray{Type: schemaKey, Items:hrefList[:]}
}

func (h HArray) Append(key HKey) HArray {
	h.Items = append(h.Items, key)
	return h
}

func (h HArray) ToObjects(s *Container, o interface{}) interface{} {
	resultType := reflect.TypeOf(o)
	slice := reflect.MakeSlice(reflect.SliceOf(resultType), 0, 0)
	for i := 0; i < len(h.Items); i++ {
		ptr := reflect.New(resultType).Interface()
		href := Href{Hash:h.Items[i], Type:h.Type}
		href.ToObject(s, ptr)
		sv := reflect.ValueOf(ptr).Elem()
		slice = reflect.Append(slice, sv)
	}
	return slice.Interface()
}

func (h HArray) Expand(source *Container, info *HrefInfo) {
	Href{Hash:h.Type}.Expand(source, info)
	for i := 0; i < len(h.Items); i++ {
		href := Href{Hash:h.Items[i], Type:h.Type}
		href.Expand(source, info)
	}
}
