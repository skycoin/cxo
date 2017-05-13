package skyobject

import (
	"errors"
	"fmt"
	"sort"
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

	db *data.DB // databse

	coreRegistry *Registry // registry witch which the container was created

	registries map[RegistryReference]*Registry
	roots      map[cipher.PubKey]*roots // root objects (pointer to slice)
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
	c.roots = make(map[cipher.PubKey]*roots)
	return
}

// registry

// AddRegistry to the Container. A registry can be removed
// by GC() or RegistiesGC() if no root refers it
func (c *Container) AddRegistry(r *Registry) {
	c.Lock()
	defer c.Unlock()
	// call Done
	r.Done()
	// don't replace
	if _, ok := c.registries[r.Reference()]; !ok {
		c.registries[r.Reference()] = r
	}
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

// WantRegistry reports true if given registry
// wanted by the Container
func (c *Container) WantRegistry(rr RegistryReference) bool {
	c.RLock()
	defer c.RUnlock()
	for _, rs := range c.roots {
		for _, r := range rs.store {
			if rr == r.RegistryReference() {
				if _, ok := c.registries[rr]; !ok {
					return true // want
				} else {
					return false // already have
				}
			}
		}
	}
	return false // don't want
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
func (c *Container) DB() *data.DB {
	return c.db
}

// Get object by Reference
func (c *Container) Get(ref Reference) (data []byte, ok bool) {
	data, ok = c.db.Get(cipher.SHA256(ref))
	return
}

// Set is shotr hand for c.DB().Det(cipher.SHA256(ref), data)
func (c *Container) Set(ref Reference, p []byte) {
	c.db.Set(cipher.SHA256(ref), p)
}

// save objects

func (c *Container) save(i interface{}) Reference {
	return Reference(c.db.AddAutoKey(encoder.Serialize(i)))
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
// NewContainer or NewContainerDB
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
// The method all create root object associated with registry
// that the container hasn't
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

func (c *Container) newRoot(pk cipher.PubKey, sk cipher.SecKey) (r *Root,
	err error) {

	if err = pk.Verify(); err != nil {
		return
	}
	if err = sk.Verify(); err != nil {
		return
	}
	//
	// TODO: (high priority) solve timestamp-seq conflicts priority
	//
	var seq uint64 = 0
	// checking for existing root objects
	if r = c.lastRoot(pk); r != nil {
		seq = r.Seq()
	}
	// create
	r = new(Root)

	r.seq = seq
	r.pub = pk
	r.sec = sk
	r.cnt = c

	return
}

// AddRootPack used to add a received root object to the
// Container. It returns an error if given data can't be decoded
// or signature is wrong
func (c *Container) AddRootPack(rp RootPack) (r *Root, err error) {

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
	r.sig = rp.Sig
	r.cnt = c

	err = cipher.VerifySignature(r.pub, rp.Sig, cipher.SumSHA256(rp.Root))
	if err != nil {
		r = nil
		return
	}

	if err = c.addRoot(r); err != nil {
		r = nil
	}
	return
}

// LastRoot returns latest root object of the feed (pk).
// It can return nil. It can return received root object
// that doesn't contain secret key
func (c *Container) LastRoot(pk cipher.PubKey) *Root {
	c.RLock()
	defer c.RUnlock()
	return c.lastRoot(pk)
}

func (c *Container) lastRoot(pk cipher.PubKey) *Root {
	if rs := c.roots[pk]; rs != nil {
		return rs.latest()
	}
	return nil
}

// LastFullRoot returns latest root object of the feed (pk) that is full.
// It can return nil. It can return received root object that doesn't
// contain secret key
func (c *Container) LastFullRoot(pk cipher.PubKey) *Root {
	c.RLock()
	defer c.RUnlock()
	if rs := c.roots[pk]; rs != nil {
		return rs.latestFull()
	}
	return nil
}

// depricated, should be replaced
func (c *Container) RootBySeq(pk cipher.PubKey, seq uint64) *Root {
	c.RLock()
	defer c.RUnlock()
	if rs := c.roots[pk]; rs != nil {
		return rs.bySeq(seq)
	}
	return nil
}

// Feeds returns public keys of feeds
// have at least one Root object
func (c *Container) Feeds() (feeds []cipher.PubKey) {
	c.RLock()
	defer c.RUnlock()

	if len(c.roots) == 0 {
		return // nil
	}
	feeds = make([]cipher.PubKey, 0, len(c.roots))
	for f, rs := range c.roots {
		if len(rs.store) == 0 { // rs must not be nil
			continue
		}
		feeds = append(feeds, f)
	}
	return
}

// WantFeed calls (*Root).WantFunc with given WantFunc
// for every Root of the feed starting from older
func (c *Container) WantFeed(pk cipher.PubKey, wf WantFunc) (err error) {
	c.RLock()
	defer c.RUnlock()
	rs := c.roots[pk]
	for _, r := range rs.store {
		if err = r.WantFunc(wf); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return
		}
	}
	return // nil
}

// GotFeed calls (*Root).GotFunc with given GotFunc
// for every Root of the feed starting from older
func (c *Container) GotFeed(pk cipher.PubKey, gf GotFunc) (err error) {
	c.RLock()
	defer c.RUnlock()
	rs := c.roots[pk]
	if rs == nil {
		return
	}
	for _, r := range rs.store {
		if err = r.GotFunc(gf); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return
		}
	}
	return // nil
}

// DelFeed deletes all root object of given feed. The
// method doesn't perform GC
func (c *Container) DelFeed(pk cipher.PubKey) {
	c.Lock()
	defer c.Unlock()
	delete(c.roots, pk)
}

// GC

// GC removes all unused objects, including Root objects and Registries
func (c *Container) GC() {
	c.Lock()
	defer c.Unlock()
	c.rootsGC()
	c.regsitryGC()
	c.objectsGC()
}

// RootsGC removes all root objects up to
// last full Root object the feed has got.
// If no full objects of a feed found then
// no Root objects removed from the feed
func (c *Container) RootsGC() {
	c.Lock()
	defer c.Unlock()
	c.rootsGC()
}

// RegsitryGC removes all unused registries
func (c *Container) RegsitryGC() {
	c.Lock()
	defer c.Unlock()
	c.regsitryGC()
}

// ObjectsGC removes all unused objects from database
func (c *Container) ObjectsGC() {
	c.Lock()
	defer c.Unlock()
	c.objectsGC()
}

// internal

func (c *Container) objectsGC() {
	gc := make(map[Reference]int)
	// fill
	c.db.Range(func(ok cipher.SHA256) {
		gc[Reference(ok)] = 0
	})
	// calculate references
	for _, rs := range c.roots {
		if rs == nil {
			continue
		}
		for _, r := range rs.store {
			r.GotFunc(func(r Reference) (_ error) {
				gc[r] = gc[r] + 1
				return
			})
		}
	}
	// remove unused objects
	for ref, i := range gc {
		if i != 0 {
			continue
		}
		c.db.Del(cipher.SHA256(ref))
	}
}

func (c *Container) rootsGC() {
	for _, rs := range c.roots {
		if rs == nil {
			continue
		}
		rs.gc()
	}
}

func (c *Container) regsitryGC() {
	gc := make(map[RegistryReference]int)
	// calculate
	for _, rs := range c.roots {
		if rs == nil {
			continue
		}
		for _, r := range rs.store {
			rr := r.RegistryReference()
			gc[rr] = gc[rr] + 1
		}
	}
	// remove
	for rr, i := range gc {
		if i != 0 {
			continue
		}
		delete(c.registries, rr)
	}
}

func (c *Container) addRoot(r *Root) (err error) {
	c.Lock()
	defer c.Unlock()

	var rs *roots
	if rs = c.roots[r.Pub()]; rs == nil {
		rs = new(roots)
		c.roots[r.Pub()] = rs
	}
	err = rs.add(r) // make a shadow copy
	return
}

// roots is list of root object of a feed sorted by Seq number
type roots struct {
	store []*Root // shadow copies
}

// sorting
func (r *roots) sort() {
	sort.Sort(r)
}

func (r *roots) Len() int {
	return len(r.store)
}

func (r *roots) Less(i, j int) bool {
	return r.store[i].Seq() < r.store[j].Seq()
}

func (r *roots) Swap(i, j int) {
	r.store[i], r.store[j] = r.store[j], r.store[i]
}

func (r *roots) add(t *Root) error {
	// TODO: reimplement using sort.Search to be faster
	for _, e := range r.store {
		// if i == 0 {
		// 	if t.Seq() < e.Seq() {
		// 		// older then first (fuck it)
		// 		return errors.New("too old") // TODO
		// 	}
		// }
		if t.Seq() == e.Seq() {
			// already have a root with the same seq (fuck it)
			return ErrAlreadyHaveThisRoot
		}
	}
	t = t.dup() // make a copy
	r.store = append(r.store, t)
	r.sort()
	return nil
}

func (r *roots) latest() (t *Root) {
	if len(r.store) > 0 {
		t = r.store[len(r.store)-1]
	}
	return
}

func (r *roots) latestFull() *Root {
	for i := len(r.store) - 1; i >= 0; i-- { // from tail
		if x := r.store[i]; x.IsFull() {
			return x
		}
	}
	return nil
}

// depricated
func (r *roots) bySeq(seq uint64) *Root {
	i := sort.Search(len(r.store), func(i int) bool {
		return r.store[i].Seq() >= seq
	})
	if i < len(r.store) && r.store[i].Seq() == seq {
		return r.store[i]
	}
	return nil // not found
}

func (r *roots) gc() {
	for i := len(r.store); i >= 0; i-- { // from tail
		if x := (r.store)[i]; x.IsFull() {
			if i > 0 { // avoid recreating slice
				r.store = r.store[i:]
			}
			return
		}
	}
}
