package skyobject

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject/registry"
)

// A CachePolicy represents cache policy
// that is obvious
type CachePolicy int

const (
	LRU CachePolicy = iota // LRU cache
	LFU                    // LFU cache
)

// String implements fmt.Stringer interface
func (c CachePolicy) String() string {
	switch c {
	case LRU:
		return "LRU"
	case LFU:
		return "LFU"
	}
	return fmt.Sprintf("CachePolicy<%d>", c)
}

type item struct {
	fwant []chan<- []byte // fillers wanters

	// the points is time.Now().Unix() or number of acesses;
	// e.g. the points is LRU or LFU number and depends on
	// the cache policy
	points int

	// not sync with DB
	cc uint32 // changed rc (cached value)

	// sync with DB
	rc  uint32 // references counter
	val []byte // value
}

// isWanted means that this item doesn't exist
func (i *item) isWanted() (yep bool) {
	return len(i.fwant) > 0
}

func (i *item) touch(cp CachePolicy) {

	switch cp {
	case LRU:
		i.points = int(time.Now().Unix()) // last access
	case LFU:
		i.points++ // access
	default:
		panic("cache with undefined CachePolicy:", cp.String())
	}

}

// itemRegistry
type itemRegistry struct {
	r      *registry.Registry // the Registry
	points int                // LRU or LFU points
}

func (i *itemRegistry) touch(cp CachePolicy) {

	switch cp {
	case LRU:
		i.points = int(time.Now().Unix()) // last access
	case LFU:
		i.points++ // access
	default:
		panic("cache with undefined CachePolicy:", cp.String())
	}

}

// A Cache is internal and used by Container.
// The Cache can't be created and used outside
type Cache struct {
	mx sync.Mutex

	db data.CXDS // database

	// use cache or not
	enable bool

	// configs
	maxAmount     int // max amount of items
	maxVolume     int // max volume in bytes of all items
	maxRegistries int // max registries

	maxItemSize int // don't cache items bigger

	policy CachePolicy // policy

	cleanAmount int // clean down to this
	cleanVolume int // clean down to this

	amount int // number of items in DB
	volume int // total volume of items in DB

	c map[cipher.SHA256]*item                // values
	r map[registry.RegistryRef]*itemRegistry // cached registries

	stat *cxdsStat

	closeo sync.Once
}

// conf should be valid, the amount and volume are values of DB
// and used by cxdsStat, the Cache fields amount and volume are
// amount and volume of the Cache (not DB)
func (c *Cache) initialize(db data.CXDS, conf *Config, amount, volume int) {

	c.db = db

	c.maxAmount = conf.CacheMaxAmount
	c.maxVolume = conf.CacheMaxVolume
	c.maxItemSize = conf.CacheMaxItemSize
	c.maxRegistries = conf.CacheRegistries

	c.policy = conf.CachePolicy

	c.cleanAmount = int(float64(c.maxAmount) * (1.0 - conf.CacheCleaning))
	c.cleanVolume = int(float64(c.maxVolume) * (1.0 - conf.CacheCleaning))

	c.amount = 0 // cache amount
	c.volume = 0 // cache volume

	c.c = make(map[cipher.SHA256]*item)
	c.r = make(map[registry.RegistryRef]*itemRegistry)

	c.stat = newCxdsStat(conf.RollAvgSamples, amount, volume)

	c.enable = !(c.maxAmount == 0 || c.maxVolume == 0)

	return
}

func (c *Cache) reset() {
	c.c = nil
	c.r = nil
	c.stat.Close()
	c.stat = nil
}

// call it under lock
func (c *Cache) addRegistryToCache(r *registry.Registry) {

	// if it already exists

	if ir, ok := c.r[r.Reference()]; ok == true {
		ir.touch(c.policy)
		return
	}

	// clean if need

	if len(c.r) == c.maxRegistries {

		var (
			tormr registry.RegistryRef // reference
			tormp int                  // points
		)

		for rr, ir := range c.r {
			if ir.points > tormp {
				tormr = rr
				tormp = ir.points
			}
		}

		delete(c.r, tormr)

	}

	// add

	var ir = &itemRegistry{r: r}

	ir.touch(c.policy)
	c.r[r.Reference()] = ir
}

