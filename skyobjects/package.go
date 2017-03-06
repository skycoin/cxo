package skyobjects

import "github.com/skycoin/skycoin/src/cipher"

// PackagedKey represents an object's key, packaged with it's schema key.
type PackagedKey struct {
	Key       cipher.SHA256
	SchemaKey cipher.SHA256
}

// PackagedKeyArray represents an array of object keys, packaged with a schema key.
// All object keys included should be of type specified by the schema key.
type PackagedKeyArray struct {
	Keys      []cipher.SHA256
	SchemaKey cipher.SHA256
}

// PackagedRoot represents a packaged root object.
// All object keys needs to linked with a schema key.
type PackagedRoot struct {
	SchemaKeys []cipher.SHA256          // Array of schema keys (linked with id).
	Children   map[cipher.SHA256]uint64 // The value of map element is the schema's id.
	Sequence   uint64
	TimeStamp  int64
}
