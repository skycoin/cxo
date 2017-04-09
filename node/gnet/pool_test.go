package gnet

import (
	"io/ioutil"
	"testing"

	"github.com/skycoin/cxo/node/log"
)

func testConfig() (c Config) {
	c = testConfigName("")
	return
}

func testConfigName(name string) (c Config) {
	c = NewConfig()
	c.ReadTimeout, c.WriteTimeout = 0, 0
	c.ReadBufferSize, c.WriteBufferSize = 0, 0
	c.PingInterval = 0
	if testing.Verbose() {
		c.Logger = log.NewLogger("["+name+"] ", true)
	} else {
		c.Logger = log.NewLogger("", false)
		c.Logger.SetOutput(ioutil.Discard)
	}
	return
}

// Any is type for tests
type Any struct{ Value string }

func TestNewPool(t *testing.T) {
	// TODO: high priority
}

func TestPool_Listen(t *testing.T) {
	// TODO: high priority
}

func TestPool_Connect(t *testing.T) {
	// TODO: high priority
}

func TestPool_Disconnect(t *testing.T) {
	// TODO: high priority
}

func TestPool_BroadcastExcept(t *testing.T) {
	// TODO: high priority
}

func TestPool_Broadcast(t *testing.T) {
	// TODO: high priority
}

func TestPool_Register(t *testing.T) {
	// TODO: high priority
}

func TestPool_Address(t *testing.T) {
	// TODO: high priority
}

func TestPool_Connections(t *testing.T) {
	// TODO: high priority
}

func TestPool_IsConnExist(t *testing.T) {
	// TODO: high priority
}

func TestPool_Receive(t *testing.T) {
	// TODO: high priority
}

func TestPool_Close(t *testing.T) {
	// TODO: high priority
}
