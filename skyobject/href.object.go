package skyobject


import (
	"github.com/skycoin/cxo/encoder"
	"github.com/skycoin/skycoin/src/cipher"
)


type HashObject Href

func NewObject(value interface{}) HashObject {
	res := HashObject{value:value}
	sm := ReadSchema(value)
	res.rtype = encoder.Serialize(sm)
	return res
}

func (h *HashObject) SetData(data []byte) {
	h.rdata = data
}

func (h *HashObject) Type() cipher.SHA256 {
	return cipher.SumSHA256(h.rtype)
}

func (h *HashObject) save(c ISkyObjects) Href {
	typeKey := c.SaveObject(_schemaType, h.rtype)
	h.rdata = encoder.Serialize(h.value)
	h.Ref = c.SaveObject(typeKey, h.rdata)
	return Href(*h)
}

func (h *HashObject) References(c ISkyObjects) RefInfoMap {
	result := RefInfoMap{}
	objSchema := Schema{}
	encoder.DeserializeRaw(h.rtype, &objSchema)
	for _, f := range objSchema.Fields {
		if (c.ValidateHashType(f.Type)) {
			var ref Href
			encoder.DeserializeField(h.rdata, objSchema.Fields, f.Name, &ref.Ref)
			mergeRefs(result, ref.References(c))
		}
	}

	return result
}

func (h *HashObject) String(c ISkyObjects) string{
	return ""
}
