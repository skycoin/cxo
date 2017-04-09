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
	// service messages
	ping = []byte{'-', '-', '-', '>', 0, 0, 0, 0} // ping message ("--->")
	pong = []byte{'<', '-', '-', '-', 0, 0, 0, 0} // pong message ("<---")
)

type Conn struct {
	conn net.Conn // connection

	r io.Reader // buffered or unbuffered reading
	w io.Writer // buffered or unbuffered writing

	lastmx sync.RWMutex // mutex for last time
	lasttm time.Time    // last time reading from/writng to connection

	bw       *bufio.Writer // if writing is buffered
	buffered bool          // is writing buffered (writing)

	pingt *time.Timer // pings

	wq     chan []byte   // write queue
	closed chan struct{} // connection was closed

	pool      *Pool // back read only reference
	closeOnce sync.Once
}

func (c *Conn) updateLastUsed() {
	c.lastmx.Lock()
	defer c.lastmx.Unlock()
	c.lasttm = time.Now()
}

// time.Now().Sub( min(lastRead, lastWrite) )
func (c *Conn) lastUsed() time.Duration {
	c.lastmx.RLock()
	defer c.lastmx.RUnlock()
	return time.Now().Sub(c.lasttm)
}

func newConn(c net.Conn, p *Pool) (x *Conn) {
	x = new(Conn)
	x.conn = c
	// set up reader
	if p.conf.ReadBufferSize > 0 { // buffered reading
		if p.conf.ReadTimeout > 0 { // with timeout
			x.r = bufio.NewReaderSize(&deadReader{x}, p.conf.ReadBufferSize)
		} else { // no timeout
			x.r = bufio.NewReaderSize(c, p.conf.ReadBufferSize)
		}
	} else { // unbuffered
		if p.conf.ReadTimeout > 0 { // with timeout
			x.r = &deadReader{x}
		} else { // no timeout
			x.r = c
		}
	}
	// set up writer
	if p.conf.WriteBufferSize > 0 { // buffered writing
		if p.conf.WriteTimeout > 0 { // with timeout
			x.w = bufio.NewWriterSize(&deadWriter{x}, p.conf.WriteBufferSize)
		} else { // no timeout
			x.w = bufio.NewWriterSize(c, p.conf.WriteBufferSize)
		}
		x.buffered, x.bw = true, x.w.(*bufio.Writer)
	} else { // unbuffered
		if p.conf.WriteTimeout > 0 { // with timeout
			x.w = &deadWriter{x}
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
	c.pool.wg.Add(2)
	go c.handleRead()
	go c.handleWrite()
}

func (c *Conn) sendEncodedMessage(m []byte) (err error) {
	select {
	case c.wq <- m:
	case <-c.closed:
		err = ErrClosedConn
	default:
		// if some connection can't send messages as fast as they
		// appears, then this connection should be closed to
		// prevent other connections be closed by timeout awaiting
		// the connection inside Broadcast* methods of Pool
		err = ErrWriteQueueFull
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

func (c *Conn) read(data []byte) (terminate bool) {
	if _, err := c.r.Read(data); err != nil {
		terminate = true
		if !c.isClosed() { // don't log about the error if the Conn closed
			c.pool.Printf("[ERR] %s reading error: %v", c.Addr(), err)
		}
	}
	return
}

func (c *Conn) write(data []byte) (terminate bool) {
	if _, err := c.w.Write(data); err != nil {
		terminate = true
		if !c.isClosed() { // don't log about the error if the Conn closed
			c.pool.Printf("[ERR] %s writing error: %v", c.Addr(), err)
		}
	}
	return
}

func (c *Conn) flush() (terminate bool) {
	if c.buffered && c.bw.Buffered() > 0 {
		if err := c.bw.Flush(); err != nil {
			if !c.isClosed() { // don't log about the error if closed
				c.pool.Printf("[ERR] %s writing error: %v", c.Addr(), err)
			}
		}
	}
	return
}

func (c *Conn) sendPing() (terminate bool) {
	// don't send ping if connection was used earler
	// then time.Now() - c.pool.conf.PingInterval
	if diff := c.pool.conf.PingInterval - c.lastUsed(); diff > minPingInterval {
		c.pingt.Reset(diff)
		return
	}
	// otherwise send the ping
	if terminate = c.write(ping); terminate {
		return
	}
	// force flushing the buffer if connection is buffered
	if terminate = c.flush(); terminate {
		return
	}
	c.pingt.Reset(c.pool.conf.PingInterval)
	return
}

func (c *Conn) handleRead() {
	c.pool.Debugf("%s start read loop", c.Addr())
	var (
		err error

		head []byte = make([]byte, PrefixLength+4) // + encoded uint32 (length)
		p    Prefix
		ln   uint32
		l    int

		body []byte

		ok  bool
		typ reflect.Type
		val reflect.Value

		terminate bool // semantic
	)
	// closing
	defer c.pool.wg.Done()
	// remove from the goroutine (no fear of deadlocks)
	defer c.Close()
	defer c.pool.Debugf("%s end read loop", c.Addr())
	// read loop
	for {
		if c.isClosed() {
			return
		}
		if terminate = c.read(head); terminate {
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
		if terminate = c.read(body); terminate {
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
		data      []byte
		pingc     <-chan time.Time
		terminate bool // for semantic
	)
	if c.pool.conf.PingInterval > 0 {
		c.pingt = time.NewTimer(c.pool.conf.PingInterval)
		pingc = c.pingt.C
		defer c.pingt.Stop()
	}
	// closing
	defer c.pool.wg.Done()
	// remove from the goroutine (no fear of deadlocks)
	defer c.Close()
	defer c.pool.Debugf("%s end write loop", c.Addr())
	// write loop
	for {
		select {
		case <-pingc:
			c.sendPing()
		case data = <-c.wq:
			if terminate = c.write(data); terminate {
				return
			}
			// may be there are more then one message to
			// use full perfomance of buffered writing
			for {
				select {
				case data = <-c.wq:
					if terminate = c.write(data); terminate {
						return
					}
					continue // drain the write queue if possible
				case <-pingc:
					c.sendPing()
					continue // drain the write quueue if possible
				default:
				}
				// flush the buffer if nothing to write anymore
				// and the buffer is not empty
				if terminate = c.flush(); terminate {
					return
				}
				break // break the loop to go to outer write loop
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
func (c *Conn) Broadcast(m interface{}) {
	c.pool.BroadcastExcept(m, c.Addr())
}

// Close connection
func (c *Conn) Close() (err error) {
	c.closeOnce.Do(func() {
		close(c.closed)         // chan
		c.pool.delete(c.Addr()) // map
		err = c.conn.Close()    // connection (inclusive release)
	})
	return
}

// read with deadline
type deadReader struct {
	c *Conn
}

// Read implements io.Reader interface
func (d *deadReader) Read(p []byte) (n int, err error) {
	err = d.c.conn.SetReadDeadline(time.Now().Add(d.c.pool.conf.ReadTimeout))
	if err != nil {
		return
	}
	if n, err = d.c.conn.Read(p); n > 0 {
		d.c.updateLastUsed()
	}
	return
}

// write with deadline
type deadWriter struct {
	c *Conn
}

// Write implements io.Writer interface
func (d *deadWriter) Write(p []byte) (n int, err error) {
	err = d.c.conn.SetWriteDeadline(time.Now().Add(d.c.pool.conf.WriteTimeout))
	if err != nil {
		return
	}
	if n, err = d.c.conn.Write(p); n > 0 {
		d.c.updateLastUsed()
	}
	return
}
