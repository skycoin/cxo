package node

import (
	"flag"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"
)

// defaults
const (

	// server defaults

	EnableRPC      bool   = true        // default RPC pin
	Listen         string = ""          // default listening address
	EnableListener bool   = true        // listen by default
	RemoteClose    bool   = false       // default remote-closing pin
	RPCAddress     string = "[::]:8878" // default RPC address
	InMemoryDB     bool   = false       // default database placement pin
	PublicServer   bool   = false       // default public-server pin

	// PingInterval is default interval by which server send pings
	// to connections that doesn't communicate. Actually, the
	// interval can be increased x2
	PingInterval time.Duration = 0 * 2 * time.Second

	// GCInterval is default valie for GC
	// triggering interval
	GCInterval time.Duration = 5 * time.Second

	// default tree is
	//   server: ~/.skycoin/cxo/server/bolt.db
	//   client: ~/.skycoin/cxo/client/bolt.db
	// todo:
	//   server should be system wide and its
	//   directory should be like /var/lib/cxo/bolt.db

	skycoinDataDir = ".skycoin"
	cxoSubDir      = "cxo"

	serverSubDir = "server"
	clientSubDir = "client"

	dbFile = "bolt.db"
)

func dataDir(sub string) string {
	// TODO: /var/lib/cxo for cxod
	usr, err := user.Current()
	if err != nil {
		panic(err) // fatal
	}
	if usr.HomeDir == "" {
		panic("empty home dir")
	}
	return filepath.Join(usr.HomeDir, skycoinDataDir, cxoSubDir, sub)
}

func initDataDir(dir string) error {
	return os.MkdirAll(dir, 0700)
}

// A NodeConfig represnets configurations
// of a Node. The config contains configurations
// for gnet.Pool and  for log.Logger. If logger of
// gnet.Config is nil, then logger of NodeConfig
// will be used
type NodeConfig struct {
	gnet.Config // pool confirations

	Log log.Config // logger configurations

	// EnableRPC server
	EnableRPC bool
	// RPCAddress if enabled
	RPCAddress string
	// Listen on address (empty for
	// arbitrary assignment)
	Listen string
	// EnableListener turns on/off listening
	EnableListener bool

	// RemoteClose allows closing the
	// server using RPC
	RemoteClose bool

	// PingInterval used to ping clients
	// Set to 0 to disable pings
	PingInterval time.Duration

	// GCInterval is interval to trigger GC
	// Set to 0 to disabel GC
	GCInterval time.Duration

	// InMemoryDB uses database in memory
	InMemoryDB bool
	// DBPath is path to database file
	DBPath string
	// DataDir is directory with data files
	DataDir string

	//
	// callbacks
	//

	// subscriptions

	OnSubscriptionAccepted func(c *gnet.Conn, feed cipher.PubKey)
	OnSubscriptionDenied   func(c *gnet.Conn, feed cipher.PubKey)

	// root objects

	// OnRootReceived is callback that called
	// when Client receive new Root object
	OnRootReceived func(root *Root)
	// OnRootFilled is callback that called when
	// Client finishes filling received Root object
	OnRootFilled func(root *Root)
}

// NewNodeConfig returns NodeConfig
// filled with default values
func NewNodeConfig() (sc NodeConfig) {
	sc.Config = gnet.NewConfig()
	sc.Log = log.NewConfig()
	sc.EnableRPC = EnableRPC
	sc.RPCAddress = RPCAddress
	sc.Listen = Listen
	sc.EnableListener = EnableListener
	sc.RemoteClose = RemoteClose
	sc.PingInterval = PingInterval
	sc.GCInterval = GCInterval
	sc.InMemoryDB = InMemoryDB
	sc.DataDir = dataDir(serverSubDir)
	sc.DBPath = filepath.Join(sc.DataDir, dbFile)
	sc.PublicServer = PublicServer
	return
}

// FromFlags obtains value from command line flags.
// Call the method before `flag.Parse` for example
//
//     c := node.NewNodeConfig()
//     c.FromFlags()
//     flag.Parse()
//
func (s *NodeConfig) FromFlags() {
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
	flag.StringVar(&s.EnableListener,
		"enable-listening",
		s.EnableListener,
		"enable listening pin")
	flag.BoolVar(&s.RemoteClose,
		"remote-close",
		s.RemoteClose,
		"allow closing the server using RPC")
	flag.DurationVar(&s.PingInterval,
		"ping",
		s.PingInterval,
		"interval to send pings (0 = disable)")
	flag.DurationVar(&s.GCInterval,
		"gc",
		s.GCInterval,
		"garbage collecting interval (0 = disable)")
	flag.BoolVar(&s.InMemoryDB,
		"mem-db",
		s.InMemoryDB,
		"use in-memory database")
	flag.StringVar(&s.DataDir,
		"data-dir",
		s.DataDir,
		"directory with data")
	flag.StringVar(&s.DBPath,
		"db-path",
		s.DBPath,
		"path to database")
	flag.BoolVar(&s.PublicServer,
		"public-server",
		s.PublicServer,
		"turn the server to be public")
	return
}
