package gnet

import (
	"bytes"
	"io/ioutil"
	"testing"
)

//
// helper functons
//

func testConfig() (c Config) {
	c = NewConfig("test", nil)
	c.PingInterval = 0 // prevent
	c.WriteTimeout = 0 // start
	c.ReadTimeout = 0  // sending pings
	if testing.Verbose() {
		c.Debug = true
	} else {
		c.Debug = false
		c.Out = ioutil.Discard
	}
	return
}

func testPool() (p *Pool) {
	p = NewPool(testConfig(), nil)
	return
}

//
// test helper functions
//

func shouldPanic(t *testing.T) {
	if recover() == nil {
		t.Error("missisng panic")
	}
}

func shouldNotPanic(t *testing.T) {
	if recover() != nil {
		t.Error("unexpected panic")
	}
}

//
// helper types
//

type Empty struct{}

func (*Empty) Handle(MessageContext, interface{}) (_ error) {
	return
}

type Big struct {
	Value string
}

func (*Big) Handle(MessageContext, interface{}) (_ error) {
	return
}

//
// test cases
//

func TestPool_encodeMessage(t *testing.T) {
	t.Run("unregistered", func(t *testing.T) {
		p := testPool()
		defer shouldPanic(t)
		_ = p.encodeMessage(&Empty{})
	})
	t.Run("registered", func(t *testing.T) {
		p := testPool()
		p.Register(NewPrefix("EMPT"), &Empty{})
		data := p.encodeMessage(&Empty{})
		if len(data) != PrefixLength+4 {
			t.Error("malformed encoded message")
			return
		}
		if string(data[:PrefixLength]) != "EMPT" {
			t.Error("wrong prefix")
		}
		if bytes.Compare(data[PrefixLength:], []byte{0, 0, 0, 0}) != 0 {
			t.Error("wrong message length")
		}
	})
	t.Run("size limit", func(t *testing.T) {
		c := testConfig()
		c.MaxMessageSize = 4
		p := NewPool(c, nil)
		p.Register(NewPrefix("BIGM"), &Big{})
		defer shouldPanic(t)
		p.encodeMessage(&Big{"FOUR+"})
	})
}

func TestPool_acquire(t *testing.T) {
	//
}

func TestPool_acquireBlock(t *testing.T) {
	//
}

func TestPool_release(t *testing.T) {
	//
}

func TestPool_BroadcastExcept(t *testing.T) {
	//
}

func TestPool_Broadcast(t *testing.T) {
	//
}

func TestPool_Listen(t *testing.T) {
	//
}

func TestPool_listen(t *testing.T) {
	//
}

func TestPool_handleConnection(t *testing.T) {
	//
}

func TestPool_removeConnection(t *testing.T) {
	//
}

func TestPool_Connect(t *testing.T) {
	//
}

func TestPool_Register(t *testing.T) {
	//
}

func TestPool_Disconnect(t *testing.T) {
	//
}

func TestPool_Address(t *testing.T) {
	//
}

func TestPool_Close(t *testing.T) {
	//
}

func TestPool_Connections(t *testing.T) {
	//
}

func TestPool_HandleMessages(t *testing.T) {
	//
}

func TestNewPool(t *testing.T) {
	//
}

func TestPool_sendPings(t *testing.T) {
	//
}
