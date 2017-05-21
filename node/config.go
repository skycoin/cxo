package node

import (
	"flag"
	"path/filepath"
	"time"

	"github.com/skycoin/skycoin/src/util"

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
	InMemoryDB  bool   = false       // default database placement pin

	// PingInterval is default interval by which server send pings
	// to connections that doesn't communicate. Actually, the
	// interval can be increased x2
	PingInterval time.Duration = 2 * time.Second

	// default tree is
	//   server: ~/.skycoin/cxo/server/bolt.db
	//   client: ~/.skycoin/cxo/client/bolt.db
	// todo:
	//   server should be system wide and its
	//   directory should be like /var/cache/cxo/bolt.db

	skycoinDataDir = ".skycoin"
	cxoSubDir      = "cxo"

	serverSubDir = "server"
	clientSubDir = "client"

	dbFile = "bolt.db"
)

func init() {
	util.InitDataDir(filepath.Join(skycoinDataDir, cxoSubDir))
}

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

	// InMemoryDB uses database in memory
	InMemoryDB bool

	// DBPath is path to database if InMemeoryDB is
	// false. If the DBPath is empty then
	// default database path used
	DBPath string
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
	sc.InMemoryDB = InMemoryDB
	sc.DBPath = filepath.Join(util.DataDir, serverSubDir, dbFile)
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
	flag.BoolVar(&s.InMemoryDB,
		"mem-db",
		s.InMemoryDB,
		"use in-memory database")
	flag.BoolVar(&s.DBPath,
		"db-path",
		s.DBPath,
		"path to database")
	return
}

// A ClientCofig represents configurations
// of a Client
type ClientConfig struct {
	gnet.Config            // pool configurations
	Log         log.Config // logger configurations

	// InMemoryDB uses database in memory
	InMemoryDB bool

	// DBPath is path to database if InMemeoryDB is
	// false. If the DBPath is empty then
	// default database path used
	DBPath string

	// handlers
	OnConnect    func()
	OnDisconenct func()
}

// NewClientConfig returns ClientConfig
// filled with default values
func NewClientConfig() (cc ClientConfig) {
	cc.Config = gnet.NewConfig()
	cc.Log = log.NewConfig()
	cc.InMemoryDB = InMemoryDB
	cc.DBPath = filepath.Join(util.DataDir, clientSubDir, dbFile)
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
	flag.BoolVar(&c.InMemoryDB,
		"mem-db",
		c.InMemoryDB,
		"use in-memory database")
	flag.BoolVar(&c.DBPath,
		"db-path",
		c.DBPath,
		"path to database")
}
