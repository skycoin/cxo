package schema

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/cxo/encoder"
)

type Ref struct {
	context interface{}
}

type HrefInfo struct {
	has []cipher.SHA256
	no  []cipher.SHA256
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
	result := Href{Type:encoder.Serialize(schema)}
	result.Hash = key
	return result
}

func (h Href) ToObject(s *Store, o interface{}) {
	data, _ := s.Get(h.Hash)
	encoder.DeserializeRaw(data, o)
}

func (h Href) Expand(source *Store, info *HrefInfo) {
	if (source.has(h.Hash)) {
		info.has = append(info.has, h.Hash)
		data, _ := source.Get(h.Hash)

		schema := StructSchema{}
		encoder.DeserializeRaw(h.Type, &schema)

		for i := 0; i < len(schema.StructFields); i++ {
			f := schema.StructFields[i]
			switch string(f.Type) {
			case "schema.Href":
				href := Href{}
				encoder.DeserializeField(data, schema.StructFields, string(f.Name), &href)
				href.Expand(source, info)
			case "schema.HArray":
				harray := HArray{}
				encoder.DeserializeField(data, schema.StructFields, string(f.Name), &harray)
				harray.Expand(source, info)
			}
		}
	} else {
		info.no = append(info.no, h.Hash)
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

