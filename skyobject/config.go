package skyobject

import (
	"time"

	"github.com/skycoin/cxo/node/log"
)

// config related constants and defaults
const (
	Prefix            string  = "[skyobject] " // default log prefix
	MerkleDegree      uint32  = 16             // default References' degree
	MerkleFillPercent float64 = 0.1            // default percentage

	StatSamples int           = 5                // it's enough
	CleanUp     time.Duration = 59 * time.Second // every minute
	KeepRoots   bool          = false            // remove
	KeepNonFull bool          = false            // remove

	CleanUpPin        log.Pin = 1 << iota // show time of CleanUp in logs
	PackSavePin                           // show time of (*Pack).Save in logs
	CleanUpVerbosePin                     // show collecting and removing times
)

// A Config represents oconfigurations
// and options of Container
type Config struct {
	// Registry that will be used as "core registry".
	// Can be nil
	Registry *Registry

	// MerkleDegree of References' Merkle trees.
	// The option affects new trees
	MerkleDegree uint32

	// MerkleFillPercent percentage of empty references in
	// Merkle trees after which depth of a tree will be
	// increased and the tree will be reconstructed.
	//
	// An element of tree can be deleted. In this case,
	// the element will be zeroed. If percentage of these
	// zero elements in tree is MerkleFillPercent then
	// tree will be reconstructed and zero elements removed
	// from tree
	//
	// The percentage measured from 0 to 1 (for example 0.1 = 10%)
	//
	// The option is machine local and not saved
	// inside tree internals
	MerkleFillPercent float64

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
	conf.MerkleFillPercent = MerkleFillPercent

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
