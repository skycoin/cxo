package skyobjects

import (
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// Container contains skyobjects.
type Container struct {
	ds      *data.DB
	rootKey cipher.SHA256
	rootSeq uint64
}

// NewContainer creates a new skyobjects container.
func NewContainer(ds *data.DB) (c *Container) {
	c = &Container{ds: ds}
	// TODO: Register default schemas in container.
	return
}

// Save saves an object into container.
func (c *Container) Save(schemaKey cipher.SHA256, data []byte) (key cipher.SHA256) {
	h := href{SchemaKey: schemaKey, Data: data}
	key = c.ds.AddAutoKey(encoder.Serialize(h))
	return
}

// func (c *Container) SaveSchema()
