package schema

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/cxo/encoder"
)

type Href struct {
	Hash cipher.SHA256
}

func (h Href) ToObject(s *Store, o interface{}) {
	data, _ := s.Get(h.Hash)
	encoder.DeserializeRaw(data, o)
}

//type HRef struct {
//	context interface{}
//}
////For implementing Functors and Applicatives
//type IHRef interface {
//	Map(Morphism) HRef
//	Value() interface{}
//}
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
