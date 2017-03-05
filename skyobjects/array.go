package skyobjects

import "github.com/skycoin/skycoin/src/cipher"

// HashArray represents an array of object keys.
type HashArray []cipher.SHA256

// NewArray creates a new array given a set of keys.
func NewArray(keys ...cipher.SHA256) (array HashArray) {
	for _, k := range keys {
		array = append(array, k)
	}
	return
}
