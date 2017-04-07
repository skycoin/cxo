package gnet

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
	"time"
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

func testConfigName(name string) (c Config) {
	c = testConfig()
	c.Name = name
	return
}

func testPool() (p *Pool) {
	p = NewPool(testConfig(), nil)
	return
}

func testPoolName(name string) (p *Pool) {
	p = NewPool(testConfigName(name), nil)
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

var receivedBy chan string

type ReceivedBy struct{}

func (*ReceivedBy) Handle(ctx MessageContext, _ interface{}) (_ error) {
	receivedBy <- ctx.Addr()
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
	t.Run("no limit", func(t *testing.T) {
		c := testConfig()
		c.MaxConnections = 0
		p := NewPool(c, nil)
		if !p.acquire() {
			t.Error("can't acquire without limit")
		}
		if len(p.sem) != 0 {
			t.Error("invalid length of limiting channel")
		}
	})
	t.Run("limited", func(t *testing.T) {
		c := testConfig()
		c.MaxConnections = 1
		p := NewPool(c, nil)
		if !p.acquire() {
			t.Error("can't acquire without limit")
		}
		if len(p.sem) != 1 {
			t.Error("invalid length of limiting channel")
		}
		if p.acquire() {
			t.Error("can acquire in spite of limit")
		}
	})
}

func TestPool_acquireBlock(t *testing.T) {
	// TODO: test blocking
}

func TestPool_release(t *testing.T) {
	t.Run("no limit", func(t *testing.T) {
		c := testConfig()
		c.MaxConnections = 0
		p := NewPool(c, nil)
		p.release()
	})
	t.Run("limited", func(t *testing.T) {
		c := testConfig()
		c.MaxConnections = 1
		p := NewPool(c, nil)
		if !p.acquire() {
			t.Error("can't acquire without limit")
		}
		if len(p.sem) != 1 {
			t.Error("invalid length of limiting channel")
		}
		p.release()
		if len(p.sem) != 0 {
			t.Error("invalid length of limiting channel")
		}
	})
}

// listening server and two clients connected to the server
// all has ReceivedBy registered
func testS2C(sn, c1n, c2n string) (s, c1, c2 *Pool, err error) {
	s = testPoolName(sn)
	c1 = testPoolName(c1n)
	c2 = testPoolName(c2n)
	if err = s.Listen(""); err != nil { // any address
		return
	}
	receivedBy = make(chan string, 10)
	s.Register(NewPrefix("RCVD"), &ReceivedBy{})
	c1.Register(NewPrefix("RCVD"), &ReceivedBy{})
	c2.Register(NewPrefix("RCVD"), &ReceivedBy{})
	address := s.Address()
	if err = c1.Connect(address); err != nil {
		s.Close()
		return
	}
	if err = c2.Connect(address); err != nil {
		s.Close()
		c1.Close()
		return
	}
	time.Sleep(50 * time.Millisecond)
	if len(s.conns) != 2 {
		s.Close()
		c1.Close()
		c2.Close()
		err = fmt.Errorf("invalid connections map length: %d", len(s.conns))
	}
	return
}

func TestPool_BroadcastExcept(t *testing.T) {
	s, c1, c2, err := testS2C("send", "recv1", "recv2")
	if err != nil {
		t.Error(err)
		return
	}
	defer s.Close()
	defer c1.Close()
	defer c2.Close()
	var (
		except string
		handle string
	)
	for a := range s.conns {
		if except == "" {
			except = a
		} else {
			handle = a
		}
		break
	}
	s.BroadcastExcept(&ReceivedBy{}, except)
	time.Sleep(100 * time.Microsecond)
	if len(receivedBy) != 1 {
		t.Error("received wrong times")
	}
	ra := <-receivedBy
	if ra == except {
		t.Error("handled by wrong side")
	} else if ra != handle {
		t.Error("handled by unknown side")
	}
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
