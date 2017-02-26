package node

import (
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/daemon/gnet"
)

func TestConfig_NewConfig(t *testing.T) {
	var c Config = NewConfig()
	if c.Name != NAME {
		t.Error("wrong defalt value for Name: ", c.Name)
	}
	if c.Debug != DEBUG {
		t.Error("wrong defalt value for Debug: ", c.Debug)
	}
	if c.Address != ADDRESS {
		t.Error("wrong defalt value for Address: ", c.Address)
	}
	if c.Port != PORT {
		t.Error("wrong defalt value for Port: ", c.Port)
	}
	if c.MaxIncomingConnections != MAX_INCOMING_CONNECTIONS {
		t.Error("wrong defalt value for MaxIncomingConnections: ",
			c.MaxIncomingConnections)
	}
	if c.MaxOutgoingConnections != MAX_OUTGOUNG_CONNECTIONS {
		t.Error("wrong defalt value for MaxOutgoingConnections: ",
			c.MaxOutgoingConnections)
	}
	if c.MaxPendingConnections != MAX_PENDING_CONNECTIONS {
		t.Error("wrong defalt value for MaxPendingConnections: ",
			c.MaxPendingConnections)
	}
	if c.MaxMessageLength != MAX_MESSAGE_LENGTH {
		t.Error("wrong defalt value for MaxMessageLength: ", c.MaxMessageLength)
	}
	if c.EventChannelSize != EVENT_CHANNEL_SIZE {
		t.Error("wrong defalt value for EventChannelSize: ", c.EventChannelSize)
	}
	if c.BroadcastResultSize != BROADCAST_RESULT_SIZE {
		t.Error("wrong defalt value for BroadcastResultSize: ",
			c.BroadcastResultSize)
	}
	if c.ConnectionWriteQueueSize != CONNECTION_WRITE_QUEUE_SIZE {
		t.Error("wrong defalt value for ConnectionWriteQueueSize: ",
			c.ConnectionWriteQueueSize)
	}
	if c.DialTimeout != DIAL_TIMEOUT {
		t.Error("wrong defalt value for DialTimeout: ", c.DialTimeout)
	}
	if c.ReadTimeout != READ_TIMEOUT {
		t.Error("wrong defalt value for ReadTimeout: ", c.ReadTimeout)
	}
	if c.WriteTimeout != WRITE_TIMEOUT {
		t.Error("wrong defalt value for WriteTimeout: ", c.WriteTimeout)
	}
	if c.HandshakeTimeout != HANDSHAKE_TIMEOUT {
		t.Error("wrong defalt value for HandshakeTimeout: ", c.HandshakeTimeout)
	}
	if c.MessageHandlingRate != MESSAGE_HANDLING_RATE {
		t.Error("wrong defalt value for MessageHandlingRate: ",
			c.MessageHandlingRate)
	}
	if c.ManageEventsChannelSize != MANAGE_EVENTS_CHANNEL_SIZE {
		t.Error("wroong default value for ManageEventsChannelSize: ",
			c.ManageEventsChannelSize)
	}
}

func TestConfig_Validate(t *testing.T) {
	var (
		c   Config
		err error
	)
	t.Run("address", func(t *testing.T) {
		c = NewConfig()
		c.Address = "__invalid__"
		if err = c.Validate(); err == nil {
			t.Errorf("Validate allows invalid address: %s:%d",
				c.Address,
				c.Port)
		}
	})
	t.Run("port", func(t *testing.T) {
		c = NewConfig()
		c.Port = -90
		if err = c.Validate(); err == nil {
			t.Errorf("Validate allows invalid port: %s:%d",
				c.Address,
				c.Port)
		}
	})
	t.Run("pending", func(t *testing.T) {
		c = NewConfig()
		c.MaxPendingConnections = 0
		if err = c.Validate(); err == nil {
			t.Error("Validate allows zero pending connections")
		}
		c.MaxPendingConnections = -10
		if err = c.Validate(); err == nil {
			t.Error("Validate allows negative number of pending connections: ",
				c.MaxPendingConnections)
		}
	})
	t.Run("message length", func(t *testing.T) {
		c = NewConfig()
		c.MaxMessageLength = 0
		if err = c.Validate(); err == nil {
			t.Error("Validate allows zero message length")
		}
		c.MaxMessageLength = -10
		if err = c.Validate(); err == nil {
			t.Error("Validate allows negative message length: ",
				c.MaxMessageLength)
		}
	})
	t.Run("new config", func(t *testing.T) {
		var c Config = NewConfig()
		if err = c.Validate(); err != nil {
			t.Error("NewConfig return invalid config: ", err)
		}
	})
}

