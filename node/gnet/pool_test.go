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

func timeLimit(tm time.Duration, fatal func()) (breakChan chan struct{}) {
	breakChan = make(chan struct{})
	go func() {
		select {
		case <-breakChan:
		case <-time.After(tm):
			fatal()
		}
	}()
	return
}

// limited closing of a pool

func closePool(t *testing.T, p *Pool, tm time.Duration) {
	done := timeLimit(50*time.Millisecond, func() {
		if _, file, no, ok := runtime.Caller(1); ok {
			file = filepath.Base(file)
			t.Fatalf("closing of pool is too slow: %s:%d", file, no)
		} else {
			t.Fatal("closing of pool is too slow")
		}
	})
	p.Close()
	close(done)
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
		closePool(t, p, 50*time.Millisecond)
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
		defer closePool(t, p, 50*time.Millisecond)
		if err := p.Listen(""); err != nil {
			t.Error("unexpected listeninig error:", err)
		}
	})
	t.Run("accept", func(t *testing.T) {
		p := NewPool(testConfigName("Listen/accept"))
		defer closePool(t, p, 50*time.Millisecond)
		if err := p.Listen(""); err != nil {
			t.Error("unexpected listeninig error:", err)
		}
		dial(t, p.l.Addr().String(), 50*time.Millisecond).Close()
	})
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
