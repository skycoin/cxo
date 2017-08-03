package skyobject

import (
	"fmt"
	"time"

	"github.com/skycoin/cxo/node/log"
)

// config related constants and defaults
const (
	Prefix       string = "[skyobject] " // default log prefix
	MerkleDegree int    = 16             // default References' degree

	StatSamples int           = 5                // it's enough
	CleanUp     time.Duration = 59 * time.Second // every minute
	KeepRoots   bool          = false            // remove
	KeepNonFull bool          = false            // remove

	CleanUpPin        log.Pin = 1 << iota // show time of CleanUp in logs
	PackSavePin                           // show time of (*Pack).Save in logs
	CleanUpVerbosePin                     // show collecting and removing times

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

	// StatSamples is number of samples of rolling average of
	// statistic of Container. Durations of (*Container).CleanUp
	// and (*Pack).Save are collected in Stat as rolling average
	// (wiki: 'moving average')
	StatSamples int

	// CleanUp by provided interval. Set to 0 to disable. The
	// cleaning can be stopped by (*Container).Close() only
	CleanUp time.Duration

	// KeepRoots instead of removing. By default (e.g. if it is false)
	// (*Container).CleanUp removes all Root obejcts of a feed before
	// last full
	KeepRoots bool

	// KeepNonFull root objects before shutdown. By default (e.g. if it is
	// false) all non-full root objects will be removed from database
	// before shutdown
	KeepNonFull bool
}

// NewConfig returns pointer to Config with default values
func NewConfig() (conf *Config) {
	conf = new(Config)

	// core configs

	conf.MerkleDegree = MerkleDegree

	// logger

	conf.Log = log.NewConfig()
	conf.Log.Prefix = Prefix
	conf.Log.Pins = CleanUpPin | PackSavePin

	// stat

	conf.StatSamples = StatSamples

	// clean up

	conf.CleanUp = CleanUp
	conf.KeepRoots = KeepRoots
	conf.KeepNonFull = KeepNonFull
	return
}

// Validate the Config
func (c *Config) Validate() error {
	if c.MerkleDegree <= 1 {
		return fmt.Errorf("skyobject.Config.MerkleDegree too small: %d",
			c.MerkleDegree)
	}
	return nil
}
