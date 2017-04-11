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
		conf.DisconnectHandler = func(*Conn) { disconnect <- struct{}{} }
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
		connected := make(chan struct{}, 2)
		disconnected := make(chan struct{}, 2)
		conf := testConfigName("Disconnect/found")
		conf.ConnectionHandler = func(*Conn) { connected <- struct{}{} }
		conf.DisconnectHandler = func(*Conn) { disconnected <- struct{}{} }
		p := NewPool(conf)
		defer p.Close()
		l := listen(t) // test listener
		defer l.Close()
		if err := p.Connect(l.Addr().String()); err != nil {
			t.Fatal("unexpected connecting error:", err)
		}
		if !readChan(TM, connected) {
			t.Fatal("slow connecting")
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
