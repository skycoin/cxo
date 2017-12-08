package node

import (
	"flag"
	"fmt"
	"time"

	"github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/cxo/skyobject/registry"
)

// default configurations
const (
	Prefix                       string        = "[node] "
	MaxConnections               int           = 1000 * 1000
	MaxFillingRootsPerConnection int           = 1000 * 1000
	MaxFillingTime               time.Duration = 10 * time.Minute
	ListenTCP                    string        = ":8870"
	ListenUDP                    string        = "" // don't listen
	RPC                          string        = ":8871"
	ResponseTimeout              time.Duration = 59 * time.Second
	Pings                        time.Duration = 118 * time.Second
	Public                       bool          = false
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

// OnRootReceivedFunc represents callback that
// called when new Root objects received. It's
// possible to reject a receved Root returning
// error by this function. The callback can be
// called many times for different connections,
// e.g. a Root object can be received from many
// connections. The callback is first filter on
// the path of the Root object. But the callback
// never called if a Root obejct received but
// already exist in DB of the node (e.g. the
// callback called for new Root objects for this
// node). The Root object can't be used
// since it is not full yet. See OnRootFilledFunc
// for Root you can use. The callback can be
// called for Root objects that never be filled
// for some reasons.
type OnRootReceivedFunc func(c *Conn, r *registry.Root) (reject error)

// OnRootFilledFunc represents callback that
// called when new Root filled and can be used.
// The Root is held by Container and can't be
// removed inside this function. After the
// call the Root will be unheld and you should
// to hold it if you want to use it after. The
// callback called once per Root.
type OnRootFilledFunc func(c *Conn, r *registry.Root)

// OnFillingBreaksFunc represents callback that
// called when a new Root object can't be filled.
// The callback called with non-full Root (that
// can't be used), connection and filling error.
// There is no way to resume the filling. If
// a remote peer push this Root obejct again,
// then it can be filled (or can be not). The
// callback called once per Root if all connections
// that fill this Root can't fill it. In the case
// with filling from many connections the callback
// will be called after last connections breaks the
// filling.
type OnFillingBreaksFunc func(c *Conn, r *registry.Root, err error)

// OnconnectFunc represents callback that called
// when a connection closed. The callback can be
// used to terminate the new connection returning
// error. In some cases it's not possible to
// disconnect (e.g. the connection will be closed,
// and then creaed again, see Config.Discovery).
// The callback called after connection established.
// In some cases connection can be rejected by
// handshake, in this cases the callback will not
// be called.
type OnConnectFunc func(c *Conn) (terminate error)

// OnDisconnectFunc represents callback that called
// when a conenction closed. The callback called
// with closing reason. The reason is nil if the
// connection closed manually. The Conn argument
// is closed connection that can't be used (e.g.
// is can be used partially).
type OnDisconnectFunc func(c *Conn, reason error)

// A Config represents configurations
// of the Node. To create Config filled
// with default values use NewConfig
// method. The recommended way of creating
// Config is the NewConfig. This Config
// contains skyobejct.Config.
type Config struct {

	// Logger contains configurations of the
	// logger
	Logger log.Config

	// Config is configurations of skyobejct.Container.
	// Use nil for defaults.
	*skyobject.Config

	// MaxConnections is limit of connections.
	// Set it to zero to disable the limit.
	MaxConnections int

	// MaxFillingRootsPerConnection is limit of
	// simultaneously filling Root object per
	// connection. A connection receiving a Root
	// object starts filling it. Only one filling
	// Root per connection-feed pair allowed at the
	// same time. New received Root objects over
	// this limit will be dropped. Set it to zero
	// to disable the limit.
	MaxFillingRootsPerConnection int

	// MaxFillingTime is time limit for filling of
	// a Root object. If a Root object fills too
	// long (longe then this limit), then it will
	// be dropped. Set it to zero to disable the
	// limit.
	MaxFillingTime time.Duration

	// ListenTCP is listening address for TCP listener.
	// Blank string disables TCP listening. Use ":0" to
	// listen on all interfaces (and ipv4 + ipv6 if
	// possible) with OS-choosed ports.
	ListenTCP string

	// ListenUDP is listening address for UDP lsitener.
	// Blank string disables UDP listening. Use ":0" to
	// listen on all interfaces (and ipv4 - ipv6 if
	// possible) with OS-choosed ports.
	ListenUDP string

	// RPC is RPC listening address. Empty string
	// disables RPC.
	RPC string

	//
	// Conenctions
	//

	// ResponseTimeout is timeout for requests (see
	// (*Conn).Subscribe, (*Conn).Unsubscribe and
	// (*Conn).RemoteFeeds). And for pings tooo.
	// Set it to zero, for infinity waiting. The
	// timeout doesn't close connections, but returns
	// ErrTimeout by the request.
	ResponseTimeout time.Duration
	// Pings is interval for pinging peers. The Node
	// sends pings only if connections not used for
	// reading and writing. E.g. a ping will be sent
	// if connection not used Pings - ResponseTimout
	// time (for reading or writing). Thus the Pings
	// should be at least two times greater then the
	// ResponseTimeout. Set it to zero to disable pings.
	// It's possible to ping a connections manually
	// calling the (*Conn).Ping method. If peer doesn't
	// response for a ping, then connection will be
	// closed with ErrTimeout.
	Pings time.Duration

	//
	// Connection callbacks
	//

	// OnConenct is callback for new established
	// conenctions. See OnConnectFunc for details.
	OnConnect OnConnectFunc
	// OnDisconenct is callback for closed
	// connections. See OnDisconnectFunc for details.
	OnDisconnect OnDisconnectFunc

	//
	// Dsicovery
	//

	// Public is flag that used to make the Node
	// public. In middle case a Node never share
	// lsit of feeds it knows. But a public node
	// share the list. E.g. the list can be
	// requested from a peer. (See also
	// (*Conn).RemoteFeeds method). The Public
	// allows to use Discovery (see below) if the
	// Public is true.
	Public bool
	// Discovery is lsit of addresses of a discovery
	// severs to connect to. The discovery servers
	// used to connect Nodes between to share feeds
	// they interested in
	Discovery Addresses

	//
	// Callbacks
	//

	// All callbacks called from goroutine of
	// connection and should not block executing,
	// because it can slow down or block handling
	// incoming messeges from the connection.

	// OnRootReceived is a callback that called
	// when new Root object received. See
	// OnRootReceivedFunc for details.
	OnRootReceived OnRootReceivedFunc

	// OnRootFilled is a callback that called
	// when new Root object filled and can be
	// used. See OnRootFilledFunc for details.
	OnRootFilled OnRootFilledFunc

	// OnRootFilled is a callback that called
	// when new Root object filled and can be
	// used. See OnRootFilledFunc for details.
	OnFillingBreaks OnFillingBreaksFunc
}

// NewConfig returns new Config with
// default values
func NewConfig() (c *Config) {

	c = new(Config)

	// logger
	c.Logger.Prefix = Prefix

	// container
	c.Config = skyobject.NewConfig()

	// node
	c.MaxConnections = MaxConnections
	c.MaxFillingRootsPerConnection = MaxFillingRootsPerConnection
	c.MaxFillingTime = MaxFillingTime
	c.ListenTCP = ListenTCP
	c.ListenUDP = ListenUDP
	c.RPC = RPC
	c.ResponseTimeout = ResponseTimeout
	c.Pings = Pings
	c.Public = Public

	return

}

// FromFlags used to get values for the
// Config from comand line flags. Take a
// look the example bleow
//
//     var c = node.NewConfig()
//
//     // change values of the Config
//     // if you want
//
//     c.FromFlags()
//
//     // other work with flags if need
//
//     flag.Parse()
//
func (c *Config) FromFlags() {

	// logger configs
	c.Logger.FromFlags()

	// contaienr
	if c.Config != nil {
		c.Config.FromFlags()
	}

	// node

	flag.IntVar(&c.MaxConnections,
		"max-connections",
		c.MaxConnections,
		"max connections, incoming and outgoing, tcp and udp")

	flag.IntVar(&c.MaxFillingRootsPerConnection,
		"max-filling-roots-per-connection",
		c.MaxFillingRootsPerConnection,
		"max filling Root objects per connection")

	flag.DurationVar(&c.MaxFillingTime,
		"max-filling-time",
		c.MaxFillingTime,
		"max time to fill a Root")

	flag.StringVar(&c.ListenTCP,
		"tcp",
		c.ListenTCP,
		"tcp listening address")

	flag.StringVar(&c.ListenUDP,
		"udp",
		c.ListenUDP,
		"udp listening address")

	flag.StringVar(&c.RPC,
		"rpc",
		c.RPC,
		"RPC listening address")

	flag.DurationVar(&c.ResponseTimeout,
		"response-timeout",
		c.ResponseTimeout,
		"response timeout")

	flag.DurationVar(&c.Pings,
		"pings",
		c.Pings,
		"pings interval")

	flag.BoolVar(&c.Public,
		"public",
		c.Public,
		"public server")

}

// Validate configurations. The Validate doesn't
// validates addresses (TCP, UDP or RPC)
func (c *Config) Validate() (err error) {

	// nothing to validate in the Logger configurations

	// container
	if c.Config != nil {
		if err = c.Config.Validate(); err != nil {
			return
		}
	}

	// nothing to validate in the Node configs

	return

}
