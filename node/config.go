package node

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/skycoin/skycoin/src/daemon/gnet"
)

// Default values for Config returned using NewConfig
const (
	NAME  string = "node" // default Name
	DEBUG bool   = false  // default Debug

	//
	// defaults
	//

	ADDRESS string = "" // default Address
	PORT    int    = 0  // default Port

	MAX_INCOMING_CONNECTIONS int = 64 // default MaxIncomingConnections
	MAX_OUTGOUNG_CONNECTIONS int = 64 // default MaxOutgoingConnections

	MAX_PENDING_CONNECTIONS int = 64 // default MaxPendingConnections

	MAX_MESSAGE_LENGTH          int = 8192 // default MaxMessageLength
	EVENT_CHANNEL_SIZE          int = 4096 // default EventChannelSize
	BROADCAST_RESULT_SIZE       int = 16   // default BroadcastResultSize
	CONNECTION_WRITE_QUEUE_SIZE int = 32   // default ConnectionWriteQueueSize

	DIAL_TIMEOUT  time.Duration = 20 * time.Second // default DialTimeout
	READ_TIMEOUT  time.Duration = 0                // default ReadTimeout
	WRITE_TIMEOUT time.Duration = 0                // default WriteTimeout

	// PING_INTERVAL is interval for send service PING message to
	// keep connection alive. If you set it to zero then pinging is
	// disabled. It's impossible to use disabled ping-interval and
	// some non-zero read and write timeout. Because if we set some
	// read timeout and no messages sent - connection is closed. Ping
	// messages are sent to incoming connections only (i.e. from feed to
	// subscribers). Thus, every feed can keep alive its subscribers, but
	// there isn't cross-ping-pong sending, that can't keep connection more
	// clever but can increase network pressure. If node is subscriber
	// and ping-interval is disabled it sends pong message back to feed
	// anyway (in response to ping from feed)
	PING_INTERVAL time.Duration = 0

	// HANDSHAKE_TIMEOUT is default HandshakeTimeout
	HANDSHAKE_TIMEOUT time.Duration = 40 * time.Second

	// MESSAGE_HANDLING_RATE is default MessageHandlingRate
	MESSAGE_HANDLING_RATE time.Duration = 50 * time.Millisecond

	// MANAGE_EVENTS_CHANNEL_SIZE size of managing events like
	// List, Connect, Terminate etc
	MANAGE_EVENTS_CHANNEL_SIZE = 1024
)

// A Config represents set of configurations of a Node
type Config struct {
	Name  string // Name is used as log-prefix
	Debug bool   // Debug enables debug logs

	// Address is listening address. Empty string allows OS to
	// choose address itself
	Address string
	// Port is listenong port. Zero allows OS to choose
	// port itself
	Port int

	// MaxIncomingConnections is maximum nuber of subscribers of the Node.
	// Set it to zero to disable listening
	MaxIncomingConnections int
	// MaxOutgoingConnections is maximum number of subscriptions of the Node.
	// If it is zero then the Node can't subscribe to anothe one
	MaxOutgoingConnections int

	// MaxPendingConnections is maximum number of incoming and outgoing
	// connections together that are not established yet (which are performing
	// handshake)
	MaxPendingConnections int

	MaxMessageLength         int // limit of message size
	EventChannelSize         int // size of events queue
	BroadcastResultSize      int // size of results queue
	ConnectionWriteQueueSize int // size of write queue for every connection

	DialTimeout  time.Duration // dial timeout, use 0 to ignore timeout
	ReadTimeout  time.Duration // read timeout, use 0 to system's default
	WriteTimeout time.Duration // writ timeout, use 0 to system's default

	PingInterval time.Duration // ping interval, use 0 to disable

	// HandshakeTimeout is timeout after which connection will be
	// closed if handshake is not performed. The hndshake is four-step
	// procedure (send->receive->send->receive or receive->send->receive->send).
	// Use 0 to ignore timeout
	HandshakeTimeout time.Duration

	// MessageHandlingRate is interval of handling messages. Set it to zero
	// if you want to handle messages in infinity loop
	MessageHandlingRate time.Duration

	// ManageEventsChannelSize is csize of channel for managing events
	// such as list connections, connect, terminate connection, etc
	ManageEventsChannelSize int
}

