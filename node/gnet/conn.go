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
)

// connection related errors
var (
	// ErrWriteQueueFull occurs when a connection writes
	// messages slower then new messages to the connection
	// appears. A message can block sending up to
	// WriteTimeout x 2
	ErrWriteQueueFull = errors.New("write queue full")
	// ErrConnAlreadyExists occurs when some the Pool already
	// have connection to the address, but new connection to
	// the address created anyway
	ErrConnAlreadyExists = errors.New("connection alredy exists")
	// ErrClosedConn occurs when connection is closed
	ErrClosedConn = errors.New("dead connection")
)

// connection related service variables
var (
	// service prefixes
	ping = []byte{'-', '-', '-', '>', 0, 0, 0, 0} // ping message
	pong = []byte{'<', '-', '-', '-', 0, 0, 0, 0} // pong message
)

type Conn struct {
	conn net.Conn // connection

	r io.Reader // buffered or unbuffered reading
	w io.Writer // buffered or unbuffered writing

	wq     chan []byte   // write queue
	closed chan struct{} // connection was closed

	pool        *Pool // back read only reference
	releaseOnce sync.Once
}

func newConn(c net.Conn, p *Pool) (x *Conn) {
	x = new(Conn)
	x.conn = c
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
		if p.conf.ReadTimeout > 0 { // with timeout
			x.r = &deadReader{p.conf.ReadTimeout, c}
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
		if p.conf.WriteTimeout > 0 { // with timeout
			x.w = &deadWriter{p.conf.WriteTimeout, c}
		} else { // no timeout
			x.w = c
		}
	}
	x.wq = make(chan []byte, p.conf.WriteQueueSize)
	x.pool = p
	x.closed = make(chan struct{})
	return
}

func (c *Conn) handle() {
	go c.handleRead()
	go c.handleWrite()
}

func (c *Conn) sendEncodedMessage(m []byte) (err error) {
	select {
	case c.wq <- m:
	case <-c.closed:
		err = ErrClosedConn
	}
	return
}

func (c *Conn) isClosed() (yep bool) {
	select {
	case <-c.closed:
		yep = true
	default:
	}
	return
}

func (c *Conn) handleRead() {
	c.pool.Debugf("%s start read loop", c.Addr())
	var (
		err error

		head []byte = make([]byte, PrefixLength+4)
		p    Prefix
		ln   uint32
		l    int

		body []byte

		ok  bool
		typ reflect.Type
		val reflect.Value
	)
	defer c.close(true, true) // remove, sync
	if c.pool.conf.Debug {
		defer c.pool.Debugf("%s end read loop", c.Addr())
	}
	for {
		if c.isClosed() {
			return
		}
		if _, err = c.r.Read(head); err != nil {
			if c.isClosed() {
				return // don't log about the error
			}
			c.pool.Printf("[ERR] %s reading error: %v", c.Addr(), err)
			return
		}
		if bytes.Compare(head, ping) == 0 { // handle pings automatically
			if err = c.sendEncodedMessage(pong); err != nil { // send pong back
				c.pool.Printf("[ERR] %s error sending PONG: %v", c.Addr(), err)
				return
			}
			continue // and continue
		}
		if bytes.Compare(head, pong) == 0 { // handle pongs automatically
			continue // and continue
		}
		copy(p[:], head) // create prefix from head[:PrefixLength]
		if typ, ok = c.pool.rev[p]; !ok {
			c.pool.Printf("[ERR] %s unregistered message received: %s",
				c.Addr(), string(p[:]))
			return
		}
		if err = encoder.DeserializeRaw(head[PrefixLength:], &ln); err != nil {
			c.pool.Printf("[ERR] %s error decoding message length: %v",
				c.Addr(), err)
			return
		}
		l = int(ln)
		if l < 0 {
			c.pool.Printf("[ERR] %s got message with negative length error: %d",
				c.Addr(), l)
			return
		}
		if c.pool.conf.MaxMessageSize > 0 && l > c.pool.conf.MaxMessageSize {
			c.pool.Printf("[ERR] %s received message exceeds max size: %d",
				c.Addr(), l)
			return
		}
		if cap(body) < l {
			body = make([]byte, l) // increase the body if need
		} else {
			body = body[:l] // but never drop it
		}
		if _, err = c.r.Read(body); err != nil {
			if c.isClosed() {
				return // don't log about the error
			}
			c.pool.Printf("[ERR] %s reading error: %v", c.Addr(), err)
			return
		}
		val = reflect.New(typ)
		if _, err = encoder.DeserializeRawToValue(body, val); err != nil {
			c.pool.Printf("[ERR] %s decoding message error: %v", c.Addr(), err)
			return
		}
		select {
		case c.pool.receive <- Message{val.Interface(), c}:
		case <-c.closed:
			return
		}
	}
}

