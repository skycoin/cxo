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
func DeadPipe(t *testing.T) (a, b net.Conn, l net.Listener) {
	var err error
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
		c := newConn(conn, p)
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
		c := newConn(conn, p)
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
		c := newConn(conn, p)
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
		c := newConn(conn, p)
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
		c := newConn(conn, p)
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
		conn, write, listener := DeadPipe(t) // for example
		defer listener.Close()
		c := newConn(conn, p)
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
		conn, read, listener := DeadPipe(t) // for example
		defer listener.Close()
		c := newConn(conn, p)
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
	// conf := testConfig()
	// conf.ReadTimeout, conf.WriteTimeout = 0, 0
	// conf.ReadBufferSize, conf.WriteBufferSize = 0, 0
	// conn, pipe := net.Pipe()
	// p := NewPool(conf)
	// p.Register("ANYM", &Any{})
	// c := newConn(conn, p)
	// defer c.close(closeDontRemove)
	// // c.pool.wg.Add(2)
	// // go c.handleRead()
	// // go c.handleWrite()
	// c.handle()
	// // prepare
	// wg := new(sync.WaitGroup)
	// wg.Add(3)
	// // write to the pipe (send)
	// //
	// // read from the pipe (receive)
	// //
}

func TestConn_sendEncodedMessage(t *testing.T) {
	//
}

func TestConn_isClosed(t *testing.T) {
	//
}

func TestConn_read(t *testing.T) {
	//
}

func TestConn_write(t *testing.T) {
	//
}

func TestConn_flush(t *testing.T) {
	//
}

func TestConn_sendPing(t *testing.T) {
	//
}

func TestConn_handleRead(t *testing.T) {
	//
}

func TestConn_handleWrite(t *testing.T) {
	//
}

func TestConn_Addr(t *testing.T) {
	//
}

func TestConn_Send(t *testing.T) {
	//
}

func TestConn_Broadcast(t *testing.T) {
	//
}

func TestConn_close(t *testing.T) {
	//
}

func TestConn_Close(t *testing.T) {
	//
}

func Test_deadReader_Read(t *testing.T) {
	//
}

func Test_deadWriter_Write(t *testing.T) {
	//
}
