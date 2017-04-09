package gnet

import (
	"bytes"
	"errors"
	"io/ioutil"
	"testing"
	"time"

	"github.com/skycoin/cxo/node/log"
)

//
// helper functons
//

func resetConfig(c *Config) {
	c.PingInterval = 0                         // prevent
	c.WriteTimeout = 0                         // start
	c.ReadTimeout = 0                          // sending pings
	c.ReadBufferSize, c.WriteBufferSize = 0, 0 // don't use buffers
	c.MaxConnections = 0                       // avoid using sem channel
}

func testConfig() (c Config) {
	c = NewConfig()
	resetConfig(&c)
	if testing.Verbose() {
		c.Logger = log.NewLogger("[test] ", true)
	} else {
		c.Logger = log.NewLogger("", false)
		c.Logger.SetOutput(ioutil.Discard)
	}
	return
}

func testConfigName(name string) (c Config) {
	c = NewConfig()
	resetConfig(&c)
	if testing.Verbose() {
		c.Logger = log.NewLogger("["+name+"] ", true)
	} else {
		c.Logger = log.NewLogger("", false)
		c.Logger.SetOutput(ioutil.Discard)
	}
	return
}

func testPool() (p *Pool) {
	p = NewPool(testConfig())
	return
}

func testPoolName(name string) (p *Pool) {
	p = NewPool(testConfigName(name))
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
		defer p.Close()
		defer shouldPanic(t)
		_ = p.encodeMessage(&Empty{})
	})
	t.Run("registered", func(t *testing.T) {
		p := testPool()
		defer p.Close()
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
		p := NewPool(c)
		defer p.Close()
		p.Register(NewPrefix("ANYM"), &Any{})
		defer shouldPanic(t)
		p.encodeMessage(&Any{"FOUR+"})
	})
}

func TestPool_acquire(t *testing.T) {
	t.Run("no limit", func(t *testing.T) {
		c := testConfig()
		c.MaxConnections = 0
		p := NewPool(c)
		defer p.Close()
		if !p.acquire() {
			t.Error("can't acquire without limit")
		}
		if len(p.sem) != 0 {
			t.Error("invalid length of limiting channel")
		}
	})
	t.Run("limited", func(t *testing.T) {
		c := testConfig()
		c.MaxConnections = 1000
		p := NewPool(c)
		defer p.Close()
		for i := 0; i < 1000; i++ {
			if !p.acquire() {
				t.Error("can't acquire:", i)
			}
		}
		if len(p.sem) != 1000 {
			t.Error("invalid length of limiting channel")
		}
		if p.acquire() {
			t.Error("can acquire in spite of limit")
		}
	})
}

func TestPool_acquireBlock(t *testing.T) {
	t.Run("no limit", func(t *testing.T) {
		c := testConfig()
		c.MaxConnections = 0
		p := NewPool(c)
		defer p.Close()
		if p.sem != nil {
			t.Error("create sem channel wihtout limit of connections")
		}
		p.acquireBlock()
	})
	t.Run("limited", func(t *testing.T) {
		c := testConfig()
		c.MaxConnections = 1000
		p := NewPool(c)
		for i := 0; i < 1000; i++ {
			p.acquireBlock()
		}
		if len(p.sem) != 1000 {
			t.Error("invalid length of limiting channel")
		}
		p.Close()
		p.acquireBlock() // don't block closed channel
	})
}

