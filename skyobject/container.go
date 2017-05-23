package skyobject

import (
	"errors"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

var (
	ErrNoCoreRegistry = errors.New(
		"missing registry, Container created without registry")
	// ErrAlreadyHaveThisRoot occurs when Container already have
	// root with the same Seq number
	ErrAlreadyHaveThisRoot = errors.New("already have the root")
)

// A Container represents ...
type Container struct {
	sync.RWMutex

	db data.DB // databse

	coreRegistry *Registry // registry witch which the container was created

	registries map[RegistryReference]*Registry
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

// registry

// AddRegistry to the Container. A registry can be removed
// by GC() or RegistiesGC() if no root refers it
func (c *Container) AddRegistry(reg *Registry) {
	c.Lock()
	defer c.Unlock()
	// call Done
	reg.Done()
	c.db.Set(cipher.SHA256(reg.Reference()), reg.Encode()) // store
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
	// c.coreRegistry is read-only and we don't need to lock/inlock
	if c.coreRegistry != nil && rr == c.coreRegistry.Reference() {
		reg = c.coreRegistry
		return
	}
	c.RLock()
	defer c.RUnlock()
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
	c.RLock()
	defer c.RUnlock()
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
				} else {
					have, stop = true, true // already have
					return
				}
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
	c.RLock()
	defer c.RUnlock()
	if len(c.registries) == 0 {
		return // nil
	}
	rrs = make([]RegistryReference, 0, len(c.registries))
	for rr := range c.registries {
		rrs = append(rrs, rr)
	}
	return
}

// database

// DB of the Container
func (c *Container) DB() data.DB {
	return c.db
}

// Get object by Reference
func (c *Container) Get(ref Reference) (data []byte, ok bool) {
	data, ok = c.db.Get(cipher.SHA256(ref))
	return
}

// Set is short hand for c.DB().Set(cipher.SHA256(ref), data)
func (c *Container) Set(ref Reference, p []byte) {
	c.db.Set(cipher.SHA256(ref), p)
}

// save objects

func (c *Container) save(i interface{}) (ref Reference) {
	data := encoder.Serialize(i)
	hash := cipher.SumSHA256(data)
	c.db.Set(hash, data)
	ref = Reference(hash)
	return
}

func (c *Container) saveArray(i ...interface{}) (refs References) {
	refs = make(References, 0, len(i))
	for _, e := range i {
		refs = append(refs, c.save(e))
	}
	return
}

// roots

// NewRoot creates new root associated with registry provided to
// NewContainer or NewContainerDB. The Root it returns is editable and
// detached. Fields Seq and Prev are actual
func (c *Container) NewRoot(pk cipher.PubKey, sk cipher.SecKey) (r *Root,
	err error) {

	c.Lock()
	defer c.Unlock()

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

	c.Lock()
	defer c.Unlock()

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
	c.RLock()
	defer c.RUnlock()
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
	c.RLock()
	defer c.RUnlock()
	lastFull = c.lastFullRoot(pk)
	return
}

func (c *Container) lastFullRoot(pk cipher.PubKey) (lastFull *Root) {
	c.db.RangeFeedReverse(pk, func(rp *data.RootPack) (stop bool) {
		var err error
		var r *Root
		if r, err = c.unpackRoot(rp); err != nil {
			panic(err) // ccritical
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
// for every Root of the feed starting from older
func (c *Container) WantFeed(pk cipher.PubKey, wf WantFunc) (err error) {
	c.RLock()
	defer c.RUnlock()
	c.db.RangeFeed(pk, func(rp *data.RootPack) (stop bool) {
		var err error
		var r *Root
		if r, err = c.unpackRoot(rp); err != nil {
			panic(err) // ccritical
		}
		r.attached = true
		if err = r.WantFunc(wf); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return true // stop
		}
		return // false (continue)
	})
	return
}

// GotFeed calls (*Root).GotFunc with given GotFunc
// for every Root of the feed starting from older
func (c *Container) GotFeed(pk cipher.PubKey, gf GotFunc) (err error) {
	c.RLock()
	defer c.RUnlock()
	c.db.RangeFeed(pk, func(rp *data.RootPack) (stop bool) {
		var err error
		var r *Root
		if r, err = c.unpackRoot(rp); err != nil {
			panic(err) // ccritical
		}
		r.attached = true
		if err = r.GotFunc(gf); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return true // stop
		}
		return // false (continue)
	})
	return
}

// DelFeed deletes all root object of given feed. The
// method doesn't perform GC
func (c *Container) DelFeed(pk cipher.PubKey) {
	c.Lock()
	defer c.Unlock() // TODO: continer locks ?
	c.db.DelFeed(pk)
}

// GC

// DelRootsBefore deletes all roots of a feed before given seq
func (c *Container) DelRootsBefore(pk cipher.PubKey, seq uint64) {
	c.Lock()
	defer c.Unlock()
	c.db.DelRootsBefore(pk, seq)
}

// GC removes all unused objects, including Root objects and Registries.
// If given argument is false, then GC deletes all root objects before
// last full root of a feed. If you want to keep all roots, then
// call the GC with true
func (c *Container) GC(dontRemoveRoots bool) {
	c.Lock()
	defer c.Unlock()
	// objects and schemas
	gc := make(map[cipher.SHA256]uint)
	// init
	c.db.Range(func(key cipher.SHA256, _ []byte) (_ bool) {
		gc[key] = 0
		return
	})
	// roots
	var rc map[cipher.PubKey]uint64
	if !dontRemoveRoots {
		rc = make(map[cipher.PubKey]uint64)
	}
	for _, pk := range c.db.Feeds() {
		c.db.RangeFeedReverse(pk, func(rp *data.RootPack) (stop bool) {
			r, err := c.unpackRoot(rp)
			if err != nil {
				panic(err) // critical
			}
			gc[cipher.SHA256(r.reg)] = gc[cipher.SHA256(r.reg)] + 1
			// ignore error
			r.GotFunc(func(r Reference) (_ error) {
				gc[cipher.SHA256(r)] = gc[cipher.SHA256(r)] + 1
				return // nil
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
	// delete roots
	for pk, seq := range rc {
		c.db.DelRootsBefore(pk, seq)
	}
	// delte objects
	for key, cn := range gc {
		if cn == 0 {
			c.db.Del(key)
		}
	}
}

// addRoot called by Root after updating to
// store new version of the root in databse
//
// a root can be attached or detached, if root is
// attached then we need to increase seq number
// and shift next/prev/hash references
func (c *Container) addRoot(r *Root) (rp data.RootPack, err error) {
	c.Lock()
	defer c.Unlock()

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
