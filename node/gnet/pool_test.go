package gnet

import (
	"io/ioutil"
	"net"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/skycoin/cxo/node/log"
)

// default timeout for some parts of tests
const TM time.Duration = 50 * time.Millisecond

func testConfig() Config { return testConfigName("") }

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

// a limited by time task

func timeLimit(tm time.Duration, limitedTask func()) (get bool) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		limitedTask()
	}()
	select {
	case <-done:
		get = true
	case <-time.After(tm):
	}
	return
}

// limited reading from struct{} channel

func readChan(tm time.Duration, c <-chan struct{}) (get bool) {
	select {
	case <-c:
		get = true
	case <-time.After(tm):
	}
	return
}

// limited closing of a pool

func closePool(t *testing.T, p *Pool, tm time.Duration) {
	get := timeLimit(tm, func() { p.Close() })
	if !get {
		if _, file, no, ok := runtime.Caller(1); ok {
			file = filepath.Base(file)
			t.Fatalf("closing of pool is too slow: %s:%d", file, no)
		} else {
			t.Fatal("closing of pool is too slow")
		}
	}
}

func shouldPanic(t *testing.T) {
	if recover() == nil {
		if _, file, no, ok := runtime.Caller(1); ok {
			file = filepath.Base(file)
			t.Fatalf("missing panicing: %s:%d", file, no)
		} else {
			t.Fatal("missing panicing")
		}
	}
}

func dial(t *testing.T, address string, tm time.Duration) (conn net.Conn) {
	var err error
	if conn, err = net.DialTimeout("tcp", address, tm); err != nil {
		if _, file, no, ok := runtime.Caller(1); ok {
			file = filepath.Base(file)
			t.Fatalf("unexpected dialing error (%s:%d): %v", file, no, err)
		} else {
			t.Fatal("unexpected dialing error:", err)
		}
	}
	return
}

func listen(t *testing.T) (l net.Listener) {
	var err error
	if l, err = net.Listen("tcp", ""); err != nil {
		if _, file, no, ok := runtime.Caller(1); ok {
			file = filepath.Base(file)
			t.Fatalf("unexpected listening error (%s:%d): %v", file, no, err)
		} else {
			t.Fatal("unexpected listening error:", err)
		}
	}
	return
}

func connectionsLength(p *Pool) int {
	p.cmx.RLock()
	defer p.cmx.RUnlock()
	return len(p.conns)
}

func firstConnection(p *Pool) (c *Conn) {
	p.cmx.RLock()
	defer p.cmx.RUnlock()
	for _, c = range p.conns {
		break
	}
	return
}

// Any is type for tests
type Any struct{ Value string }