func TestConfig_FromFlags(t *testing.T) {
	var c Config = NewConfig()
	c.FromFlags()

	flag.Set("name", "zorro")
	if c.Name != "zorro" {
		t.Error("wrong value for Name: ", c.Name)
	}
	flag.Set("d", "off")
	if c.Debug != false {
		t.Error("wrong value for Debug: ", c.Debug)
	}

	flag.Set("a", "192.168.0.1")
	if c.Address != "192.168.0.1" {
		t.Error("wrong defalt value for Address: ", c.Address)
	}
	flag.Set("p", "7789")
	if c.Port != 7789 {
		t.Error("wrong defalt value for Port: ", c.Port)
	}

	flag.Set("max-incoming", "10")
	if c.MaxIncomingConnections != 10 {
		t.Error("wrong defalt value for MaxIncomingConnections: ",
			c.MaxIncomingConnections)
	}
	flag.Set("max-outgoing", "11")
	if c.MaxOutgoingConnections != 11 {
		t.Error("wrong defalt value for MaxOutgoingConnections: ",
			c.MaxOutgoingConnections)
	}

	flag.Set("max-pending", "12")
	if c.MaxPendingConnections != 12 {
		t.Error("wrong defalt value for MaxPendingConnections: ",
			c.MaxPendingConnections)
	}

	flag.Set("max-msg-len", "1024")
	if c.MaxMessageLength != 1024 {
		t.Error("wrong defalt value for MaxMessageLength: ", c.MaxMessageLength)
	}
	flag.Set("event-queue", "13")
	if c.EventChannelSize != 13 {
		t.Error("wrong defalt value for EventChannelSize: ", c.EventChannelSize)
	}
	flag.Set("result-queue", "14")
	if c.BroadcastResultSize != 14 {
		t.Error("wrong defalt value for BroadcastResultSize: ",
			c.BroadcastResultSize)
	}
	flag.Set("write-queue", "15")
	if c.ConnectionWriteQueueSize != 15 {
		t.Error("wrong defalt value for ConnectionWriteQueueSize: ",
			c.ConnectionWriteQueueSize)
	}

	flag.Set("dt", "1s")
	if c.DialTimeout != 1*time.Second {
		t.Error("wrong defalt value for DialTimeout: ", c.DialTimeout)
	}
	flag.Set("rt", "2s")
	if c.ReadTimeout != 2*time.Second {
		t.Error("wrong defalt value for ReadTimeout: ", c.ReadTimeout)
	}
	flag.Set("wt", "3s")
	if c.WriteTimeout != 3*time.Second {
		t.Error("wrong defalt value for WriteTimeout: ", c.WriteTimeout)
	}

	flag.Set("ht", "4s")
	if c.HandshakeTimeout != 4*time.Second {
		t.Error("wrong defalt value for HandshakeTimeout: ", c.HandshakeTimeout)
	}

	flag.Set("rate", "5s")
	if c.MessageHandlingRate != 5*time.Second {
		t.Error("wrong defalt value for MessageHandlingRate: ",
			c.MessageHandlingRate)
	}

	flag.Set("man-chan-size", "87")
	if c.ManageEventsChannelSize != 87 {
		t.Error("wrong defalt value for ManageEventsChannelSize: ",
			c.ManageEventsChannelSize)
	}

}

func TestConfig_gnetConfig(t *testing.T) {
	var (
		c  Config      = NewConfig()
		gc gnet.Config = c.gnetConfig()
	)
	if gc.Address != c.Address {
		t.Error("wrong Address: ", gc.Address)
	}
	if gc.Port != uint16(c.Port) {
		t.Error("wrong Port: ", gc.Port)
	}
	if gc.MaxConnections != (c.MaxIncomingConnections +
		c.MaxOutgoingConnections) {
		t.Error("wrong MaxConnections: ", gc.MaxConnections)
	}
	if gc.MaxMessageLength != c.MaxMessageLength {
		t.Error("wrong MaxMessageLength: ", gc.MaxMessageLength)
	}
	if gc.DialTimeout != c.DialTimeout {
		t.Error("wrong DialTimeout: ", gc.DialTimeout)
	}
	if gc.ReadTimeout != c.ReadTimeout {
		t.Error("wrong ReadTimeout: ", gc.ReadTimeout)
	}
	if gc.WriteTimeout != c.WriteTimeout {
		t.Error("wrong WriteTimeout: ", gc.WriteTimeout)
	}
	if gc.EventChannelSize != c.EventChannelSize {
		t.Error("wrong EventChannelSize: ", gc.EventChannelSize)
	}
	if gc.BroadcastResultSize != c.BroadcastResultSize {
		t.Error("wrong BroadcastResultSize: ", gc.BroadcastResultSize)
	}
	if gc.ConnectionWriteQueueSize != c.ConnectionWriteQueueSize {
		t.Error("wrong ConnectionWriteQueueSize: ", gc.ConnectionWriteQueueSize)
	}
}

func ExampleConfig_HumanString() {
	var c Config = NewConfig()

	fmt.Println(c.HumanString())

	// Output:
	//
	// 	name:       node
	// 	debug logs: disabled
	//
	// 	address:    auto
	// 	port:       auto
	//
	// 	max incoming connections: 64
	// 	max outgoing connections: 64
	// 	max pending connections:  64
	//
	// 	max message length:          8192
	// 	event channel size:          4096
	// 	broadcast result size:       16
	// 	connection write queue size: 32
	//
	// 	dial timeout:  20s
	// 	read timeout:  system default
	// 	write timeout: system default
	//
	// 	handshake timeout: 40s
	//
	// 	messages handling rate: 50ms
	//
	// 	managing events channel size: 20
}
