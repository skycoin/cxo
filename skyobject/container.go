package skyobject

import (
	"errors"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

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
	reg   *Registry // shared registery
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
	c.reg = NewRegistery(db)
	return
}

//
// root object
//

// NewRoot creates new empty root object. The method doesn't put the root
// to the Container
func (c *Container) NewRoot(pk cipher.PubKey) (root *Root) {
	root = new(Root)
	root.reg = c.reg // shared registery
	root.cnt = c
	root.Pub = pk
	root.Time = time.Now().UnixNano()
	return
}

// Root returns root object by its public key
func (c *Container) Root(pk cipher.PubKey) *Root {
	return c.roots[pk]
}

// AddRoot add/replace given root object to the Container if timestamp of
// given root is greater than timestamp of existsing root object. It's
// possible to add a root object only if the root created by this container
func (c *Container) AddRoot(root *Root) (set bool) {
	if root.cnt != c {
		panic("trying to add root object from a side")
	}
	if rt, ex := c.roots[root.Pub]; !ex {
		c.roots[root.Pub], set = root, true
	} else if rt.Time < root.Time {
		c.roots[root.Pub], set = root, true
	}
	return
}

// SetEncodedRoot set given data as root object of the container.
// It returns an error if the data can't be encoded. It returns
// true if the root is set
func (c *Container) SetEncodedRoot(p []byte) (ok bool, err error) {
	var x struct {
		Root Root
		Nmr  []struct{ K, V string } // map[string]string
		Reg  []struct {              // map[string]cipher.SHA256
			K string
			V cipher.SHA256
		}
	}
	if err = encoder.DeserializeRaw(p, &x); err != nil {
		return
	}
	var root *Root = &x.Root
	root.cnt = c
	root.reg = c.reg
	for _, v := range x.Nmr {
		root.reg.nmr[v.K] = v.V
	}
	for _, v := range x.Reg {
		root.reg.reg[v.K] = v.V
	}
	ok = c.AddRoot(root)
	return
}

//
// database wrappers (Reference <-> cipher.SHA256)
//

func (c *Container) get(r Reference) (v []byte, ok bool) {
	v, ok = c.db.Get(cipher.SHA256(r))
	return
}
