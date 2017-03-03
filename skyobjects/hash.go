package skyobjects

import "github.com/skycoin/skycoin/src/cipher"

// HashArray represents an array of object keys.
type HashArray []cipher.SHA256

// HashObject represents a single object key.
// type HashObject cipher.SHA256

type href struct {
	SchemaKey cipher.SHA256
	Data      []byte
}
