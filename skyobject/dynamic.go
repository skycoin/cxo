package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

var (
	dynamicHrefSchema    = getSchema(DynamicHref{})
	dynamicHrefSchemaKey = cipher.SumSHA256(
		encoder.Serialize(dynamicHrefSchema))
)

// DynamicHref is reference to object with schema
type DynamicHref struct {
	Schema cipher.SHA256
	ObjKey cipher.SHA256
}

// NewDynamicHref creates DynamicHref from given object saving serialized
// object in DB
func (c *Container) NewDynamicHref(i interface{}) DynamicHref {
	return DynamicHref{
		Schema: c.Save(getSchema(i)),
		ObjKey: c.Save(i),
	}
}