func TestNewPool(t *testing.T) {
	t.Run("configs", func(t *testing.T) {
		t.Run("defaults", func(t *testing.T) {
			p := NewPool(Config{
				MaxConnections:  -1,
				MaxMessageSize:  -1,
				DialTimeout:     -1,
				ReadTimeout:     -1,
				WriteTimeout:    -1,
				ReadBufferSize:  -1,
				WriteBufferSize: -1,
				ReadQueueSize:   -1,
				WriteQueueSize:  -1,
				PingInterval:    -1,
			})
			if p.conf.MaxConnections != MaxConnections {
				t.Error("negative MaxConnections wasn't set to default value:",
					p.conf.MaxConnections)
			}
			if p.conf.MaxMessageSize != MaxMessageSize {
				t.Error("negative MaxMessageSize wasn't set to default value:",
					p.conf.MaxMessageSize)
			}
			if p.conf.DialTimeout != DialTimeout {
				t.Error("negative DialTimeout wasn't set to default value:",
					p.conf.DialTimeout)
			}
			if p.conf.ReadTimeout != ReadTimeout {
				t.Error("negative ReadTimeout wasn't set to default value:",
					p.conf.ReadTimeout)
			}
			if p.conf.WriteTimeout != WriteTimeout {
				t.Error("negative WriteTimeout wasn't set to default value:",
					p.conf.WriteTimeout)
			}
			if p.conf.ReadBufferSize != ReadBufferSize {
				t.Error("negative ReadBufferSize wasn't set to default value:",
					p.conf.ReadBufferSize)
			}
			if p.conf.WriteBufferSize != WriteBufferSize {
				t.Error("negative WriteBufferSize wasn't set to default value:",
					p.conf.WriteBufferSize)
			}
			if p.conf.ReadQueueSize != ReadQueueSize {
				t.Error("negative ReadQueueSize wasn't set to default value:",
					p.conf.ReadQueueSize)
			}
			if p.conf.WriteQueueSize != WriteQueueSize {
				t.Error("negative WriteQueueSize wasn't set to default value:",
					p.conf.WriteQueueSize)
			}
			if p.conf.PingInterval != PingInterval {
				t.Error("negative PingInterval wasn't set to default value:",
					p.conf.PingInterval)
			}
		})
		t.Run("min timeouts", func(t *testing.T) {
			p := NewPool(Config{
				ReadTimeout:  minTimeout - 1*time.Nanosecond,
				WriteTimeout: minTimeout - 1*time.Nanosecond,
				PingInterval: minPingInterval - 1*time.Nanosecond,
			})
			if p.conf.ReadTimeout != minTimeout {
				t.Error("ReadTimeout is lesser then allowed minimum")
			}
			if p.conf.WriteTimeout != minTimeout {
				t.Error("WriteTimeout is lesser then allowed minimum")
			}
			if p.conf.PingInterval != minPingInterval {
				t.Error("PingInterval is lesser then allowed minimum")
			}
		})
		t.Run("logger", func(t *testing.T) {
			p := NewPool(Config{})
			if p.Logger == nil {
				t.Error("logger of pool wasn't created")
			}
			l := log.NewLogger("prefix", false)
			p = NewPool(Config{Logger: l})
			if p.Logger != l {
				t.Error("logger of pool was replaced")
			}
		})
	})
	t.Run("immediate close", func(t *testing.T) {
		p := NewPool(Config{})
		closePool(t, p, TM)
	})
}

func TestPool_Listen(t *testing.T) {
	t.Run("invalid address", func(t *testing.T) {
		p := NewPool(testConfigName("Listen/invalid address"))
		defer p.Close()
		if err := p.Listen("-1-2-3-4-5-"); err == nil {
			t.Error("missing error listening on invalid address")
		}
	})
	t.Run("arbitrary address", func(t *testing.T) {
		p := NewPool(testConfigName("Listen/arbitrary address"))
		defer closePool(t, p, TM)
		if err := p.Listen(""); err != nil {
			t.Error("unexpected listeninig error:", err)
		}
	})
	t.Run("accept", func(t *testing.T) {
		p := NewPool(testConfigName("Listen/accept"))
		defer closePool(t, p, TM)
		if err := p.Listen(""); err != nil {
			t.Error("unexpected listeninig error:", err)
		}
		dial(t, p.l.Addr().String(), TM).Close()
	})
	t.Run("closed pool", func(t *testing.T) {
		p := NewPool(testConfigName("Listen/closed pool"))
		closePool(t, p, TM)
		if err := p.Listen("127.0.0.1:3000"); err == nil {
			t.Error("missing error listening on closed pool")
		} else if err != ErrClosed {
			t.Errorf("unexpected error: want %q, got %q", ErrClosed, err)
		}
	})
	t.Run("connections limit", func(t *testing.T) {
		connect := make(chan struct{}, 2)
		disconnect := make(chan struct{}, 2)
		conf := testConfigName("Listen/connections limit")
		conf.MaxConnections = 1 // 0 - unlimited
		conf.ConnectionHandler = func(*Conn) { connect <- struct{}{} }
		conf.DisconnectHandler = func(*Conn, error) { disconnect <- struct{}{} }
		p := NewPool(conf)
		defer closePool(t, p, TM)
		if err := p.Listen(""); err != nil {
			t.Fatal("unexpected listening error:", err)
		}
		c1 := dial(t, p.l.Addr().String(), TM)
		c2 := dial(t, p.l.Addr().String(), TM)
		if !readChan(TM, connect) {
			t.Fatal("connecting is too slow")
		}
		if connectionsLength(p) != 1 {
			t.Fatal("accepted out of limit")
		}
		c1.Close() // release
		if !readChan(TM, disconnect) {
			t.Error("disconnecting is too slow")
		}
		if !readChan(TM, connect) {
			t.Fatal("connecting is too slow")
		}
		if connectionsLength(p) != 1 {
			t.Fatal("accepted out of limit")
		}
		c2.Close()
		if !readChan(TM, disconnect) {
			t.Error("disconnecting is too slow")
		}
		if connectionsLength(p) != 0 {
			t.Error("doesn't release closed connections")
		}
	})
}

