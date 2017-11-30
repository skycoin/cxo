package skyobject

import (
	"sync"
	"time"

	"github.com/skycoin/cxo/skyobject/statutil"
)

type cxdsStat struct {
	drps  *statutil.Float // reads per second (DB)
	reads int             // reads for current second

	dwps   *statutil.Float // writes per second (DB)
	writes int             // writes for current second

	crps   *statutil.Float // reads per second (cache)
	creads int             // cache reads for current second

	cwps    *statutil.Float // writes per second (cache)
	cwrites int             // cache writes for current second

	amount int // amount of items in DB
	volume int // volume of items in DB

	mx     sync.Mutex
	quit   chan struct{}
	closeo sync.Once
}

func newCxdsStat(rollAvgSamples int, amount, volume uint32) (c *cxdsStat) {

	c = new(cxdsStat)

	c.drps = statutil.NewFloat(rollAvgSamples)
	c.dwps = statutil.NewFloat(rollAvgSamples)
	c.crps = statutil.NewFloat(rollAvgSamples)
	c.cwps = statutil.NewFloat(rollAvgSamples)

	c.amount = int(amount)
	c.volume = int(volume)

	c.quit = make(chan struct{})

	go c.secondLoop()

	return
}

func (c *cxdsStat) addToDB(vol int) {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.amount++
	c.volume += vol
	c.writes++ // writing request
}

func (c *cxdsStat) delFromDB(vol int) {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.amount--
	c.volume -= vol
	c.writes++ // writing request
}

func (c *cxdsStat) addDBGet(inc int, rc uint32, val []byte, err error) {
	if inc == 0 {
		c.addReadingDBRequest() // reading request
		return
	}

	if err != nil {
		c.addWritingDBRequest() // writing request failed
		return
	}

	if rc == 0 {
		c.delFromDB(len(val)) // removed
	} else {
		c.addWritingDBRequest() // just change
	}

}

func (c *cxdsStat) addCacheGet(inc int) {
	if inc == 0 {
		c.addReadingCacheRequest()
		return
	}
	c.addWritingCacheRequest()
}

// current amoutn and volume of DB
func (c *cxdsStat) amountAndVolume() (amount, volume uint32) {
	c.mx.Lock()
	defer c.mx.Unlock()

	return uint32(c.amount), uint32(c.volume)
}

func (c *cxdsStat) addWritingDBRequest() {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.writes++
}

func (c *cxdsStat) addReadingDBRequest() {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.reads++
}

func (c *cxdsStat) addReadingCacheRequest() {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.creads++
}

func (c *cxdsStat) addWritingCacheRequest() {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.cwrites++
}

func (c *cxdsStat) dbRPS() float64 {
	return c.drps.Value()
}

func (c *cxdsStat) dbWPS() float64 {
	return c.dwps.Value()
}

func (c *cxdsStat) cRPS() float64 {
	return c.crps.Value()
}

func (c *cxdsStat) cWPS() float64 {
	return c.cwps.Value()
}

func (c *cxdsStat) secondLoop() {

	var (
		tk = time.NewTicker(time.Second)
		tc = tk.C
	)

	defer tk.Stop()

	for {
		select {
		case <-tc:
			c.second()
		case <-c.quit:
			return
		}
	}

}

func (c *cxdsStat) second() {
	c.mx.Lock()
	defer c.mx.Unlock()

	// read DB
	c.drps.Add(float64(c.reads))
	c.reads = 0

	// write DB
	c.dwps.Add(float64(c.writes))
	c.writes = 0

	// read cache
	c.crps.Add(float64(c.creads))
	c.creads = 0

	// write cache
	c.cwps.Add(float64(c.cwrites))
	c.cwrites = 0

}

func (c *cxdsStat) Close() {
	c.closeo.Do(func() {
		close(c.quit)
	})
}
