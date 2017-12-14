package skyobject

import (
	"errors"
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/skyobject/statutil"
)

// common errors
var (
	ErrStopIteration            = errors.New("stop iteration")
	ErrInvalidArgument          = errors.New("invalid argument")
	ErrMissingTypes             = errors.New("missing Types maps")
	ErrMissingDirectMapInTypes  = errors.New("missing Direct map in Types")
	ErrMissingInverseMapInTypes = errors.New("missing Inverse map in Types")
	ErrEmptyRegistryRef         = errors.New("empty registry reference")
	ErrNotFound                 = errors.New("not found")
	ErrEmptyRootHash            = errors.New("empty hash of Root")
	ErrMissingDB                = errors.New(
		"skyobject.NewContainer: missing *data.DB")
)

// A Container represents overlay for database
// and all related to CX objects. The Container
// manages DB, Registries, Root objects. The
// Container also provides a way to create, explore
// and modify Root objects, add/remove feeds, etc
type Container struct {
	log.Logger

	conf Config   // configurations
	db   *data.DB // database

	coreRegistry *Registry // core registry or nil

	rootsHolder // hold Roots

	// stat
	packSave statutil.RollAvg
	cleanUp  statutil.RollAvg // TODO (kostyarin): cleaning up
}

// NewContainer creates new Container instance by given DB
// and configurations. The DB must be used by this container
// exclusively (e.g. no one another Container should use the DB).
// The db argument must not be nil. If conf argument is nil,
// then default configurations used (see NewConfig for details).
// This function also panics if given configs contains invalid
// values
func NewContainer(db *data.DB, conf *Config) (c *Container, err error) {

	if db == nil {
		err = ErrMissingDB
		return
	}

	if conf == nil {
		conf = NewConfig()
	}

	if err = conf.Validate(); err != nil {
		return
	}

	c = new(Container)
	c.Logger = log.NewLogger(conf.Log)

	c.db = db
	c.conf = *conf // copy configs

	if conf.Registry != nil {
		c.coreRegistry = conf.Registry
	}

	c.rootsHolder.holded = make(map[holdedRoot]int)

	c.packSave = statutil.NewRollAvg(conf.RollAvgSamples)
	c.cleanUp = statutil.NewRollAvg(conf.RollAvgSamples)
	return
}

// DB returns underlying *data.DB
func (c *Container) DB() *data.DB {
	return c.db
}

// CoreRegistry of the Container or nil if
// the Container created without a Registry
func (c *Container) CoreRegistry() *Registry {
	c.Debugln(VerbosePin, "CoreRegistry", c.coreRegistry != nil)

	return c.coreRegistry
}

// Registry by RegistryRef. The method can returns database error or
// idxdb.ErrNotfound if requested registry has not been found
func (c *Container) Registry(rr RegistryRef) (r *Registry, err error) {
	c.Debugln(VerbosePin, "Registry", rr.Short())

	var val []byte
	if val, _, err = c.DB().CXDS().Get(cipher.SHA256(rr)); err != nil {
		return
	}
	r, err = DecodeRegistry(val)
	return
}

// Unpack given Root object. Use flags by your needs
//
//     r, err := c.Root(pk, 500)
//     if err != nil {
//         // handle error
//         return
//     }
//
//     theFlagsIUsuallyUse := EntireTree | HashTableIndex
//
//     pack, err := c.Unpack(r, theFlagsIUsuallyUse, c.CoreRegistry().Types(),
//         cipher.SecKey{})
//     if err != nil {
//         // handle error
//         return
//     }
//
//     // use the pack
//
// The sk argument should not be empty if you want to modify the Root
// and publish changes. In any other cases it is not necessary and can be
// passed like cipher.SecKey{}. If the sk is empty then ViewOnly flag
// will be set
//
// The Unpack holds given Root object. Thus, you have to Close the
// Pack after use. Feel free to close it many times. But keep in
// mind that after closing Pack can't be used
func (c *Container) Unpack(r *Root, flags Flag, types *Types,
	sk cipher.SecKey) (pack *Pack, err error) {

	// check arguments

	if r == nil {
		c.Debugln(VerbosePin, "Unpack nil")
		err = ErrInvalidArgument
		return
	}

	// debug log
	c.Debugln(VerbosePin, "Unpack", r.Pub.Hex()[:7], r.Seq)

	if err = r.Pub.Verify(); err != nil {
		err = fmt.Errorf("invalid public key of given Root: %v", err)
		return
	}

	if sk == (cipher.SecKey{}) {
		flags = flags | ViewOnly // can't modify
	} else {
		if err = sk.Verify(); err != nil {
			err = fmt.Errorf("invalid secret key: %v", err)
			return
		}
	}

	if types == nil {
		err = ErrMissingTypes
		return
	} else if types.Direct == nil {
		err = ErrMissingDirectMapInTypes
		return
	} else if types.Inverse == nil {
		err = ErrMissingInverseMapInTypes
		return
	}

	// check registry presence

	if r.Reg == (RegistryRef{}) {
		err = ErrEmptyRegistryRef
		return
	}

	pack = new(Pack)
	pack.r = r

	if c.coreRegistry == nil {
		err = errors.New("missing core registry")
		pack = nil // release for GC
		return
	}

	if r.Reg != c.coreRegistry.Reference() {
		err = errors.New("Root not points to core registry of the Continer")
		return
	}

	pack.reg = c.coreRegistry

	// create the pack

	pack.flags = flags
	pack.types = types

	pack.unsaved = make(map[cipher.SHA256][]byte)

	pack.c = c
	pack.sk = sk

	if err = pack.init(); err != nil { // initialize
		pack = nil // release for GC
		return
	}

	if flags&freshRoot == 0 {
		c.Hold(r.Pub, r.Seq)
	}

	return

}

