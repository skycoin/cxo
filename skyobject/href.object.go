package skyobject

import (
	"github.com/skycoin/cxo/encoder"
	"github.com/skycoin/skycoin/src/cipher"
)

var _linkSchemaKey cipher.SHA256 = cipher.SumSHA256(encoder.Serialize(*ReadSchema(HashLink{})))

type HashLink Href

func NewLink(value interface{}) HashLink {
	res := HashLink{value:value}
	return res
}

func (h *HashLink) SetData(data []byte) {
	h.rdata = data
}

func (h *HashLink) save(c ISkyObjects) Href {
	objSchema := ReadSchema(h.value)
	objData := encoder.Serialize(h.value)
	objHash := href{Type:c.SaveObject(*objSchema), Data:objData}
	objKey := c.SaveObject(objHash)

	h.Ref = c.SaveObject(&href{Type:h.Type(), Data:objKey[:]})
	h.rdata = objKey[:]
	return Href(*h)
}

func (h *HashLink) Type() cipher.SHA256 {
	return _linkSchemaKey
}

func (h *HashLink) References(c ISkyObjects) RefInfoMap {
	result := RefInfoMap{}

	var objKey cipher.SHA256
	objKey.Set(h.rdata)
	objHash := href{}
	objData, _ := c.Get(objKey)
	encoder.DeserializeRaw(objData, &objHash)
	result[objKey] = int32(len(objData))

	objSchema := Schema{}
	objSchemaData, _ := c.Get(objHash.Type)
	encoder.DeserializeRaw(objSchemaData, &objSchema)
	result[objHash.Type] = int32(len(objSchemaData))

	for _, f := range objSchema.Fields {
		if (c.ValidateHashType(f.Type)) {
			var ref Href
			encoder.DeserializeField(objHash.Data, objSchema.Fields, f.Name, &ref.Ref)
			mergeRefs(result, ref.References(c))
		}
	}
	return result
}

func (h *HashLink) String(c ISkyObjects) string{
	var objKey cipher.SHA256
	objKey.Set(h.rdata)
	objHash := href{}
	objData, _ := c.Get(objKey)
	encoder.DeserializeRaw(objData, &objHash)

	objSchema := Schema{}
	objSchemaData, _ := c.Get(objHash.Type)
	encoder.DeserializeRaw(objSchemaData, &objSchema)
	return objSchema.Name
}
