package gnet

import (
	"crypto/tls"
	"flag"
	"fmt"
	"time"

	"github.com/skycoin/cxo/node/log"
)

// defaults
const (
	MaxConnections int = 10 * 1000 // default connections limit
	MaxMessageSize int = 16 * 1024 // default message size

	DialTimeout  time.Duration = 5 * time.Second     // default timeout
	ReadTimeout  time.Duration = 0 * 5 * time.Second // default timeout
	WriteTimeout time.Duration = 0 * 5 * time.Second // default timeout

	ReadQueueLen  int = 128 // default
	WriteQueueLen int = 128 // default

	RedialTimeout    time.Duration = 100 * time.Millisecond // default
	MaxRedialTimeout time.Duration = 5 * time.Second        // default
	DialsLimit       int           = 0                      // 0 - infinity

	ReadBufferSize  int = 0 * 4096 // default
	WriteBufferSize int = 0 * 4096 // default
)

// OnCreateConnection represents connectinig callback.
// The callback performed while *Conn created. The callback
// has nothing to do with underlying net.Conn(s)
type OnCreateConnection func(c *Conn)

// OnCloseConnection represents disconnecting callback.
// The callback performed while *Conn closed. The callback
// has nothing to do with underlying net.Conn(s)
type OnCloseConnection func(c *Conn)

// OnDial represetns dialing callback. The err argumetn
// represtns first error by which previous connection was
// closed. If this function returns an error it terminates
// dialing and connection. The err is nil if it's first
// dialing
type OnDial func(c *Conn, err error) error

// A Config represents configurations of a Pool
type Config struct {
	// MaxConnections is connections limit for both, incoming and
	// outgoing connections
	MaxConnections int
	// MaxMessageSize limits messages to send. If the size is 0
	// then no limit used. But if the size is set and a connection
	// receive a message greater then the size, the Pool close
	// this connection. Sending a message greater then the size
	// causes panic (!)
	MaxMessageSize int //

	DialTimeout  time.Duration // dial timeout
	ReadTimeout  time.Duration // read deadline
	WriteTimeout time.Duration // write deadline

	ReadQueueLen  int // reading queue length
	WriteQueueLen int // // writing queue length

	RedialTimeout    time.Duration // timeout between redials
	MaxRedialTimeout time.Duration // increase RedilaTimeout up to it every time
	DialsLimit       int           // max dials allowed

	ReadBufferSize  int // reading buffer
	WriteBufferSize int // writing buffer

	OnCreateConnection OnCreateConnection // on conenct callback
	OnCloseConnection  OnCloseConnection  // on disconnect callback

	// OnDial is callback that called before dialing and
	// the callback used to stop dialing and make
	// some work you may need. The callback receive
	// error by which connection fails. For first dialing
	// (for outgoing connections) the error is nil. If the
	// callback returns non-nil error then dialing will not
	// be performed and connection (gnet.Conn) will be closed
	// While the callback called, underlying net.Conn already
	// closed or not created yet
	OnDial OnDial

	TLSConfig *tls.Config // use TLS if it's not nil

	Logger log.Logger // use the logger
}

// NewConfig creates new configurations
// filled with default values
func NewConfig() (c Config) {
	c.MaxConnections = MaxConnections
	c.MaxMessageSize = MaxMessageSize

	c.DialTimeout = DialTimeout
	c.ReadTimeout = ReadTimeout
	c.WriteTimeout = WriteTimeout

	c.ReadQueueLen = ReadQueueLen
	c.WriteQueueLen = WriteQueueLen

	c.RedialTimeout = RedialTimeout
	c.MaxRedialTimeout = MaxRedialTimeout
	c.DialsLimit = DialsLimit

	c.ReadBufferSize = ReadBufferSize
	c.WriteBufferSize = WriteBufferSize
	return
}

// FromFlags obtains configurations from command
// line flags. Call the method before `flag.Parse()`
// For example
//
//     c := gnet.NewConfig()
//     c.FromFlags()
//     flag.Parse()
//
func (c *Config) FromFlags() {
	flag.IntVar(&c.MaxConnections,
		"max-conns",
		c.MaxConnections,
		"max connections (0 - no limit)")
	flag.IntVar(&c.MaxMessageSize,
		"max-msg-size",
		c.MaxMessageSize,
		"max message size (0 - no limit)")

	flag.DurationVar(&c.DialTimeout,
		"dial-timeout",
		c.DialTimeout,
		"dial timeout (0 - no limit)")
	flag.DurationVar(&c.ReadTimeout,
		"read-timeout",
		c.ReadTimeout,
		"read timeout (0 - no limit)")
	flag.DurationVar(&c.WriteTimeout,
		"write-timeout",
		c.WriteTimeout,
		"write timeout (0 - no limit)")

	flag.IntVar(&c.ReadQueueLen,
		"read-qlen",
		c.ReadQueueLen,
		"read queue length")
	flag.IntVar(&c.WriteQueueLen,
		"write-qlen",
		c.WriteQueueLen,
		"write queue length")

	flag.DurationVar(&c.RedialTimeout,
		"redial-timeout",
		c.RedialTimeout,
		"redial timeout")
	flag.DurationVar(&c.MaxRedialTimeout,
		"max-redial-timeout",
		c.MaxRedialTimeout,
		"max redial timeout")
	flag.IntVar(&c.DialsLimit,
		"redials-limit",
		c.DialsLimit,
		"redials limit (0 - no limit)")

	flag.IntVar(&c.ReadBufferSize,
		"read-buf",
		c.ReadBufferSize,
		"read buffer size (0 - unbuffered)")
	flag.IntVar(&c.WriteBufferSize,
		"write-buf",
		c.WriteBufferSize,
		"write buffer size (0 - unbuffered)")
}

// Validate the Config
func (c *Config) Validate() (err error) {
	if c.MaxConnections < 0 {
		err = fmt.Errorf("negative MaxConnections %v", c.MaxConnections)
	} else if c.MaxMessageSize < 0 {
		err = fmt.Errorf("negative MaxMessageSize %v", c.MaxMessageSize)
	} else if c.DialTimeout < 0 {
		err = fmt.Errorf("negative DialTimeout %v", c.DialTimeout)
	} else if c.ReadTimeout < 0 {
		err = fmt.Errorf("negative ReadTimeout %v", c.ReadTimeout)
	} else if c.WriteTimeout < 0 {
		err = fmt.Errorf("negative WriteTimeout %v", c.WriteTimeout)
	} else if c.ReadQueueLen < 0 {
		err = fmt.Errorf("negative ReadQueueLen %v", c.ReadQueueLen)
	} else if c.WriteQueueLen < 0 {
		err = fmt.Errorf("negative WriteQueueLen %v", c.WriteQueueLen)
	} else if c.RedialTimeout < 0 {
		err = fmt.Errorf("negative RedialTimeout %v", c.RedialTimeout)
	} else if c.MaxRedialTimeout < 0 {
		err = fmt.Errorf("negative MaxRedialTimeout %v", c.MaxRedialTimeout)
	} else if c.DialsLimit < 0 {
		err = fmt.Errorf("negative DialsLimit %v", c.DialsLimit)
	} else if c.ReadBufferSize < 0 {
		err = fmt.Errorf("negative ReadBufferSize %v", c.ReadBufferSize)
	} else if c.WriteBufferSize < 0 {
		err = fmt.Errorf("negative WriteBufferSize %v", c.WriteBufferSize)
	}
	return
}