// NewConfig returns Config filled down with default values
func NewConfig() (c Config) {
	c.Name = NAME
	c.Debug = DEBUG

	c.Address = ADDRESS
	c.Port = PORT

	c.MaxIncomingConnections = MAX_INCOMING_CONNECTIONS
	c.MaxOutgoingConnections = MAX_OUTGOUNG_CONNECTIONS

	c.MaxPendingConnections = MAX_PENDING_CONNECTIONS

	c.MaxMessageLength = MAX_MESSAGE_LENGTH
	c.EventChannelSize = EVENT_CHANNEL_SIZE
	c.BroadcastResultSize = BROADCAST_RESULT_SIZE
	c.ConnectionWriteQueueSize = CONNECTION_WRITE_QUEUE_SIZE

	c.DialTimeout = DIAL_TIMEOUT
	c.ReadTimeout = READ_TIMEOUT
	c.WriteTimeout = WRITE_TIMEOUT

	c.PingInterval = PING_INTERVAL

	c.HandshakeTimeout = HANDSHAKE_TIMEOUT

	c.MessageHandlingRate = MESSAGE_HANDLING_RATE

	c.ManageEventsChannelSize = MANAGE_EVENTS_CHANNEL_SIZE
	return
}

// Validate config values
func (c *Config) Validate() (err error) {
	_, err = net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", c.Address, c.Port))
	if err != nil {
		return
	}
	if c.MaxPendingConnections <= 0 {
		err = errors.New("no pending connections allowed: 0")
		return
	}
	if c.MaxMessageLength <= 0 {
		err = errors.New("max mesage length is zero of below")
	}
	// if we accept connections ...
	if c.MaxIncomingConnections > 0 {
		// ...and read or write interval is set...
		if c.ReadTimeout > 0 || c.WriteTimeout > 0 {
			// .. we must have some ping interval...
			if c.PingInterval == 0 {
				err = errors.New("ping interval required")
				return
			} else if c.PingInterval > c.ReadTimeout {
				err = errors.New("ping interval is greater than read timeout")
				return
			} else if c.PingInterval > c.WriteTimeout {
				err = errors.New("ping interval is greater than write timeout")
				return
			}
		}
	}
	return
}

// FromFlags obtains configurations from command line flags.
// The method doesn't call flag.Parse
func (c *Config) FromFlags() {
	flag.StringVar(&c.Name,
		"name",
		NAME,
		"node name (log prefix)")
	flag.BoolVar(&c.Debug,
		"d",
		DEBUG,
		"show debug logs")

	flag.StringVar(&c.Address,
		"a",
		ADDRESS,
		"listening address (set to empty string for arbitrary assignment)")
	flag.IntVar(&c.Port,
		"p",
		PORT,
		"listening port number (set to zero for arbitrary assignment)")

	flag.IntVar(&c.MaxIncomingConnections,
		"max-incoming",
		MAX_INCOMING_CONNECTIONS,
		"maximum incoming connections (set to zero to desable listening)")
	flag.IntVar(&c.MaxOutgoingConnections,
		"max-outgoing",
		MAX_OUTGOUNG_CONNECTIONS,
		"maximum outgoing connections (set to zero to diable subscriptions)")

	flag.IntVar(&c.MaxPendingConnections,
		"max-pending",
		MAX_PENDING_CONNECTIONS,
		"maximum pending connections (must be above zero)")

	flag.IntVar(&c.MaxMessageLength,
		"max-msg-len",
		MAX_MESSAGE_LENGTH,
		"maximum message size (must be above zero)")
	flag.IntVar(&c.EventChannelSize,
		"event-queue",
		EVENT_CHANNEL_SIZE,
		"event queue size")
	flag.IntVar(&c.BroadcastResultSize,
		"result-queue",
		BROADCAST_RESULT_SIZE,
		"result queue size")
	flag.IntVar(&c.ConnectionWriteQueueSize,
		"write-queue",
		CONNECTION_WRITE_QUEUE_SIZE,
		"write queue size of every connection")

	flag.DurationVar(&c.DialTimeout,
		"dt",
		DIAL_TIMEOUT,
		"dial timeout (set to zero to ignore)")
	flag.DurationVar(&c.ReadTimeout,
		"rt",
		READ_TIMEOUT,
		"reading timeout (set to zero to use system's default)")
	flag.DurationVar(&c.WriteTimeout,
		"wt",
		WRITE_TIMEOUT,
		"write timeout (set to zero to use system's default)")

	flag.DurationVar(&c.PingInterval,
		"ping",
		PING_INTERVAL,
		"ping interval (set to zero to disable pinging)")

	flag.DurationVar(&c.HandshakeTimeout,
		"ht",
		HANDSHAKE_TIMEOUT,
		"handshake timeout (set to zero to disable)")

	flag.DurationVar(&c.MessageHandlingRate,
		"rate",
		MESSAGE_HANDLING_RATE,
		"messages handling rate (set to zero to immediate handling)")

	flag.IntVar(&c.ManageEventsChannelSize,
		"man-chan-size",
		MANAGE_EVENTS_CHANNEL_SIZE,
		"size of managing events channel")
}

