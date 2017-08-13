package skyobject

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

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
)

// A Container represents container of Root
// objects
type Container struct {
	log.Logger
	conf Config
	db   data.DB
	stat

	// registries
	coreRegistry *Registry

	rmx  sync.RWMutex
	regs map[RegistryRef]*Registry

	// clean up
	cleanmx sync.Mutex // clean up mutex

	closeq chan struct{}  //
	closeo sync.Once      // clean up by interval
	await  sync.WaitGroup //
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
	if err := conf.Validate(); err != nil {
		panic(err)
	}
	c = new(Container)
	c.db = db
	c.closeq = make(chan struct{})
	c.Logger = log.NewLogger(conf.Log)
	c.regs = make(map[RegistryRef]*Registry)
	// copy configs
	c.conf = *conf
	c.stat.init(c.conf.StatSamples)

	if conf.Registry != nil {
		c.coreRegistry = conf.Registry
		if err := c.AddRegistry(conf.Registry); err != nil {
			panic(err) // fatality
		}
	}

	if c.conf.CleanUp > 0 {
		c.await.Add(1)
		go c.cleanUpByInterval()
	}

	return
}

// saveRegistry in database
func (c *Container) saveRegistry(reg *Registry) error {
	c.Debug(VerbosePin, "saveRegsitry ", reg.Reference().Short())

	return c.DB().Update(func(tx data.Tu) error {
		objs := tx.Objects()
		return objs.Set(cipher.SHA256(reg.Reference()), reg.Encode())
	})
}

// add registry that already saved in database
func (c *Container) addRegistry(reg *Registry) {
	c.Debug(VerbosePin, "addRegistry ", reg.Reference().Short())

	c.rmx.Lock()
	defer c.rmx.Unlock()

	if _, ok := c.regs[reg.Reference()]; !ok {
		c.regs[reg.Reference()] = reg
		c.stat.addRegistry(1)
	}
	return
}

// AddRegistry to the Container and save it into database until
// it removed by CelanUp
func (c *Container) AddRegistry(reg *Registry) (err error) {
	c.Debug(VerbosePin, "AddRegistry ", reg.Reference().Short())

	c.rmx.Lock()
	defer c.rmx.Unlock()

	if _, ok := c.regs[reg.Reference()]; !ok {
		if err = c.saveRegistry(reg); err == nil {
			c.regs[reg.Reference()] = reg
			c.stat.addRegistry(1)
		}
	}
	return
}

// DB returns underlying data.DB
func (c *Container) DB() data.DB {
	return c.db
}

// Set saves single object into database
func (c *Container) Set(hash cipher.SHA256, val []byte) (err error) {
	c.Debugln(VerbosePin, "Set", hash.Hex()[:7])

	return c.DB().Update(func(tx data.Tu) error {
		return tx.Objects().Set(hash, val)
	})
}

// Get returns data by hash. Result is nil if data not found
func (c *Container) Get(hash cipher.SHA256) (value []byte) {
	c.Debugln(VerbosePin, "Get", hash.Hex()[:7])

	err := c.db.View(func(tx data.Tv) (_ error) {
		value = tx.Objects().GetCopy(hash)
		return
	})
	if err != nil {
		panic("database error: " + err.Error())
	}
	return
}

// CoreRegisty of the Container or nil if
// the Container created without a Regsitry
func (c *Container) CoreRegistry() *Registry {
	c.Debugln(VerbosePin, "CoreRegistry", c.coreRegistry != nil)

	return c.coreRegistry
}

// Registry by RegistryRef. It returns nil if
// the Container doesn't contain required Registry
func (c *Container) Registry(rr RegistryRef) *Registry {
	c.Debugln(VerbosePin, "Registry", rr.Short())

	c.rmx.RLock()
	defer c.rmx.RUnlock()

	return c.regs[rr]
}

// Root of a feed by seq. If err is nil then the Root is not
func (c *Container) Root(pk cipher.PubKey, seq uint64) (r *Root, err error) {
	c.Debugln(VerbosePin, "Root", pk.Hex()[:7], seq)

	var rp *data.RootPack
	err = c.db.View(func(tx data.Tv) (_ error) {
		roots := tx.Feeds().Roots(pk)
		if roots == nil {
			return ErrNoSuchFeed
		}
		rp = roots.Get(seq)
		return
	})
	if err != nil {
		return
	}
	if rp == nil {
		err = fmt.Errorf("root %d of %s not found", seq, pk.Hex()[:7])
		return
	}
	r, err = c.unpackRoot(pk, rp)
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
// If the EntireTree flag provided then given Root (entire tree) will be
// unpacked inside the Unpack method call. Unpack wihtout GoTypes
// flag will not wrok, because the feature is not implemented yet
//
// The sk argument should not be empty if you want to modify the Root
// and pulish changes. In any ither cases it is not necessary and can be
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
		err = ErrEmptyRegsitryRef
		return
	}

	pack = new(Pack)
	pack.r = r

	if pack.reg = c.Registry(r.Reg); pack.reg == nil {
		err = fmt.Errorf("missing registry [%s] of Root %s",
			r.Reg.Short(),
			r.Short())
		pack = nil // release for GC
		return
	}

	// create the pack

	pack.flags = flags
	pack.types = types

	pack.unsaved = make(map[cipher.SHA256][]byte)

	pack.c = c
	pack.sk = sk

	if err = pack.init(); err != nil { // initialize
		pack = nil // release for GC
	}

	return

}

