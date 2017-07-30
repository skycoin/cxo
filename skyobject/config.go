package skyobject

import (
	"time"

	"github.com/skycoin/cxo/node/log"
)

// config related constants
const (
	Prefix       string = "[skyobject] " // default log prefix
	MerkleDegree uint32 = 16             // default References' degree

	StatSamples int           = 5                // default
	CleanUp     time.Duration = 59 * time.Second // default interval
	KeepRoots   bool          = false            // keep roots before last full

	CleanUpPin  log.Pin = 1 << iota // show time of CleanUp in logs
	PackSavePin                     // show time of (*Pack).Save in logs
)

// A Config represents oconfigurations
// and options of Container
type Config struct {
	Registry     *Registry     // core registry
	MerkleDegree uint32        // degree of References' Merkle trees
	Log          log.Config    // logging
	StatSamples  int           // samples of rolling average of stat times
	CleanUp      time.Duration // clean up by interval (use 0 to disable)
	KeepRoots    bool          // keep roots before last full
}

// NewConfig returns pointer to Config with default values
func NewConfig() (conf *Config) {
	conf = new(Config)
	conf.MerkleDegree = MerkleDegree // defautl
	conf.Log = log.NewConfig()
	conf.Log.Prefix = Prefix
	conf.StatSamples = StatSamples
	conf.CleanUp = CleanUp
	conf.KeepRoots = KeepRoots
	return
}