// AddRegistryToCache adds given registry to Cache
func (c *Cache) AddRegistryToCache(r *registry.Registry) {

	if c.maxRegistries <= 0 {
		return
	}

	c.mx.Lock()
	defer c.mx.Unlock()

	c.addRegistryToCache(r)

}

// Registry returns Registry by reference. The
// Registry looks the Cache first. If it gets Registry
// from DB, then it put the Registry to the Cache
func (c *Cache) Registry(
	rr registry.RegistryRef, // : the reference
) (
	r *registry.Registry, //    : registry
	err error, //               : an error
) {

	c.mx.Lock()
	defer c.mx.Unlock()

	// check out cache first

	if ir, ok := c.r[rr]; ok == true {
		r = ir.r
		ir.touch(c.policy)
		return
	}

	// get from DB and add to cache after

	var val []byte
	if val, _, err = c.get(cipher.SHA256(rr), 0); err != nil {
		return
	}

	if r, err = registry.DecodeRegistry(val); err != nil {
		return
	}

	// TOTH (kostyarin): to add or not to add? That is the fucking question

	if c.maxRegistries >= 0 {
		c.addRegistryToCache(r)
	}

	return

}

// Close cache releasing associated resuorces
// and closing CXDS saving statistics. The Close
// method syncs cached values with saved in DB
func (c *Cache) Close() (err error) {
	c.mx.Lock()
	defer c.mx.Unlock()

	// sync items
	for k, it := range c.c {
		if _, err = c.sync(it); err != nil {
			break // break on first error
		}
	}

	// save stat only if no errors
	if err == nil {
		err = c.db.SetStat(uint32(c.amount), uint32(c.volume))
	}

	if err == nil {
		err = c.db.Close()
	} else {
		c.db.Close() // ignore error
	}

	c.stat.Close() // release associated resources
	return
}

// sync with DB removing item from cache if it is removed from DB
func (c *Cache) sync(key cipher.SHA256, it *item) (removed bool, err error) {

	if it.cc == it.rc {
		return // already synchronized
	}

	var inc = int(cc) - int(rc) // 20 - 19 = 1, 19 - 20 = -1

	if err = c.db.Inc(key, inc); err != nil {
		c.stat.addWritingDBRequest() // just request
		return                       // failure
	}

	it.rc = it.cc // synchronized

	// if the item has been removed from DB
	if it.cc == 0 {
		c.amount--
		c.volume -= len(it.val)
		delete(c.c, key)
		removed = true // has been removed

		c.stat.delFromDB(len(it.val)) // delete
	} else {
		c.stat.addWritingDBRequest() // update
	}

	return
}

// change rc in the cache (cc), removing item if rc turns to be 0
func (c *Cache) incr(
	key cipher.SHA256,
	it *item,
	inc int,
) (
	removed bool,
	err error,
) {

	switch {

	case inc == 0:

		// leave as is

	case inc < 0:

		inc = -inc             // make it positive
		var uinc = uint32(inc) // to uint32

		if uinc < i.cc {
			i.cc -= uinc // reduce
		} else {
			// else ->  reduce to zero (sync to remove)
			if removed, err = c.sync(key, it); err == nil {
				i.cc = 0 // sync is sync
			}
		}

	case inc > 0:

		i.cc += uint32(inc) // increase

	}

	return
}

