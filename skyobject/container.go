package skyobject

import (
	"errors"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

var (
	ErrStopRange = errors.New("stop range")
)

type Container struct {
	db data.DB

	coreRegistry *Registry

	rmx  sync.RWMutex
	regs map[RegistryReference]*Registry
}

// NewContainer by given database (required) and Registry
// (optional). Given Registry will be CoreRegsitry of the
// Container
func NewContainer(db data.DB, reg *Registry) (c *Container) {
	if db == nil {
		panic("missing data.DB")
	}
	c = new(Container)
	c.regs = make(map[RegistryReference]*Registry)

	if reg != nil {
		c.coreRegistry = reg
		if err = c.AddRegistry(reg); err != nil {
			c.db.Close() // to be safe
			panic(err)   // fatality
		}
	}

	return
}

// saveRegistry in database
func (c *Container) saveRegistry(reg *Registry) error {
	return db.Update(func(tx data.Tu) error {
		objs := tx.Objects()
		return objs.Set(cipher.SHA256(reg.Reference()), reg.Encode())
	})
}

// AddRegistry to the Container and save it database until
// it removed by CelanUp
func (c *Container) AddRegistry(reg *Registry) (err error) {
	c.rmx.Lock()
	defer c.rmx.Unlock()

	if _, ok := c.regs[reg.Reference()]; !ok {
		if err = c.saveRegistry(reg); err == nil {
			c.regs[reg.Reference()] = reg
		}
	}
	return
}

// DB returns underlying data.DB
func (c *Container) DB() data.DB {
	return c.db
}

// CoreRegisty of the Container or nil if
// the Container created without a Regsitry
func (c *Container) CoreRegistry() *Registry {
	return c.coreRegistry
}

// Registry by RegistryReference. It returns nil if
// the Container doesn't contain required Registry
func (c *Container) Registry(rr RegistryReference) *Registry {
	c.rmx.RLock()
	defer c.rmx.RUnlock()

	return c.regs[rr]
}

// CelanUp removes unused objects from database
func CleanUp() (err error) {
	// TODO
	return
}
