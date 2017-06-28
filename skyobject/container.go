package skyobject

import (
	"errors"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

var (
	// ErrNoCoreRegistry occurs while you are trying to call NewRoot
	// of Container that created without Registry
	ErrNoCoreRegistry = errors.New(
		"missing registry, Container created without registry")
	// ErrAlreadyHaveThisRoot occurs when Container already have
	// root with the same Seq number
	ErrAlreadyHaveThisRoot = errors.New("already have the root")
)

// A Container represents ...
type Container struct {
	// lock database (for GC)
	dbmx sync.RWMutex
	db   data.DB // databse

	// lock registries
	rmx          sync.RWMutex
	coreRegistry *Registry // registry witch which the container was created
	registries   map[RegistryReference]*Registry

	gcmx sync.Mutex
}

// NewContainer creates new Container using given databse and
// optional Registry. If Registry is no nil, then the registry
// will be used to create Root objects by NewRoot. The registry
// is just usablility trick. You can create a Container without
// registry, add a Registry using AddRegistry method and
// create a Root using NewRootReg method
func NewContainer(db data.DB, reg *Registry) (c *Container) {
	if db == nil {
		panic("missing data.DB")
	}
	c = new(Container)
	c.db = db
	c.registries = make(map[RegistryReference]*Registry)
	if reg != nil {
		reg.Done()
		// store registry in database
		c.db.Set(cipher.SHA256(reg.Reference()), reg.Encode())
		c.coreRegistry = reg
		c.registries[reg.Reference()] = reg
	}
	return
}

// database

// DB returns unerlying database. It's unsafe to
// insert some data to database if GC of Container
// called. Use Set and Get method of Container
// or similar methods of Root to be safe
func (c *Container) DB() data.DB {
	return c.db
}

// registry

// AddRegistry to the Container. A registry can be removed
// by GC() or RegistiesGC() if no root refers it
func (c *Container) AddRegistry(reg *Registry) {
	reg.Done() // call Done

	// use Reference instead of RegistryReference to use
	// Set method (with locks), the Reference will be
	// converted to cipher.SHA256 and then to []byte
	// anyway (it doesn't make sense, the Set method
	// requires it to be type of Reference)
	c.Set(Reference(reg.Reference()), reg.Encode()) // store

	c.rmx.Lock()
	defer c.rmx.Unlock()

	// don't replace
	if _, ok := c.registries[reg.Reference()]; !ok {
		c.registries[reg.Reference()] = reg
	}
}

// CoreRegistry returns registry with which the Container
// was created. It can returns nil
func (c *Container) CoreRegistry() *Registry {
	return c.coreRegistry
}

// Registry by reference
func (c *Container) Registry(rr RegistryReference) (reg *Registry, err error) {
	// c.coreRegistry is read-only and we don't need to lock/unlock
	if c.coreRegistry != nil && rr == c.coreRegistry.Reference() {
		reg = c.coreRegistry
		return
	}

	c.rmx.RLock()
	defer c.rmx.RUnlock()

	// never lookup database keeping all registries as a hot-list,
	// because registries are slow to unpack, and a Root object
	// has short-hand reference to related Registry
	var ok bool
	if reg, ok = c.registries[rr]; !ok {
		err = &MissingRegistryError{rr}
	}
	return
}

func (c *Container) unpackRoot(rp *data.RootPack) (r *Root, err error) {
	var x encodedRoot
	if err = encoder.DeserializeRaw(rp.Root, &x); err != nil {
		return
	}
	r = new(Root)
	r.refs = x.Refs
	r.reg = x.Reg
	r.time = x.Time
	r.seq = x.Seq
	r.pub = x.Pub
	r.cnt = c

	r.prev = x.Prev

	r.sig = rp.Sig
	r.hash = rp.Hash
	return
}

// WantRegistry returns true if given registry wanted by the Container
func (c *Container) WantRegistry(rr RegistryReference) (want bool) {

	// don't need to lock database because we don't change it

	var have bool
	for _, pk := range c.db.Feeds() {
		c.db.RangeFeed(pk, func(rp *data.RootPack) (stop bool) {
			var x encodedRoot
			if err := encoder.DeserializeRaw(rp.Root, &x); err != nil {
				panic(err) // critical
			}
			if x.Reg == rr {
				if _, ok := c.registries[rr]; !ok {
					want, stop = true, true // want
					return
				}
				have, stop = true, true // already have
				return
			}
			return // continue
		})
		if want || have {
			return // break feeds loop if we already know all we need
		}
	}
	return // don't want
}

// Registries returns registries that the Container has got
func (c *Container) Registries() (rrs []RegistryReference) {

	c.rmx.RLock()
	defer c.rmx.RUnlock()

	if len(c.registries) == 0 {
		return // nil
	}
	rrs = make([]RegistryReference, 0, len(c.registries))
	for rr := range c.registries {
		rrs = append(rrs, rr)
	}
	return
}

// Get object by Reference
func (c *Container) Get(ref Reference) (data []byte, ok bool) {

	// don't need to lock database, because we don't change it

	data, ok = c.db.Get(cipher.SHA256(ref))
	return
}

// Set adds given value to database using given reference
// as key for the value
func (c *Container) Set(ref Reference, p []byte) {

	// we need to lock database, because of GC
	c.dbmx.Lock()
	defer c.dbmx.Unlock()

	c.db.Set(cipher.SHA256(ref), p)
}

// save objects

func (c *Container) save(i interface{}) (ref Reference) {
	data := encoder.Serialize(i)
	hash := cipher.SumSHA256(data)

	// we need to lock database, because of GC
	c.dbmx.Lock()
	defer c.dbmx.Unlock()

	c.db.Set(hash, data)
	ref = Reference(hash)
	return
}

func (c *Container) saveArray(i ...interface{}) (refs References) {
	if len(i) == 0 {
		return // don't create empty slice
	}

	var data []byte
	var hash cipher.SHA256

	refs = make(References, 0, len(i)) // prepare

	// we need to lock database, because of GC
	c.dbmx.Lock()
	defer c.dbmx.Unlock()

	for _, e := range i {
		data = encoder.Serialize(e)
		hash = cipher.SumSHA256(data)

		c.db.Set(hash, data)

		refs = append(refs, Reference(hash))
	}
	return
}

// roots

// NewRoot creates new root associated with registry provided to
// NewContainer or NewContainerDB. The Root it returns is editable and
// detached. Fields Seq and Prev are actual
func (c *Container) NewRoot(pk cipher.PubKey, sk cipher.SecKey) (r *Root,
	err error) {

	if c.coreRegistry == nil {
		err = ErrNoCoreRegistry
		return
	}

	if r, err = c.newRoot(pk, sk); err != nil {
		return
	}

	r.reg = c.coreRegistry.Reference()
	r.rsh = c.coreRegistry

	return
}

// NewRootReg creates new root object with provided registry
// The method can create root object associated with registry
// that the container hasn't got. The Root it returns, is
// editable and detached. Fields Seq and Prev are actual
func (c *Container) NewRootReg(pk cipher.PubKey, sk cipher.SecKey,
	rr RegistryReference) (r *Root, err error) {

	if r, err = c.newRoot(pk, sk); err != nil {
		return
	}

	r.reg = rr

	return
}

// returns detached editable root with actual seq number and prev reference
func (c *Container) newRoot(pk cipher.PubKey, sk cipher.SecKey) (r *Root,
	err error) {

	if err = pk.Verify(); err != nil {
		return
	}

	if err = sk.Verify(); err != nil {
		return
	}

	var rp *data.RootPack
	var ok bool

	if rp, ok = c.db.LastRoot(pk); ok {
		if r, err = c.unpackRoot(rp); err != nil {
			panic(err) // critical
		}

		// set and clean nessesary fields

		r.sec = sk               // make it editable
		r.prev = r.hash          // shift
		r.hash = cipher.SHA256{} // clear
		r.seq++                  // increase
		r.refs = nil             // clear
		r.sig = cipher.Sig{}     // clear

	} else {
		r = new(Root)
		r.seq = 0
		r.sec = sk
		r.pub = pk
		r.cnt = c
	}

	return
}

// AddRootPack used to add a received root object to the
// Container. It returns an error if given data can't be decoded
// or signature is wrong. It also returns error if
// something wrong with prev/next/hash or seq
func (c *Container) AddRootPack(rp *data.RootPack) (r *Root, err error) {

	if r, err = c.unpackRoot(rp); err != nil {
		return
	}

	err = cipher.VerifySignature(r.pub, rp.Sig, rp.Hash)
	if err != nil {
		r = nil
		return
	}

	// lock database before modifing, because of GC
	c.dbmx.Lock()
	defer c.dbmx.Unlock()

	if err = c.db.AddRoot(r.pub, rp); err != nil {
		return
	}

	r.attached = true // the root is attached and it's seq is actual
	return
}

// LastRoot returns latest root object of the feed (pk).
// It can return nil. It can return received root object
// that doesn't contain secret key
func (c *Container) LastRoot(pk cipher.PubKey) (r *Root) {
	if rp, ok := c.db.LastRoot(pk); ok {
		var err error
		if r, err = c.unpackRoot(rp); err != nil {
			panic(err) // critical
		}
		r.attached = true // it's attached
	}
	return
}

// LastRootSk is equal to call LastRoot and then Edit
func (c *Container) LastRootSk(pk cipher.PubKey, sk cipher.SecKey) (r *Root) {
	if r = c.LastRoot(pk); r != nil {
		r.Edit(sk)
	}
	return
}

// LastFullRoot returns latest root object of the feed (pk) that is full.
// It can return nil. It can return received root object that doesn't
// contain secret key
func (c *Container) LastFullRoot(pk cipher.PubKey) (lastFull *Root) {
	c.db.RangeFeedReverse(pk, func(rp *data.RootPack) (stop bool) {
		var err error
		var r *Root
		if r, err = c.unpackRoot(rp); err != nil {
			panic(err) // critical
		}
		r.attached = true
		// first full from tail
		if r.IsFull() {
			lastFull, stop = r, true
			return // break
		}
		return // false (continue)
	})
	return
}

// Feeds returns public keys of feeds
// have at least one Root object
func (c *Container) Feeds() []cipher.PubKey {
	return c.db.Feeds()
}

// WantFeed calls (*Root).WantFunc with given WantFunc
// for every Root of the feed starting from older. Unlike
// (*Root).WantFunc the WantFeed ingores all errors
// (except ErrStopRange) trying to find all wanted objects
// even if some root has not related Registry or contains
// malformed data
func (c *Container) WantFeed(pk cipher.PubKey, wf WantFunc) {
	c.db.RangeFeed(pk, func(rp *data.RootPack) (stop bool) {
		var r *Root
		var err error
		if r, err = c.unpackRoot(rp); err != nil {
			panic(err) // critical
		}
		r.attached = true
		if err = r.WantFunc(wf); err == ErrStopRange {
			return true // stop
		}
		return // false (continue)
	})
}

// GotFeed calls (*Root).GotFunc with given GotFunc
// for every Root of the feed starting from older.
// Unlike (*Root).GotFunc the GotFeed ingores all errors
// (except ErrStopRange) trying to find all objects even
// if some root has not related Registry or contains
// malformed data
func (c *Container) GotFeed(pk cipher.PubKey, gf GotFunc) {
	c.db.RangeFeed(pk, func(rp *data.RootPack) (stop bool) {
		var err error
		var r *Root
		if r, err = c.unpackRoot(rp); err != nil {
			panic(err) // critical
		}
		r.attached = true
		if err = r.GotFunc(gf); err == ErrStopRange {
			return true // stop
		}
		return // false (continue)
	})
}

// AddFeed to database of do nothing if given
// feed already exists in the database
func (c *Container) AddFeed(pk cipher.PubKey) {
	c.dbmx.Lock()
	defer c.dbmx.Unlock()

	c.db.AddFeed(pk)
}

// HasFeed looks database about given feed
func (c *Container) HasFeed(pk cipher.PubKey) (yes bool) {
	return c.db.HasFeed(pk)
}

// DelFeed deletes all root object of given feed. The
// method doesn't perform GC
func (c *Container) DelFeed(pk cipher.PubKey) {
	c.dbmx.Lock()
	defer c.dbmx.Unlock()

	c.db.DelFeed(pk)
}

// RangeFeedFunc used by RangeFeed method of Container. If the
// function returns ErrStopRange then itteration terminates
// but not returns the error
type RangeFeedFunc func(r *Root) (err error)

// RangeFeed itterates root obejcts of given feed from old to new.
// Given RangeFeedFunc must be read-only
func (c *Container) RangeFeed(pk cipher.PubKey, fn RangeFeedFunc) (err error) {
	c.db.RangeFeed(pk, func(rp *data.RootPack) (stop bool) {
		var r *Root
		if r, err = c.unpackRoot(rp); err != nil {
			panic(err) // critical
		}
		r.attached = true
		if err = fn(r); err != nil {
			stop = true // break
			if err == ErrStopRange {
				err = nil
			}
		}
		return // continue
	})
	return
}

// RootByHash return Root by its hash. It returns (nil, fasle)
// if there is not
func (c *Container) RootByHash(hash RootReference) (r *Root, ok bool) {
	var rp *data.RootPack
	if rp, ok = c.db.GetRoot(cipher.SHA256(hash)); !ok {
		return
	}
	var err error
	if r, err = c.unpackRoot(rp); err != nil {
		panic(err) // critical
	}
	r.attached = true
	return
}

// GC

// DelRootsBefore deletes all roots of a feed before given seq
func (c *Container) DelRootsBefore(pk cipher.PubKey, seq uint64) {
	c.dbmx.Lock()
	defer c.dbmx.Unlock()

	c.db.DelRootsBefore(pk, seq)
}

// RemoveNonFullRoots removes all non-full root objects of a feed from
// database. The method required by a node to clean up during shutdown. If
// there are any non-full roots then a node are not able to fill them next
// time and it's better to remove them to reduce database size. The method
// doesn't remove related objects. Use GC method to do that
func (c *Container) RemoveNonFullRoots(pk cipher.PubKey) {
	c.dbmx.Lock()
	defer c.dbmx.Unlock()

	// TODO (kostyarin): optimisation
	// -------------------------------------------------------------------------
	//       We can't call r.IsFull() inside RangeFeedDelete because of
	//       database locks. So, this approach is slow, because we have to
	//       range a feed twice
	// -------------------------------------------------------------------------

	// build map of non-full roots
	nonFull := make(map[cipher.SHA256]struct{})

	// collect root objects
	c.db.RangeFeed(pk, func(rp *data.RootPack) (_ bool) {
		var r *Root
		var err error
		if r, err = c.unpackRoot(rp); err != nil {
			panic(err) // critical
		}
		r.attached = true
		// But, we can call IsFull from if database locked for reading
		if r.IsFull() {
			return
		}
		nonFull[cipher.SHA256(r.Hash())] = struct{}{}
		return
	})

	// delete
	c.db.RangeFeedDelete(pk, func(rp *data.RootPack) (del bool) {
		_, del = nonFull[rp.Hash]
		return
	})
}

// GC removes all unused objects, including Root objects and Registries.
// If given argument is false, then GC deletes all root objects before
// last full root of a feed. If you want to keep all roots, then
// call the GC with true
func (c *Container) GC(dontRemoveRoots bool) {

	c.gcmx.Lock()
	defer c.gcmx.Unlock()

	// gc lock
	c.dbmx.Lock()
	defer c.dbmx.Unlock()

	gc, sc, rc := c.collect(dontRemoveRoots)

	// delete roots
	for pk, seq := range rc {
		c.db.DelRootsBefore(pk, seq)
	}

	// delete schemas
	for sr, cn := range sc {
		if cn == 0 {
			// delete registry
			delete(c.registries, RegistryReference(sr))
			continue
		}
		gc[sr] = 1 // keep encoded registry in database
	}

	// delte objects
	c.db.RangeDelete(func(key cipher.SHA256) (del bool) {
		return gc[key] == 0
	})
}

// collect garbage (don't remove)
func (c *Container) collect(dontRemoveRoots bool) (gc,
	sc map[cipher.SHA256]uint, rc map[cipher.PubKey]uint64) {

	// objects
	gc = make(map[cipher.SHA256]uint)
	// schemas
	sc = make(map[cipher.SHA256]uint)

	// core
	if c.coreRegistry != nil {
		sc[cipher.SHA256(c.coreRegistry.Reference())] = 1
	}
	// roots
	if !dontRemoveRoots {
		rc = make(map[cipher.PubKey]uint64)
	}

	for _, pk := range c.db.Feeds() {
		c.db.RangeFeedReverse(pk, func(rp *data.RootPack) (stop bool) {
			r, err := c.unpackRoot(rp)
			if err != nil {
				panic(err) // critical
			}
			r.attached = true
			sc[cipher.SHA256(r.reg)] = 1
			// ignore error
			r.RefsFunc(func(r Reference) (skip bool, _ error) {
				if _, skip = gc[cipher.SHA256(r)]; skip {
					return // skip entire branch
				}
				gc[cipher.SHA256(r)] = 1
				return
			})
			if !dontRemoveRoots {
				if r.IsFull() {
					rc[pk] = r.Seq()
					return true // stop
				}
			}
			return // continue
		})
	}
	return
}

// addRoot called by Root after updating to
// store new version of the root in databse
//
// a root can be attached or detached, if root is
// attached then we need to increase seq number
// and shift next/prev/hash references
func (c *Container) addRoot(r *Root) (rp data.RootPack, err error) {
	c.dbmx.Lock()
	defer c.dbmx.Unlock()

	// the Root locked, we can access all its fields
	if r.attached {
		r.seq++
		r.prev = r.hash // must have valid hash
		// encode, sign and update hash of the root
	} else {
		// actual seq and prev, and cleared next
		r.attached = true // make it attached
	}
	rp = r.encode()
	err = c.db.AddRoot(r.pub, &rp)
	return
}

// LockGC locks mutex of GC
func (c *Container) LockGC() {
	c.gcmx.Lock()
}

//UnlockGC unlocks mutex of GC
func (c *Container) UnlockGC() {
	c.gcmx.Unlock()
}
