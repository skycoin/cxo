package skyobject

import (
	"github.com/skycoin/cxo/node/log"
)

// config related constants
const (
	Prefix       string = "[skyobject] " // default log prefix
	MerkleDegree uint32 = 8              // default References' degree
)

// A Config represents oconfigurations
// and options of Container
type Config struct {
	Registry     *Registry  // core registry
	MerkleDegree uint32     // degree of References' Merkle trees
	Log          log.Config // logging
}

// NewConfig returns pointer to Config with default values
func NewConfig() (conf *Config) {
	conf = new(Config)
	conf.MerkleDegree = MerkleDegree // defautl
	conf.Log = log.NewConfig()
	conf.Log.Prefix = Prefix
	return
}
