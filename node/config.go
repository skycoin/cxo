package node

import (
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/skycoin/skycoin/src/daemon/gnet"

	"github.com/skycoin/cxo/db"
	"github.com/skycoin/cxo/enc"
)

const (
	NAME  string = "node"
	DEBUG bool   = true

	//
	// defaults
	//

	// connection pools related defaults
	ADDRESS                     string        = "" // "" is auto
	PORT                        int           = 0  // 0 is auto
	MAX_CONNECTIONS             int           = 0  // 0 is unlimited
	MAX_MESSAGE_LENGTH          int           = 8192
	DIAL_TIMEOUT                time.Duration = 20 * time.Second
	READ_TIMEOUT                time.Duration = 0  // 0 is unlimited
	WRITE_TIMEOUT               time.Duration = 0  // 0 is unlimited
	EVENT_CHANNEL_SIZE          int           = 20 //
	BROADCAST_RESULT_SIZE       int           = 20 //
	CONNECTION_WRITE_QUEUE_SIZE int           = 20 //
	// node related defaults
	HANDSHAKE_TIMEOUT time.Duration = 20 * time.Second
)

type Config struct {
	// connection pool related configs
	Address                  string
	Port                     int
	MaxConnections           int
	MaxMessageLength         int
	DialTimeout              time.Duration
	ReadTimeout              time.Duration
	WriteTimeout             time.Duration
	EventChannelSize         int
	BroadcastResultSize      int
	ConnectionWriteQueueSize int
	// node related configs
	HandshakeTimeout time.Duration

	SecretKey string
	Name      string // [log prefix]

	DB      db.DB
	Encoder enc.Encoder

	Debug bool

	ReceiveCallback ReceiveCallback
}

// NewConfig retusn Config with default values
func NewConfig() *Config {
	return &Config{
		Address: ADDRESS,
		Port:    PORT,

		MaxConnections:           MAX_CONNECTIONS,
		MaxMessageLength:         MAX_MESSAGE_LENGTH,
		DialTimeout:              DIAL_TIMEOUT,
		ReadTimeout:              READ_TIMEOUT,
		WriteTimeout:             WRITE_TIMEOUT,
		EventChannelSize:         EVENT_CHANNEL_SIZE,
		BroadcastResultSize:      BROADCAST_RESULT_SIZE,
		ConnectionWriteQueueSize: CONNECTION_WRITE_QUEUE_SIZE,
		HandshakeTimeout:         HANDSHAKE_TIMEOUT,

		Name:  NAME,
		Debug: DEBUG,
	}
}

// Fill up Config using flags. The method doesn't include flag.Parse()
// function call
func (c *Config) FromFlags() {
	// connection pool related configs
	flag.StringVar(&c.Address,
		"a",
		ADDRESS,
		"listening address")
	flag.IntVar(&c.Port,
		"p",
		PORT,
		"port number")
	flag.IntVar(&c.MaxConnections,
		"max-conn",
		MAX_CONNECTIONS,
		"max connections (0 is unlimited)")
	flag.IntVar(&c.MaxMessageLength,
		"max-msg-len",
		MAX_MESSAGE_LENGTH,
		"max message length (0 is unlimited)")
	flag.DurationVar(&c.DialTimeout,
		"dt",
		DIAL_TIMEOUT,
		"dial timeout (0 is unlimited)")
	flag.DurationVar(&c.ReadTimeout,
		"rt",
		READ_TIMEOUT,
		"read timeout (0 is unlimited)")
	flag.DurationVar(&c.WriteTimeout,
		"wt",
		WRITE_TIMEOUT,
		"write timeout (0 is unlimited)")
	flag.IntVar(&c.EventChannelSize,
		"event-chan-szie",
		EVENT_CHANNEL_SIZE,
		"channel size for events")
	flag.IntVar(&c.BroadcastResultSize,
		"broadcast-result-size",
		BROADCAST_RESULT_SIZE,
		"breadcast result size")
	flag.IntVar(&c.ConnectionWriteQueueSize,
		"conn-write-queue-size",
		CONNECTION_WRITE_QUEUE_SIZE,
		"write queue size of connection")
	// node related configs
	flag.DurationVar(&c.HandshakeTimeout,
		"ht",
		HANDSHAKE_TIMEOUT,
		"handshake timeout (0 is unlimited)")

	flag.StringVar(&c.SecretKey,
		"sec",
		"",
		"hexadecimal encoded secret key")
	flag.StringVar(&c.Name,
		"name",
		"node",
		"name of node for logs")
	flag.BoolVar(&c.Debug,
		"d",
		DEBUG,
		"enable debug mode")
}

func (c *Config) gnetConfig() (gc gnet.Config) {
	gc.Address = c.Address
	gc.Port = uint16(c.Port)
	gc.MaxConnections = c.MaxConnections
	gc.MaxMessageLength = c.MaxMessageLength
	gc.DialTimeout = c.DialTimeout
	gc.ReadTimeout = c.ReadTimeout
	gc.WriteTimeout = c.WriteTimeout
	gc.EventChannelSize = c.EventChannelSize
	gc.BroadcastResultSize = c.BroadcastResultSize
	gc.ConnectionWriteQueueSize = c.ConnectionWriteQueueSize
	return
}

func (c *Config) HumanString() string {
	return fmt.Sprintf(`
	address:                     %s
	port:                        %s
	max connections:             %s
	max message length:          %s
	dial timeout:                %s
	read timeout:                %s
	write timeout:               %s
	event channel size:          %s
	broadcast result size:       %s
	connection write queue size: %s

	handshake timeout:           %s

	name:                        %s
	secret key:                  %s

`,
		c.humanAddress(),
		c.humanPort(),

		humanInt(c.MaxConnections),

		humanInt(c.MaxMessageLength),

		humanDuration(c.DialTimeout),
		humanDuration(c.ReadTimeout),
		humanDuration(c.WriteTimeout),

		humanInt(c.EventChannelSize),
		humanInt(c.BroadcastResultSize),
		humanInt(c.ConnectionWriteQueueSize),

		humanDuration(c.HandshakeTimeout),
		c.Name,
		c.humanSecretKey(),
	)
}

func (c *Config) humanAddress() string {
	if c.Address == "" {
		return "auto"
	}
	return c.Address
}

func (c *Config) humanPort() string {
	if c.Port == 0 {
		return "auto"
	}
	return strconv.Itoa(int(c.Port))
}

func (c *Config) humanSecretKey() string {
	if !c.Debug {
		return "[hidden]"
	}
	if c.SecretKey == "" {
		return "[not provided]"
	}
	return c.SecretKey
}

// where 0 is unlimited
func humanInt(i int) string {
	if i == 0 {
		return "unlimited"
	}
	return strconv.Itoa(i)
}

// where 0 is unlimited
func humanDuration(d time.Duration) string {
	if d == 0 {
		return "unlimited"
	}
	return d.String()
}
