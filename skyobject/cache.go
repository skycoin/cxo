package skyobject

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
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

// a rootKey used to select a Root object
type rootKey struct {
	pk    cipher.PubKey
	nonce uint64
	seq   uint64
}

// TODO
type cache struct {
	mx sync.Mutex

	db data.CXDS // database

	// use cache or not
	enable bool

	// configs
	maxAmount int // max amount of items
	maxVolume int // max volume in bytes of all items

	maxItemSize int // don't cache items bigger

	policy CachePolicy // policy

	cleanAmount int // clean down to this
	cleanVolume int // clean down to this

	amount int // number of items in DB
	volume int // total volume of items in DB

	c map[cipher.SHA256]*item // values
	r map[rootKey]int         // root objects (fill, preview, hold)

	stat *cxdsStat

	closeo sync.Once
}

// conf should be valid
func newCache(db data.CXDS, conf *Config, amount, volume int) (c *cache) {
	c = new(cache)

	c.db = db

	c.maxAmount = conf.CacheMaxAmount
	c.maxVolume = conf.CacheMaxVolume
	c.maxItemSize = conf.CacheMaxItemSize

	c.policy = conf.CachePolicy

	c.cleanAmount = int(float64(c.maxAmount) * (1.0 - conf.CacheCleaning))
	c.cleanVolume = int(float64(c.maxVolume) * (1.0 - conf.CacheCleaning))

	c.amount = amount
	c.volume = volume

	c.c = make(map[cipher.SHA256]*item)
	c.r = make(map[rootKey]int)

	c.stat = newCxdsStat(conf.RollAvgSamples)

	c.enable = !(c.maxAmount == 0 || c.maxVolume == 0)

	return
}

// Close cache releasing associated resuorces
// and closing CXDS saving statistics
func (c *cache) Close() (err error) {
	c.mx.Lock()
	defer c.mx.Unlock()

	if err = c.db.SetStat(uint32(c.amount), uint32(c.volume)); err != nil {
		c.db.Close() // ignore error
	} else {
		err = c.db.Close()
	}

	c.stat.Close()
	return
}

// clean the cache down, the vol is size of item to insert to the cache
// the items can't be inserted, because the cache is full; we are using
// this vol to clean the cache down
func (c *cache) cleanDown(vol int) {

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
		i  int // last removed element
		ri *rankItem
	)

	// clean by amount first (if need)
	if c.amount == c.maxAmount {

		for i, ri = range rank {
			if c.amount <= c.cleanAmount { // actually, ==
				break // done
			}
			delete(c.c, ri.key)
			c.amount--
			c.volume -= len(ri.it.val)
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

		delete(c.c, ri.key)
		c.amount--
		c.volume -= len(ri.it.val)
		rank[i] = nil // GC for the item

	}

	// ok

}

// put item to the cache
func (c *cache) putItem(key cipher.SHA256, val []byte, rc uint32) {

	if len(val) > c.maxItemSize {
		return // can't put, the item is too big
	}

	if c.amount+1 >= c.maxAmount || c.volume+len(val) >= c.maxVolume {
		c.cleanDown(len(val)) // clean the cache first
	}

	var it = &item{
		rc:  rc,
		val: val,
	}

	it.touch(c.policy)

	c.c[key] = it
	return

}

// CXDS wrapper

// Get from CXDS or the cache
func (c *cache) Get(
	key cipher.SHA256, // :
	inc int, //           :
) (
	val []byte, //        :
	rc uint32, //         :
	err error, //         :
) {

	if c.enable == false {
		return c.db.Get(key, inc) // cache disabled
	}

	c.mx.Lock()
	defer c.mx.Unlock()

	var it, ok = c.c[key]

	if ok {

		// if item is wanted, then the CXDS
		// doesn't contains the item

		if it.isWanted() == true {
			return // not found
		}

		// change rc in db (sync)
		if inc != 0 {
			if _, rc, err = c.db.Get(key, inc); err != nil {
				return // an error
			}
			it.rc = rc // update cached value
		}

		it.touch(c.policy)      // touch
		val, rc = it.val, it.rc // found in the cache
		return
	}

	// not found in the cache, let's look DB
	if val, rc, err = c.db.Get(key, inc); err != nil {
		return
	}

	// put to cache
	c.putItem(key, val, rc)
	return
}

// never block
func sendWanted(gc chan<- []byte, val []byte) {
	select {
	case gc <- val:
	default:
	}
}

// Set to CXDS. Don't put to cache. But
// touch if it already cached
func (c *cache) Set(
	key cipher.SHA256, // : key
	val []byte, //        : value
	inc int, //           : >= 0
) (
	rc uint32, //         : new rc of the value
	err error, //         : an error
) {

	c.mx.Lock()
	defer c.mx.Unlock()

	// TODO (kostyarin): avoid write to DB if value alreaady exist, or
	//                   reduce number of writes in this case

	// the item can be wanted, but we have to save it first anyway
	if rc, err = c.db.Set(key, val, inc); err != nil {
		return
	}

	// check out wanted items first, even if the cache is disabled
	// it used for the wanted items
	var it, ok = c.c[key]

	// does not exist
	if ok == false {
		return // don't put to cache
	}

	// ok == true (exist)

	// wanted item
	if it.isWanted() == true {
		for _, gc := range it.fwant {
			sendWanted(gc, val)
		}

		// don't put to cache

		delete(c.c, key)
		return
	}

	// item already in the cache, let's touch it
	it.rc = rc
	it.touch(c.policy)

	return

}

