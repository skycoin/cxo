package skyobjects

import "github.com/skycoin/skycoin/src/cipher"

// HashObject represents an single object key.
type HashObject cipher.SHA256

// NewObject creates a new object reference.
func NewObject(key cipher.SHA256) HashObject {
	return HashObject(key)
}
