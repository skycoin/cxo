package node

import (
	"flag"
	"io"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"syscall"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/skyobject"
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

	// PingInterval is default interval by which server send pings
	// to connections that doesn't communicate. Actually, the
	// interval can be increased x2
	PingInterval    time.Duration = 2 * time.Second
	ResponseTimeout time.Duration = 5 * time.Second // default
	PublicServer    bool          = false           // default

	// default tree is
	//   server: ~/.skycoin/cxo/bolt.db

	skycoinDataDir = ".skycoin"
	cxoSubDir      = "cxo"

	dbFilePrefix = "cxodb"
)

// log pins
const (
	MsgPin    log.Pin = 1 << iota // msgs
	SubscrPin                     // subscriptions
	ConnPin
	RootPin
	FillPin
)

func dataDir() string {
	usr, err := user.Current()
	if err != nil {
		panic(err) // fatal
	}
	if usr.HomeDir == "" {
		panic("empty home dir")
	}
	return filepath.Join(usr.HomeDir, skycoinDataDir, cxoSubDir)
}

func initDataDir(dir string) error {
	return os.MkdirAll(dir, 0700)
}

// A Config represnets configurations
// of a Node. The config contains configurations
// for gnet.Pool and  for log.Logger. If logger of
// gnet.Config is nil, then logger of Config
// will be used
type Config struct {
	gnet.Config // pool confirations

	Log log.Config // logger configurations (logger of Node)

	// Skyobject configuration
	Skyobject *skyobject.Config

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

	// ResponseTimeout used by methods that requires response.
	// Zero timeout means infinity. Negative timeout causes panic
	ResponseTimeout time.Duration

	// InMemoryDB uses database in memory
	InMemoryDB bool
	// DBPath is path to database file
	DBPath string
	// DataDir is directory with data files
	DataDir string

	// PublicServer never keeps secret feeds it share
	PublicServer bool

	// DropNonFullRoots removes Root object from
	// DB if it can't be filled
	DropNonFullRotos bool

	//
	// callbacks
	//

	// connections create/close

	OnCreateConnection func(n *Node, c *gnet.Conn)
	OnCloseConnection  func(n *Node, c *gnet.Conn)

	// subscribe/unsubscribe from a remote peer

	// Both OnSubscribeRemote and OnUnsubscribeRemote
	// called while some mutexes are locked and it's
	// imposisble to call following methods of the Node
	// in this callbacks: Subscribe, Unsubscribe, Feeds,
	// SubscribeResponse and SubscribeResponseTimeout

	// OnSubscribeRemote called while a remote peer wants to
	// subscribe to feed of this (local) node. This callback
	// never called if subscription rejected by any reason.
	// If this callback returns a non-nil error the subscription
	// willl be rejected, even if it's ok
	OnSubscribeRemote func(n *Node, c *gnet.Conn,
		feed cipher.PubKey) (reject error)
	// OnUnsubscribeRemote called while a remote peer wants
	// to unsubscribe from feed of this (local) node. This
	// callback never called if remote peer is not susbcribed
	OnUnsubscribeRemote func(n *Node, c *gnet.Conn, feed cipher.PubKey)

	// replies for subscriptions

	// OnSubscriptionAccepted called when a remote peer accepts
	// you subscription. It never called if remote peer already
	// subscribed to the feed by this (local) node
	OnSubscriptionAccepted func(n *Node, c *gnet.Conn, feed cipher.PubKey)
	// OnSubscriptionRejected called when a remote peer rejects
	// you subscription
	OnSubscriptionRejected func(n *Node, c *gnet.Conn, feed cipher.PubKey)

	// root objects

	// OnRootReceived is callback that called
	// when Client receive new Root object.
	// The callback never called for rejected
	// Roots (including "already exists")
	OnRootReceived func(n *Node, c *gnet.Conn, root *skyobject.Root)
	// OnRootFilled is callback that called when
	// Client finishes filling received Root object
	OnRootFilled func(n *Node, c *gnet.Conn, root *skyobject.Root)
	// OnFillingBreaks occurs when a filling Root
	// can't be filled up because connection breaks.
	// The Root can be removed or can not, depending
	// DropNonFullRoots option. In many cases, connection
	// will be recreated and the Root will be filled up.
	// no reason to remove it. But. This callback also called
	// if any other error occured, such as DB can't
	// save received object, the Root it fills malformed
	// and can't be filled, etc
	OnFillingBreaks func(n *Node, c *gnet.Conn, root *skyobject.Root, err error)
}

// NewConfig returns Config
// filled with default values
func NewConfig() (sc Config) {
	sc.Config = gnet.NewConfig()
	sc.Log = log.NewConfig()
	sc.Skyobject = skyobject.NewConfig()
	sc.EnableRPC = EnableRPC
	sc.RPCAddress = RPCAddress
	sc.Listen = Listen
	sc.EnableListener = EnableListener
	sc.RemoteClose = RemoteClose
	sc.PingInterval = PingInterval
	sc.InMemoryDB = InMemoryDB
	sc.DataDir = dataDir()
	sc.DBPath = filepath.Join(sc.DataDir, dbFilePrefix)
	sc.ResponseTimeout = ResponseTimeout
	sc.PublicServer = PublicServer
	sc.Config.OnDial = OnDialFilter
	return
}

// FromFlags obtains value from command line flags.
// Call the method before `flag.Parse` for example
//
//     c := node.NewConfig()
//     c.FromFlags()
//     flag.Parse()
//
func (s *Config) FromFlags() {
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
	flag.BoolVar(&s.EnableListener,
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
	flag.DurationVar(&s.ResponseTimeout,
		"response-tm",
		s.ResponseTimeout,
		"response timeout (0 = infinity)")
	flag.BoolVar(&s.PublicServer,
		"public-server",
		s.PublicServer,
		"make the server public")

	// TODO: skyobejct.Configs from flags

	return
}

// OnDialFilter is gnet.OnDial callback that rejects redialing if
// remote peer closes connection
func OnDialFilter(c *gnet.Conn, err error) (reject error) {
	if ne, ok := err.(net.Error); ok {
		if ne.Temporary() {
			return // nil
		}
		if oe, ok := err.(*net.OpError); ok {
			// reading fails with EOF if remote peer closes connection,
			// but writing ...
			if se, ok := oe.Err.(*os.SyscallError); ok {
				if errno, ok := se.Err.(syscall.Errno); ok {
					if errno == 0x68 {
						// "connection reset by peer"
						return err // reject
					}
				}
			}
		}
	} else if err == io.EOF { // connection has been closed by peer
		return err // reject
	}
	return // nil (unknowm case)
}
