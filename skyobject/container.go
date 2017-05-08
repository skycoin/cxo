package skyobject

import (
	"fmt"
	"sort"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
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

// save objects

func (c *Container) save(i interface{}) Reference {
	return Reference(c.db.AddAutoKey(encoder.Serialize(i)))
}

func (c *Container) Save(i interface{}) Reference {
	c.RLock()
	defer c.RUnlock() // locks required for GC
	return c.save(i)
}

func (c *Container) SaveArray(i ...interface{}) (refs References) {
	c.RLock()
	defer c.RUnlock() // locks required for GC
	refs = make(References, 0, len(i))
	for _, e := range i {
		refs = append(refs, c.save(e))
	}
	return
}

func (c *Container) Dynamic(i interface{}) (dr Dynamic) {
	if c.coreRegistry == nil {
		panic("unable to create Dynamic, Container created without registry")
	}
	c.RLock()
	defer c.RUnlock() // locks required for GC
	s, err := c.coreRegistry.SchemaByInterface(i)
	if err != nil {
		panic(err)
	}
	dr.Schema = s.Reference()
	dr.Object = c.save(i)
	return
}

// roots

// LastRoot returns latest root object of the feed (pk).
// It can return nil
func (c *Container) LastRoot(pk cipher.PubKey) *Root {
	c.RLock()
	defer c.RUnlock()
	if rs := c.roots[pk]; rs != nil {
		return rs.latest()
	}
	return nil
}

// LastFullRoot returns latest root object of the feed (pk) that is full.
// It can return nil
func (c *Container) LastFullRoot(pk cipher.PubKey) *Root {
	c.RLock()
	defer c.RUnlock()
	if rs := c.roots[pk]; rs != nil {
		return rs.latestFull()
	}
	return nil
}

func (c *Container) RootBySeq(pk cipher.PubKey, seq uint64) *Root {
	c.RLock()
	defer c.RUnlock()
	if rs := c.roots[pk]; rs != nil {
		return rs.bySeq(seq)
	}
	return nil
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
		for _, r := range *rs {
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
		for _, r := range *rs {
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

func (c *Container) addRoot(r *Root) {
	c.Lock()
	defer c.Unlock()

	var rs *roots
	if rs = c.roots[r.Pub()]; rs == nil {
		rsd := make(roots, 0)
		rs = &rsd
		c.roots[r.Pub()] = rs
	}
	rs.add(r)
}

// roots is list of root object of a feed sorted by Seq number
type roots []*Root

// sorting
func (r roots) sort()              { sort.Sort(r) }
func (r roots) Len() int           { return len(r) }
func (r roots) Less(i, j int) bool { return r[i].Seq() < r[j].Seq() }
func (r roots) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

func (r *roots) add(t *Root) {
	*r = append(*r, t)
	r.sort()
}

func (r roots) latest() (t *Root) {
	if len(r) > 0 {
		t = r[len(r)-1]
	}
	return
}

func (r roots) latestFull() *Root {
	for i := len(r); i >= 0; i-- { // from tail
		if x := r[i]; x.IsFull() {
			return x
		}
	}
	return nil
}

func (r roots) bySeq(seq uint64) *Root {
	// TODO: make it faster using sort.Search
	for _, x := range r {
		if x.Seq() == seq {
			return x
		}
	}
	return nil // not found
}

func (r *roots) gc() {
	for i := len(*r); i >= 0; i-- { // from tail
		if x := (*r)[i]; x.IsFull() {
			if i > 0 { // avoid recreating slice
				(*r) = (*r)[i:]
			}
			return
		}
	}
}