func TestPool_Connect(t *testing.T) {
	t.Run("invalid address", func(t *testing.T) {
		p := NewPool(testConfigName("Connect/invalid address"))
		defer closePool(t, p, TM)
		if err := p.Connect("-1-2-3-4-5-"); err == nil {
			t.Error("mising error connecting to invalid address")
		}
	})
	t.Run("closed pool", func(t *testing.T) {
		p := NewPool(testConfigName("Connect/closed pool"))
		closePool(t, p, TM)
		if err := p.Connect("127.0.0.1:3000"); err == nil {
			t.Error("missing error connecting on closed pool")
		} else if err != ErrClosed {
			t.Errorf("unexpected error: want %q, got %q", ErrClosed, err)
		}
	})
	t.Run("connections limit", func(t *testing.T) {
		conf := testConfigName("Listen/connections limit")
		conf.MaxConnections = 1 // 0 - unlimited
		p := NewPool(conf)
		defer closePool(t, p, TM)
		l1 := listen(t) // test listener
		defer l1.Close()
		if err := p.Connect(l1.Addr().String()); err != nil {
			t.Fatal("unexpected connecting error:", err)
		}
		l2 := listen(t) // another listener required because of unique addresses
		defer l2.Close()
		if err := p.Connect(l2.Addr().String()); err == nil {
			t.Fatal("allow connection out of limit")
		} else if err != ErrConnectionsLimit {
			t.Errorf("unexpected error: want %q, got %q",
				ErrConnectionsLimit, err)
		}
	})
	// TODO: low priority
	//   - dial timeout test
}

func TestPool_Disconnect(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		p := NewPool(testConfigName("Disconnect/not_found"))
		defer p.Close()
		if err := p.Disconnect("127.0.0.1:1599"); err == nil {
			t.Errorf("missing %q error", ErrNotFound)
		} else if err != ErrNotFound {
			t.Errorf("unexpected error: want %q, got %q", ErrNotFound, err)
		}
	})
	t.Run("found", func(t *testing.T) {
		disconnected := make(chan struct{}, 2)
		conf := testConfigName("Disconnect/found")
		conf.DisconnectHandler = func(*Conn, error) {
			disconnected <- struct{}{}
		}
		p := NewPool(conf)
		defer p.Close()
		l := listen(t) // test listener
		defer l.Close()
		if err := p.Connect(l.Addr().String()); err != nil {
			t.Fatal("unexpected connecting error:", err)
		}
		// unfortunately we can't use l.Addr().String to disconnect the
		// connection because it returns [::]:<port> but connection
		// stored in map using [::1]:<port>. There is a way to use
		// RemoteAdrr of the underlying net.Conn that is (*Conn).Addr()
		if err := p.Disconnect(firstConnection(p).Addr()); err != nil {
			t.Error("unexpected disconnecting error:", err)
		}
		if !readChan(TM, disconnected) {
			t.Fatal("slow disconnecting")
		}
		if connectionsLength(p) != 0 {
			t.Error("map is not clear")
		}
	})
}

