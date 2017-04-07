package gnet

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/node/log"
)

// connection related constants
const (
	ReadBufferSize  int = 4096 // default redading buffer
	WriteBufferSize int = 4096 // default writing buffer
)

// connection related errors
var (
	// ErrWriteQueueFull occurs when a connection writes
	// messages slower then new messages to the connection
	// appears. A message can block sending up to
	// WriteTimeout x 2
	ErrWriteQueueFull = errors.New("write queue full")
)

// connection related service variables
var (
	// service prefixes
	ping = []byte{'-', '-', '-', '>', 0, 0, 0, 0} // ping message
	pong = []byte{'<', '-', '-', '-', 0, 0, 0, 0} // pong message
)

// receivedMessage represents received and
// decoded message that ready to be handled
type receivedMessage struct {
	msg  Message
	conn *conn
}

type conn struct {
	conn net.Conn // connection

	r io.Reader // buffered or unbuffered reading
	w io.Writer // buffered or unbuffered writing

	rq chan receivedMessage // read queue
	wr chan []byte          // write queue

	pool        *Pool // back read only reference
	releaseOnce sync.Once
}

func newConn(c net.Conn, p *Pool) (x *conn) {
	x = new(conn)
	x.c = c
	// set up reader
	if p.conf.ReadBufferSize > 0 { // buffered reading
		if p.conf.ReadTimeout > 0 { // with timeout
			x.r = bufio.NewReaderSize(&deadReader{
				c: c,
				t: p.conf.ReadTimeout,
			}, p.conf.ReadBufferSize)
		} else { // no timeout
			x.r = bufio.NewReaderSize(c, p.conf.ReadBufferSize)
		}
	} else { // unbuffered
		if p.conf.ReadTimeout { // with timeout
			x.r = &deadReader{c, p.conf.ReadTimeout}
		} else { // no timeout
			x.r = c
		}
	}
	// set up writer
	if p.conf.WriteBufferSize > 0 { // buffered writing
		if p.conf.WriteTimeout > 0 { // with timeout
			x.w = bufio.NewWriterSize(&deadWriter{
				c: c,
				t: p.conf.WriteTimeout,
			}, p.conf.WriteBufferSize)
		} else { // no timeout
			x.w = bufio.NewWriterSize(c, p.conf.WriteBufferSize)
		}
	} else { // unbuffered
		if p.conf.WriteTimeout { // with timeout
			x.w = &deadWriter{c, p.conf.WriteTimeout}
		} else { // no timeout
			x.w = c
		}
	}
}

func (c *conn) sendEncodedMessage(m []byte) (err error) {
	var tm *time.Timer
	var tc <-chan Time
	if c.pool.conf.WriteTimeout > 0 {
		tm = time.NewTimer(c.pool.conf.WriteTimeout)
		tc = tm.C
		defer tm.Stop()
	}
	select {
	case c.wq <- m:
	case <-c.pool.quit:
		err = ErrClosed
	case <-tc: // write timeout
		c.Close() // terminate connection
		err = ErrWriteQueueFull
	}
	return
}

func (c *conn) handleRead() {
	var (
		err  error
		head []byte = make([]byte, 4+PrefixLength)
		typ  reflect.Type
		ok   bool
		ln   uint32
		l    int
		body []byte
		val  reflect.Value
	)
	defer c.Close()
	for {
		select {
		case <-c.pool.quit:
			return
		default:
		}
		if _, err = c.r.Read(head); err != nil {
			// TODO: handle error
			return
		}
		if bytes.Compare(head, ping) == 0 { // handle pings automatically
			if err = c.sendEncodedMessage(pong); err != nil { // send pong back
				// TODO: handle error
				return
			}
			continue // and continue
		}
		if typ, ok = c.pool.rev[string(head[:4])]; !ok {
			// TODO: handle error
			return
		}
		if err = encoder.DeserializeRaw(head[4:], &ln); err != nil {
			// TODO: handle error
			return
		}
		l = int(ln)
		if l < 0 {
			// TODO: handle error
			return
		}
		if l > c.pool.conf.MaxMessageSize {
			// TODO: handle error
			return
		}
		if cap(body) < l {
			body = make([]byte, l) // increase the body if need
		} else {
			body = body[:l] // but never drop it
		}
		if _, err = c.r.Read(body); err != nil {
			return
		}
		val = reflect.New(typ)
		if err = encoder.DeserializeRawToValue(body, val); err != nil {
			// TODO: handle error
			return
		}
		select {
		case c.pool.receive <- receivedMessage{val.Interface().(Message), c}:
		case <-c.pool.quit:
			return
		}
	}
}

// handleWrite writes encoded messages to remote connection
// using write-buffer (if configured) and sending pings
// (if configured)
func (c *conn) handleWrite() {
	var (
		pingt *time.Ticker
		pincc <-chan time.Time

		data []byte

		bw *bufio.Writer
		ok bool

		err error
	)
	if c.pool.conf.PingInterval > 0 {
		pingt = time.NewTicker(c.pool.conf.PingInterval)
		pingc = pingt.C
		defer pingt.Stop()
	}
	if c.pool.conf.WriteBufferSize > 0 {
		if bw, ok = c.w.(*bufio.Writer); !ok {
			c.pool.Panicf("buffered writer is not *bufio.Writer: %T", c.w)
		}
	}
	defer c.Close()
	for {
		select {
		case data = <-c.wr:
			if _, err = c.w.Write(data); err != nil {
				// TODO: handle error
				return
			}
			// may be there are more then one message to
			// use full perfomance of buffered writing
			continue
		case <-pingc:
			if _, err = c.w.Write(ping); err != nil {
				// TODO: handle error
				return
			}
			if bw != nil && bw.Buffered() > 0 { // force the ping to be sent
				if err = bw.Flush(); err != nil {
					// TODO: handle error
					return
				}
			}
		case <-c.pool.quit:
			return
		default:
		}
		// flush the buffer if nothing to write anymore
		// and the buffer is not empty
		if bw != nil && bw.Buffered() > 0 {
			if err = bw.Flush(); err != nil {
				// TODO: handle error
				return
			}
		}
	}
}

// Addr returns remote address
func (c *conn) Addr() string {
	return c.conn.RemoteAddr().String()
}

// Send given message to the connection
func (c *conn) Send(m Message) (err error) {
	err = c.sendEncodedMessage(c.pool.encodeMessage(m))
	return
}

// Broadcast the message to all other connections except this one
func (c *conn) Broadcast(m Message) {
	c.pool.BroadcastExcept(m, c.Addr())
}

// Close connection
func (c *conn) Close() (err error) {
	c.releaseOnce.Do(func() {
		c.pool.release()
	})
	err = c.conn.Close()
	return
}

// read with deadline
type deadReader struct {
	t time.Duration
	c net.Conn
}

// Read implements io.Reader interface
func (d *deadReader) Read(p []byte) (n int, err error) {
	if err = d.c.SetReadDeadline(time.Now().Add(time.Duration)); err != nil {
		return
	}
	n, err = d.c.Read(p)
	return
}

// write with deadline
type deadWriter struct {
	t time.Duration
	c net.Conn
}

// Write implements io.Writer interface
func (d *deadWriter) Write(p []byte) (n int, err error) {
	if err = d.c.SetWriteDeadline(time.Now().Add(d.t)); err != nil {
		return
	}
	n, err = d.Write(p)
	return
}
