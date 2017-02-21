package node

import (
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/daemon/gnet"
)

func TestNewConfig(t *testing.T) {
	c := NewConfig()
	if c == nil {
		t.Fatal("NewConfig retusn nil")
	}
	if c.Address != ADDRESS ||
		c.Port != PORT ||
		c.MaxIncomingConnections != MAX_INCOMING_CONNECTIONS ||
		c.MaxOutgoingConnections != MAX_OUTGOING_CONNECTIONS ||
		c.MaxMessageLength != MAX_MESSAGE_LENGTH ||
		c.DialTimeout != DIAL_TIMEOUT ||
		c.ReadTimeout != READ_TIMEOUT ||
		c.WriteTimeout != WRITE_TIMEOUT ||
		c.EventChannelSize != EVENT_CHANNEL_SIZE ||
		c.BroadcastResultSize != BROADCAST_RESULT_SIZE ||
		c.ConnectionWriteQueueSize != CONNECTION_WRITE_QUEUE_SIZE ||
		c.HandshakeTimeout != HANDSHAKE_TIMEOUT {
		t.Error("wrong default values for NewConfig")
	}
}

func TestConfig_FromFlags(t *testing.T) {
	// local
	const (
		ADDRESS                     = "192.168.0.1"
		PORT                        = "1599"
		MAX_INCOMING_CONNECTIONS    = "666"
		MAX_OUTGOING_CONNECTIONS    = "777"
		MAX_MESSAGE_LENGTH          = "20"
		DIAL_TIMEOUT                = "5s"
		READ_TIMEOUT                = "6s"
		WRITE_TIMEOUT               = "7s"
		EVENT_CHANNEL_SIZE          = "21"
		BROADCAST_RESULT_SIZE       = "22"
		CONNECTION_WRITE_QUEUE_SIZE = "23"
		HANDSHAKE_TIMEOUT           = "8s"
	)

	c := NewConfig()
	c.FromFlags()

	flag.Set("a", ADDRESS)
	flag.Set("p", PORT)
	flag.Set("max-in", MAX_INCOMING_CONNECTIONS)
	flag.Set("max-out", MAX_OUTGOING_CONNECTIONS)
	flag.Set("max-msg-len", MAX_MESSAGE_LENGTH)
	flag.Set("dt", DIAL_TIMEOUT)
	flag.Set("rt", READ_TIMEOUT)
	flag.Set("wt", WRITE_TIMEOUT)
	flag.Set("event-chan-szie", EVENT_CHANNEL_SIZE)
	flag.Set("broadcast-result-size", BROADCAST_RESULT_SIZE)
	flag.Set("conn-write-queue-size", CONNECTION_WRITE_QUEUE_SIZE)
	flag.Set("ht", HANDSHAKE_TIMEOUT)

	// namespace isolation
	func(c *Config) {
		const (
			ADDRESS                     = "192.168.0.1"
			PORT                        = 1599
			MAX_INCOMING_CONNECTIONS    = 666
			MAX_OUTGOING_CONNECTIONS    = 777
			MAX_MESSAGE_LENGTH          = 20
			DIAL_TIMEOUT                = 5 * time.Second
			READ_TIMEOUT                = 6 * time.Second
			WRITE_TIMEOUT               = 7 * time.Second
			EVENT_CHANNEL_SIZE          = 21
			BROADCAST_RESULT_SIZE       = 22
			CONNECTION_WRITE_QUEUE_SIZE = 23
			HANDSHAKE_TIMEOUT           = 8 * time.Second
		)
		if c.Address != ADDRESS ||
			c.Port != PORT ||
			c.MaxIncomingConnections != MAX_INCOMING_CONNECTIONS ||
			c.MaxOutgoingConnections != MAX_OUTGOING_CONNECTIONS ||
			c.MaxMessageLength != MAX_MESSAGE_LENGTH ||
			c.DialTimeout != DIAL_TIMEOUT ||
			c.ReadTimeout != READ_TIMEOUT ||
			c.WriteTimeout != WRITE_TIMEOUT ||
			c.EventChannelSize != EVENT_CHANNEL_SIZE ||
			c.BroadcastResultSize != BROADCAST_RESULT_SIZE ||
			c.ConnectionWriteQueueSize != CONNECTION_WRITE_QUEUE_SIZE ||
			c.HandshakeTimeout != HANDSHAKE_TIMEOUT {
			t.Error("wrong configs given from flags")
			t.Log(c.HumanString())
		}
	}(c)

}

