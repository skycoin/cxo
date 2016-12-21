package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/cxo/encoder"
	"fmt"
)

//var _rootSchemaKey cipher.SHA256 = cipher.SumSHA256(encoder.Serialize(ReadSchema(HashRoot{})))

type rootObject struct {
	Root HashObject
	Size int32
	Sign cipher.Sig
}

type HashRoot Href

func newRoot(ref Href, sign cipher.Sig) HashRoot {
	res := HashRoot{value:rootObject{Root:HashObject{Ref:ref.Ref}, Sign:sign}}
	return res
}

func (h *HashRoot) SetData(tp []byte, data []byte) {
	h.rdata = data
}

func (h *HashRoot) Type() cipher.SHA256 {
	return cipher.SumSHA256(h.rtype)
}

func (h *HashRoot) save(c ISkyObjects) Href {
	value := h.value.(rootObject)
	for _, s := range value.Root.References(c) {
		value.Size += s
	}
	objSchema := ReadSchema(h.value)
	schemaKey := c.SaveData(_schemaType, encoder.Serialize(objSchema))
	h.rdata = encoder.Serialize(h.value)
	h.Ref = c.SaveData(schemaKey, h.rdata)
	h.value = value
	return Href(*h)
}

func (h *HashRoot) References(c ISkyObjects) RefInfoMap {
	fmt.Println("Root References ")
	value := rootObject{}
	encoder.DeserializeRaw(h.rdata, &value)
	return value.Root.References(c)
}

func (h *HashRoot) String(c ISkyObjects) string {
	return ""
}