// NewRoot associated with core regsitry. It panics if the container created
// without core regsitry
func (c *Container) NewRoot(pk cipher.PubKey, sk cipher.SecKey, flags Flag,
	types *Types) (pack *Pack, err error) {

	c.Debugln(VerbosePin, "NewRoot", pk.Hex()[:7])

	if c.coreRegistry == nil {
		err = errors.New("can't create new Root: missing core Registry")
		return
	}

	r := new(Root)
	r.Pub = pk
	r.Reg = c.coreRegistry.Reference()
	return c.Unpack(r, flags, types, sk)
}

// NewRootReg associated with given regsitry
func (c *Container) NewRootReg(pk cipher.PubKey, sk cipher.SecKey,
	reg RegistryRef, flags Flag, types *Types) (pack *Pack, err error) {

	c.Debugln(VerbosePin, "NewRoot", pk.Hex()[:7])

	r := new(Root)
	r.Pub = pk
	r.Reg = reg
	return c.Unpack(r, flags, types, sk)
}

// CelanUp removes unused objects from database
func (c *Container) CleanUp(keepRoots bool) (err error) {

	c.Debugln(VerbosePin, "CleanUp, keep roots:", keepRoots)

	// avoid simultaneous CleanUps,
	// otherwise timing would be wrong
	c.cleanmx.Lock()
	defer c.cleanmx.Unlock()

	tp := time.Now()
	var elapsed, verboseElapsed time.Duration

	// hash -> needed (> 0)
	coll := make(map[cipher.SHA256]int)

	// remove roots before (seq of last full)
	collRoots := make(map[cipher.PubKey]uint64)

	// we have to use single transaction

	err = c.DB().Update(func(tx data.Tu) (err error) {

		//
		// collect
		//

		err = c.cleanUpCollect(tx, coll, collRoots, keepRoots)

		if err != nil {
			c.Debugf(CleanUpPin,
				"CleanUp failed after collecting, took: %v, error: %v",
				time.Now().Sub(tp),
				err)
			return
		}

		if c.Logger.Pins()&CleanUpVerbosePin != 0 {
			verboseElapsed = time.Now().Sub(tp)
			c.Debug(CleanUpVerbosePin, "CleanUp collecting took: ",
				verboseElapsed)
		}

		//
		// delete
		//

		if cr := c.CoreRegistry(); cr != nil {
			coll[cipher.SHA256(cr.Reference())] = 1
		}

		if len(coll) != 0 && len(collRoots) != 0 {
			err = c.cleanUpRemove(tx, coll, collRoots, keepRoots)
		}

		return
	})

	elapsed = time.Now().Sub(tp)
	c.stat.addCleanUp(elapsed)

	if err != nil {
		c.Printf("CleanUp failed after %v: %v", elapsed, err)
		return
	}

	if c.Logger.Pins()&CleanUpVerbosePin != 0 {
		verboseElapsed = time.Now().Sub(tp) - verboseElapsed
		c.Debug(CleanUpVerbosePin, "CleanUp removing took: ",
			verboseElapsed)
	}

	c.Debugln(CleanUpPin, "CleanUp", elapsed)
	return
}

func (c *Container) cleanUpCollect(tx data.Tu, coll map[cipher.SHA256]int,
	collRoots map[cipher.PubKey]uint64, keepRoots bool) error {

	feeds := tx.Feeds()
	objs := tx.Objects()

	// range over roots

	return feeds.Ascend(func(pk cipher.PubKey) (err error) {

		roots := feeds.Roots(pk)

		var lastFull uint64
		var hasLastFull bool

		err = roots.Descend(func(rp *data.RootPack) (err error) {

			var r *Root
			if r, err = c.unpackRoot(pk, rp); err != nil {
				return
			}

			kerr := c.knowsAbout(r, objs, func(hash cipher.SHA256) (deeper bool,
				_ error) {

				if _, ok := coll[hash]; !ok {
					coll[hash] = 1
					return true, nil // go deeper
				}
				return // already known the object
			})
			if kerr != nil {
				c.Printf("[ERR] knowsAbout of %s error: %v",
					r.Short(),
					kerr)
			}

			// ignore all possible errors of the knowsAbout
			// becasue we need to walk through all Roots
			//
			// TOTH (kostyarin): ignore? or be strict?

			if rp.IsFull && !keepRoots {
				lastFull, hasLastFull = rp.Seq, true
				// we will delete roots below last full
				return data.ErrStopIteration
			}

			return
		})

		if hasLastFull && !keepRoots {
			collRoots[pk] = lastFull
		}

		return

	})

}