func humanBool(b bool, t, f string) string {
	if b {
		return t
	}
	return f
}

func humanDur(d time.Duration, zero string) string {
	if d == 0 {
		return zero
	}
	return d.String()
}

func humanInt(i int, zero string) string {
	if i == 0 {
		return zero
	}
	return strconv.Itoa(i)
}

func humanString(s, empty string) string {
	if s == "" {
		return empty
	}
	return s
}

// HumanString retursn human readable representation of Config
func (c *Config) HumanString() (s string) {
	s = fmt.Sprintf(`	name:       %s
	debug logs: %s

	address:    %s
	port:       %s

	max incoming connections: %s
	max outgoing connections: %s
	max pending connections:  %d

	max message length:          %d
	event channel size:          %d
	broadcast result size:       %d
	connection write queue size: %d

	dial timeout:  %s
	read timeout:  %s
	write timeout: %s

	ping interval: %s

	handshake timeout: %s

	messages handling rate: %s

	managing events channel size: %d`,

		c.Name,
		humanBool(c.Debug, "enabled", "disabled"),

		humanString(c.Address, "auto"),
		humanInt(c.Port, "auto"),

		humanInt(c.MaxIncomingConnections, "disabled"),
		humanInt(c.MaxOutgoingConnections, "disabled"),
		c.MaxPendingConnections,

		c.MaxMessageLength,
		c.EventChannelSize,
		c.BroadcastResultSize,
		c.ConnectionWriteQueueSize,

		humanDur(c.DialTimeout, "ignore"),
		humanDur(c.ReadTimeout, "system default"),
		humanDur(c.WriteTimeout, "system default"),

		humanDur(c.PingInterval, "disabled"),

		humanDur(c.HandshakeTimeout, "ignore"),

		c.MessageHandlingRate.String(),
		c.ManageEventsChannelSize)

	return
}

// gnetConfig generates gnet.Config based on Config
func (c *Config) gnetConfig() (gc gnet.Config) {
	gc.Address = c.Address
	gc.Port = uint16(c.Port)
	gc.MaxConnections = c.MaxIncomingConnections + c.MaxOutgoingConnections
	gc.MaxMessageLength = c.MaxMessageLength
	gc.DialTimeout = c.DialTimeout
	gc.ReadTimeout = c.ReadTimeout
	gc.WriteTimeout = c.WriteTimeout
	gc.EventChannelSize = c.EventChannelSize
	gc.BroadcastResultSize = c.BroadcastResultSize
	gc.ConnectionWriteQueueSize = c.ConnectionWriteQueueSize
	return
}
