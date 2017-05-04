// Package skyobject represents skyobject
package skyobject

import (
	"errors"
	"fmt"
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
	c.reg = newRegistery(db)
	return
}

//
// root object
//

// NewRoot creates new empty root object. The method doesn't put the root
// to the Container. Seq of the root is 0, Timestamp of the root set to now.
func (c *Container) NewRoot(pk cipher.PubKey, sk cipher.SecKey) (root *Root) {
	root = new(Root)
	root.reg = c.reg // shared registery
	root.cnt = c
	root.Pub = pk
	root.Time = time.Now().UnixNano()
	root.Sign(sk)
	c.roots[pk] = root
	return
}

// Roots retusn list of all public keys of the Container
func (c *Container) Roots() (list []cipher.PubKey) {
	if len(c.roots) == 0 {
		return
	}
	list = make([]cipher.PubKey, 0, len(c.roots))
	for pub := range c.roots {
		list = append(list, pub)
	}
	return
}

// Root returns root object by its public key
func (c *Container) Root(pk cipher.PubKey) (r *Root) {
	return c.roots[pk]
}

func (c *Container) addRoot(root *Root) (set bool) {
	if rt, ex := c.roots[root.Pub]; !ex {
		c.roots[root.Pub], set = root, true
	} else if rt.Time < root.Time {
		c.roots[root.Pub], set = root, true
	}
	return
}

func decodeRoot(p []byte) (re rootEncoding, err error) {
	err = encoder.DeserializeRaw(p, &re)
	return
}

// AddEncodedRoot set given data as root object of the container.
// It returns an error if the data can't be encoded. It returns
// true if the root is set
func (c *Container) AddEncodedRoot(p []byte, // root.Encode()
	pub cipher.PubKey, sig cipher.Sig) (ok bool, err error) {

	err = cipher.VerifySignature(pub, sig, cipher.SumSHA256(p))
	if err != nil {
		return
	}

	var re rootEncoding
	if re, err = decodeRoot(p); err != nil {
		return
	}
	var root *Root = &Root{
		Time: re.Time,
		Seq:  re.Seq,
		Refs: re.Refs,
	}
	root.Pub = pub
	root.Sig = sig
	root.cnt = c
	root.reg = c.reg
	for _, v := range re.Reg {
		if sck, ae := c.reg.reg[v.K]; ae {
			if sck != v.V {
				err = fmt.Errorf("conflict between registered types %q", v.K)
				return
			}
		} else {
			c.reg.reg[v.K] = v.V
		}
	}
	ok = c.addRoot(root)
	return
}

//
// database wrappers (Reference <-> cipher.SHA256)
//

func (c *Container) get(r Reference) (v []byte, ok bool) {
	v, ok = c.db.Get(cipher.SHA256(r))
	return
}

//
// schemas and objects
//

// SchemaByReference returns *Schema by reference if the Container know
// about the schema
func (c *Container) SchemaByReference(sr Reference) (s *Schema, err error) {
	if sr.IsBlank() {
		err = ErrEmptySchemaKey
		return
	}
	s, err = c.reg.SchemaByReference(sr)
	return
}

// Save an object to db and get reference-key to it
func (c *Container) Save(i interface{}) Reference {
	return Reference(c.db.AddAutoKey(encoder.Serialize(i)))
}

// SaveArray of objects and get array of references-keys to them
func (c *Container) SaveArray(ary ...interface{}) (rs References) {
	if len(ary) == 0 {
		return
	}
	rs = make(References, 0, len(ary))
	for _, a := range ary {
		rs = append(rs, c.Save(a))
	}
	return
}

// SchemaReference returns reference-key to schema of given vlaue. It panics
// if the schema is not registered
func (c *Container) SchemaReference(i interface{}) (ref Reference) {
	return c.reg.SchemaReference(i)
}

// Dynamic saves object and its schema in db and returns dynamic reference,
// that points to the object and the schema
func (c *Container) Dynamic(i interface{}) (dn Dynamic) {
	dn.Object = c.Save(i)
	dn.Schema = c.SchemaReference(i)
	return
}

// Register schema of given object with given name
func (c *Container) Register(ni ...interface{}) {
	c.reg.Register(ni...)
}
