package skyobjects

import (
	"bytes"
	"fmt"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// Container represents a SkyObjects container.
type Container struct {
	ds      *data.DB
	schemas []schemaRef
	root    cipher.SHA256
	rootSeq uint64
}

// NewContainer creates a new SkyObjects container.
func NewContainer(ds *data.DB) (c *Container) {
	c = &Container{ds: ds}
	c.RegisterSchema(RootReference{}, ObjectReference{}, ArrayReference{})
	newRootReference([]cipher.SHA256{}, []cipher.SHA256{}, 1).save(c)
	return
}

// Store stores an item.
func (c *Container) Store(ref IReference) IReference {
	ref.save(c)
	return ref
}

// StoreToRoot stores item, and sets it as child of root.
func (c *Container) StoreToRoot(ref IReference) IReference {
	ref.save(c)
	keys, deletedKeys := []cipher.SHA256{}, []cipher.SHA256{}
	if c.rootSeq > 0 {
		var prevRoot rootObject
		c.Get(c.root).Deserialize(&prevRoot)
		keys = prevRoot.Keys
		deletedKeys = prevRoot.DeletedKeys
	}
	keys = append(keys, ref.GetKey())
	rootRef := newRootReference(keys, deletedKeys, c.rootSeq+1)
	rootRef.save(c)
	return ref
}

// RemoveFromRoot removes an item from root.
func (c *Container) RemoveFromRoot(key cipher.SHA256) error {
	if c.rootSeq == 0 {
		return fmt.Errorf("no root avaliable")
	}
	var root rootObject
	c.Get(c.root).Deserialize(&root)

	for i, prevKey := range root.Keys {
		if prevKey == key {
			// Delete algorithm.
			root.Keys[len(root.Keys)-1], root.Keys[i] = root.Keys[i], root.Keys[len(root.Keys)-1]
			root.Keys = root.Keys[:len(root.Keys)-1]
			root.DeletedKeys = append(root.DeletedKeys, key)

			rootRef := newRootReference(root.Keys, root.DeletedKeys, c.rootSeq+1)
			rootRef.save(c)
			return nil
		}
	}
	return fmt.Errorf("item of key '%s' is not a child of root, or is already deleted", key.Hex())
}

// Delete deletes an item from container.
func (c *Container) Delete(key cipher.SHA256) {
	c.ds.Remove(key)
}

// Get gets an item using a key.
func (c *Container) Get(key cipher.SHA256) IReference {
	ref := c.dbGet(c.ds, key)
	if len(ref.Data) == 0 {
		return nil
	}
	switch c.GetSchemaOfKey(ref.Type).Name {
	case "ArrayReference":
		return &ArrayReference{key, ref.Type, ref.Data, nil}
	case "ArrayRoot":
	}
	return &ObjectReference{key, ref.Type, ref.Data, nil}
}

// GetAllOfSchema gets all keys of objects of specified schemaKey.
func (c *Container) GetAllOfSchema(schemaKey cipher.SHA256) []cipher.SHA256 {
	query := func(key cipher.SHA256, data []byte) bool {
		return bytes.Compare(schemaKey[:32], data[:32]) == 0
	}
	return c.ds.Where(query)
}

// GetRootChildren gets all the keys of the current root's children.
func (c *Container) GetRootChildren() (values []IReference) {
	var prevRoot rootObject
	c.Get(c.root).Deserialize(&prevRoot)
	for _, key := range prevRoot.Keys {
		values = append(values, c.Get(key))
	}
	return
}

// GetRootDescendants gets all the root descendants into a map.
// The map value is a boolean of whether we have the referenced object or not.
func (c *Container) GetRootDescendants() (descendants map[cipher.SHA256]IReference) {
	descendants = make(map[cipher.SHA256]IReference)
	for _, ref := range c.GetRootChildren() {
		ref.GetDescendants(c, descendants)
	}
	return
}

// GetRootTimestamp returns the time the root object was updated.
func (c *Container) GetRootTimestamp() int64 {
	var root rootObject
	c.Get(c.root).Deserialize(&root)
	return root.Timestamp
}

// Inspect prints information about the container.
func (c *Container) Inspect() {
	query := func(key cipher.SHA256, data []byte) bool {
		hr := href{}
		encoder.DeserializeRaw(data, &hr)
		smKey := hr.Type
		smKey.Set(data[:32])

		var sm = Schema{}
		if smKey == _schemaType {
			encoder.DeserializeRaw(hr.Data, &sm)
			fmt.Println("[Schema] ", sm)
		} else {
			schemaData, _ := c.ds.Get(smKey)
			shr := href{}
			encoder.DeserializeRaw(schemaData, &shr)
			if shr.Type != _schemaType {
				panic("Reference mast be an schema type")
			}
			encoder.DeserializeRaw(shr.Data, &sm)
			fmt.Println("\t\t\t\t\t\t\t\tObject: ", sm)
		}
		return false
	}
	c.ds.Where(query)
}
