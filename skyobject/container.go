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
	ErrStopRange                = errors.New("stop range")
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

	db data.DB

	trackSeq
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
//
// TODO (kostyarin): move the db argument to Config creating
// in-memory DB by default (if nil). Think about panics
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

	c.closeq = make(chan struct{})

	c.Logger = log.NewLogger(conf.Log)
	c.regs = make(map[RegistryRef]*Registry)

	// copy configs
	c.conf = *conf

	c.initSeqTrackStat()

	if conf.Registry != nil {
		c.coreRegistry = conf.Registry
		if err := c.AddRegistry(conf.Registry); err != nil {
			c.db.Close() // to be safe
			panic(err)   // fatality
		}
	}

	if c.conf.CleanUp > 0 {
		c.await.Add(1)
		c.cleanUpByInterval()
	}

	return
}

func (c *Container) initSeqTrackStat() {
	// initialize trackSeq
	c.trackSeq.init()
	c.stat.init(c.conf.StatSamples)
	// range over feeds of DB and fill up the trackSeq
	err := c.DB().View(func(tx data.Tv) error {
		feeds := tx.Feeds()
		return feeds.Range(func(pk cipher.PubKey) (err error) {
			roots := feeds.Roots(pk)
			var seq, last uint64 // last seq and seq of last full
			var hasLast bool     // really has last full root (if last is 0)
			err = roots.Reverse(func(rp *data.RootPack) (_ error) {
				if seq == 0 {
					seq = rp.Seq
				}
				if rp.IsFull {
					last, hasLast = rp.Seq, true
					return data.ErrStopRange // break
				}
				return // continue, finding last full
			})
			if err != nil {
				return
			}
			c.trackSeq.addSeq(pk, seq, false)
			if hasLast {
				c.trackSeq.addSeq(pk, last, true)
				return
			}
			return
		})
	})
	if err != nil {
		c.DB().Close()
		c.Panic("unexpected database error: ", err)
	}
}

// saveRegistry in database
func (c *Container) saveRegistry(reg *Registry) error {
	return c.DB().Update(func(tx data.Tu) error {
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
		c.stat.addRegistry(1)
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
	return c.DB().Update(func(tx data.Tu) error {
		return tx.Objects().Set(hash, val)
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

// Registry by RegistryRef. It returns nil if
// the Container doesn't contain required Registry
func (c *Container) Registry(rr RegistryRef) *Registry {
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

	if err = pack.init(); err != nil { // initialize
		pack = nil // release for GC
	}

	return

}

// CelanUp removes unused objects from database
func (c *Container) CleanUp(keepRoots bool) (err error) {

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

	err = c.DB().Update(func(tx data.Tu) (_ error) {

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
			c.Debugf(CleanUpVerbosePin, "CleanUp collecting took: ",
				verboseElapsed)
		}

		//
		// delete
		//

		for k, v := range coll {
			if v != 0 {
				delete(coll, k) // remove necessary objects from the map
			}
		}

		if len(coll) != 0 && len(collRoots) != 0 {
			err = c.cleanUpRemove(tx, coll, collRoots, keepRoots)
		}

		return
	})

	elapsed = time.Now().Sub(tp)
	c.stat.addCleanUp(elapsed)

	if err != nil {
		c.Debugf(CleanUpPin, "CleanUp failed after %v: %v", elapsed, err)
		return
	}

	if c.Logger.Pins()&CleanUpVerbosePin != 0 {
		verboseElapsed = time.Now().Sub(tp) - verboseElapsed
		c.Debugf(CleanUpVerbosePin, "CleanUp removing took: ",
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

	return feeds.Range(func(pk cipher.PubKey) (err error) {

		roots := feeds.Roots(pk)

		var lastFull uint64
		var hasLastFull bool

		err = roots.Reverse(func(rp *data.RootPack) (err error) {

			if rp.IsFull {
				lastFull, hasLastFull = rp.Seq, true
				if !keepRoots {
					// we will delete roots below last full
					return data.ErrStopRange
				}
			}

			var r *Root
			if r, err = c.unpackRoot(pk, rp); err != nil {
				return
			}

			c.knowsAbout(r, objs, func(hash cipher.SHA256) (deeper bool,
				err error) {

				if _, ok := coll[hash]; !ok {
					coll[hash] = 1
					return true, nil // go deeper
				}
				return // already known the object
			})

			// ignore all possible errors of the knowsAbout
			// becasue we need to walk through all Roots
			//
			// TOTH (kostyarin): ignore? or be strict?

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
		return objs.RangeDel(func(key cipher.SHA256, _ []byte) (del bool,
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

	r = new(Root)
	if err = encoder.DeserializeRaw(rp.Root, r); err != nil {
		// detailed error
		err = fmt.Errorf("error decoding root"+
			" (feed %s, seq %d, hash %s): %v",
			pk.Hex()[:7],
			rp.Seq,
			rp.Hash.Hex()[:7],
			err)
		r = nil
	}
	return
}

// removeNonFullRoots removes all non-full
// Root objects from database
func (c *Container) removeNonFullRoots() error {

	// don't perform simultaneously with CleanUp
	c.cleanmx.Lock()
	defer c.cleanmx.Unlock()

	return c.DB().Update(func(tx data.Tu) error {
		feeds := tx.Feeds()
		return feeds.Range(func(pk cipher.PubKey) error {
			roots := feeds.Roots(pk)
			return roots.RangeDel(func(rp *data.RootPack) (del bool, _ error) {
				del = !rp.IsFull
				return
			})
		})
	})
}

// Close the Container. The
// closing doesn't close DB
func (c *Container) Close() error {

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
	return c.DB().Update(func(tx data.Tu) error {
		return tx.Feeds().Add(pk)
	})
}

// DelFeed. The method never returns "not found" errors
func (c *Container) DelFeed(pk cipher.PubKey) (err error) {
	err = c.DB().Update(func(tx data.Tu) error {
		return tx.Feeds().Del(pk)
	})
	if err == nil {
		c.trackSeq.delFeed(pk)
	}
	return
}
