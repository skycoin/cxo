package skyobjects

import "github.com/skycoin/skycoin/src/cipher"

type href struct {
	SchemaKey cipher.SHA256
	Data      []byte
}