// Inc increase or reduce references counter.
// Touch appropriate item, only if the item
// already in cache
func (c *cache) Inc(
	key cipher.SHA256, // :
	inc int, //           :
) (
	rc uint32, //         :
	err error, //         :
) {

	// avoid unneccessary lock
	if c.enable == false {
		return c.db.Inc(key, inc)
	}

	c.mx.Lock()
	defer c.mx.Unlock()

	// undex lock
	if rc, err = c.db.Inc(key, inc); err != nil {
		return
	}

	var it, ok = c.c[key]

	if ok == false {
		return // doesn't exist in cache
	}

	// ok == true (exist)

	if rc == 0 {
		delete(c.c, key) // removed from DB
		return
	}

	it.rc = rc         // update
	it.touch(c.policy) // and touch

	return

}

// Want requests value from the cache or from DB and if it
// doesn't exist, it makes it wanted. And next Set with given
// key sends value to given channel. The cahnnel should reads
// (e.g. it's better to use buffered channel to be able to
// call the Want from any goroutine). The Want used by the
// node package for filling and for preview. If the value
// exists in the cache or in the DB, the Want sends it through
// the channel
func (c *cache) Want(
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
		it.touch(c.policy)  // touch the item
		return

	}

	// ok == false (doen't exist in the cache)

	var (
		val []byte
		rc  uint32
	)

	if val, rc, err = c.db.Get(key, 0); err != nil {
		if err == data.ErrNotFound {
			c.c[key] = &item{fwant: []chan<- []byte{gc}} // add to wanted
			return nil                                   // no errors
		}
		return // database error
	}

	// so, since the Want is the same as Get, then
	// we put the item to the cache; and since
	// the item doesn't exist in the cache, we are
	// creating it here
	c.putItem(key, val, rc)

	return
}

// HoldRoot used to avoid removing of a
// Root object with all related objects.
// A Root can be used for end-user needs
// and also to share it with other nodes.
// Thus, in some cases a Root can't be
// removed, and should be held
func (c *cache) HoldRoot(
	pk cipher.PubKey, // :
	nonce uint64, //     :
	seq uint64, //       :
) {

	var rk = rootKey{pk, nonce, seq}

	c.mx.Lock()
	defer c.mx.Unlock()

	c.r[rk]++
}

// UnholdRoot used to unhold
// previously held Root object
func (c *cache) UnholdRoot(
	pk cipher.PubKey, // :
	nonce uint64, //     :
	seq uint64, //       :
) {

	var rk = rootKey{pk, nonce, seq}

	c.mx.Lock()
	defer c.mx.Unlock()

	var hc, ok = c.r[rk]

	if ok == false {
		return
	}

	if hc <= 1 {
		delete(c.r, rk)
		return
	}

	c.r[rk]--
}

// IsRootHeld returns true if Root with
// given feed, head, seq is held
func (c *cache) IsRootHeld(
	pk cipher.PubKey, // :
	nonce uint64, //     :
	seq uint64, //       :
) (
	held bool, //        :
) {

	var rk = rootKey{pk, nonce, seq}

	c.mx.Lock()
	defer c.mx.Unlock()

	_, held = c.r[rk]

	return
}

/*

	// Get from cache or DB, increasing, leaving as is, or reducing
	// references counter. For most reading cases the inc argument should
	// be zero. If value doesn't exist the Get returns data.ErrNotFound
	Get(key cipher.SHA256, inc int) (val []byte, rc uint32, err error)

	// Set to DB, increasing references counter. The inc argument must
	// be 1 or greater, otherwise the Set panics. Owerwriting incareases
	// references counter of existing value
	Set(key cipher.SHA256, val []byte, inc int) (rc uint32, err error)

	// Inc used to increase or reduce references counter. The Inc with
	// zero inc argument can be used to check presence of a value. If
	// value doesn't exists the Inc returns data.ErrNotFound. If the
	// Inc reduce references counter to zero or less, then value will
	// be deleted
	Inc(key cipher.SHA256, inc int) (rc uint32, err error)

	// Want is service method. It used by the node package for filling.
	// If wanted value exists in DB, it will eb sent through given
	// channel. Otherwise, it will be sent when it comes
	Want(key cipher.SHA256, gc chan<- []byte) (err error)

*/

/*

	AddFeed(pk cipher.PubKey) (err error)
	AddHead(pk cipher.PubKey, nonce uint64) (err error)

	DelFeed(pk cipher.PubKey) (err error)
	DelHead(pk cipher.PubKey, nonce uint64) (err error)
	DelRoot(pk cipher.PubKey, nonce, seq uint64) (err error)

*/