// clean the cache down, the vol is size of item to insert to the cache
// the items can't be inserted, because the cache is full; we are using
// this vol to clean the cache down
func (c *Cache) cleanDown(vol int) (err error) {

	type rankItem struct {
		key cipher.SHA256
		it  *item
	}

	var rank = make([]*rankItem, 0, len(c.c))

	for k, it := range c.c {

		if it.isWanted() == true {
			continue // skip wanted items
		}

		rank = append(rank, &rankItem{k, it}) // add the item to the rank
	}

	// sort the rank by points

	sort.Slice(rank, func(i, j int) bool {
		rank[i].it.points < rank[j].it.points
	})

	// clean down

	// celan by amount of items only if c.amount == c.maxAmount

	var (
		i       int       // last removed element
		ri      *rankItem // key->item
		removed bool      // removed by sync
	)

	// clean by amount first (if need)
	if c.amount == c.maxAmount {

		for i, ri = range rank {
			if c.amount <= c.cleanAmount { // actually, ==
				break // done
			}

			if removed, err = c.sync(ri.key, ri.it); err != nil {
				return // fail on first error
			} else if removed == false {
				delete(c.c, ri.key)        // delete from cache
				c.amount--                 // cache amount
				c.volume -= len(ri.it.val) // cache volume
			}

			rank[i] = nil // GC for the item
		}

		// and if, after the cleaning, the cache has
		// enouth palce to fit new item, we are done

		if c.volume+vol <= c.maxVolume {
			return // done
		}

		// otherwise, we are cleaning by volume
		// down to the c.cleanVolume (down to
		// c.cleanVolume - vol)

	}

	// clean by volume if need
	for ; i < len(rank); i++ {

		if c.volume+vol <= c.cleanVolume {
			break // done
		}

		ri = rank[i]

		if removed, err = c.sync(ri.key, ri.it); err != nil {
			return // fail on first error
		} else if removed == false {
			delete(c.c, ri.key)
			c.amount--
			c.volume -= len(ri.it.val)
		}

		rank[i] = nil // GC for the item
	}

	return // ok
}

// put item to the cache
func (c *Cache) putItem(key cipher.SHA256, val []byte, rc uint32) (err error) {

	if len(val) > c.maxItemSize {
		return // can't put, the item is too big
	}

	if c.amount+1 >= c.maxAmount || c.volume+len(val) >= c.maxVolume {
		if err = c.cleanDown(len(val)); err != nil { // clean the cache first
			return
		}
	}

	var it = &item{
		cc:  rc, // synchronized
		rc:  rc,
		val: val,
	}

	it.touch(c.policy)

	c.c[key] = it // add to cache
	return

}

// CXDS wrappers

// call it under lock
func (c *Cache) get(
	key cipher.SHA256, // :
	inc int, //           :
) (
	val []byte, //        :
	rc uint32, //         :
	err error, //         :
) {

	var it, ok = c.c[key]

	if c.enable == false {

		// if cache disabled then an item can be wanted,
		// and looking cache for wanted items, we can avoid
		// DB lookup (disck access); here is cache is distabled
		// then c.c map used for wanted items only, and we
		// can avoid it.isWanted() call
		if ok == true {
			return data.ErrNotFound // if it wanted, then it not found
		}

		val, rc, err = c.db.Get(key, inc)
		c.stat.addDBGet(inc, rc, val, err) // stat
		return
	}

	if ok == true {

		// if item is wanted, then the CXDS
		// doesn't contains the item

		if it.isWanted() == true {
			return data.ErrNotFound // not found
		}

		// change cc (cached rc)
		var removed bool
		if removed, err = c.incr(key, it, inc); err != nil {
			return
		}

		if removed == false {
			c.stat.addCacheGet(inc)
			it.touch(c.policy) // touch item
		}

		val, rc = it.val, it.cc
		return
	}

	// not found in the cache, let's look DB
	val, rc, err = c.db.Get(key, inc)
	c.stat.addDBGet(inc, rc, val, err)

	if err != nil {
		return
	}

	// put to cache
	err = c.putItem(key, val, rc)
	return
}

// Get from cache or DB, increasing, leaving as is, or reducing
// references counter. For most reading cases the inc argument should
// be zero. If value doesn't exist the Get returns data.ErrNotFound.
// It's possible to get and remove value using the inc argument. The
// Get returns value and new references counter. The Get can returns
// an error that doen'r relate to given element. Putting element to
// the Cache the Get can clean up cache to free space, syncing
// wiht DB. In this case if the synching fails, the Get can returns
// valid value and rc and some error. Anyway, any DB failure should be
// fatal. E.g. all errors except data.ErrNotFound means that data
// in the DB can be lost
func (c *Cache) Get(
	key cipher.SHA256, // :
	inc int, //           :
) (
	val []byte, //        :
	rc uint32, //         :
	err error, //         :
) {

	c.mx.Lock()
	defer c.mx.Unlock()

	return c.get(key, inc)
}

// never block
func sendWanted(gc chan<- []byte, val []byte) {
	select {
	case gc <- val:
	default:
	}
}