// handleWrite writes encoded messages to remote connection
// using write-buffer (if configured) and sending pings
// (if configured)
func (c *Conn) handleWrite() {
	c.pool.Debugf("%s start write loop", c.Addr())
	var (
		data []byte

		bw *bufio.Writer
		ok bool

		err error
	)
	if c.pool.conf.WriteBufferSize > 0 {
		if bw, ok = c.w.(*bufio.Writer); !ok {
			c.pool.Panicf("buffered writer is not *bufio.Writer: %T", c.w)
		}
	}
	defer c.close(true, true) // remove, sync
	if c.pool.conf.Debug {
		defer c.pool.Debugf("%s end write loop", c.Addr())
	}
WriteLoop:
	for {
		select {
		case data = <-c.wq:
			if _, err = c.w.Write(data); err != nil {
				if c.isClosed() {
					return // don't log about the error
				}
				c.pool.Printf("[ERR] %s writing error: %v", c.Addr(), err)
				return
			}
			// may be there are more then one message to
			// use full perfomance of buffered writing
			for {
				select {
				case data = <-c.wq:
					if _, err = c.w.Write(data); err != nil {
						if c.isClosed() {
							return // don't log about the error
						}
						c.pool.Printf("[ERR] %s writing error: %v",
							c.Addr(), err)
						return
					}
				default:
					// flush the buffer if nothing to write anymore
					// and the buffer is not empty
					if bw != nil && bw.Buffered() > 0 {
						if err = bw.Flush(); err != nil {
							if c.isClosed() {
								return // don't log about the error
							}
							c.pool.Printf("[ERR] %s writing error: %v",
								c.Addr(), err)
							return
						}
					}
					continue WriteLoop // wait for new data
				}
			}
		case <-c.closed:
			return
		}
	}
}

// Addr returns remote address
func (c *Conn) Addr() string {
	return c.conn.RemoteAddr().String()
}

// Send given message to the connection
func (c *Conn) Send(m interface{}) {
	var err error
	if err = c.sendEncodedMessage(c.pool.encodeMessage(m)); err != nil {
		c.pool.Printf("[ERR] %s error sending message: %v", c.Addr(), err)
		c.Close() // terminate the connection
	}
	return
}

// Broadcast the message to all other connections except this one
func (c *Conn) Broadcast(m Message) {
	c.pool.BroadcastExcept(m, c.Addr())
}

func (c *Conn) close(remove, async bool) (err error) {
	c.releaseOnce.Do(func() {
		close(c.closed)
		c.pool.release()
		if remove {
			if async {
				go c.pool.removeConnection(c.Addr()) // async
			} else {
				c.pool.removeConnection(c.Addr()) // same goroutine
			}
		}
	})
	err = c.conn.Close()
	return
}

// Close connection
func (c *Conn) Close() (err error) {
	err = c.close(true, true) // remove async
	return
}

// read with deadline
type deadReader struct {
	t time.Duration
	c net.Conn
}

// Read implements io.Reader interface
func (d *deadReader) Read(p []byte) (n int, err error) {
	if err = d.c.SetReadDeadline(time.Now().Add(d.t)); err != nil {
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
