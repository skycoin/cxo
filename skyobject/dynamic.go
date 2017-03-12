package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
)

// DynamicHref is reference to object with schema
type DynamicHref struct {
	Schema cipher.SHA256
	ObjKey cipher.SHA256
}

// NewDynamicHref creates DynamicHref from given object saving serialized
// object in DB
func (c *Container) SaveDynamicHref(i interface{}) cipher.SHA256 {
	return c.Save(DynamicHref{
		Schema: c.Save(getSchema(i)),
		ObjKey: c.Save(i),
	})
}
