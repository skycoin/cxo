package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
)

// Dynamic is reference to object with schema
type Dynamic struct {
	Schema cipher.SHA256 `skyobject:"dynamic_schema"`
	ObjKey cipher.SHA256 `skyobject:"dynamic_objkey"`
}

// NewDynamic creates Dynamic from given object saving serialized
// object in DB
func (c *Container) NewDynamic(i interface{}) (dh Dynamic) {
	dh.Schema = c.Save(getSchema(i))
	dh.ObjKey = c.Save(i)
	return
}