func TestPool_release(t *testing.T) {
	t.Run("no limit", func(t *testing.T) {
		c := testConfig()
		c.MaxConnections = 0
		p := NewPool(c)
		defer p.Close()
		p.release()
	})
	t.Run("limited", func(t *testing.T) {
		c := testConfig()
		c.MaxConnections = 1000
		p := NewPool(c)
		defer p.Close()
		if !p.acquire() {
			t.Error("can't acquire")
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

func connsCount(p *Pool) int {
	p.RLock()
	defer p.RUnlock()
	return len(p.conns)
}

func closePool(p *Pool) (err error) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		p.Close()
	}()
	select {
	case <-time.After(100 * time.Millisecond):
		err = errors.New("slow closing")
	case <-done:
	}
	return
}

// listening server and two clients connected to the server
// all has Any registered
func testS2C(t *testing.T, sn, c1n, c2n string) (s, c1, c2 *Pool) {
	s = testPoolName(sn)
	c1 = testPoolName(c1n)
	c2 = testPoolName(c2n)
	if err := s.Listen(""); err != nil { // any address
		t.Fatal(err)
	}
	s.Register(NewPrefix("ANYM"), &Any{})
	c1.Register(NewPrefix("ANYM"), &Any{})
	c2.Register(NewPrefix("ANYM"), &Any{})
	if err := c1.Connect(s.Address()); err != nil {
		if err := closePool(s); err != nil {
			t.Fatal(err)
		}
	}
	if err := c2.Connect(s.Address()); err != nil {
		if err := closePool(c1); err != nil {
			t.Fatal(err)
		}
		if err := closePool(s); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(50 * time.Millisecond)
	if cc := connsCount(s); cc != 2 {
		if err := closePool(c2); err != nil {
			t.Fatal(err)
		}
		if err := closePool(c1); err != nil {
			t.Fatal(err)
		}
		if err := closePool(s); err != nil {
			t.Fatal(err)
		}
		t.Fatalf("invalid connections map length: %d", cc)
	}
	time.Sleep(100 * time.Millisecond)
	return
}

func TestPool_BroadcastExcept(t *testing.T) {
	s, h, e := testS2C(t, "send", "recv1", "recv2")
	// broadcast
	defer func() {
		if err := closePool(e); err != nil {
			t.Error(err)
		}
		if err := closePool(h); err != nil {
			t.Error(err)
		}
		if err := closePool(s); err != nil {
			t.Error(err)
		}
	}()
	var except string
	if fc := firstConnection(e); fc == nil {
		t.Error("wrong connections count")
	} else {
		except = fc.conn.LocalAddr().String()
	}
	t.Log("BroadcastExcept")
	s.BroadcastExcept(&Any{"data"}, except)
	select {
	case m := <-h.Receive():
		if a, ok := m.Value.(*Any); !ok {
			t.Errorf("wrog=ng message received: %T", m.Value)
		} else if a.Value != "data" {
			t.Error("wrong messagereceived: ", a.Value)
		}
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
	s, c1, c2 := testS2C(t, "send", "recv1", "recv2")
	defer func() {
		if err := closePool(c2); err != nil {
			t.Error(err)
		}
		if err := closePool(c1); err != nil {
			t.Error(err)
		}
		if err := closePool(s); err != nil {
			t.Error(err)
		}
	}()
	t.Log("Broadcast")
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
	t.Run("arbitrary", func(t *testing.T) {
		p := testPool()
		defer p.Close()
		if err := p.Listen(""); err != nil {
			t.Error(err)
		}
	})
	t.Run("invalid address", func(t *testing.T) {
		p := testPool()
		defer p.Close()
		if err := p.Listen("-1-2-3-4-5-"); err == nil {
			t.Error("listen with invalid address")
		}
	})
}

func firstConnection(p *Pool) (fc *Conn) {
	p.RLock()
	defer p.RUnlock()
	for _, c := range p.conns {
		fc = c
		return
	}
	return
}

func TestPool_listen(t *testing.T) {
	var serverConn, clientConn = make(chan string, 1), make(chan string, 1)
	pc := testConfigName("server")
	pc.ConnectionHandler = func(c *Conn) { serverConn <- c.Addr() }
	p := NewPool(pc)
	defer func() {
		if err := closePool(p); err != nil {
			t.Error(err)
		}
	}()
	if err := p.Listen(""); err != nil {
		t.Error(err)
		return
	}
	cc := testConfigName("client")
	cc.ConnectionHandler = func(c *Conn) { clientConn <- c.Addr() }
	c := NewPool(cc)
	defer func() {
		if err := closePool(c); err != nil {
			t.Error(err)
		}
	}()
	if err := c.Connect(p.Address()); err != nil {
		t.Error(err)
		return
	}
	select {
	case sc := <-serverConn:
		select {
		case cc := <-clientConn:
			// take a look at server connection
			if fs := firstConnection(c); fs == nil {
				t.Error("missing client connection")
			} else if la := fs.conn.LocalAddr().String(); la != sc {
				t.Error("wrong connection address", la, sc)
			}
			// take a look at client connection
			if fc := firstConnection(p); fc == nil {
				t.Error("missing server connection")
			} else if la := fc.conn.LocalAddr().String(); la != cc {
				t.Error("wrong connection address", la, cc)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("slow")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("slow")
	}
}

func TestPool_handleConnection(t *testing.T) {
	//
}

func TestPool_removeConnection(t *testing.T) {
	//
}

func TestPool_Connect(t *testing.T) {
	t.Run("fields", func(t *testing.T) {
		sc := testConfigName("server")
		sc.MaxConnections = 1000
		s := NewPool(sc)
		if err := s.Listen(""); err != nil {
			t.Fatal(err)
		}
		cc := testConfigName("client")
		cc.MaxConnections = 1000
		c := NewPool(cc)
		if err := c.Connect(s.Address()); err != nil {
			t.Fatal(err)
		}
		var sf, cf *Conn
		if sf = firstConnection(s); sf == nil {
			t.Fatal("missing connection")
		}
		if cf = firstConnection(c); cf == nil {
			t.Fatal("missing connection")
		}
		if sf.Addr() != cf.conn.LocalAddr().String() {
			t.Error("missmatch adresses")
		}
		if cf.Addr() != sf.conn.LocalAddr().String() {
			t.Error("missmatch adresses")
		}
		if len(s.sem) != 2 { // connection + next accept
			t.Error("wrong sem len of server:", len(s.sem))
		}
		if len(c.sem) != 1 {
			t.Error("wrong sem len of client:", len(s.sem))
		}
		// close the server
		s.Close()
		// TODO
	})
}

func TestPool_Register(t *testing.T) {
	t.Run("norm", func(t *testing.T) {
		defer shouldNotPanic(t)
		p := testPool()
		p.Register(NewPrefix("ANYM"), &Any{})
	})
	t.Run("type twice", func(t *testing.T) {
		defer shouldPanic(t)
		p := testPool()
		p.Register(NewPrefix("ANYM"), &Any{})
		p.Register(NewPrefix("SOME"), &Any{})
	})
	t.Run("prefix twice", func(t *testing.T) {
		type Some struct {
			Int int64
		}
		defer shouldPanic(t)
		p := testPool()
		p.Register(NewPrefix("ANYM"), &Any{})
		p.Register(NewPrefix("ANYM"), &Some{})
	})
	t.Run("invalid type", func(t *testing.T) {
		type Some interface {
			Some() int64
		}
		defer shouldPanic(t)
		p := testPool()
		p.Register(NewPrefix("Some"), Some(nil))
	})
	t.Run("invalid prefix", func(t *testing.T) {
		defer shouldPanic(t)
		p := testPool()
		p.Register(Prefix{'-', '-', '-', '>'}, &Any{})
	})
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
	t.Run("test", func(t *testing.T) {
		p := testPool()
		// todo
		_ = p
	})
}
