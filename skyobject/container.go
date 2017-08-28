package skyobject

import (
	"errors"
	"fmt"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/data/idxdb"
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

	hmx  sync.Mutex
	hold map[holdedRoot]int
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

	c.hold = make(map[holdedRoot]int)

	return
}

type holdedRoot struct {
	Pub cipher.PubKey
	Seq uint64
}

func (c *Container) holdRoot(pk cipher.PubKey, seq uint64) {
	c.hmx.Lock()
	defer c.hmx.Unlock()

	hr := holdedRoot{pk, seq}

	c.hold[hr] = c.hold[hr] + 1
}

func (c *Container) unholdRoot(pk cipher.PubKey, seq uint64) {
	c.hmx.Lock()
	defer c.hmx.Unlock()

	hr := holdedRoot{pk, seq}

	if v, ok := c.hold[hr]; ok {
		if v == 1 {
			delete(c.hold, hr)
		} else {
			c.hold[hr] = c.hold[hr] - 1
		}
	}
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

// Root of a feed by seq. The method can returns database error or
// idxdb.ErrNotfound if requested Root has not been found
func (c *Container) Root(pk cipher.PubKey, seq uint64) (r *Root, err error) {
	c.Debugln(VerbosePin, "Root", pk.Hex()[:7], seq)

	// request index

	var ir *idxdb.Root
	err = c.DB().IdxDB().Tx(func(feeds idxdb.Feeds) (err error) {
		var rs idxdb.Roots
		if rs, err = feeds.Roots(pk); err != nil {
			return
		}
		ir, err = rs.Get(seq)
		return
	})
	if err != nil {
		return
	}

	// request CXDS

	var val []byte
	if val, _, err = c.DB().CXDS().Get(ir.Hash); err != nil {
		return
	}

	// decode

	r = new(Root)
	if err = encoder.DeserializeRaw(val, r); err != nil {
		return
	}

	// copy meta information

	r.Sig = ir.Sig
	r.Hash = ir.Hash
	r.IsFull = ir.IsFull

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
		c.holdRoot(r.Pub, r.Seq)
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
	c.Debug(VerbosePin, "Close")

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

	return c.DB().IdxDB().Tx(func(feeds idxdb.Feeds) error {
		return feeds.Add(pk)
	})
}

// DelFeed deletes feed and all related Root obejcts and objects
// relates to the Root obejcts. The method never returns
// "not found" errors
func (c *Container) DelFeed(pk cipher.PubKey) error {
	c.Debugln(VerbosePin, "DelFeed", pk.Hex()[:7])

	return c.DB().IdxDB().Tx(func(feeds idxdb.Feeds) error {
		return feeds.Del(pk)
	})
}
