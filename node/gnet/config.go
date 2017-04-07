package gnet

import (
	"time"
)

// config defaults
const (
	MaxConnections  int           = 1024
	MaxMessageSize  int           = 8192
	DialTimeout     time.Duration = 5 * time.Second
	ReadTimeout     time.Duration = 5 * time.Second
	WriteTimeout    time.Duration = 5 * time.Second
	ReadBufferSize  int           = 4096
	WriteBufferSize int           = 4096
	ReadQueueSize   int           = 64 * 256 // 1/4
	WriteQueueSize  int           = 64
	PingInterval    time.Duration = 5 * time.Second
	Debug           bool          = false
)

// ConnectionHandler represents function that used
// to do some work on new connections
type ConnectionHandler func(c *Conn)

type Config struct {
	// MaxConnections - incoming and outgoing
	// together
	MaxConnections int
	// MaxMessageSize is limit of message size to
	// prevent reading any malformed big message
	MaxMessageSize int

	// DialTimeout, ReadTimeout and WriteTimeout are
	// used to read, write and dial with provided
	// timeout. If timeout is zero then no timeout
	// used (no time limit). The ReadTimeout is hard
	// but the WriteTimeout can be x2 greater then
	// provided
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

	Debug bool   // print debug logs
	Name  string // name for logs (used as prefix)

	// ConnectionHandler is a handler that called whe
	// a new connections was created
	ConnectionHandler ConnectionHandler
}

// NewConfig returns Config filled with defaults valus
// and given name
func NewConfig(name string, handler ConnectionHandler) (c Config) {
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
	c.Debug = Debug
	c.Name = name
	c.ConnectionHandler = handler
	return
}

// replace negative values wiht defaults;
// set PingInterval to min(ReadTimeout, WriteTimeout)
// if the interval is less then the minimum
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
	var mt time.Duration
	if c.ReadTimeout < c.WriteTimeout {
		mt = c.WriteTimeout
	} else {
		mt = c.ReadTimeout
	}
	if c.PingInterval < mt {
		c.PingInterval = mt
	}
	return
}
