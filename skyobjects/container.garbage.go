package skyobjects

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

// ClearRoot clears the root object.
func (c *Container) ClearRoot() {
	for key := range c.GetAllToRemove() {
		c.Delete(key)
	}
	var oldRoot rootObject
	c.Get(c.root).Deserialize(&oldRoot)

	schemaKey, _ := c.GetSchemaOfTypeName("RootReference")
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
	schemaKey, _ := c.GetSchemaOfTypeName("RootReference")
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

// GetAllToRemove get's all root children to be removed by garbage collector.
func (c *Container) GetAllToRemove() (values map[cipher.SHA256]IReference) {
	values = make(map[cipher.SHA256]IReference)
	// Loop though keys of objects to be removed.
	for _, ref := range c.getRootRemoved() {
		c.getAllToRemove(ref, values)
	}
	return
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
