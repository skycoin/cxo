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

// An item is cache item that can be
//
//     1. db item as is (mirror)
//     2. fillers item (not in DB, but going to be)
//     3. preview item (not in DB)
//
// Fillers and previewers keeps its rc of an item
// outside the cache. Calling Set method after.
//
// Also the item can be wanted. E.g. if DB doesn't
// have an item its fwant field contains channels
// to get the item when it will be received. In this
// case item has only this fwant cahnnels, other
// fields has no meaning
//
//
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

	policy CachePolicy // policy

	cleanAmount int // clean down to this
	cleanVolume int // clean down to this

	amount int // amount of items in the cache (except wanted)
	volume int // volume of all items of the cache

	c map[cipher.SHA256]*item    // values
	r map[rootKey]*registry.Root // root objects (fill, preview, hold)
}

// clean the cache down, the vol is size of item to insert to the cache
// the items can't be inserted, because the caceh is full; we are using
// this vol to clean the caceh down
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

	for _, ri := range rank {
		//
	}

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

	c.mx.Lock()
	defer c.mx.Unlock()

	var it, ok = c.c[key]

	if ok {

		if it.isWanted() == true {
			return // not found
		}

		it.inc(inc)        // change the rc (lrc)
		it.touch(c.policy) // touch

		val, rc = it.val, it.rc // found in the cache
		return
	}

	// not found in the cache, let's look DB
	if val, rc, err = c.db.Get(key, inc); err != nil {
		return
	}

	// let's create item

	return
}

// Set to the cache
func (c *cache) Set(
	key cipher.SHA256, // :
	val []byte, //        :
	inc int, //           :
) (
	rc uint32, //         :
	err error, //         :
) {

	c.mx.Lock()
	defer c.mx.Unlock()

	//

	return

}

func (c *cache) Inc(
	key cipher.SHA256, // :
	inc int, //           :
) (
	rc uint32, //         :
	err error, //         :
) {

	c.mx.Lock()
	defer c.mx.Unlock()

	//

	return

}

// filling

// get and mark as short lived
func (c *cache) getShort(key cipher.SHA256) (val []byte, err error) {
	//
}

// del short lived item
func (c *cache) delShort(key cipher.SHA256) {
	//
}

// preview

func (c *cache) setPreview(key cipher.SHA256) {
	//
}

func (c *cache) delPreview(key cipher.SHA256) {
	//
}

// fill
