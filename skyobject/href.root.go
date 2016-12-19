package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/cxo/encoder"
)

var _rootSchemaKey cipher.SHA256 = cipher.SumSHA256(encoder.Serialize(ReadSchema(HashRoot{})))

type rootObject struct {
	Root Href
	Size int32
	Sign cipher.Sig
}

type HashRoot Href

func newRoot(ref Href, sign cipher.Sig) HashRoot {
	res := HashRoot{value:rootObject{Root:ref, Sign:sign}}
	return res
}

func (h *HashRoot) SetData(data []byte) {
	h.rdata = data
}

func (h *HashRoot) Type() cipher.SHA256 {
	return _rootSchemaKey
}

func (h *HashRoot) save(c ISkyObjects) Href {
	value := h.value.(rootObject)
	//for _, s := range value.Root.References(c) {
	//	value.Size += s
	//}
	h.value = value
	//
	//objSchema := ReadSchema(h.value)
	//objData := encoder.Serialize(h.value)
	//objHash := href{Type:c.SaveObject(*objSchema), Data:objData}
	//objKey := c.SaveObject(objHash)
	//
	//h.Ref = c.SaveObject(href{Type:h.Type(), Data:objKey[:]})
	//h.rdata = objKey[:]
	return Href(*h)
}

func (h *HashRoot) References(c ISkyObjects) RefInfoMap {
	value := rootObject{}

	var objKey cipher.SHA256
	objKey.Set(h.rdata)
	objHash := href{}

	objData, _ := c.Get(objKey)
	encoder.DeserializeRaw(objData, &objHash)
	encoder.DeserializeRaw(objHash.Data, &value)
	return value.Root.References(c)
}

func (h *HashRoot) String(c ISkyObjects) string {
	return ""
}
