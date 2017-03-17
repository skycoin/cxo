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

// A Container represents type helper to manage root objects
type Container struct {
	root *Root
	db   *data.DB
}

// NewContainer creates container using given db. If the db is nil
// then the function panics
func NewContainer(db *data.DB) (c *Container) {
	if db == nil {
		panic("missisng db")
	}
	c = new(Container)
	c.db = db
	return
}

//
// root object
//

// NewRoot creates new empty root object. The method doesn't put the root
// to the Container
func (c *Container) NewRoot() (root *Root) {
	root = new(Root)
	root.reg = newSchemaReg(c.db)
	root.cnt = c
	return
}

// Root returns root object of the Container
func (c *Container) Root() *Root {
	return c.root
}

// SetRoot set given root object to the Container if timestamp of
// given root is greater than timestamp of existsing root object
func (c *Container) SetRoot(root *Root) (ok bool) {
	if c.root == nil {
		c.root, ok = root, true
		root.cnt = c // be sure that the root referes to the container
		return
	}
	if c.root.Time < root.Time {
		c.root, ok = root, true
		root.cnt = c // be sure that the root referes to the container
		return
	}
	return // false
}

//
// database wrappers (Reference <-> cipher.SHA256)
//

func (c *Container) get(r Reference) (v []byte, ok bool) {
	v, ok = c.db.Get(cipher.SHA256(r))
	return
}
