package gnet

import (
	"flag"
	"time"

	"github.com/skycoin/cxo/node/log"
)

// config defaults
const (
	MaxConnections  int           = 1024
	MaxMessageSize  int           = 8192
	DialTimeout     time.Duration = 25 * time.Second
	ReadTimeout     time.Duration = 25 * time.Second
	WriteTimeout    time.Duration = 25 * time.Second
	ReadBufferSize  int           = 4096
	WriteBufferSize int           = 4096
	ReadQueueSize   int           = 64 * 256 // 1/4
	WriteQueueSize  int           = 64
	PingInterval    time.Duration = 23 * time.Second

	minPingInterval time.Duration = 400 * time.Millisecond
	minTimeout      time.Duration = 2 * minPingInterval
)

// ConnectionHandler represents function that used
// to do some work on new connections
type ConnectionHandler func(c *Conn)

type Config struct {
	// MaxConnections - incoming and outgoing
	// together
	MaxConnections int
	// MaxMessageSize is limit of message size to
	// prevent reading any malformed big message.
	// A Pool will panic if you try to send message
	// exceeds the limit. If a Pool receive a message
	// that exceeds the limit then the Pool closes
	// connection from which the message coming from
	MaxMessageSize int

	// DialTimeout, ReadTimeout and WriteTimeout are
	// used to read, write and dial with provided
	// timeout. If timeout is zero then no timeout
	// used (no time limit)
	DialTimeout  time.Duration // dial timeout
	ReadTimeout  time.Duration // read timeout
	WriteTimeout time.Duration // write timeout

	// ReadBufferSize and WriteBufferSize are used for
	// buffered reading and writing. If the value is
	// zero then no buffers are used. If the value is
	// negative then default buffer size is used
	ReadBufferSize  int
	WriteBufferSize int

	// ReadQueueSize is size of reading queue. The queue
	// is shared for all connections. All connections
	// read messages and put them to the queue
	ReadQueueSize int
	// WriteQueueSize is size of connection related queue.
	// Every connection has its own write queue
	WriteQueueSize int

	// PingInterval used to send pings every
	// PingInterval time. If the interval is zero
	// then sending of pings is not used. But the
	// interval can't be less then
	// min(ReadTimeout, WriteTimeout). I.e. if
	// a timeout grater then zero, then the
	// interval is greater too
	PingInterval time.Duration // ping interval

	// ConnectionHandler is a handler that called whe
	// a new connections was created
	ConnectionHandler ConnectionHandler

	// Logger to use. If it's nil then default logger used
	Logger log.Logger
}

// NewConfig returns Config filled with defaults valus
// and given name
func NewConfig() (c Config) {
	c.MaxConnections = MaxConnections
	c.MaxMessageSize = MaxMessageSize
	c.DialTimeout = DialTimeout
	c.ReadTimeout = ReadTimeout
	c.WriteTimeout = WriteTimeout
	c.ReadBufferSize = ReadBufferSize
	c.WriteBufferSize = WriteBufferSize
	c.ReadQueueSize = ReadQueueSize
	c.WriteQueueSize = WriteQueueSize
	c.PingInterval = PingInterval
	return
}

// replace negative values wiht defaults;
// set PingInterval to (min(ReadTimeout,
// WriteTimeout) - minPingInterval) if the
// interval is lesser
func (c *Config) applyDefaults() {
	if c.MaxConnections < 0 {
		c.MaxConnections = MaxConnections
	}
	if c.MaxMessageSize < 0 {
		c.MaxMessageSize = MaxMessageSize
	}
	if c.DialTimeout < 0 {
		c.DialTimeout = DialTimeout
	}
	if c.ReadTimeout < 0 {
		c.ReadTimeout = ReadTimeout
	}
	if c.WriteTimeout < 0 {
		c.WriteTimeout = WriteTimeout
	}
	if c.ReadBufferSize < 0 {
		c.ReadBufferSize = ReadBufferSize
	}
	if c.WriteBufferSize < 0 {
		c.WriteBufferSize = WriteBufferSize
	}
	if c.ReadQueueSize < 0 {
		c.ReadQueueSize = ReadQueueSize
	}
	if c.WriteQueueSize < 0 {
		c.WriteQueueSize = WriteQueueSize
	}
	if c.PingInterval < 0 {
		c.PingInterval = PingInterval
	}
	// min timeouts
	if c.ReadTimeout > 0 && c.ReadTimeout < minTimeout {
		c.ReadTimeout = minTimeout
	}
	if c.WriteTimeout > 0 && c.WriteTimeout < minTimeout {
		c.WriteTimeout = minTimeout
	}
	// ping interval
	if c.PingInterval > 0 && c.PingInterval < minPingInterval {
		c.PingInterval = minPingInterval
	}
	return
}

// FromFlags is helper to obtain value s from
// commandline flags. Values of the struct will
// be set after flag.Parse() call. I.e. call the
// method before flag.Parse(), use the Config after
// flag.Parse()
func (c *Config) FromFlags() {
	flag.IntVar(&c.MaxConnections,
		"max-conn",
		MaxConnections,
		"max connections")
	flag.IntVar(&c.MaxMessageSize,
		"max-msg-size",
		MaxMessageSize,
		"max message size")

	flag.DurationVar(&c.DialTimeout,
		"dial-timeout",
		DialTimeout,
		"dial timeout")
	flag.DurationVar(&c.ReadTimeout,
		"read-timeout",
		ReadTimeout,
		"read timeout")
	flag.DurationVar(&c.WriteTimeout,
		"write-timeout",
		WriteTimeout,
		"write timeout")

	flag.IntVar(&c.ReadBufferSize,
		"read-buf",
		ReadBufferSize,
		"reading buffer size")
	flag.IntVar(&c.WriteBufferSize,
		"write-buf",
		WriteBufferSize,
		"writing buffer size")

	flag.IntVar(&c.ReadQueueSize,
		"readq",
		ReadQueueSize,
		"shared reading queue size")
	flag.IntVar(&c.WriteQueueSize,
		"writeq",
		WriteQueueSize,
		"write queue size")

	flag.DurationVar(&c.PingInterval,
		"ping",
		PingInterval,
		"interval to send pings")
}
