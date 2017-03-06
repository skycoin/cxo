package skyobjects

/* SHOULD WE DEPRECATE THIS? */

import "github.com/skycoin/skycoin/src/cipher"

// HashObject represents an single object key.
type HashObject cipher.SHA256

// NewObject creates a new object reference.
func NewObject(key cipher.SHA256) HashObject {
	return HashObject(key)
}

// HashArray represents an array of object keys.
type HashArray []cipher.SHA256

// NewArray creates a new array given a set of keys.
func NewArray(keys ...cipher.SHA256) (array HashArray) {
	for _, k := range keys {
		array = append(array, k)
	}
	return
}
