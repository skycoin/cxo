package skyobjects

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// RootReference struct.
type RootReference Reference

type rootObject struct {
	Keys        []cipher.SHA256
	DeletedKeys []cipher.SHA256
	Sequence    uint64
	Timestamp   int64
}

func newRootReference(keys, deletedKeys []cipher.SHA256, seq uint64) *RootReference {
	rootObj := rootObject{
		Keys:        keys,
		DeletedKeys: deletedKeys,
		Sequence:    seq,
		Timestamp:   time.Now().UnixNano(),
	}
	return &RootReference{value: rootObj}
}

func (r *RootReference) save(c *Container) {
	r.schemaKey = c.dbSaveSchema(c.ds, encoder.Serialize(ReadSchema(RootReference{})))
	r.data = encoder.Serialize(r.value.(rootObject))
	r.key = c.dbSave(c.ds, r.schemaKey, r.data)

	c.root = r.key
	c.rootSeq = r.value.(rootObject).Sequence
}

// GetKey returns the identifier.
func (r *RootReference) GetKey() cipher.SHA256 {
	return r.key
}

// GetType get's the Schema Key of the ArrayReference.
func (r *RootReference) GetType() cipher.SHA256 {
	return r.schemaKey
}

// GetSchema gets the Schema.
func (r *RootReference) GetSchema(c *Container) *Schema {
	return c.Schema.GetOfKey(r.schemaKey)
}

// GetField gets a field as key.
func (r *RootReference) GetField(c *Container, fieldName string) IReference {
	return nil
}

// GetChildren gets all the children of root object.
func (r *RootReference) GetChildren(c *Container) (children []IReference) {
	var root rootObject
	r.Deserialize(&root)
	for _, key := range root.Keys {
		children = append(children, c.Get(key))
	}
	return
}

// GetDescendants gets all the descendants.
func (r *RootReference) GetDescendants(c *Container, descendants map[cipher.SHA256]IReference) {
	for _, child := range r.GetChildren(c) {
		if _, has := descendants[child.GetKey()]; has {
			continue
		}
		descendants[child.GetKey()] = child
		child.GetDescendants(c, descendants)
	}
}

// Deserialize deserializes the array into 'dataOut'.
func (r *RootReference) Deserialize(dataOut interface{}) error {
	return encoder.DeserializeRaw(r.data, dataOut)
}
