package node

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"syscall"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/skyobject"
)

// defaults
const (

	// server defaults
	Listen         string = ""   // default listening address
	EnableListener bool   = true // listen by default

	EnableRPC   bool   = true        // default RPC pin
	RemoteClose bool   = false       // default remote-closing pin
	RPCAddress  string = "[::]:8878" // default RPC address

	// PingInterval is default interval by which server send pings
	// to connections that doesn't communicate. Actually, the
	// interval can be increased x2
	PingInterval    time.Duration = 2 * time.Second
	ResponseTimeout time.Duration = 5 * time.Second // default
	PublicServer    bool          = false           // default
)

// log pins
const (
	MsgPin       log.Pin = 1 << iota // msgs
	SubscrPin                        // subscriptions
	ConnPin                          // connect/disconnect
	RootPin                          // root receive etc
	FillPin                          // fill/drop
	HandlePin                        // handle a message
	DiscoveryPin                     // discovery
)

// Addresses are discovery addresses
type Addresses []string

// String implements flag.Value interface
func (a *Addresses) String() string {
	return fmt.Sprintf("%v", []string(*a))
}

// Set implements flag.Value interface
func (a *Addresses) Set(addr string) error {
	*a = append(*a, addr)
	return nil
}

// A Config represents configurations
// of a Node. The config contains configurations
// for gnet.Pool and  for log.Logger. If logger of
// gnet.Config is nil, then logger of Config
// will be used
type Config struct {
	gnet.Config // pool configurations

	Log log.Config // logger configurations (logger of Node)

	// Skyobject configuration
	Skyobject *skyobject.Config

	// RPC

	// EnableRPC server
	EnableRPC bool
	// RPCAddress if enabled
	RPCAddress string
	// RemoteClose allows closing the
	// server using RPC
	RemoteClose bool

	// Listener

	// Listen on address (empty for
	// arbitrary assignment)
	Listen string
	// EnableListener turns on/off listening
	EnableListener bool

	// conenction configs

	// PingInterval used to ping clients
	// Set to 0 to disable pings
	PingInterval time.Duration
	// ResponseTimeout used by methods that requires response.
	// Zero timeout means infinity. Negative timeout causes panic
	ResponseTimeout time.Duration

	// node configs

	// PublicServer never keeps secret feeds it share
	PublicServer bool
	// ServiceDiscovery addresses
	DiscoveryAddresses Addresses

	//
	// callbacks
	//

	// connections create/close, this callbacks
	// perform in own goroutines

	OnCreateConnection func(c *Conn)
	OnCloseConnection  func(c *Conn)

	// subscribe/unsubscribe from a remote peer

	// OnSubscribeRemote called while a remote peer wants to
	// subscribe to feed of this (local) node. This callback
	// never called if subscription rejected by any reason.
	// If this callback returns a non-nil error the subscription
	// will be rejected, even if it's ok. This callback should
	// not block, because it performs inside message handling
	// goroutine and long freeze breaks connection
	OnSubscribeRemote func(c *Conn, feed cipher.PubKey) (reject error)
	// OnUnsubscribeRemote called while a remote peer wants
	// to unsubscribe from feed of this (local) node. This
	// callback never called if remote peer is not susbcribed.
	// This callback should not block, because it performs inside
	// message handling goroutine and long freeze breaks connection
	OnUnsubscribeRemote func(c *Conn, feed cipher.PubKey)

	// root objects

	// OnRootReceived is callback that called
	// when Client receive new Root object.
	// The callback never called for rejected
	// Roots (including "already exists"). This callback
	// performs in own goroutine. You can't use
	// Root of this callback anywhere because it
	// is not saved and filled yet. This callback doesn't
	// called if received a Roto already exists
	OnRootReceived func(c *Conn, root *skyobject.Root)
	// OnRootFilled is callback that called when
	// Client finishes filling received Root object.
	// This callback performs in own goroutine. The
	// Root is full and holded during this callabck.
	// You can use it anywhere
	OnRootFilled func(c *Conn, root *skyobject.Root)
	// OnFillingBreaks occurs when a filling Root
	// can't be filled up because connection breaks.
	// The Root will be removed after this callback
	// with all related objects. The Root is not full
	// and can't be used in skyobject methods.This
	// callback should not block because it performs
	// in handling goroutine
	OnFillingBreaks func(c *Conn, root *skyobject.Root, err error)
}

// NewConfig returns Config
// filled with default values
func NewConfig() (sc Config) {

	sc.Config = gnet.NewConfig()
	sc.Log = log.NewConfig()
	sc.Skyobject = skyobject.NewConfig()

	sc.Listen = Listen
	sc.EnableListener = EnableListener

	sc.EnableRPC = EnableRPC
	sc.RPCAddress = RPCAddress
	sc.RemoteClose = RemoteClose

	sc.PingInterval = PingInterval
	sc.ResponseTimeout = ResponseTimeout

	sc.PublicServer = PublicServer
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

	if s.Skyobject != nil {
		s.Skyobject.FromFlags()
	}

	flag.BoolVar(&s.EnableRPC,
		"rpc",
		s.EnableRPC,
		"enable RPC server")
	flag.StringVar(&s.RPCAddress,
		"rpc-address",
		s.RPCAddress,
		"address of RPC server")
	flag.BoolVar(&s.RemoteClose,
		"remote-close",
		s.RemoteClose,
		"allow closing the server using RPC")

	flag.StringVar(&s.Listen,
		"address",
		s.Listen,
		"listening address (pass empty string to arbitrary assignment by OS)")
	flag.BoolVar(&s.EnableListener,
		"enable-listening",
		s.EnableListener,
		"enable listening pin")

	flag.DurationVar(&s.PingInterval,
		"ping",
		s.PingInterval,
		"interval to send pings (0 = disable)")
	flag.DurationVar(&s.ResponseTimeout,
		"response-tm",
		s.ResponseTimeout,
		"response timeout (0 = infinity)")

	flag.BoolVar(&s.PublicServer,
		"public-server",
		s.PublicServer,
		"make the server public")
	flag.Var(&s.DiscoveryAddresses,
		"discovery-address",
		"address of service discovery")

	return
}
