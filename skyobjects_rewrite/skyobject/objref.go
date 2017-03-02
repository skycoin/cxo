package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// ObjRef is an convenience struct for extracting objects from SkyObjects.
type ObjRef struct {
	key       cipher.SHA256
	schemaKey cipher.SHA256
	data      []byte
	c         ISkyObjects
}

// GetKey returns the identifier of the skyObject.
func (r *ObjRef) GetKey() cipher.SHA256 {
	return r.key
}

// GetSchemaKey returns the skyObject's schema identifier.
func (r *ObjRef) GetSchemaKey() cipher.SHA256 {
	return r.schemaKey
}

// GetSchema returns the skyObject's schema.
func (r *ObjRef) GetSchema() *Schema {
	return r.c.GetSchemaFromKey(r.schemaKey)
}

// Deserialize deserializes the skyObject into 'dataOut'.
// 'dataOut' should be a pointer interface.
func (r *ObjRef) Deserialize(dataOut interface{}) error {
	return encoder.DeserializeRaw(r.data, dataOut)
}

// GetFieldAsKey returns the given field of the skyObject as an identifier.
// Returns an empty key if skyObject is not a hash object.
func (r *ObjRef) GetFieldAsKey(fieldName string) (key cipher.SHA256, e error) {
	e = encoder.DeserializeField(r.data, r.GetSchema().Fields, fieldName, &key)
	return
}

// GetFieldAsObj returns the given field of the skyObject as a skyObject.
// Returns an empty skyObject if skyObject is not a hash object.
func (r *ObjRef) GetFieldAsObj(fieldName string) (obj *ObjRef, e error) {
	var fieldKey cipher.SHA256
	e = encoder.DeserializeField(r.data, r.GetSchema().Fields, fieldName, &fieldKey)
	if e != nil {
		return
	}
	obj = r.c.GetObjRef(fieldKey)
	return
}

// GetValuesAsKeyArray assumes that skyObject is a hash array, and returns the
// array values as an identifier array.
// Returns an empty array if skyObject is not a hash array.
func (r *ObjRef) GetValuesAsKeyArray() (keyArray []cipher.SHA256, e error) {
	e = r.Deserialize(&keyArray)
	return
}

// GetValuesAsObjArray assumes that skyObject is a hash array, and returns the
// array values as a skyObject array.
// Returns an empty array if skyObject is not a hash array.
func (r *ObjRef) GetValuesAsObjArray() (objArray []*ObjRef, e error) {
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
