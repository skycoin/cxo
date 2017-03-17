package skyobject

import (
	"errors"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

var (
	ErrMissingRoot        = errors.New("misisng root object")
	ErrShortBuffer        = errors.New("short buffer")
	ErrInvalidSchema      = errors.New("invalid schema")
	ErrMissingInDB        = errors.New("missing in db")
	ErrUnregisteredSchema = errors.New("unregistered schema")
	ErrInvalidTag         = errors.New("invalid tag")
	ErrMissingSchemaTag   = errors.New("missing schema tag")
)

type Container struct {
	root *Root
	db   *data.DB
}

//
// root object
//

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
		return
	}
	return // false
}

//
// database wrappers
//

func (c *Container) get(r Reference) (v []byte, ok bool) {
	v, ok = c.db.Get(cipher.SHA256(r))
	return
}