// Set to DB, increasing references counter. The inc argument must
// be 1 or greater, otherwise the Set panics. Owerwriting incareases
// references counter of existing value
func (c *Cache) Set(
	key cipher.SHA256, // : key
	val []byte, //        : value
	inc int, //           : >= 0
) (
	rc uint32, //         : new rc of the value
	err error, //         : an error
) {

	if inc <= 0 {
		panic("invalid inc argument of Set method: " + fmt.Sprint(inc))
	}

	c.mx.Lock()
	defer c.mx.Unlock()

	// check out cache first
	var it, ok = c.c[key]

	if ok == true {

		if it.isWanted() == true {

			// we have to save the item first to guarantee
			// that the element will exist in DB
			if rc, err = c.db.Set(key, val, inc); err != nil {
				c.stat.addWritingDBRequest() // just request
				return                       // an error
			}

			c.stat.addToDB(len(val)) // stat

			// send to wanters
			for _, gc := range it.fwant {
				sendWanted(gc, val)
			}

			delete(c.c, key) // remove from wanted

			if c.enable == false {
				return // done
			}

			// put to cache (replace the wanted)
			err = c.putItem(key, val, rc)
			return

		}

		// not wanted, just chagne the rc
		// (here the Cache is enabled), we
		// don't update DB value

		it.cc += uint32(inc) // increase
		c.stat.addWritingCacheRequest()
		it.touch(c.policy) // touch item

		return

	}

	// not found in the Cache
	if rc, err = c.db.Set(key, val, inc); err != nil || c.enable == false {
		c.stat.addWritingDBRequest() // just request
		return
	}

	// cache enabled and err is nil
	err = c.putItem(key, val, rc)
	return

}

// Inc used to increase or reduce references counter. The Inc with
// zero inc argument can be used to check presence of a value. If
// value doesn't exists the Inc returns data.ErrNotFound. If the
// Inc reduce references counter to zero or less, then value will
// be deleted
func (c *Cache) Inc(
	key cipher.SHA256, // :
	inc int, //           :
) (
	rc uint32, //         :
	err error, //         :
) {

	c.mx.Lock()
	defer c.mx.Unlock()

	// check out cache first
	var it, ok = c.c[key]

	if ok == true {
		// found

		if it.isWanted() == true {
			err = data.ErrNotFound // wanted item
			return
		}

		// here the c.enable is true
		var removed bool
		if removed, err = c.incr(key, it, inc); err != nil {
			return
		}

		if removed == false {
			c.stat.addCacheGet(inc) // Inc not Get, but who fucking cares
			it.touch(c.policy)      // touch item
		}

		val, rc = it.val, it.cc
		return

	}

	// not found in cache

	// use Get instead of the Inc, to get val;
	// the val required for stat to track DB
	// volume
	var val []byte
	val, rc, err = c.db.Get(key, inc)
	c.stat.addDBGet(inc, rc, val, err)

	if err != nil {
		return
	}

	// put to cache
	err = c.putItem(key, val, rc)

	return

}

// Want is service method. It used by the node package for filling.
// If wanted value exists in DB, it will eb sent through given
// channel. Otherwise, it will be sent when it comes
func (c *Cache) Want(
	key cipher.SHA256, // : requested value
	gc chan<- []byte, //  : channel to receive value
) (
	err error, //         : the err never be data.ErrNotFound
) {

	c.mx.Lock()
	defer c.mx.Unlock()

	var it, ok = c.c[key]

	if ok == true {

		if it.isWanted() == true {
			it.fwant = append(it.fwant, gc) // add to the list
			return
		}

		// exist (not wanted)

		sendWanted(gc, val) // never block

		c.stat.addReadingCacheRequest() // got from cache
		it.touch(c.policy)              // touch the item
		return

	}

	// ok == false (doen't exist in the cache)

	var (
		val []byte
		rc  uint32
	)

	val, rc, err = c.db.Get(key, 0)
	c.stat.addDBGet(inc, rc, val, err)

	if err == data.ErrNotFound {
		c.c[key] = &item{fwant: []chan<- []byte{gc}} // add to wanted
		return nil                                   // no errors
	} else if err != nil {
		return // database failure
	}

	// found
	sendWanted(gc, val)

	err = c.putItem(key, val, rc)
	return
}
