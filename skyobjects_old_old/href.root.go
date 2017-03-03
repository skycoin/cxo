package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// HashRoot is a root hash item.
type HashRoot Href

type rootObject struct {
	linked   HashArray
	deleted  HashArray
	sequence uint64
}

func newRoot(linked, deleted HashArray, sequence uint64) HashRoot {
	h := HashRoot{
		value: rootObject{linked: linked, deleted: deleted, sequence: sequence},
	}
	h.rtype = encoder.Serialize(ReadSchema(HashRoot{}))
	return h
}

// SetData sets data within HashRoot.
func (h *HashRoot) SetData(tp []byte, data []byte) {
	h.rdata = data
}

func (h *HashRoot) save(c ISkyObjects) Href {
	typeKey := c.SaveData(_schemaType, h.rtype)
	h.rdata = encoder.Serialize(h.value)
	h.Ref = c.SaveData(typeKey, h.rdata)
	c.setRoot(h.Ref)
	return Href(*h)
}

// Type returns the "type": schemaKey.
func (h *HashRoot) Type() cipher.SHA256 {
	return cipher.SumSHA256(h.rtype)
}

// References Returns keys of the linked objects from linked array.
func (h *HashRoot) References(c ISkyObjects) RefInfoMap {
	value := rootObject{}
	encoder.DeserializeRaw(h.rdata, &value)
	return value.linked.References(c)
	// 	return value.Root.References(c)
}

func (h *HashRoot) String(c ISkyObjects) string {
	return ""
}

//var _rootSchemaKey cipher.SHA256 = cipher.SumSHA256(encoder.Serialize(ReadSchema(HashRoot{})))

// type rootObject struct {
// 	Root HashObject
// 	Size int32
// 	Sign cipher.Sig
// }
//
// type HashRoot Href
//
// func newRoot(ref Href, sign cipher.Sig) HashRoot {
// 	res := HashRoot{value: rootObject{Root: HashObject{Ref: ref.Ref}, Sign: sign}}
// 	return res
// }
//
// func (h *HashRoot) SetData(tp []byte, data []byte) {
// 	h.rdata = data
// }
//
// func (h *HashRoot) Type() cipher.SHA256 {
// 	return cipher.SumSHA256(h.rtype)
// }
//
// func (h *HashRoot) save(c ISkyObjects) Href {
// 	value := h.value.(rootObject)
// 	for _, s := range value.Root.References(c) {
// 		value.Size += s
// 	}
// 	objSchema := ReadSchema(h.value)
// 	schemaKey := c.SaveData(_schemaType, encoder.Serialize(objSchema))
// 	h.rdata = encoder.Serialize(h.value)
// 	h.Ref = c.SaveData(schemaKey, h.rdata)
// 	h.value = value
// 	return Href(*h)
// }
//
// func (h *HashRoot) References(c ISkyObjects) RefInfoMap {
// 	value := rootObject{}
// 	encoder.DeserializeRaw(h.rdata, &value)
// 	return value.Root.References(c)
// }
//
// func (h *HashRoot) String(c ISkyObjects) string {
// 	return ""
// }
