package skyobject

import (
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

var (
	hrefTypeName      = typeName(reflect.TypeOf(cipher.SHA256{}))
	hrefArrayTypeName = typeName(reflect.TypeOf([]cipher.SHA256{}))
)

type Container struct {
	db   *data.DB
	root *Root

	registry map[string]Schema
}

func NewContainer(db *data.DB) *Container {
	return &Container{
		db:       db,
		registry: make(map[string]Schema),
	}
}

func (c *Container) Root() *Root {
	return c.root
}

func (c *Container) SetRoot(root *Root) (ok bool) {
	if c.root == nil {
		c.root, ok = root, true
		return
	}
	if c.root.Time < root.Time {
		c.root, ok = root, true
	}
	return
}

func (c *Container) Register(name string, i interface{}) {
	c.registry[name] = getSchema(i)
}

func (c *Container) Childs(schema Schema, data []byte) (ch []cipher.SHA256) {
	for _, sf := range schema.Fields {
		switch sf.Type {
		case hrefTypeName:
			//
		case hrefArrayTypeName:
			//
		}
	}
}

func hrefChilds(sf encoder.StructField, data []byte) (cipher.SHA256, ok bool) {
	
}
