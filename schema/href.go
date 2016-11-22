package schema

import (
	"github.com/skycoin/skycoin/src/cipher"
)

//type HRef struct {
//	context interface{}
//}

type HrefTyped struct {
	Href
	Type []byte //the schema of object
}

type Href struct {
	Hash cipher.SHA256
}


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