func TestPool_BroadcastExcept(t *testing.T) {
	p1 := NewPool(testConfigName("BroadcastExcept (broadcaster)"))
	p1.Register(NewPrefix("ANYM"), &Any{})
	defer p1.Close()
	p2 := NewPool(testConfigName("BroadcastExcept (receiver)"))
	p2.Register(NewPrefix("ANYM"), &Any{})
	defer p2.Close()
	p3 := NewPool(testConfigName("BroadcastExcept (excepted)"))
	p3.Register(NewPrefix("ANYM"), &Any{})
	defer p3.Close()
	if err := p1.Listen(""); err != nil {
		t.Fatal("unexpected listening error:", err)
	}
	if err := p2.Connect(p1.Address()); err != nil {
		t.Fatal("unexpected connecting error:", err)
	}
	if err := p3.Connect(p1.Address()); err != nil {
		t.Fatal("unexpected connecting error:", err)
	}
	except := firstConnection(p3).conn.LocalAddr().String() // remote for p1
	p1.BroadcastExcept(&Any{"first"}, except)
	p1.Broadcast(&Any{"second"}) // without exception
	select {
	case m := <-p2.Receive():
		if a, ok := m.Value.(*Any); !ok {
			t.Errorf("unexpected type of received message: %T", m.Value)
		} else if a.Value != "first" {
			t.Errorf("unexepeced data received: want %q, got %q",
				"first", a.Value)
		}
	case <-time.After(TM):
		t.Error("slow receiving")
	}
	// drop p2 "second"
	select {
	case m := <-p3.Receive():
		if a, ok := m.Value.(*Any); !ok {
			t.Errorf("unexpected type of received message: %T", m.Value)
		} else if a.Value != "second" {
			if a.Value == "first" {
				t.Error("received from excepted connection")
			} else {
				t.Errorf("unexepeced data received: want %q, got %q",
					"second", a.Value)
			}
		}
	case <-time.After(TM):
		t.Error("slow receiving")
	}
}

func TestPool_Broadcast(t *testing.T) {
	// TODO: low priority (implemented in Pool_BroadcastExcept)
}

func TestPool_Register(t *testing.T) {
	t.Run("invalid prefix", func(t *testing.T) {
		p := NewPool(testConfigName("Register/invalid prefix"))
		defer p.Close()
		defer shouldPanic(t)
		p.Register(Prefix{'-', '-', '-', '-'}, &Any{})
	})
	t.Run("invalid type", func(t *testing.T) {
		type Some interface{}
		p := NewPool(testConfigName("Register/invalid type"))
		defer p.Close()
		defer shouldPanic(t)
		p.Register(NewPrefix("SOME"), Some(nil))
	})
	t.Run("register type twice", func(t *testing.T) {
		p := NewPool(testConfigName("Register/register type twice"))
		defer p.Close()
		p.Register(NewPrefix("ANYM"), &Any{})
		defer shouldPanic(t)
		p.Register(NewPrefix("SOME"), &Any{})
	})
	t.Run("register prefix twice", func(t *testing.T) {
		p := NewPool(testConfigName("Register/register prefix twice"))
		defer p.Close()
		p.Register(NewPrefix("ANYM"), &Any{})
		type Some struct {
			Int int64
		}
		defer shouldPanic(t)
		p.Register(NewPrefix("ANYM"), &Some{})
	})
}

func TestPool_Address(t *testing.T) {
	// TODO: mid. priority
}

func TestPool_Connections(t *testing.T) {
	// TODO: mid. priority
}

func TestPool_Connection(t *testing.T) {
	// TODO: low priority
}

func TestPool_Receive(t *testing.T) {
	// TODO: low priority (implemented in Conn_Send)
}

func TestPool_Close(t *testing.T) {
	// TODO: high priority
}

