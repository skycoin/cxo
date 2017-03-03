package skyobjects

import (
	"bytes"
	"fmt"
	"time"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// Container represents a SkyObjects container.
type Container struct {
	Schema  ISchemaManager
	ds      data.IDataSource
	root    cipher.SHA256
	rootSeq uint64
}

// NewContainer creates a new SkyObjects container.
func NewContainer(ds data.IDataSource) (c *Container) {
	c = &Container{
		Schema: &schemaManager{c: c, ds: ds},
		ds:     ds,
	}
	c.Schema.Register(RootReference{}, ObjectReference{}, ArrayReference{})
	newRootReference([]cipher.SHA256{}, []cipher.SHA256{}, 1).save(c)
	return
}

// Store stores an item.
func (c *Container) Store(ref IReference) IReference {
	ref.save(c)
	return ref
}

// StoreToRoot stores item to root.
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
	switch c.Schema.GetOfKey(ref.Type).Name {
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

// GetRootValues gets the root values.
func (c *Container) GetRootValues() (values []IReference) {
	var prevRoot rootObject
	c.Get(c.root).Deserialize(&prevRoot)
	for _, key := range prevRoot.Keys {
		values = append(values, c.Get(key))
	}
	return
}

// GetRootTimestamp returns the time the root object was updated.
func (c *Container) GetRootTimestamp() int64 {
	var root rootObject
	c.Get(c.root).Deserialize(&root)
	return root.Timestamp
}

// GetAllToRemove get's all root children to be removed by garbage collector.
func (c *Container) GetAllToRemove() (values map[cipher.SHA256]IReference) {
	values = make(map[cipher.SHA256]IReference)
	// Loop though keys of objects to be removed.
	for _, ref := range c.getRootRemoved() {
		c.getAllToRemove(ref, values)
	}
	return
}

// ClearRoot clears the root object.
func (c *Container) ClearRoot() {
	for key := range c.GetAllToRemove() {
		c.Delete(key)
	}
	var oldRoot rootObject
	c.Get(c.root).Deserialize(&oldRoot)

	schemaKey, _ := c.Schema.GetOfType("RootReference")
	for _, key := range c.GetAllOfSchema(schemaKey) {
		c.Delete(key)
	}

	newRoot := newRootReference(oldRoot.Keys, []cipher.SHA256{}, 1)
	newRoot.save(c)
}

// CollectGarbage collects garbage thats older than specified time.
func (c *Container) CollectGarbage(duration int64) {
	var timeFrom = time.Now().UnixNano() - duration
	var toDelete = make(map[cipher.SHA256]IReference)
	schemaKey, _ := c.Schema.GetOfType("RootReference")
	// Loop through all root objects and descendants. Add to toDelete map.
	for _, key := range c.GetAllOfSchema(schemaKey) {
		var root rootObject
		// Don't delete roots that haven't expired.
		if c.Get(key).Deserialize(&root); root.Timestamp > timeFrom {
			continue
		}
		// Special algorithm for current root.
		if key == c.root {
			c.ClearRoot()
			return
		}
		// Add all descendants to toDelete map.
		for _, dKey := range root.DeletedKeys {
			dRef := c.Get(dKey)
			if dRef == nil {
				continue
			}
			toDelete[dKey] = dRef
			dRef.GetDescendants(c, toDelete)
		}
	}
	// Delete objects.
	for dKey := range toDelete {
		c.Delete(dKey)
	}
}

// CollectGarbage3Days is a convenience function.
func (c *Container) CollectGarbage3Days() {
	duration, _ := time.ParseDuration("72h")
	c.CollectGarbage(int64(duration))
}

func (c *Container) getAllToRemove(ref IReference, values map[cipher.SHA256]IReference) {
	values[ref.GetKey()] = ref
	// Loop through children.
	for _, childRef := range ref.GetChildren(c) {
		c.getAllToRemove(childRef, values)
	}
}

func (c *Container) getRootRemoved() (values []IReference) {
	var root rootObject
	c.Get(c.root).Deserialize(&root)
	for _, key := range root.DeletedKeys {
		values = append(values, c.Get(key))
	}
	return
}

func (c *Container) dbSave(ds data.IDataSource, schemaKey cipher.SHA256, data []byte) (key cipher.SHA256) {
	h := href{Type: schemaKey, Data: data}
	key = ds.AddAutoKey(encoder.Serialize(h))
	return
}

func (c *Container) dbSaveSchema(ds data.IDataSource, data []byte) (schemaKey cipher.SHA256) {
	h := href{Type: _schemaType, Data: data}
	return ds.AddAutoKey(encoder.Serialize(h))
}

func (c *Container) dbGet(ds data.IDataSource, key cipher.SHA256) (ref *href) {
	ref = &href{}
	data, ok := ds.Get(key)
	if ok == false {
		return
	}
	encoder.DeserializeRaw(data, ref)
	return
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
