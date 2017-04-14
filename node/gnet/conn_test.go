package gnet

import (
	"bufio"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// unfortunately, net.Pipe returns connections
// that doesn't supports deadlines
func DeadPipe(t *testing.T) (a, b net.Conn) {
	var err error
	var l net.Listener
	if l, err = net.Listen("tcp", ""); err != nil {
		t.Fatal("can't create mock listener:", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		var err error
		if b, err = l.Accept(); err != nil {
			t.Fatal("can't accept mock connection:", err)
		}
		l.Close()
	}()
	a, err = net.DialTimeout("tcp", l.Addr().String(), 100*time.Millisecond)
	if err != nil {
		t.Fatal("can't dial to mock listener:", err)
	}
	<-done
	return
}

func Test_newConn(t *testing.T) {
	// no buffers, no timeouts
	t.Run("unbuffered no timeouts", func(t *testing.T) {
		conf := testConfig()
		conf.ReadBufferSize, conf.WriteBufferSize = 0, 0
		conf.ReadTimeout, conf.WriteTimeout = 0, 0
		p := NewPool(conf)
		conn, _ := net.Pipe() // for example
		c := newConn(conn, p, false)
		if c == nil {
			t.Fatal("newConn return nil")
		}
		if c.conn != conn {
			t.Error("wrong connection")
		}
		if x, ok := c.r.(net.Conn); !ok {
			t.Error("wrong type of reader")
		} else if x != conn {
			t.Error("wrong value of reader")
		}
		if x, ok := c.w.(net.Conn); !ok {
			t.Error("wrong type of writer")
		} else if x != conn {
			t.Error("wrong value of writer")
		}
		if !c.lasttm.IsZero() {
			t.Error("non zero last time")
		}
		if c.bw != nil {
			t.Error("unexpected *bufio.Writer")
		}
		if c.buffered {
			t.Error("must be unbuffered")
		}
		if c.wq == nil {
			t.Error("write queue is nil")
		}
		if c.closed == nil {
			t.Error("closed chan is nil")
		}
		if c.pool != p {
			t.Error("wrong value of back reference to pool")
		}
	})
	// unbuffered with timeouts (TODO: dead pipe)
	t.Run("unbuffered read timeout", func(t *testing.T) {
		conf := testConfig()
		conf.ReadBufferSize, conf.WriteBufferSize = 0, 0
		conf.ReadTimeout, conf.WriteTimeout = 5*time.Second, 0
		p := NewPool(conf)
		conn, _ := net.Pipe() // for example
		c := newConn(conn, p, false)
		if x, ok := c.r.(*deadReader); !ok {
			t.Error("wrong type of reader")
		} else if x.c != c {
			t.Error("wrong value of reader")
		}
		// SetReadDeadline error (net.Pipe doesn't supports SetReadDeadline)
		if _, err := c.r.Read(nil); err == nil {
			t.Error("missing error of SetReadDeadline")
		}
	})
	t.Run("unbuffered write timeout", func(t *testing.T) {
		conf := testConfig()
		conf.ReadBufferSize, conf.WriteBufferSize = 0, 0
		conf.ReadTimeout, conf.WriteTimeout = 0, 5*time.Second
		p := NewPool(conf)
		conn, _ := net.Pipe() // for example
		c := newConn(conn, p, false)
		if x, ok := c.w.(*deadWriter); !ok {
			t.Error("wrong type of writer")
		} else if x.c != c {
			t.Error("wrong value of writer")
		}
		// SetWriteDeadline error (net.Pipe doesn't supports SetWriteDeadline)
		if _, err := c.w.Write(nil); err == nil {
			t.Error("missing error of SetWriteDeadline")
		}
	})
	// buffered
	t.Run("read buffer no timeouts", func(t *testing.T) {
		conf := testConfig()
		conf.ReadBufferSize, conf.WriteBufferSize = 4096, 0
		conf.ReadTimeout, conf.WriteTimeout = 0, 0
		p := NewPool(conf)
		conn, write := net.Pipe() // for example
		c := newConn(conn, p, false)
		if _, ok := c.r.(*bufio.Reader); !ok {
			t.Error("wrong type of reader")
		}
		if c.buffered {
			t.Error("buffered must be false")
		}
		var example string = "example"
		// test reading from c.r
		done := make(chan struct{})
		await := new(sync.WaitGroup)
		await.Add(2)
		go func() {
			defer await.Done()
			select {
			case <-done:
			case <-time.After(100 * time.Millisecond):
			}
			write.Close()
			conn.Close()
		}()
		go func() {
			defer await.Done()
			defer close(done)
			if _, err := write.Write([]byte(example)); err != nil {
				t.Error("writing error:", err)
			}
		}()
		b := make([]byte, len(example))
		if _, err := c.r.Read(b); err != nil && err != io.EOF {
			t.Error("reading error:", err)
			return
		}
		if string(b) != example {
			t.Error("wrong value read:", string(b))
		}
		await.Wait()
	})
	t.Run("write buffer no timeouts", func(t *testing.T) {
		conf := testConfig()
		conf.ReadBufferSize, conf.WriteBufferSize = 0, 4096
		conf.ReadTimeout, conf.WriteTimeout = 0, 0
		p := NewPool(conf)
		conn, read := net.Pipe() // for example
		c := newConn(conn, p, false)
		if _, ok := c.w.(*bufio.Writer); !ok {
			t.Error("wrong type of reader")
		}
		if !c.buffered {
			t.Error("buffered must be true")
		}
		if c.bw == nil {
			t.Error("missing *bufio.Writer")
		}
		var example string = "example"
		// test reading from c.r
		done := make(chan struct{})
		await := new(sync.WaitGroup)
		await.Add(2)
		go func() {
			defer await.Done()
			select {
			case <-done:
			case <-time.After(100 * time.Millisecond):
			}
			read.Close()
			conn.Close()
		}()
		go func() {
			defer await.Done()
			defer close(done)
			b := make([]byte, len(example))
			if _, err := read.Read(b); err != nil && err != io.EOF {
				t.Error("reading error:", err)
				return
			}
			if string(b) != example {
				t.Error("wron value read:", string(b))
			}
		}()
		if _, err := c.w.Write([]byte(example)); err != nil {
			t.Error("writing error:", err)
		}
		if err := c.bw.Flush(); err != nil {
			t.Error("flushing error:", err)
		}
		await.Wait()
	})
	// buffered + timeouts
	t.Run("read buffer with timeout", func(t *testing.T) {
		conf := testConfig()
		conf.ReadBufferSize, conf.WriteBufferSize = 4096, 0
		conf.ReadTimeout, conf.WriteTimeout = minTimeout, 0
		p := NewPool(conf)
		conn, write := DeadPipe(t) // for example
		c := newConn(conn, p, false)
		if _, ok := c.r.(*bufio.Reader); !ok {
			t.Error("wrong type of reader")
		}
		if c.buffered {
			t.Error("buffered must be false")
		}
		var example string = "example"
		// test reading from c.r
		done := make(chan struct{})
		await := new(sync.WaitGroup)
		await.Add(2)
		go func() {
			defer await.Done()
			select {
			case <-done:
			case <-time.After(100*time.Millisecond + minTimeout):
			}
			write.Close()
			conn.Close()
		}()
		go func() {
			defer await.Done()
			defer close(done)
			// first, write without deadline errors to
			// check out last used time of connections
			if _, err := write.Write([]byte(example)); err != nil {
				t.Error("writing error:", err)
				return
			}
			// second, trigger deadline timeout error
			time.Sleep(minTimeout)
		}()
		b := make([]byte, len(example))
		// read and check last used time
		st := time.Now()
		if _, err := c.r.Read(b); err != nil {
			t.Error("reading error:", err)
			c.conn.Close()
		} else {
			et := time.Now()
			if c.lasttm.Before(st) || c.lasttm.After(et) {
				t.Error("wrong last used time")
			}
			if _, err := c.r.Read(b); err == nil {
				t.Error("missing read-deadline error")
			}
		}
		await.Wait()
	})
	t.Run("write buffer with timeout", func(t *testing.T) {
		conf := testConfig()
		conf.ReadBufferSize, conf.WriteBufferSize = 0, 4096
		conf.ReadTimeout, conf.WriteTimeout = 0, minTimeout
		p := NewPool(conf)
		conn, read := DeadPipe(t) // for example
		c := newConn(conn, p, false)
		if _, ok := c.w.(*bufio.Writer); !ok {
			t.Error("wrong type of writer")
		}
		if !c.buffered {
			t.Error("buffered must be true")
		}
		if c.bw == nil {
			t.Error("missing *bufio.Writer")
			return
		}
		var example string = "example"
		// test reading from c.r
		done := make(chan struct{})
		await := new(sync.WaitGroup)
		await.Add(2)
		go func() {
			defer await.Done()
			select {
			case <-done:
			case <-time.After(100*time.Millisecond + minTimeout):
			}
			read.Close()
			conn.Close()
		}()
		go func() {
			defer await.Done()
			defer close(done)
			// first, read without deadline errors to
			// check out last used time of connections
			b := make([]byte, len(example))
			if _, err := read.Read(b); err != nil {
				t.Error("reading error:", err)
			}
		}()
		// write and check last used time
		st := time.Now()
		if _, err := c.w.Write([]byte(example)); err != nil {
			t.Error("writing error:", err)
			c.conn.Close()
		} else if err := c.bw.Flush(); err != nil {
			t.Error("flushing error:", err)
			c.conn.Close()
		} else {
			et := time.Now()
			if c.lasttm.Before(st) || c.lasttm.After(et) {
				t.Error("wrong last used time")
			}
			// set WriteTimout to negative value to get the error
			c.pool.conf.WriteTimeout = -5 * time.Second
			if _, err := c.w.Write([]byte(example)); err == nil {
				if err := c.bw.Flush(); err == nil {
					t.Error("missing write-deadline error")
				}
			}
		}
		await.Wait()
	})
}

func TestConn_updateLastUsed(t *testing.T) {
	c := new(Conn)
	c.updateLastUsed()
	st := c.lasttm
	c.updateLastUsed()
	et := c.lasttm
	if c.lasttm.Before(st) || c.lasttm.After(et) {
		t.Error("wrong last used time")
	}
}

func TestConn_lastUsed(t *testing.T) {
	c := new(Conn)
	c.updateLastUsed()
	st := time.Now()
	lu := c.lastUsed()
	et := time.Now()
	if st.Sub(c.lasttm) > lu {
		t.Error("smal lastUsed duration")
	}
	if et.Sub(c.lasttm) < lu {
		t.Error("big lastUsed duration")
	}
}

func TestConn_handle(t *testing.T) {
	conf := testConfig()
	conf.ReadTimeout, conf.WriteTimeout = 0, 0
	conf.ReadBufferSize, conf.WriteBufferSize = 0, 0
	conf.PingInterval = 0
	conn, pipe := net.Pipe()
	p := NewPool(conf)
	defer p.Close()
	p.Register(NewPrefix("ANYM"), &Any{})
	c := newConn(conn, p, false)
	defer c.Close()
	// ----
	// c.pool.wg.Add(2)
	// go c.handleRead()
	// go c.handleWrite()
	// ----
	p.acquire() // add connection
	c.handle()
	// write to the pipe (send)
	if _, err := pipe.Write(p.encodeMessage(&Any{"read"})); err != nil {
		t.Fatal(err)
	}
	select {
	case m := <-p.Receive():
		if a, ok := m.Value.(*Any); !ok {
			t.Errorf("wrong type of message received: %T", a.Value)
		} else if a.Value != "read" {
			t.Error("wrong value received:", a.Value)
		} else {
			// write to the connection
			m.Conn.Send(&Any{"write"})
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("slow receiving")
	}
	// read from the pipe (receive)
	// PREFIX, msg len, string len, string
	reply := make([]byte, PrefixLength+4+4+len("write"))
	if n, err := io.ReadFull(pipe, reply); err != nil {
		t.Fatal("error reading reply:", err)
	} else if n != PrefixLength+4+4+len("write") {
		t.Fatal("wrong reply length:", n)
	}
	if string(reply[:PrefixLength]) != "ANYM" {
		t.Error("wrong message prefix")
	}
	if string(reply[PrefixLength+4+4:]) != "write" {
		t.Error("wrong data")
	}
}

//
// public methods
//

func TestConn_Addr(t *testing.T) {
	p := NewPool(testConfigName("Addr"))
	defer p.Close()
	l := listen(t)
	defer l.Close()
	if err := p.Connect(l.Addr().String()); err != nil {
		t.Fatal("unexpeceted connecting error:", err)
	}
	c := firstConnection(p)
	if want, got := c.conn.RemoteAddr().String(), c.Addr(); want != got {
		t.Errorf("unexpected Addr() value: want %q, got %q", want, got)
	}
}

func TestConn_Send(t *testing.T) {
	t.Run("send receive", func(t *testing.T) {
		r := NewPool(testConfigName("Send (receiver)"))
		r.Register(NewPrefix("ANYM"), &Any{})
		defer r.Close()
		if err := r.Listen(""); err != nil {
			t.Error("unexpected listening error:", err)
		}
		s := NewPool(testConfigName("Send (sender)"))
		s.Register(NewPrefix("ANYM"), &Any{})
		defer s.Close()
		if err := s.Connect(r.Address()); err != nil {
			t.Fatal("unexpected conneccting error:", err)
		}
		c := firstConnection(s) // sending to the connection
		c.Send(&Any{"data"})
		select {
		case m := <-r.Receive():
			la := m.Conn.conn.LocalAddr().String()
			ra := c.conn.RemoteAddr().String()
			if la != ra {
				t.Errorf("received from unexpeted connection: %v - %v", la, ra)
			}
			if a, ok := m.Value.(*Any); !ok {
				t.Errorf("unexpected type of received message: %T", m.Value)
			} else if a.Value != "data" {
				t.Errorf("unexepeced data received: want %q, got %q",
					"data", a.Value)
			}
		case <-time.After(TM):
			t.Fatal("slow receiving")
		}
	})
	t.Run("send size limit", func(t *testing.T) {
		r := NewPool(testConfigName("Send (receiver)"))
		r.Register(NewPrefix("ANYM"), &Any{})
		defer r.Close()
		if err := r.Listen(""); err != nil {
			t.Error("unexpected listening error:", err)
		}
		sc := testConfigName("Send (sender)")
		sc.MaxMessageSize = 1 // 1 byte max allowed
		s := NewPool(sc)
		s.Register(NewPrefix("ANYM"), &Any{})
		defer s.Close()
		if err := s.Connect(r.Address()); err != nil {
			t.Fatal("unexpected conneccting error:", err)
		}
		c := firstConnection(s) // sending to the connection
		defer shouldPanic(t)
		c.Send(&Any{"data"})
	})
	t.Run("receive size limit", func(t *testing.T) {
		disconnect := make(chan struct{}, 1)
		rc := testConfigName("Send (receiver)")
		rc.MaxMessageSize = 1 // 1 byte max allowed
		rc.DisconnectHandler = func(*Conn, error) { disconnect <- struct{}{} }
		r := NewPool(rc)
		r.Register(NewPrefix("ANYM"), &Any{})
		defer r.Close()
		if err := r.Listen(""); err != nil {
			t.Error("unexpected listening error:", err)
		}
		s := NewPool(testConfigName("Send (sender)"))
		s.Register(NewPrefix("ANYM"), &Any{})
		defer s.Close()
		if err := s.Connect(r.Address()); err != nil {
			t.Fatal("unexpected conneccting error:", err)
		}
		c := firstConnection(s) // sending to the connection
		c.Send(&Any{"data"})
		// receiver must handle the head of the message and
		// close the connection because the message exceeds max
		// size allowed for the receiver
		if !readChan(TM, disconnect) {
			t.Error("slow or missing disconnecting")
		}
	})
}

func TestConn_Broadcast(t *testing.T) {
	// TODO: low priority (implemented in Pool_BroadcastExcept)
}

func TestConn_Close(t *testing.T) {
	p1 := NewPool(testConfigName("Close 1"))
	defer p1.Close()
	if err := p1.Listen(""); err != nil {
		t.Error("unexpected listening error:", err)
	}
	conf2 := testConfigName("Close 1")
	conf2.MaxConnections = 1 // to be sure that the limit is set
	p2 := NewPool(conf2)
	defer p2.Close()
	if err := p2.Connect(p1.Address()); err != nil {
		t.Fatal("unexpected conneccting error:", err)
	}
	firstConnection(p2).Close()
	if connectionsLength(p2) != 0 {
		t.Error("closng connectin doesn't delete the connection from pool")
	}
	if len(p2.sem) != 0 {
		t.Error("closing connection doesn't release limit")
	}
}