func TestPool_conf_AcceptFailureHandler(t *testing.T) {
	// FROZEN due to never happens.
	//
	// the error can hapens if reconnecting is faster then closing
	// a connection

	// conf := testConfigName("conf AcceptFailureHandler")
	// acceptFailure := make(chan error, 1)
	// conf.AcceptFailureHandler = func(c net.Conn, err error) {
	// 	acceptFailure <- err
	// }
	// p := NewPool(conf)
	// defer p.Close()
	// if err := p.Listen(""); err != nil {
	// 	t.Fatal("unexpected listening error:", err)
	// }
	// c1 := dial(t, p.Address(), TM) // invoke fatal if some error occurs
	// defer c1.Close()
	// dl := net.Dialer{}
	// dl.Timeout = TM
	// dl.LocalAddr = c1.LocalAddr()
	// c2, err := dl.Dial("tcp", p.Address())
	// if err != nil {
	// 	t.Fatal("unexpected dialing error:", err)
	// }
	// defer c2.Close()
	// select {
	// case err := <-acceptFailure:
	// 	if err != ErrConnAlreadyExists {
	// 		t.Error("AcceptFailureHandler got unexpected error:", err)
	// 	}
	// case <-time.After(TM):
	// 	t.Error("AcceptFailureHandler wasn't called a few time")
	// }
}

func TestPool_AddSendFilter(t *testing.T) {
	t.Run("no filters", func(t *testing.T) {
		p := NewPool(testConfigName("AddSendFilter/no filters"))
		defer p.Close()
		p.Register(NewPrefix("ANYM"), &Any{})
		p.Broadcast(&Any{})
	})
	t.Run("filter", func(t *testing.T) {
		p := NewPool(testConfigName("AddSendFilter/filter"))
		anyp := NewPrefix("ANYM")
		p.Register(anyp, &Any{})
		p.AddSendFilter(func(p Prefix) bool { return p == anyp })
		p.Broadcast(&Any{})
	})
	t.Run("deny", func(t *testing.T) {
		p := NewPool(testConfigName("AddSendFilter/deny"))
		defer p.Close()
		type Some struct{ Int int64 }
		p.Register(NewPrefix("ANYM"), &Any{})
		somep := NewPrefix("SOME")
		p.Register(somep, &Some{})
		p.AddSendFilter(func(p Prefix) bool { return p == somep })
		defer shouldPanic(t)
		p.Broadcast(&Any{})
	})
}

