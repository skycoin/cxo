package schema

import (
	"reflect"
)

type HArray struct {
	Items  []HKey
	Schema HKey
}

func newArray(schemaKey HKey, hrefList ...HKey) HArray {
	return HArray{Schema: schemaKey, Items:hrefList[:]}
}

func (h HArray) Append(key HKey) HArray {
	h.Items = append(h.Items, key)
	return h
}

func (h HArray) ToObjects(s *Store, o interface{}) interface{} {
	resultType := reflect.TypeOf(o)
	slice := reflect.MakeSlice(reflect.SliceOf(resultType), 0, 0)
	for i := 0; i < len(h.Items); i++ {
		ptr := reflect.New(resultType).Interface()
		schema, _ := s.Get(h.Schema)
		href := Href{Hash:h.Items[i],Type:schema}
		href.ToObject(s, ptr)
		sv := reflect.ValueOf(ptr).Elem()
		slice = reflect.Append(slice, sv)
	}
	return slice.Interface()
}



func (h HArray) Expand(source *Store, info *HrefInfo) {
	Href{Hash:h.Schema}.Expand(source, info)
	for i := 0; i < len(h.Items); i++ {
		schema, _ := source.Get(h.Schema)
		href := Href{Hash:h.Items[i],Type:schema}
		//fmt.Println("Item expand for href: ", href)
		href.Expand(source, info)
	}
}
