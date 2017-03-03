package skyobjects

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// ObjectReference struct.
type ObjectReference Reference

// NewObjectReference creates a new object reference.
func NewObjectReference(value interface{}) *ObjectReference {
	objectRef := &ObjectReference{value: value}
	return objectRef
}

func (r *ObjectReference) save(c *Container) {
	r.data = encoder.Serialize(r.value)
	r.schemaKey = c.dbSaveSchema(c.ds, encoder.Serialize(ReadSchema(r.value)))
	r.key = c.dbSave(c.ds, r.schemaKey, r.data)
}

// GetKey returns the identifier.
func (r *ObjectReference) GetKey() cipher.SHA256 {
	return r.key
}

// GetType get's the Schema Key of the ObjectReference.
func (r *ObjectReference) GetType() cipher.SHA256 {
	return r.schemaKey
}

// GetSchema gets the Schema.
func (r *ObjectReference) GetSchema(c *Container) *Schema {
	return c.GetSchemaOfKey(r.schemaKey)
}

// GetField gets a field as key.
func (r *ObjectReference) GetField(c *Container, fieldName string) IReference {
	var key cipher.SHA256
	encoder.DeserializeField(r.data, r.GetSchema(c).Fields, fieldName, &key)
	return c.Get(key)
}

// GetChildren gets all the children of the object.
func (r *ObjectReference) GetChildren(c *Container) (children []IReference) {
	for _, field := range r.GetSchema(c).Fields {
		switch field.Type {
		case "arrayreference", "objectreference":
			fieldRef := r.GetField(c, field.Name)
			if fieldRef == nil {
				continue
			}
			children = append(children, fieldRef)
		}
	}
	return
}

// GetDescendants gets all the descendants.
func (r *ObjectReference) GetDescendants(c *Container, descendants map[cipher.SHA256]IReference) {
	for _, child := range r.GetChildren(c) {
		if _, has := descendants[child.GetKey()]; has {
			continue
		}
		descendants[child.GetKey()] = child
		child.GetDescendants(c, descendants)
	}
}

// Deserialize deserializes the object into 'dataOut'.
func (r *ObjectReference) Deserialize(dataOut interface{}) error {
	return encoder.DeserializeRaw(r.data, dataOut)
}
