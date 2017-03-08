package skyobject

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

// RootObject represents a root object in container.
type RootObject struct {
	Children  []cipher.SHA256 `skyobject:"href"`
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
