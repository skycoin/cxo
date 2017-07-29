package skyobject

import (
	"errors"
	"fmt"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node/log"
)

// common errors
var (
	ErrStopRange                          = errors.New("stop range")
	ErrInvalidArgument                    = errors.New("invalid argument")
	ErrMissingTypesButGoTypesFlagProvided = errors.New(
		"missing Types maps, but GoTypes flag provided")
	ErrMissingDirectMapInTypes  = errors.New("missing Direct map in Types")
	ErrMissingInverseMapInTypes = errors.New("missing Inverse map in Types")
)

// A Container represents container of Root
// objects
type Container struct {
	log.Logger

	db data.DB

	coreRegistry *Registry

	rmx  sync.RWMutex
	regs map[RegistryReference]*Registry

	filler *Filler // related filler
}

// NewContainer by given database (required) and Registry
// (optional). Given Registry will be CoreRegsitry of the
// Container
func NewContainer(db data.DB, conf *Config) (c *Container) {

	if db == nil {
		panic("missing data.DB")
	}

	if conf == nil {
		conf = NewConfig()
	}

	c = new(Container)
	c.Logger = log.NewLogger(conf.Log)
	c.regs = make(map[RegistryReference]*Registry)

	if conf.Registry != nil {
		c.coreRegistry = conf.Registry
		if err = c.AddRegistry(conf.Registry); err != nil {
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

// add registry that already saved in database
func (c *Container) addRegistry(reg *Registry) {
	c.rmx.Lock()
	defer c.rmx.Unlock()

	if _, ok := c.regs[reg.Reference()]; !ok {
		c.regs[reg.Reference()] = reg
	}
	return
}

// AddRegistry to the Container and save it into database until
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

// Set saves single object into database
func (c *Container) Set(hash cipher.SHA256, data []byte) (err error) {
	return c.db.Update(func(tx data.Tu) error {
		return tx.Objects().Set(hash, data)
	})
}

// Get returns data by hash. Result is nil if data not found
func (c *Container) Get(hash cipher.SHA256) (value []byte) {
	err := c.db.View(func(tx data.Tv) (_ error) {
		value = tx.Objects().Get(hash)
		return
	})
	if err != nil {
		c.db.Close() // to be safe (don't corrupt database-file)
		c.Fatalf("[ALERT] database error: %v", err)
	}
	return
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

func (c *Container) Root(pk cipher.PubKey, seq uint64) (r *Root, err error) {
	var rp *data.RootPack
	err = c.db.View(func(tx data.Tv) (_ error) {
		roots := tx.Feeds().Roots(pk)
		if roots == nil {
			return data.ErrNotFound
		}
		rp := roots.Get(seq)
		return
	})
	if err != nil {
		return
	}
	//
	return
}

// Unpack given Root obejct. Use flags by your needs. To use GoTypes
// flag, provide Types instance, for example:
//
//     r, err := c.Root(pk, 500)
//     if err != nil {
//         // handle error
//         return
//     }
//
//     theFlagsIUsuallyUse := EntireMerkleTrees |
//         EntireTree |
//         HashTableIndex |
//         GoTypes
//
//     pack, err := c.Unpack(r, theFlagsIUsuallyUse, &c.CoreRegistry().Types())
//     if err != nil {
//         // handle error
//         return
//     }
//
//     // use the pack
//
// If the EntireTree flag provided then given Root (entire tree) will be
// unpacked inside the Unpack method call. Unpack wihtout GoTypes
// flag will not wrok, because the feature is not implemented yet
func (c *Container) Unpack(r *Root, flags Flag, types *Types) (pack *Pack,
	err error) {

	// check arguments

	if r == nil {
		err = ErrInvalidArgument
		return
	}

	if flags & GoTypes {
		if types == nil {
			err = ErrMissingTypesButGoTypesFlagProvided
			return
		}
	} else {
		// TODO (kostyarin): provide a way to unpack to Value
		//                   if there are not Types
		err = errors.New("not possible to unpack wihtout GoTypes yet, cheese")
		return
	}

	if types != nil {
		if types.Direct == nil {
			err = ErrMissingDirectMapInTypes
			return
		}
		if types.Inverse == nil {
			err = ErrMissingInverseMapInTypes
			return
		}
	}

	// check registry presence

	if r.Reg == (RegistryReference{}) {
		err = ErrEmptyRegsitryReference
		return
	}

	pack = new(Pack)
	pack.r = r

	if pack.reg = c.Registry(r.Reg); pack.reg == nil {
		err = fmt.Errorf("missing registry [%s] of Root %s",
			r.Reg.Short(),
			r.PH())
		pack = nil // release for GC
		return
	}

	// create the pack

	pack.flags = flags
	pack.types = types

	pack.cache = make(map[cipher.SHA256][]byte)
	pack.unsaved = make(map[cipher.SHA256][]byte)

	pack.c = c

	if err = pack.init(); err != nil { // initialize
		pack = nil // release for GC
	}

	return

}

// CelanUp removes unused objects from database
func CleanUp() (err error) {
	// TODO
	return
}
