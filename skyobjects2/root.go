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
func (r *RootObject) GetDescendants(c *Container) (desMap map[cipher.SHA256]bool) {
	desMap = make(map[cipher.SHA256]bool)
	// TODO: Implement.
	return
}