func TestPool_AddReceiveFilter(t *testing.T) {
	t.Run("no filters", func(t *testing.T) {
		connected := make(chan struct{}, 1)
		rconf := testConfigName("AddSendFilter/no filters: receiver")
		rconf.ConnectionHandler = func(*Conn) { connected <- struct{}{} }
		r := NewPool(rconf)
		defer r.Close()
		r.Register(NewPrefix("ANYM"), &Any{})
		if err := r.Listen(""); err != nil {
			t.Fatal("unexpected listening error:", err)
		}
		s := NewPool(testConfigName("AddSendFilter/no filters: sender"))
		defer s.Close()
		s.Register(NewPrefix("ANYM"), &Any{})
		if err := s.Connect(r.Address()); err != nil {
			t.Fatal("unexpected connecting error:", err)
		}
		if !readChan(TM, connected) {
			t.Fatal("slow connecting")
		}
		s.Broadcast(&Any{"data"})
		select {
		case <-r.Receive():
		case <-time.After(TM):
			t.Error("slow receiving")
		}
	})
	t.Run("filter", func(t *testing.T) {
		connected := make(chan struct{}, 1)
		rconf := testConfigName("AddSendFilter/no filters: receiver")
		rconf.ConnectionHandler = func(*Conn) { connected <- struct{}{} }
		r := NewPool(rconf)
		defer r.Close()
		anyp := NewPrefix("ANYM")
		r.Register(anyp, &Any{})
		r.AddReceiveFilter(func(p Prefix) bool { return p == anyp })
		if err := r.Listen(""); err != nil {
			t.Fatal("unexpected listening error:", err)
		}
		s := NewPool(testConfigName("AddSendFilter/no filters: sender"))
		defer s.Close()
		s.Register(NewPrefix("ANYM"), &Any{})
		if err := s.Connect(r.Address()); err != nil {
			t.Fatal("unexpected connecting error:", err)
		}
		if !readChan(TM, connected) {
			t.Fatal("slow connecting")
		}
		s.Broadcast(&Any{"data"})
		select {
		case <-r.Receive():
		case <-time.After(TM):
			t.Error("slow receiving")
		}
	})
	t.Run("deny", func(t *testing.T) {
		connected := make(chan struct{}, 1)
		disconnected := make(chan struct{}, 1)
		rconf := testConfigName("AddSendFilter/no filters: receiver")
		rconf.ConnectionHandler = func(*Conn) { connected <- struct{}{} }
		rconf.DisconnectHandler = func(_ *Conn, err error) {
			if err != ErrRejectedByReceiveFilter {
				t.Error("unexpected disconnecting error:", err)
			}
			disconnected <- struct{}{}
		}
		r := NewPool(rconf)
		defer r.Close()
		type Some struct{ Int int64 }
		anyp := NewPrefix("ANYM")
		r.Register(anyp, &Any{})
		r.Register(NewPrefix("SOME"), &Some{})
		// Allow Any but deny all other
		r.AddReceiveFilter(func(p Prefix) bool { return p == anyp })
		if err := r.Listen(""); err != nil {
			t.Fatal("unexpected listening error:", err)
		}
		s := NewPool(testConfigName("AddSendFilter/no filters: sender"))
		defer s.Close()
		s.Register(NewPrefix("ANYM"), &Any{})
		s.Register(NewPrefix("SOME"), &Some{})
		if err := s.Connect(r.Address()); err != nil {
			t.Fatal("unexpected connecting error:", err)
		}
		if !readChan(TM, connected) {
			t.Fatal("slow connecting")
		}
		s.Broadcast(&Some{98})
		select {
		case <-disconnected:
		case <-time.After(TM):
			t.Error("slow or missing disconnecting")
		}
	})
}

func TestPool_SendTo(t *testing.T) {
	connected := make(chan struct{}, 3)
	sconf := testConfigName("SendTo server")
	sconf.ConnectionHandler = func(c *Conn) { connected <- struct{}{} }
	s := NewPool(sconf)
	s.Register(NewPrefix("ANYM"), &Any{})
	defer s.Close()
	c1 := NewPool(testConfigName("SendTo client 1"))
	defer c1.Close()
	c1.Register(NewPrefix("ANYM"), &Any{})
	c2 := NewPool(testConfigName("SendTo client 2"))
	defer c2.Close()
	c2.Register(NewPrefix("ANYM"), &Any{})
	c3 := NewPool(testConfigName("SendTo client 3"))
	defer c3.Close()
	c3.Register(NewPrefix("ANYM"), &Any{})
	if err := s.Listen(""); err != nil {
		t.Fatal("unexpected listening error:", err)
	}
	if err := c1.Connect(s.Address()); err != nil {
		t.Fatal("unexpected connecting error:", err)
	}
	if err := c2.Connect(s.Address()); err != nil {
		t.Fatal("unexpected connecting error:", err)
	}
	if err := c3.Connect(s.Address()); err != nil {
		t.Fatal("unexpected connecting error:", err)
	}
	for i := 0; i < 3; i++ {
		select {
		case <-connected:
		case <-time.After(TM):
			t.Fatal("slow or missing connecting")
		}
	}
	s.SendTo(&Any{"ha-ha"}, []string{
		firstConnection(c1).conn.LocalAddr().String(),
		firstConnection(c2).conn.LocalAddr().String(),
		// don't send to c3
	})
	for _, cc := range []*Pool{c1, c2} {
		select {
		case <-cc.Receive():
		case <-time.After(TM):
			t.Error("slow receiving")
		}
	}
	select {
	case <-c3.Receive():
		t.Error("SendTo sends to connection not in list")
	case <-time.After(TM):
	}
}
