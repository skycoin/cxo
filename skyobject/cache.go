package skyobject

import (
	"fmt"
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
	preview int // previews (not sync with DB)
	fill    int // fillers  (not sync with DB)
	short   int // short lived item

	fwant []chan<- []byte // fillers wanters

	// the points is time.Now().Unix() or number of acesses;
	// e.g. the points is LRU or LFU number and depends on
	// the cache policy
	points int

	rc  uint32 // references counter (sync with DB)
	val []byte // value
}

type rootKey struct {
	pk    cipher.PubKey
	nonce uint64
	seq   uint64
}

// A cache represents cache for CXO. The cache wrapps
// data.CXDS and, it also used to hold Root objects
// to preven removing them. The cache used for
//
//     - increase reading access
//     - feed preview
//     - filling new root objects
//     - hold root objects (to send or use)
//     - keep short lived items
//
// See, Config to look at configs of the cache.
//
// The cache is thread safe, and the cache is part of
// skyobject.Container and it can't be created and used
// outside
//
// There is ablility to keep short lived items (for
// filling)
type cache struct {
	mx sync.RWMutex

	db data.DB // database

	// use cache or not
	enable bool

	// configs
	maxItems int // max items
	maxSize  int // max size in bytes

	policy CachePolicy // policy

	cleanItems int // clean down to this
	cleanSize  int // clean down to this

	c map[cipher.SHA256]*item    // values
	r map[rootKey]*registry.Root // root objects (fill, preview, hold)
}

// CXDS wrapper

func (c *cache) touchItem(it *item) {

	switch c.policy {
	case LRU:
		it.points = int(time.Now().Unix())
	case LFU:
		it.points++
	default:
		panic("cache with undefined CachePolicy:", c.policy.String())
	}

}

// Get from CXDS or the cache
func (c *cache) Get(
	key cipher.SHA256, // :
	inc int, //           :
) (
	val []byte, //        :
	rc uint32, //         :
	err error, //         :
) {

	c.mx.RLock()
	defer c.mx.RUnlock()

	var it, ok = c.c[key]

	if ok {
		val, rc = it.val, it.rc
		return
	}

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
