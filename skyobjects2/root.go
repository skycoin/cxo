package skyobjects

import "github.com/skycoin/skycoin/src/cipher"

// RootObject represents a root object in container.
type RootObject struct {
	Children  []cipher.SHA256
	Sequence  uint64
	TimeStamp int64
}