func cmpConfigGnetConfig(c *Config, gc *gnet.Config, mc int) bool {
	return gc.Address == c.Address ||
		gc.Port == uint16(c.Port) ||
		gc.MaxConnections == mc ||
		gc.MaxMessageLength == c.MaxMessageLength ||
		gc.DialTimeout == c.DialTimeout ||
		gc.ReadTimeout == c.ReadTimeout ||
		gc.WriteTimeout == c.WriteTimeout ||
		gc.EventChannelSize == c.EventChannelSize ||
		gc.BroadcastResultSize == c.BroadcastResultSize ||
		gc.ConnectionWriteQueueSize == c.ConnectionWriteQueueSize ||
		gc.DisconnectCallback == nil ||
		gc.ConnectCallback == nil
}

func TestConfig_gnetConfig(t *testing.T) {
	var (
		c  *Config     = NewConfig()
		gc gnet.Config = c.gnetConfig()
	)
	if !cmpConfigGnetConfig(c, &gc, 0) {
		t.Error("(*Config).gnetConfig returns wrong result")
	}
}

func TestConfig_gnetConfigInflow(t *testing.T) {
	var (
		c  *Config     = NewConfig()
		gc gnet.Config = c.gnetConfig()
	)
	if !cmpConfigGnetConfig(c, &gc, c.MaxIncomingConnections) {
		t.Error("(*Config).gnetConfigInflow returns wrong result")
	}
}

func TestConfig_gnetConfigFeed(t *testing.T) {
	var (
		c  *Config     = NewConfig()
		gc gnet.Config = c.gnetConfig()
	)
	if !cmpConfigGnetConfig(c, &gc, c.MaxOutgoingConnections) {
		t.Error("(*Config).gnetConfigFeed returns wrong result")
	}
}

func TestConfig_humanPort(t *testing.T) {
	c := NewConfig()
	c.Port = 0
	if c.humanPort() != "auto" {
		t.Error("(*Config).humanPort doen't returns 'auto'" +
			" if port is zero")
	}
}

func TestConfig_humanAddress(t *testing.T) {
	c := NewConfig()
	c.Address = ""
	if c.humanAddress() != "auto" {
		t.Error("(*Config).humanAddress doen't returns 'auto'" +
			" if address is empty")
	}
}

func Test_humanInt(t *testing.T) {
	if s := humanInt(100); s != "100" {
		t.Error("humanInt(100) error: want 100, got ", s)
	}
	if s := humanInt(0); s != "unlimited" {
		t.Errorf(`humanInt(100) error: want "unlimited", got %q`, s)
	}
}

func Test_humanDuration(t *testing.T) {
	if s := humanDuration(1 * time.Second); s != "1s" {
		t.Error("humanDuration(1 * time.Second) error: want 1s, got ", s)
	}
	if s := humanDuration(0); s != "unlimited" {
		t.Errorf(`humanDuration(0) error: want "unlimited", got %q`, s)
	}
}

//
// debug print example
//

func ExampleConfig_HumanString() {
	c := NewConfig()
	fmt.Println(c.HumanString())

	// Output:
	//
	// 	address:                     127.0.0.1
	// 	port:                        7899
	// 	max subscriptions:           unlimited
	// 	max subscribers:             unlimited
	// 	max message length:          8192
	// 	dial timeout:                20s
	// 	read timeout:                unlimited
	// 	write timeout:               unlimited
	// 	event channel size:          20
	// 	broadcast result size:       20
	// 	connection write queue size: 20
	//
	// 	handshake timeout:           20s
}
