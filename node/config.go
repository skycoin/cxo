package node

import (
	"flag"
	"time"

	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"
)

// defaults
const (

	// server defaults

	EnableRPC   bool   = true        // default RPC pin
	Listen      string = ""          // default listening address
	RemoteClose bool   = false       // default remote-closing pin
	RPCAddress  string = "[::]:8878" // default RPC address

	PingInterval time.Duration = 2 * time.Second
)

// A ServerConfig represnets configurations
// of a Server
type ServerConfig struct {
	gnet.Config // pool confirations

	Log log.Config // logger configurations

	// EnableRPC server
	EnableRPC bool
	// RPCAddress if enabled
	RPCAddress string
	// Listen on address (empty for
	// arbitrary assignment)
	Listen string

	// RemoteClose allows closing the
	// server using RPC
	RemoteClose bool

	// PingInterval used to ping clients
	// Set to 0 to disable pings
	PingInterval time.Duration
}

// NewServerConfig returns ServerConfig
// filled with default values
func NewServerConfig() (sc ServerConfig) {
	sc.Config = gnet.NewConfig()
	sc.Log = log.NewConfig()
	sc.EnableRPC = EnableRPC
	sc.RPCAddress = RPCAddress
	sc.Listen = Listen
	sc.RemoteClose = RemoteClose
	sc.PingInterval = PingInterval
	return
}

// FromFlags obtains value from command line flags.
// Call the method before `flag.Parse` for example
//
//     c := node.NewServerConfig()
//     c.FromFlags()
//     flag.Parse()
//
func (s *ServerConfig) FromFlags() {
	s.Config.FromFlags()
	s.Log.FromFlags()

	flag.BoolVar(&s.EnableRPC,
		"rpc",
		s.EnableRPC,
		"enable RPC server")
	flag.StringVar(&s.RPCAddress,
		"rpc-address",
		s.RPCAddress,
		"address of RPC server")
	flag.StringVar(&s.Listen,
		"address",
		s.Listen,
		"listening address (pass empty string to arbitrary assignment by OS)")
	flag.BoolVar(&s.RemoteClose,
		"remote-close",
		s.RemoteClose,
		"allow closing the server using RPC")
	flag.DurationVar(&s.PingInterval,
		"ping",
		s.PingInterval,
		"interval to send pings (0 = disable)")
	return
}

// A ClientCofig represents configurations
// of a Client
type ClientConfig struct {
	gnet.Config            // pool configurations
	Log         log.Config // logger configurations

	// handlers
	OnConnect    func()
	OnDisconenct func()
}

// NewClientConfig returns ClientConfig
// filled with default values
func NewClientConfig() (cc ClientConfig) {
	cc.Config = gnet.NewConfig()
	cc.Log = log.NewConfig()
	return
}

// FromFlags obtains value from command line flags.
// Call the method before `flag.Parse` for example
//
//     c := node.NewClientConfig()
//     c.FromFlags()
//     flag.Parse()
//
func (c *ClientConfig) FromFlags() {
	c.Config.FromFlags()
	c.Log.FromFlags()
}
