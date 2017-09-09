package skyobject

import (
	"fmt"

	"github.com/skycoin/cxo/node/log"
)

// config related constants and defaults
const (
	Prefix       string = "[skyobject] " // default log prefix
	MerkleDegree int    = 16             // default References' degree

	RollAvgSamples int = 5 // rolling average samples

	PackSavePin       log.Pin = 1 << iota // show time of (*Pack).Save in logs
	CleanUpVerbosePin                     // show collecting and removing times
	FillVerbosePin                        // show filling debug logs
	FillPin                               // show filling time

	VerbosePin // to many logs to show
)

// A Config represents oconfigurations
// and options of Container
type Config struct {
	// Registry that will be used as "core registry".
	// Can be nil
	Registry *Registry

	// MerkleDegree of References' Merkle trees.
	// The option affects new trees
	MerkleDegree int

	// Log configs
	Log log.Config // logging

	RollAvgSamples int // calculate rolling average for stat
}

// NewConfig returns pointer to Config with default values
func NewConfig() (conf *Config) {
	conf = new(Config)

	// core configs

	conf.MerkleDegree = MerkleDegree

	// logger

	conf.Log = log.NewConfig()
	conf.Log.Prefix = Prefix
	conf.Log.Pins = PackSavePin
	conf.RollAvgSamples = RollAvgSamples
	return
}

// Validate the Config
func (c *Config) Validate() error {
	if c.MerkleDegree <= 1 {
		return fmt.Errorf("skyobject.Config.MerkleDegree too small: %d",
			c.MerkleDegree)
	}
	if c.RollAvgSamples < 1 {
		return fmt.Errorf("skyobject.Config.RollAvgSampels too small: %d",
			c.RollAvgSamples)
	}
	return nil
}