// NewRoot associated with core registry
func (c *Container) NewRoot(pk cipher.PubKey, sk cipher.SecKey, flags Flag,
	types *Types) (pack *Pack, err error) {

	c.Debugln(VerbosePin, "NewRoot", pk.Hex()[:7])

	if c.coreRegistry == nil {
		err = errors.New("can't create new Root: missing core Registry")
		return
	}

	if sk == (cipher.SecKey{}) {
		err = errors.New("empty secret key")
		return
	}

	r := new(Root)
	r.Pub = pk
	r.Reg = c.coreRegistry.Reference()
	return c.Unpack(r, flags|freshRoot, types, sk)
}

// Close the Container. The
// closing doesn't close DB
func (c *Container) Close() error {
	c.Debug(VerbosePin, "Close()")

	// TODO (kostyarin): cleaning up
	return nil
}

//
// feeds
//

// AddFeed add feed to DB. The method never returns to
// make the Container able to handle Root object related to
// the feed. The feed stored in DB. This method never
// returns "already exists" errors
func (c *Container) AddFeed(pk cipher.PubKey) error {
	c.Debugln(VerbosePin, "AddFeed", pk.Hex()[:7])

	return c.DB().IdxDB().Tx(func(feeds data.Feeds) error {
		return feeds.Add(pk)
	})
}

// DelFeed deletes feed and all related Root objects and objects
// related to the Root objects (if this objects not used by some other feeds).
// The method never returns "not found" errors. The method returns error
// if there are held Root objects of the feed. Close all Pack
// instances that uses Root objects of the feed to release them
func (c *Container) DelFeed(pk cipher.PubKey) error {
	c.Debugln(VerbosePin, "DelFeed", pk.Hex()[:7])

	return c.DB().IdxDB().Tx(func(feeds data.Feeds) (err error) {

		if false == feeds.Has(pk) {
			return // nothing to delete
		}

		var rs data.Roots
		if rs, err = feeds.Roots(pk); err != nil {
			return
		}

		if rs.Len() == 0 {
			return feeds.Del(pk) // delete empty feed
		}

		// remove all Root objects first
		// (and decrement refs count of all related objects)
		// checking out held Roots

		err = rs.Ascend(func(ir *data.Root) (err error) {
			if err = c.CanRemove(pk, ir.Seq); err != nil {
				return
			}

			if err = c.decrementAll(ir); err != nil {
				return
			}

			return rs.Del(ir.Seq) // delete the Root
		})

		if err != nil {
			return
		}

		return feeds.Del(pk)
	})
}

// DelRoot deletes Root of given feed with given seq number. If the Root
// doesn't exists, then this method doesn't returns an error. Even if the feed
// does not exist
func (c *Container) DelRoot(pk cipher.PubKey, seq uint64) (err error) {
	c.Debugln(VerbosePin, "DelRoot", pk.Hex()[:7])

	return c.DB().IdxDB().Tx(func(feeds data.Feeds) (err error) {
		if false == feeds.Has(pk) {
			return // nothing to delete
		}
		var rs data.Roots
		if rs, err = feeds.Roots(pk); err != nil {
			return
		}
		if rs.Len() == 0 {
			return feeds.Del(pk) // delete empty feed
		}
		var ir *data.Root
		if ir, err = rs.Get(seq); err != nil {
			if err == data.ErrNotFound {
				err = nil
			}
			return
		}
		if err = c.CanRemove(pk, ir.Seq); err != nil {
			return
		}
		if err = c.decrementAll(ir); err != nil {
			return
		}
		return rs.Del(ir.Seq) // delete the Root
	})
}
