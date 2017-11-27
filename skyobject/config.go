package skyobject

import (
	"fmt"

	"github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/skyobject/registry"
)

// config related constants and defaults
const (
	Prefix         string = "[skyobject] " // default log prefix
	RollAvgSamples int    = 5              // rolling average samples

	CacheMaxItems int     = 1024 * 1024 // binary million
	CacheMaxSize  int     = 1024 * 1024 // 1M
	CacheCleaning float64 = 0.8         // down to 80%

	// CacheMaxItemSize is around 100K
	CacheMaxItemSize int = int(float64(CacheMaxSize)*(1.0-CacheCleaning)) / 2

	// default CachePolicy is LRU

	PackSavePin       log.Pin = 1 << iota // show time of (*Pack).Save in logs
	CleanUpVerbosePin                     // show collecting and removing times
	FillVerbosePin                        // show filling debug logs
	FillPin                               // show filling time

	VerbosePin // too many logs to show
)

// A Config represents configurations
// and options of Container
type Config struct {
	// Registry that will be used as "core registry".
	// Can be nil
	Registry *registry.Registry

	// MerkleDegree of References' Merkle trees.
	// The option affects new trees onyl and not
	// changes existsing registry.Refs
	MerkleDegree int

	// Cache configs. Set the CacheMaxItems or the CacheMaxSize to
	// zero to switch the cache off

	// CacheMaxItems is maximum number of items the cache can fit.
	// See also, CacheCleaning field
	CacheMaxItems int
	// CacheMaxSize is maximum total length of all elements of the caceh.
	// See also CacheCleaning field
	CacheMaxSize int
	// CachePolicy is policy of the Cache. By default it's LRU,
	// but it's possible to choose LFU if you want
	CachePolicy CachePolicy
	// CacheCleaning is a flaot point number from 0.5 to 0.9.
	// The number is percent. If the Cache is full, then the
	// will be cleaned down to this percent of fullness. E.g.
	// the cache never exceeds the CacheMaxItems and the
	// CacheMaxSize boundaries. Reaching them, the Cache
	// cleans itself down to the percent using given
	// CachePolicy
	CacheCleaning float64
	// CacheMaxItemSize is max size of an item the cache will
	// keep inside. All items bigger will be put to DB directly.
	// This required to don't keep very big items in the cache.
	// This items can will break caching algorithms. The
	// CacheMaxItemSize can't be bigger then
	// CacheMaxSize*(1.0 - CacheCleaning)
	CacheMaxItemSize int

	// Log configs
	Log log.Config // logging

	RollAvgSamples int // calculate rolling average for stat
}

// NewConfig returns pointer to Config with default values
func NewConfig() (conf *Config) {
	conf = new(Config)

	// core configs

	conf.MerkleDegree = registry.Degree

	// cache configs

	conf.CacheMaxItems = CacheMaxItems
	conf.CacheMaxSize = CacheMaxSize
	conf.CachePolicy = LRU
	conf.CacheCleaning = CacheCleaning
	conf.CacheMaxItemSize = CacheMaxItemSize

	// logger

	conf.Log = log.NewConfig()
	conf.Log.Prefix = Prefix
	conf.Log.Pins = PackSavePin
	conf.RollAvgSamples = RollAvgSamples
	return
}

// Validate the Config
func (c *Config) Validate() error {

	if c.MerkleDegree <= 2 {
		return fmt.Errorf("skyobject.Config.MerkleDegree too small: %d",
			c.MerkleDegree)
	}
	if c.RollAvgSamples < 1 {
		return fmt.Errorf("skyobject.Config.RollAvgSampels too small: %d",
			c.RollAvgSamples)
	}

	if c.CacheMaxItems < 0 {
		return fmt.Errorf("skyobject.Config.CacheMaxItems is negaive: %d",
			c.CacheMaxItems)
	}

	if c.CacheMaxSize < 0 {
		return fmt.Errorf("skyobject.Config.CacheMaxSize is negaive: %d",
			c.CacheMaxSize)
	}

	if c.CachePolicy != LRU && c.CachePolicy != LFU {
		return fmt.Errorf(
			"skyobject.Config.CachePolicy is unknown: %d (choose LRU or LFU)",
			c.CachePolicy)
	}

	if c.CacheCleaning < 0.5 {
		return fmt.Errorf(
			"skyobject.Config.CacheCleaning is too small: %f (< 0.5)",
			c.CacheCleaning)
	}

	if c.CacheCleaning > 0.9 {
		return fmt.Errorf(
			"skyobject.Config.CacheCleaning is too big: %f (> 0.9)",
			c.CacheCleaning)
	}

	var cacheMaxItemSize = int(float64(CacheMaxSize) * (1.0 - CacheCleaning))

	if c.CacheMaxItemSize > cacheMaxItemSize {
		return fmt.Errorf(
			"skyobject.Config.CacheaxItemSize is too big: %f (> %f)",
			c.CacheMaxItemSize, cacheMaxItemSize)
	}

	return nil
}
