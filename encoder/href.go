package encoder

import (
	"fmt"
	"github.com/skycoin/skycoin/src/cipher"
)

type Ref struct {
	context interface{}
}

type HrefInfo struct {
	has []Href
	no  []Href
}

////For implementing Functors and Applicatives
type IRef interface {
	//Map(Morphism) HRef
	//Value() interface{}
	Expand(source *Store, info *HrefInfo)
}

func (h *Ref) Expand(source *Store, info *HrefInfo) {
	h.context.(IRef).Expand(source, info)
}

type Href struct {
	Hash cipher.SHA256
	Type []byte //the schema of object
}

func NewHref(key cipher.SHA256, value interface{}) Href {
	schema := ExtractSchema(value)
	result := Href{Type:Serialize(schema)}
	result.Hash = key
	return result
}

func (h Href) ToObject(s *Store, o interface{}) {
	data, _ := s.Get(h.Hash)
	DeserializeRaw(data, o)
}

func (h Href) Expand(source *Store, info *HrefInfo) {
	if (source.has(h.Hash)) {
		info.has = append(info.has, h)
		data, _ := source.Get(h.Hash)

		schema := StructSchema{}
		DeserializeRaw(h.Type, &schema)

		for i := 0; i < len(schema.StructFields); i++ {
			f := schema.StructFields[i]
			switch string(f.FieldType) {
			case "encoder.Href":
				href := Href{}
				DeserializeField(data, schema, string(f.FieldName), &href)
				href.Expand(source, info)
			case "encoder.HArray":
				harray := HArray{}
				DeserializeField(data, schema, string(f.FieldName), &harray)
				harray.Expand(source, info)
			}
		}
	} else {
		fmt.Println("Source doesn't have a Key")
		info.no = append(info.no, h)
	}
}


//
//type Morphism func(*Store, interface{}) interface{}
//
////FMap is a Haskel fmap implementation
//func (h HRef) Map(m Morphism) HRef {
//	return h.context.(IHRef).Map(m)
//}
//
////Value extracts from a functor
//func (h HRef) Value() interface{} {
//	return h.context.(IHRef).Value()
//}

//var HrefToBinary Morphism = func(source *Store, item interface{}) interface{} {
//	obj, _ := source.Get(item.(HrefStatic).Hash)
//	return obj
//}

