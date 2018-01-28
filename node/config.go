package node

import (
	"flag"
	"fmt"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/cxo/skyobject/registry"
)

// default configurations
const (
	Prefix          string        = "[node] "
	MaxConnections  int           = 1000 * 1000
	MaxFillingTime  time.Duration = 10 * time.Minute
	MaxHeads        int           = 10
	ListenTCP       string        = ":8870"
	ListenUDP       string        = "" // don't listen
	RPCAddress      string        = ":8871"
	ResponseTimeout time.Duration = 59 * time.Second
	Pings           time.Duration = 118 * time.Second
	Public          bool          = false
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
// possible to reject a received Root returning
// error by this function.
//
// The callback never called if (1) node has a
// newer Root, (2) node already have the Root,
// (3) node or connection doesn't interest the
// Root.
//
// If a Root received then it doesn't mean that
// the Root will be filled and if this callback
// called then it can be silently dropped by
// some reasons.
//
// Short words, the callback called if a received
// Root is going to be filled
type OnRootReceivedFunc func(c *Conn, r *registry.Root) (reject error)

// OnRootFilledFunc represents callback that
// called when new Root filled and can be used.
// The callback called once per Root.
type OnRootFilledFunc func(n *Node, r *registry.Root)

// OnFillingBreaksFunc represents callback that
// called when a new Root object can't be filled.
// The callback called with non-full Root (that
// can't be used), and with filling error.
// There is no way to resume the filling. If
// a remote peer push this Root object again,
// then the Root can be filled (or can be not).
type OnFillingBreaksFunc func(n *Node, r *registry.Root, err error)

// OnConnectFunc represents callback that called
// when a connection created and established. It's
// possible to terminate connection returning error
type OnConnectFunc func(c *Conn) (terminate error)

// OnDisconnectFunc represents callback that called
// when a connection closed. The callback called
// with closing reason. The reason is nil if the
// connection closed manually. The Conn argument
// is closed connection that can't be used (e.g.
// is can be used partially).
type OnDisconnectFunc func(c *Conn, reason error)

// OnSubscribeRemoteFunc represents callback that
// called when a remote peer subscribes to some
// feed of the Node. The Node accepts the subscription
// if the Node share the feed. You can add the feed
// to the Node inside the callback to allow the
// subscription even if the Node doesn't share the
// feed. Also, you can reject this subscription
// returning error
type OnSubscribeRemoteFunc func(c *Conn, feed cipher.PubKey) (reject error)

// OnUnsubscribeRemoteFunc represent callback that
// called when a remote peer unsubscribes from
// some feed. The callback have informative role
// only
type OnUnsubscribeRemoteFunc func(c *Conn, feed cipher.PubKey)

// NetConfig represents configurations of
// a TCP or UDP network
type NetConfig struct {
	// Listen is listening address. Blank string
	// disables listening. Use ":0" to listen on all
	// interfaces (and ipv4 + ipv6 if possible) with
	// OS-choosed ports.
	Listen string

	// Discovery is lsit of addresses of a discovery
	// severs to connect to. The discovery servers
	// used to connect Nodes between to share feeds
	// they interested in
	Discovery Addresses

	// ResponseTimeout is timeout for requests (see
	// (*Conn).Subscribe, (*Conn).Unsubscribe and
	// (*Conn).RemoteFeeds). And for pings too.
	// Set it to zero, for infinity waiting. The
	// timeout doesn't close connections. A request
	// returns ErrTimeout if time is out.
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
}

// A Config represents configurations
// of the Node. To create Config filled
// with default values use NewConfig
// method. The recommended way of creating
// Config is the NewConfig. This Config
// contains skyobject.Config.
type Config struct {

	// Logger contains configurations of logger
	Logger log.Config

	// Config is configurations of skyobject.Container.
	// Use nil for defaults.
	*skyobject.Config

	// MaxConnections is limit of connections.
	// Set it to zero to disable the limit.
	MaxConnections int

	// MaxHeads is limit of heads per feed. A head
	// allocates some resources in the Node. And
	// this limit required to protect the Node against
	// misuse and against intentional DoS attacks. A
	// user can generate infinity number of heads.
	// This limit dismiss heads Root objects of
	// which received most early. It's impossible to
	// know which heads will be rejected, and which
	// not. But in reality, it is not a problem.
	//
	// For example, if a node subscribed to feed A,
	// and a peer receive 10 Root object one by one
	// with different heads. And, if the MaxHeads
	// limit is 5, then first 5 received Root objects
	// will be rejected when last 5 received.
	//
	// In reality, this porblem can't occur or be
	// a real problem
	//
	// Set the limit to zero, to turn it off.
	MaxHeads int

	// MaxFillingTime is time limit for filling of
	// a Root object. If a Root object fills too
	// long (longe then this limit), then it will
	// be dropped. Set it to zero to disable the
	// limit.
	MaxFillingTime time.Duration

	// RPC is RPC listening address. Empty string
	// disables RPC.
	RPC string

	//
	// Networks
	//

	// TCP configurations
	TCP NetConfig

	// UDP configurations
	UDP NetConfig

	//
	// Connection callbacks
	//

	// OnConenct is callback for new established
	// connections. See OnConnectFunc for details.
	OnConnect OnConnectFunc
	// OnDisconenct is callback for closed
	// connections. See OnDisconnectFunc for details.
	OnDisconnect OnDisconnectFunc

	//
	// Discovery
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

	//
	// Subscription related callbacks
	//

	// OnSubscribeRemote is callback for remote
	// subscriptions. See OnSubscribeRemote for
	// details
	OnSubscribeRemote OnSubscribeRemoteFunc

	// OnUnsubscribeRemote is callback for remote
	// unsubscriptions. See OnUnsubscribeRemote for
	// details
	OnUnsubscribeRemote OnUnsubscribeRemoteFunc

	//
	// Root related callbacks
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
	c.MaxFillingTime = MaxFillingTime
	c.MaxHeads = MaxHeads

	c.TCP.Listen = ListenTCP
	c.TCP.Pings = Pings
	c.TCP.ResponseTimeout = ResponseTimeout

	c.UDP.Listen = ListenUDP
	c.UDP.ResponseTimeout = ResponseTimeout

	c.RPC = RPCAddress
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

	flag.DurationVar(&c.MaxFillingTime,
		"max-filling-time",
		c.MaxFillingTime,
		"max time to fill a Root")

	flag.IntVar(&c.MaxHeads,
		"max-heads",
		c.MaxHeads,
		"max heads of a feed allowed")

	flag.StringVar(&c.RPC,
		"rpc",
		c.RPC,
		"RPC listening address")

	// TCP

	flag.StringVar(&c.TCP.Listen,
		"tcp",
		c.TCP.Listen,
		"tcp listening address")

	flag.DurationVar(&c.TCP.ResponseTimeout,
		"tcp-response-timeout",
		c.TCP.ResponseTimeout,
		"response timeout of TCP connections")

	flag.DurationVar(&c.TCP.Pings,
		"tcp-pings",
		c.TCP.Pings,
		"pings interval of TCP connections")

	// UDP

	flag.StringVar(&c.UDP.Listen,
		"udp",
		c.UDP.Listen,
		"udp listening address")

	flag.DurationVar(&c.UDP.ResponseTimeout,
		"udp-response-timeout",
		c.UDP.ResponseTimeout,
		"response timeout of UDP connections")

	flag.DurationVar(&c.UDP.Pings,
		"udp-pings",
		c.UDP.Pings,
		"pings interval of UDP connections")

	// public

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
