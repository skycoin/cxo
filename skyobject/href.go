package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// Href is an objects data wrapped with it's Schema.
type Href struct {
	SchemaKey cipher.SHA256
	Data      []byte
}

// Deserialize deserializes the href.
func (h Href) Deserialize(dataOut interface{}) {
	encoder.DeserializeRaw(h.Data, dataOut)
}
