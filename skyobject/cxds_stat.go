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

	mx     sync.Mutex
	quit   chan struct{}
	closeo sync.Once
}

func newCxdsStat(rollAvgSamples int, amount, volume uint32) (c *cxdsStat) {

	c = new(cxdsStat)

	c.drps = statutil.NewFloat(rollAvgSamples)
	c.dwps = statutil.NewFloat(rollAvgSamples)
	c.crps = statutil.NewFloat(rollAvgSamples)

	c.quit = make(chan struct{})

	go c.secondLoop()

	return
}

func (c *cxdsStat) addReadingDBRequest() {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.reads++
}

func (c *cxdsStat) addWritingDBRequest() {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.writes++
}

func (c *cxdsStat) addReadingCacheRequest() {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.creads++
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

}

func (c *cxdsStat) Close() {
	c.closeo.Do(func() {
		close(c.quit)
	})
}
