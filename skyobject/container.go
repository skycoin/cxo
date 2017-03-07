package skyobject

import (
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

type Container struct {
	root     *Root
	registry map[string]reflect.Type  // type name to reflect type registry
	schemas  map[string]cipher.SHA256 // type name to schema

	db *data.DB
}

func (c *Container) Root() *Root {
	return c.root
}

func (c *Container) SetRoot(root *Root) bool {
	if c.root == nil {
		c.root = root
		return true
	}
	if c.root.Time < root.Time && c.root.Seq < root.Seq {
		c.root = root
		return true
	}
	return false
}

func (c *Container) RegisterType(i interface{}) {
	typ := reflect.TypeOf(i)
	c.registry[typ.Name()] = typ
}

func (c *Container) RegisterSchema(i interface{}) {
	//
}
