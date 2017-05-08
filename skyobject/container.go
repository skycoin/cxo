package skyobject

import (
	"fmt"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

// A Container represents ...
type Container struct {
	sync.RWMutex

	db *data.DB // databse

	coreRegistry *Registry // registry witch which the container was created

	registries map[RegistryReference]*Registry
	roots      map[cipher.PubKey][]*Root // root objects
}

// NewContainer is like NewContainerDB but database created
// implicitly. See documentation of NewContainerDB for details
func NewContainer(reg *Registry) *Container {
	return NewContainerDB(data.NewDB(), reg)
}

// NewContainerDB creates new Container using given databse and
// optional Registry. If Registry is no nil, then the registry
// will be used to create Dynamic objects. The Registry will be
// used as registry of all Root objects created by the Container.
// If Regsitry is nil then the Container can be used server-side.
// Creating Dynamic and Root objects without Registry causes panic
func NewContainerDB(db *data.DB, reg *Registry) (c *Container) {
	if db == nil {
		panic("nil db")
	}
	c = new(Container)
	c.db = db
	c.registries = make(map[RegistryReference]*Registry)
	if reg != nil {
		reg.Done()
		c.coreRegistry = reg
		c.registries[reg.Reference()] = reg
	}
	c.roots = make(map[cipher.PubKey][]*Root)
	return
}

// CoreRegistry returns registry witch wich the Container
// was created. It can returns nil
func (c *Container) CoreRegistry() *Registry {
	return c.coreRegistry
}

// Registry by reference
func (c *Container) Registry(rr RegistryReference) (reg *Registry, err error) {
	c.RLock()
	defer c.RUnlock()

	var ok bool
	if reg, ok = c.registries[rr]; !ok {
		err = fmt.Errorf("missing registry %q", rr.String())
	}
	return
}

// DB of the Container
func (c *Container) DB() *data.DB {
	return c.db
}

// Get object by Reference
func (c *Container) Get(ref Reference) (data []byte, ok bool) {
	data, ok = c.db.Get(cipher.SHA256(ref))
	return
}

func (c *Container) save(i interface{}) Reference {
	return Reference(c.db.AddAutoKey(encoder.Serialize(i)))
}

func (c *Container) Save(i interface{}) Reference {
	c.RLock()
	defer c.RUnlock() // locks required for GC
	return c.save(i)
}

func (c *Container) SaveArray(i ...interface{}) (refs References) {
	c.RLock()
	defer c.RUnlock() // locks required for GC
	refs = make(References, 0, len(i))
	for _, e := range i {
		refs = append(refs, c.save(e))
	}
	return
}

func (c *Container) Dynamic(i interface{}) (dr Dynamic) {
	if c.coreRegistry == nil {
		panic("unable to create Dynamic, Container created without registry")
	}
	c.RLock()
	defer c.RUnlock() // locks required for GC
	s, err := c.coreRegistry.SchemaByInterface(i)
	if err != nil {
		panic(err)
	}
	dr.Schema = s.Reference()
	dr.Object = c.save(i)
	return
}

// GC removes all unused objects
func (c *Container) GC() {
	//
}
