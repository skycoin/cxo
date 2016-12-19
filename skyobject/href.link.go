package skyobject

import (
	"github.com/skycoin/cxo/encoder"
	"github.com/skycoin/skycoin/src/cipher"
	"fmt"
)

var linkSchema cipher.SHA256 = cipher.SumSHA256(encoder.Serialize(ReadSchema(HashLink{})))

type linkValue cipher.SHA256

type HashLink Href

func NewLink(value interface{}) HashLink {
	res := HashLink{value:value}
	return res
}

func (h *HashLink) SetData(data []byte) {
	h.rdata = data
}

func (h *HashLink) Type() cipher.SHA256 {
	return linkSchema
}

func (h *HashLink) save(c ISkyObjects) Href {

	//objSchema := ReadSchema(h.value)
	objData := encoder.Serialize(h.value)
	//objHash := href{Type:c.SaveObject(*objSchema), Data:objData}
	//objKey := c.SaveObject(objHash)
	//
	h.Ref = c.SaveObject(h.Type(), objData)
	return Href(*h)
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

	fmt.Println("Lokking object referencies for data ", objData)
	encoder.DeserializeRaw(objSchemaData, &objSchema)
	fmt.Println("Lokking object referencies for type ", objSchema.Name)
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

//func (h *HashLink) Fields(key cipher.SHA256) (map[string]string){}

func (h *HashLink) String(c ISkyObjects) string {
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
