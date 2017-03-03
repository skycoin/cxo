package skyobjects

import "github.com/skycoin/skycoin/src/cipher"

// RootObject represents a root object in container.
type RootObject struct {
	Children  []cipher.SHA256
	Sequence  uint64
	TimeStamp int64
}

// GetDescendants gets all keys of descendants of a root object.
// Boolean value:
// * TRUE: We have a copy of this object in container.
// * FALSE: We don't have a copy of this object in container.
func (r *RootObject) GetDescendants(c *Container) (dMap map[cipher.SHA256]bool) {
	dMap = make(map[cipher.SHA256]bool)
	// TODO: Implement.
	for _, key := range r.Children {
		c.getDescendants(key, dMap)
	}
	return
}
