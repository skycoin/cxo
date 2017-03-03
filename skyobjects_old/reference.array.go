package skyobjects

import (
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// ArrayReference struct.
type ArrayReference Reference

// NewArrayReference creates a new array reference.
func NewArrayReference(items ...interface{}) *ArrayReference {
	arrayRef := &ArrayReference{value: items}
	return arrayRef
}

func (r *ArrayReference) save(c *Container) {
	r.schemaKey = c.dbSaveSchema(c.ds, encoder.Serialize(ReadSchema(ArrayReference{})))
	var objKeys []cipher.SHA256
	// Some crazy hack. -------------------- ^(*0*)^
	for _, obj := range InterfaceSlice((r.value.([]interface{}))[0]) {
		objRef := NewObjectReference(obj)
		objRef.save(c)
		objKeys = append(objKeys, objRef.key)
	}
	r.data = encoder.Serialize(objKeys)
	r.key = c.dbSave(c.ds, r.schemaKey, r.data)
}

// GetKey returns the identifier.
func (r *ArrayReference) GetKey() cipher.SHA256 {
	return r.key
}

// GetType get's the Schema Key of the ArrayReference.
func (r *ArrayReference) GetType() cipher.SHA256 {
	return r.schemaKey
}

// GetSchema gets the Schema.
func (r *ArrayReference) GetSchema(c *Container) *Schema {
	return c.GetSchemaOfKey(r.schemaKey)
}

// GetField gets a field as key.
func (r *ArrayReference) GetField(c *Container, fieldName string) IReference {
	var key cipher.SHA256
	encoder.DeserializeField(r.data, r.GetSchema(c).Fields, fieldName, &key)
	return c.Get(key)
}

// GetChildren gets all the children of the array.
func (r *ArrayReference) GetChildren(c *Container) (children []IReference) {
	var keys []cipher.SHA256
	encoder.DeserializeRaw(r.data, &keys)
	for _, key := range keys {
		ref := c.Get(key)
		if ref == nil {
			continue
		}
		children = append(children, ref)
	}
	return
}

// GetDescendants gets all the descendants.
func (r *ArrayReference) GetDescendants(c *Container, descendants map[cipher.SHA256]IReference) {
	for _, child := range r.GetChildren(c) {
		if _, has := descendants[child.GetKey()]; has {
			continue
		}
		descendants[child.GetKey()] = child
		child.GetDescendants(c, descendants)
	}
}

// Deserialize deserializes the array into 'dataOut'.
func (r *ArrayReference) Deserialize(dataOut interface{}) error {
	return encoder.DeserializeRaw(r.data, dataOut)
}

// InterfaceSlice returns an array of interfaces from an interface.
func InterfaceSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return []interface{}{}
	}
	ret := make([]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}
	return ret
}
