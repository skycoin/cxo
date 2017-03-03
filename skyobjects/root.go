package skyobjects

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

// RootObject represents a root object in container.
type RootObject struct {
	Children  []cipher.SHA256
	Sequence  uint64
	TimeStamp int64
}

// NewRoot creates a new root object of the current timestamp.
func NewRoot(seq uint64) RootObject {
	return RootObject{
		Sequence:  seq,
		TimeStamp: time.Now().UnixNano(),
	}
}

// AddChildren adds a child to root without duplication.
func (r *RootObject) AddChildren(keys ...cipher.SHA256) {
	for _, key := range keys {
		uniq := true
		for _, k := range r.Children {
			if key == k {
				uniq = false
				break
			}
		}
		if uniq {
			r.Children = append(r.Children, key)
		}
	}
}

// UpdateTimeStamp updates the timestamp to now.
func (r *RootObject) UpdateTimeStamp() {
	r.TimeStamp = time.Now().UnixNano()
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