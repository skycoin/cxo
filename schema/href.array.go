package schema

import (
	"reflect"
	"github.com/skycoin/skycoin/src/cipher"
)

type HArray struct {
	Items [] cipher.SHA256
	Type  cipher.SHA256
}

func newArray(schemaKey cipher.SHA256, hrefList ...cipher.SHA256) HArray {
	return HArray{Type: schemaKey, Items:hrefList[:]}
}

func (h HArray) Append(key cipher.SHA256) HArray {
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

func (h HArray) ExpandBySchema(source *Container, schemaKey cipher.SHA256, query *HrefQuery) {
	if (h.Type == schemaKey){
		query.Items = append(query.Items, h.Items...)
		return
	}
	for i := 0; i < len(h.Items); i++ {
		href := Href{Hash:h.Items[i], Type:h.Type}
		href.ExpandBySchema(source, schemaKey, query)
	}
}
