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

type Any struct {
	Value string
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
		p.Register(NewPrefix("ANYM"), &Any{})
		defer shouldPanic(t)
		p.encodeMessage(&Any{"FOUR+"})
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
// all has Any registered
func testS2C(sn, c1n, c2n string) (s, c1, c2 *Pool, err error) {
	s = testPoolName(sn)
	c1 = testPoolName(c1n)
	c2 = testPoolName(c2n)
	if err = s.Listen(""); err != nil { // any address
		return
	}
	s.Register(NewPrefix("ANYM"), &Any{})
	c1.Register(NewPrefix("ANYM"), &Any{})
	c2.Register(NewPrefix("ANYM"), &Any{})
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
	s, h, e, err := testS2C("send", "recv1", "recv2")
	if err != nil {
		t.Error(err)
		return
	}
	defer s.Close() // broadcast
	defer h.Close() // handle
	defer e.Close() // except
	var except string
	if len(e.conns) != 1 {
		t.Error("wrong connections size:", len(except))
		return
	}
	for _, c := range e.conns {
		except = c.conn.LocalAddr().String()
	}
	s.BroadcastExcept(&Any{"data"}, except)
	select {
	case <-h.Receive():
		select {
		case <-e.Receive():
			t.Error("received by excepted connection")
		case <-time.After(100 * time.Millisecond): // to be sure
		}
	case <-e.Receive():
		t.Error("received by excepted connection")
	case <-time.After(100 * time.Millisecond):
		t.Error("slow")
	}
}

func TestPool_Broadcast(t *testing.T) {
	s, c1, c2, err := testS2C("send", "recv1", "recv2")
	if err != nil {
		t.Error(err)
		return
	}
	defer s.Close()  // broadcast
	defer c1.Close() // receive
	defer c2.Close() // receive
	s.Broadcast(&Any{"data"})
	select {
	case <-c1.Receive():
		select {
		case <-c2.Receive():
		case <-time.After(100 * time.Millisecond):
			t.Error("slow")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("slow")
	}
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

func TestPool_Receive(t *testing.T) {
	//
}

func TestNewPool(t *testing.T) {
	//
}

func TestPool_sendPings(t *testing.T) {
	//
}