func (c *Container) cleanUpRemove(tx data.Tu, coll map[cipher.SHA256]int,
	collRoots map[cipher.PubKey]uint64, keepRoots bool) (err error) {

	// remove unused registries

	c.cleanUpRemoveRegistries(coll)

	// remove roots

	if len(collRoots) > 0 {
		feeds := tx.Feeds()
		for pk, before := range collRoots {
			if roots := feeds.Roots(pk); roots != nil {
				if err = roots.DelBefore(before); err != nil {
					return
				}
			}
		}
	}

	// remove objects

	if len(coll) > 0 {
		objs := tx.Objects()
		return objs.AscendDel(func(key cipher.SHA256, _ []byte) (del bool,
			_ error) {

			if _, ok := coll[key]; !ok {
				del = true
			}
			return
		})
	}

	return
}

func (c *Container) cleanUpRemoveRegistries(coll map[cipher.SHA256]int) {
	c.rmx.Lock()
	defer c.rmx.Unlock()

	for k := range c.regs {
		if _, ok := coll[cipher.SHA256(k)]; !ok {
			delete(c.regs, k)
			c.stat.addRegistry(-1)
		}
	}
}

func (c *Container) cleanUpByInterval() {
	defer c.await.Done()

	c.Debug(CleanUpPin, "start CleanUp loop by ", c.conf.CleanUp)
	defer c.Debug(CleanUpPin, "stop CleanUp loop")

	tk := time.NewTicker(c.conf.CleanUp)
	defer tk.Stop()

	tick := tk.C

	var err error

	for {
		select {
		case <-tick:
			if err = c.CleanUp(c.conf.KeepRoots); err != nil {
				c.Print("[ERR] CleanUp error: ", err)
			}
		case <-c.closeq:
			return
		}
	}
}

func (c *Container) unpackRoot(pk cipher.PubKey, rp *data.RootPack) (r *Root,
	err error) {

	c.Debugln(VerbosePin, "unpackRoot", pk.Hex()[:7], rp.Seq)

	r = new(Root)
	if err = encoder.DeserializeRaw(rp.Root, r); err != nil {
		// detailed error
		err = fmt.Errorf("error decoding root"+
			" {%s:%d}: %v",
			pk.Hex()[:7],
			rp.Seq,
			err)
		r = nil
		return
	}
	r.Sig = rp.Sig
	r.Hash = rp.Hash
	return
}

// removeNonFullRoots removes all non-full
// Root objects from database
func (c *Container) removeNonFullRoots() error {
	c.Debug(VerbosePin, "removeNonFullRoots")

	// don't perform simultaneously with CleanUp
	c.cleanmx.Lock()
	defer c.cleanmx.Unlock()

	return c.DB().Update(func(tx data.Tu) error {
		feeds := tx.Feeds()
		return feeds.Ascend(func(pk cipher.PubKey) error {
			roots := feeds.Roots(pk)
			return roots.AscendDel(func(rp *data.RootPack) (del bool, _ error) {
				del = !rp.IsFull
				return
			})
		})
	})
}

// Close the Container. The
// closing doesn't close DB
func (c *Container) Close() error {
	c.Debug(VerbosePin, "Close")

	// close cleanUpByInterval

	c.closeo.Do(func() {
		close(c.closeq)
	})
	c.await.Wait()

	if !c.conf.KeepNonFull {
		if err := c.removeNonFullRoots(); err != nil {
			c.Print("[ERR] error removing non-full roots:", err)
		}
	}

	// and remove all possible
	return c.CleanUp(c.conf.KeepRoots)
}

//
// feeds
//

// AddFeed. The method never retruns "already exists" errors
func (c *Container) AddFeed(pk cipher.PubKey) error {
	c.Debugln(VerbosePin, "AddFeed", pk.Hex()[:7])

	return c.DB().Update(func(tx data.Tu) error {
		return tx.Feeds().Add(pk)
	})
}

// DelFeed. The method never returns "not found" errors
func (c *Container) DelFeed(pk cipher.PubKey) (err error) {
	c.Debugln(VerbosePin, "DelFeed", pk.Hex()[:7])

	err = c.DB().Update(func(tx data.Tu) error {
		return tx.Feeds().Del(pk)
	})
	return
}
