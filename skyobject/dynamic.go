package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// DynamicHref is reference to object with schema
type DynamicHref struct {
	Schema Schema
	ObjKey cipher.SHA256
}

// NewDynamicHref creates DynamicHref from given object saving serialized
// object in DB
func (c *Container) NewDynamicHref(i interface{}) *DynamicHref {
	return &DynamicHref{
		Schema: *getSchema(i),
		ObjKey: c.Save(i),
	}
}
