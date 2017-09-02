package skyobject

import (
	"errors"
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node/log"
)

// common errors
var (
	ErrStopIteration            = errors.New("stop iteration")
	ErrInvalidArgument          = errors.New("invalid argument")
	ErrMissingTypes             = errors.New("missing Types maps")
	ErrMissingDirectMapInTypes  = errors.New("missing Direct map in Types")
	ErrMissingInverseMapInTypes = errors.New("missing Inverse map in Types")
	ErrEmptyRegistryRef         = errors.New("empty refsitry reference")
)

// A Container represents overlay for database
// and all realted to CX obejcts. The Container
// manages DB, Registries, Root obejcts. The
// Container also provides a way to create, explore
// and modify Root obejcts, add/remove feeds, etc
type Container struct {
	log.Logger

	conf Config   // configurations
	db   *data.DB // database

	coreRegistry *Registry // core registry or nil

	rootsHolder // hold Roots
}

// NewContainer creates new Container instance by given DB
// and configurations. The DB must be used by this container
// exclusively (e.g. no one another Container should use the DB).
// The db argument must not be nil. If conf argument is nil,
// then default configurations used (see NewConfig for details).
// This function also panics if given configs contains invalid
// values
func NewContainer(db *data.DB, conf *Config) (c *Container) {

	if db == nil {
		panic("missing *data.DB")
	}

	if conf == nil {
		conf = NewConfig()
	}

	if err := conf.Validate(); err != nil {
		panic(err)
	}

	c = new(Container)
	c.Logger = log.NewLogger(conf.Log)

	c.db = db
	c.conf = *conf // copy configs

	if conf.Registry != nil {
		c.coreRegistry = conf.Registry
	}

	c.rootsHolder.holded = make(map[holdedRoot]int)
	return
}

// DB returns underlying *data.DB
func (c *Container) DB() *data.DB {
	return c.db
}

// CoreRegisty of the Container or nil if
// the Container created without a Regsitry
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

// Unpack given Root obejct. Use flags by your needs
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
		err = fmt.Errorf("invalud public key of given Root: %v", err)
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

// NewRoot associated with core regsitry
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

	c.Print("[CONTAINER CLOSING] holded:", c.holded)

	// TODO (kostyarin): cleaning up
	return nil
}

//
// feeds
//

// AddFeed add feed to DB. The method never retruns to
// make the Container able handle Root object related to
// the feed. The feed stored in DB. This mehtod never
// returns "already exists" errors
func (c *Container) AddFeed(pk cipher.PubKey) error {
	c.Debugln(VerbosePin, "AddFeed", pk.Hex()[:7])

	return c.DB().IdxDB().Tx(func(feeds data.Feeds) error {
		return feeds.Add(pk)
	})
}

// DelFeed deletes feed and all related Root obejcts and objects
// relates to the Root obejcts (if this obejcts not used by some other feeds).
// The method never returns "not found" errors. The method returns error
// if there are holded Root obejcts of the feed. Close all Pack
// instances that uses Root obejcts of the feed to release them
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
		// (and decrement refs count of all related obejcts)
		// checking out holded Roots

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
