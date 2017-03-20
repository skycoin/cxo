package skyobject

import (
	"errors"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

var (
	// ErrInvalidSchema occurs any time using the schema that is invalid
	ErrInvalidSchema = errors.New("invalid schema")
	// ErrEmptySchemaKey occurs if you want to get schema by its key, but
	// the key is empty
	ErrEmptySchemaKey = errors.New("empty schema key")
	// ErrTypeNameNotFound oocurs if you want to get schema by type name
	// but the Container knows nothing about the name
	ErrTypeNameNotFound = errors.New("type name not found")
	// ErrInvalidReference occurs when some dynamic reference is invalid
	ErrInvalidReference = errors.New("invalid reference")
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
// to the Container. Seq of the root is 0, Timestamp of the root set to now.
func (c *Container) NewRoot(pk cipher.PubKey) (root *Root) {
	root = new(Root)
	root.reg = c.reg // shared registery
	root.cnt = c
	root.Pub = pk
	root.Time = time.Now().UnixNano()
	return
}

// Root returns root object by its public key
func (c *Container) Root(pk cipher.PubKey) (r *Root) {
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
