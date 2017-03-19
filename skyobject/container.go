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
	ErrUnregisteredSchema = errors.New("unregistered schema")
	ErrInvalidTag         = errors.New("invalid tag")
	ErrMissingSchemaTag   = errors.New("missing schema tag")
	ErrEmptySchemaKey     = errors.New("empty schema key")
	ErrNotFound           = errors.New("not found")
)

// A Container represents type helper to manage root objects
type Container struct {
	roots map[cipher.PubKey]*Root // feed -> root of the feed
	db    *data.DB
}

// NewContainer creates container using given db. If the db is nil
// then the function panics
func NewContainer(db *data.DB) (c *Container) {
	if db == nil {
		panic("missisng db")
	}
	c = new(Container)
	c.db = db
	c.roots = make(map[cipher.PubKey]*Root)
	return
}

//
// root object
//

// NewRoot creates new empty root object. The method doesn't put the root
// to the Container
func (c *Container) NewRoot(pk cipher.PubKey) (root *Root) {
	root = new(Root)
	root.reg = NewRegistery(c.db)
	root.cnt = c
	root.Pub = pk
	return
}

// Root returns root object of the Container
func (c *Container) Root(pk cipher.PubKey) *Root {
	return c.roots[pk]
}

// AddRoot add/replace given root object to the Container if timestamp of
// given root is greater than timestamp of existsing root object
func (c *Container) AddRoot(root *Root) (set bool) {
	if rt, ex := c.roots[root.Pub]; !ex {
		root.cnt = c
		c.roots[root.Pub], set = root, true
	} else if rt.Time < root.Time {
		root.cnt = c
		c.roots[root.Pub], set = root, true
	}
	return
}

//
// database wrappers (Reference <-> cipher.SHA256)
//

func (c *Container) get(r Reference) (v []byte, ok bool) {
	v, ok = c.db.Get(cipher.SHA256(r))
	return
}
