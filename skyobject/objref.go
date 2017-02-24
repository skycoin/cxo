package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

type ObjRef struct {
	key       cipher.SHA256
	schemaKey cipher.SHA256
	data      []byte
	c         ISkyObjects
}

func (r *ObjRef) GetKey() cipher.SHA256 {
	return r.key
}

func (r *ObjRef) GetSchemaKey() cipher.SHA256 {
	return r.schemaKey
}

func (r *ObjRef) GetSchema() *Schema {
	return r.c.GetSchemaFromKey(r.schemaKey)
}

func (r *ObjRef) Deserialize(dataOut interface{}) error {
	return encoder.DeserializeRaw(r.data, dataOut)
}

func (r *ObjRef) GetFieldAsKey(fieldName string) (key cipher.SHA256, e error) {
	e = encoder.DeserializeField(r.data, r.GetSchema().Fields, fieldName, &key)
	return
}

func (r *ObjRef) GetFieldAsObj(fieldName string) (obj ObjRef, e error) {
	var fieldKey cipher.SHA256

	e = encoder.DeserializeField(r.data, r.GetSchema().Fields, fieldName, &fieldKey)
	if e != nil {
		return
	}

	obj = r.c.GetObjRef(fieldKey)
	return
}

func (r *ObjRef) GetValuesAsKeyArray() (keyArray []cipher.SHA256, e error) {
	e = r.Deserialize(&keyArray)
	return
}

func (r *ObjRef) GetValuesAsObjArray() (objArray []ObjRef, e error) {
	var keyArray []cipher.SHA256
	if e = r.Deserialize(&keyArray); e != nil {
		return
	}

	for _, key := range keyArray {
		childObj := r.c.GetObjRef(key)
		objArray = append(objArray, childObj)
	}
	return
}
