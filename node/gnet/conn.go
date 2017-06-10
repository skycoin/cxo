package gnet

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// A ConnState represents connection state
type ConnState int

// there are three possible states of a connection
const (
	ConnStateConnected ConnState = iota // connection works
	ConnStateDialing                    // dialing
	ConnStateClosed                     // closed connection
)

var connStateString = [...]string{
	ConnStateConnected: "CONNECTED",
	ConnStateDialing:   "DIALING",
	ConnStateClosed:    "CLOSED",
}

// String implements fmt.Stringer interface
func (c ConnState) String() string {
	if c >= 0 && int(c) < len(connStateString) {
		return connStateString[c]
	}
	return fmt.Sprintf("ConnState<%d>", c)
}

type Conn struct {
	address string // diaing address

	// cmx locks conn, state, dialo and diallm fields
	cmx   sync.Mutex // connection lock for redialing
	conn  net.Conn
	state ConnState

	r io.Reader
	w io.Writer

	bw *bufio.Writer // if writing is buffered

	incoming bool

	readq  chan []byte
	writeq chan []byte

	diallm int           // dials limit (countdown)
	dialo  *sync.Once    // trigger dialing once per connection fail
	dialtr chan error    // trigger dialing
	dialrl chan struct{} // redialing (lock read)
	dialwl chan struct{} // redialing (lock write)

	//
	// last read and last write
	//

	lrmx     sync.Mutex // lock for lastRead
	lastRead time.Time

	lwmx      sync.Mutex /// lock for lastWrite
	lastWrite time.Time

	p *Pool // logger and configs

	vmx sync.Mutex // lock for value
	val interface{}

	closeo sync.Once
	closed chan struct{}
}

// accept connection by listener
func (p *Pool) acceptConnection(c net.Conn) (cn *Conn, err error) {
	p.Debug("accept connection ", c.RemoteAddr().String())

	p.cmx.Lock()
	defer p.cmx.Unlock()
	var got bool
	if _, got = p.conns[c.RemoteAddr().String()]; got {
		err = fmt.Errorf("connection already exists %s",
			c.RemoteAddr().String())
		return
	}
	cn = new(Conn)

	cn.address = c.RemoteAddr().String()

	p.conns[c.RemoteAddr().String()] = cn // save
	cn.p = p

	cn.updateConnection(c)

	cn.incoming = true

	cn.readq = make(chan []byte, p.conf.ReadQueueLen)
	cn.writeq = make(chan []byte, p.conf.WriteQueueLen)

	cn.dialrl = make(chan struct{})
	cn.dialwl = make(chan struct{})

	close(cn.dialrl) // never block
	close(cn.dialwl) // never block

	cn.closed = make(chan struct{})

	p.await.Add(2)
	go cn.read()
	go cn.write()

	return
}

// create outgoing connections
func (p *Pool) createConnection(address string) (cn *Conn) {
	p.Debug("create connection: ", address)

	cn = new(Conn)

	p.conns[address] = cn // save
	cn.p = p

	cn.address = address
	cn.incoming = false

	cn.readq = make(chan []byte, p.conf.ReadQueueLen)
	cn.writeq = make(chan []byte, p.conf.WriteQueueLen)

	cn.diallm = p.conf.DialsLimit
	cn.dialo = new(sync.Once)
	cn.dialtr = make(chan error)
	cn.dialrl = make(chan struct{})
	cn.dialwl = make(chan struct{})

	cn.closed = make(chan struct{})

	p.await.Add(3)
	go cn.read()
	go cn.write()
	go cn.dial()

	cn.triggerDialing(nil)

	return
}

// ========================================================================== //
//                             dial/read/write                                //
// ========================================================================== //

func (c *Conn) closeConnection() {
	c.p.Debug("close connection of: ", c.address)

	c.cmx.Lock()
	defer c.cmx.Unlock()

	if c.conn != nil {
		c.conn.Close()
		c.state = ConnStateClosed
	}
}

func (cn *Conn) dialing() (c net.Conn, err error) {
	cn.p.Debug("dialing to ", cn.address)

	// tcp or tls
	if cn.p.conf.TLSConfig == nil {
		// timeout or not
		if cn.p.conf.DialTimeout > 0 {
			c, err = net.DialTimeout("tcp", cn.address, cn.p.conf.DialTimeout)
		} else {
			c, err = net.Dial("tcp", cn.address)
		}
	} else {
		// timeout or not
		if cn.p.conf.DialTimeout > 0 {
			var d net.Dialer
			d.Timeout = cn.p.conf.DialTimeout
			c, err = tls.DialWithDialer(&d, "tcp", cn.address,
				cn.p.conf.TLSConfig)
		} else {
			c, err = tls.Dial("tcp", cn.address, cn.p.conf.TLSConfig)
		}
	}
	return
}

func (c *Conn) triggerDialing(err error) {
	c.p.Debug("trigger dialing of: ", c.address)

	c.cmx.Lock()
	defer c.cmx.Unlock()

	c.dialo.Do(func() {
		// check out dialing limit
		c.diallm--
		if c.diallm == 0 {
			c.p.Debug("dials limit exceeded: ", c.address)
			// close the connection

			// we need to unlock cmx to close the connection
			c.cmx.Unlock() // unlock -> lock again (because of deferred unlock)
			{
				c.Close()
			}
			c.cmx.Lock()

			return
		}

		// change state
		c.state = ConnStateDialing

		// make sure that conn is closed
		if c.conn != nil {
			c.conn.Close()
		}

		//
		// the callback should be prefomed with unlocked cmx
		//

		// perform dialing callback
		if callback := c.p.conf.OnDial; callback != nil {

			// we need to unlock cmx to close the connection
			c.cmx.Unlock() // unlock -> lock again (because of deferred unlock)
			{
				if err = callback(c, err); err != nil {

					// we don't want to redial anymore
					c.Close()
					c.cmx.Lock()
					return

				}
			}
			c.cmx.Lock()

		}

		// trigger redialing
		select {
		case c.dialtr <- err:
		case <-c.closed:
		}
	})

}

func (cn *Conn) updateConnection(c net.Conn) {
	cn.p.Debug("update connection of ", cn.address)

	cn.cmx.Lock()
	defer cn.cmx.Unlock()

	cn.dialo = new(sync.Once) // refresh

	cn.conn = c
	cn.state = ConnStateConnected

	// r io.Reader
	if cn.p.conf.ReadBufferSize > 0 { // buffered
		cn.r = bufio.NewReaderSize(&timedReadWriter{cn, c},
			cn.p.conf.ReadBufferSize)
	} else { // unbuffered
		cn.r = &timedReadWriter{cn, c}
	}
	// w io.Writer
	if cn.p.conf.WriteBufferSize > 0 {
		cn.bw = bufio.NewWriterSize(&timedReadWriter{cn, c},
			cn.p.conf.WriteBufferSize)
		cn.w = cn.bw
	} else {
		cn.w = &timedReadWriter{cn, c}
	}
}

// update connection and trigger read and
// write loops after successful dialing
func (c *Conn) triggerReadWrite(conn net.Conn) {
	c.p.Debug("trigger read/write loops of ", c.address)

	c.updateConnection(conn)
	select {
	case c.dialrl <- struct{}{}:
		select {
		case c.dialwl <- struct{}{}:
		case <-c.closed:
			return
		}
	case c.dialwl <- struct{}{}:
		select {
		case c.dialrl <- struct{}{}:
		case <-c.closed:
			return
		}
	case <-c.closed:
		return
	}
}

func (c *Conn) dial() {
	c.p.Debug("start dial loop ", c.address)
	defer c.p.Debug("stop dial loop ", c.address)

	defer c.p.await.Done()
	var (
		conn net.Conn
		err  error

		tm time.Duration // redial timeout
	)
TriggerLoop:
	for {
		select {
		case <-c.dialtr: // trigger
			c.p.Debug("redialing ", c.address)
			tm = c.p.conf.RedialTimeout // set/reset
		DialLoop:
			for {
				if conn, err = c.dialing(); err != nil {
					c.p.Printf("[ERR] error dialing %s: %v", c.address, err)
					if c.p.conf.MaxRedialTimeout > tm {
						if tm == 0 {
							tm = 100 * time.Millisecond
						} else {
							tm = tm * 2
						}
						if tm > c.p.conf.MaxRedialTimeout {
							tm = c.p.conf.MaxRedialTimeout
						}
					}
					if tm > 0 { // with timeout
						select {
						case <-time.After(tm):
							continue DialLoop
						case <-c.closed:
							return
						}
					} else { // witout timeout
						select {
						case <-c.closed:
							return
						default:
							continue DialLoop
						}
					}
				}
				// success
				c.triggerReadWrite(conn) // and update connection
				continue TriggerLoop     // (break DialLoop)
			}
		case <-c.closed:
			return
		}
	}
}

func (c *Conn) read() {
	c.p.Debug("start read loop ", c.address)
	defer c.p.Debug("stop read loop ", c.address)

	defer c.p.await.Done()
	defer c.Close()
	var (
		head []byte = make([]byte, 4)
		body []byte

		ln uint32
		l  int

		err error

		r io.Reader
	)
DialLoop:
	for {
		select {
		case <-c.dialrl: // waiting for dialing
			c.cmx.Lock() //	{
			r = c.r
			c.cmx.Unlock() // }
		case <-c.closed:
			return
		}
		c.p.Debug("start reading in loop ", c.address)
	ReadLoop:
		for {
			c.p.Debug("read message ", c.address)
			if _, err = io.ReadFull(r, head); err != nil {
				select {
				case <-c.closed:
					return
				default:
				}
				c.p.Printf("[ERR] %s reading error: %v", c.address, err)
				if c.incoming {
					return // don't redial if the connection is incoming
				}
				c.triggerDialing(err)
				continue DialLoop // waiting for redialing
			}
			// the head contains message length
			ln = binary.LittleEndian.Uint32(head)
			l = int(ln)
			if l < 0 { // negative length (32-bit CPU)
				c.p.Printf("[ERR] %s negative messge length %d", c.address, l)
				return // fatal
			}
			if c.p.conf.MaxMessageSize > 0 && l > c.p.conf.MaxMessageSize {
				c.p.Printf("[ERR] %s got messge exceeds max size allowed %d",
					c.address,
					l)
				return // fatal
			}
			body = make([]byte, l) // create new slice
			// and read it
			if _, err = io.ReadFull(r, body); err != nil {
				select {
				case <-c.closed:
					return
				default:
				}
				c.p.Printf("[ERR] %s reading error: %v",
					c.address,
					err)
				if c.incoming {
					return // don't redial if the connection is incoming
				}
				c.triggerDialing(err)
				continue DialLoop // waiting for redialing
			}
			select {
			case c.readq <- body: // receive
				c.p.Debug("msg enqueued to ReceiveQueue ", c.address)
			case <-c.closed:
				return
			}
			continue ReadLoop // semantic and code readablility
		}
	}
}

func (c *Conn) writeMsg(w io.Writer, body []byte) (terminate,
	redial bool, err error) {

	c.p.Debug("write message to ", c.address)

	if c.p.conf.MaxMessageSize > 0 &&
		len(body) > c.p.conf.MaxMessageSize {
		c.p.Panicf(
			"[CRIT] attempt to send a message exceeds"+
				" configured max size %d", len(body))
		return // terminate everything
	}

	var head []byte = make([]byte, 4)

	binary.LittleEndian.PutUint32(head, uint32(len(body)))

	// write the head
	if _, err = w.Write(head); err != nil {
		select {
		case <-c.closed:
			terminate = true
			return
		default:
		}
		c.p.Printf("[ERR] %s writing error: %v",
			c.address,
			err)
		if !c.incoming { // don't redial if the connection is incoming
			redial = true
		}
		return
	}

	// write the body
	if _, err = w.Write(body); err != nil {
		select {
		case <-c.closed:
			terminate = true
			return
		default:
		}
		c.p.Printf("[ERR] %s writing error: %v",
			c.address,
			err)
		if !c.incoming { // don't redial if the connection is incoming
			redial = true
		}
	}

	return
}

func (c *Conn) write() {
	c.p.Debug("start write loop of ", c.address)
	defer c.p.Debug("stop write loop of ", c.address)

	defer c.p.await.Done()
	defer c.Close()
	var (
		body []byte

		err error

		terminate, redial bool

		w  io.Writer
		bw *bufio.Writer
	)
DialLoop:
	for {
		select {
		case <-c.dialwl: // waiting for dialing
			c.cmx.Lock() // {
			w, bw = c.w, c.bw
			c.cmx.Unlock() // }
		case <-c.closed:
			return
		}
		c.p.Debug("start writing in loop ", c.address)
	WriteLoop:
		for {
			select {
			case body = <-c.writeq: // send
				c.p.Debug("msg was dequeued from SendQueue ", c.address)
			case <-c.closed:
				return
			}

			if terminate, redial, err = c.writeMsg(w, body); terminate {
				return
			} else if redial {
				c.triggerDialing(err)
				continue DialLoop
			}

			// write all possible messages
			for {
				select {
				case body = <-c.writeq:
					c.p.Debug("msg was dequeued from SendQueue ", c.address)
					if terminate, redial, err = c.writeMsg(w, body); terminate {
						return
					} else if redial {
						c.triggerDialing(err)
						continue DialLoop
					}
				default:
					// flush the buffer if writing is buffered
					if bw != nil {
						c.p.Debug("flush write buffer ", c.address)
						if err = bw.Flush(); err != nil {
							select {
							case <-c.closed:
								return
							default:
							}
							c.p.Printf("[ERR] %s flushing buffer error: %v",
								c.conn.RemoteAddr().String(),
								err)
							if c.incoming {
								return // don't redial if the connection is inc.
							}
							c.triggerDialing(err)
							continue DialLoop
						}
					}
				}

				continue WriteLoop // break this small write+flush loop
			}

			continue WriteLoop // semantic and code readablility
		}
	}
}

// ========================================================================== //
//                            an attached value                               //
// ========================================================================== //

// The Value returns value provided using SetValue method
func (c *Conn) Value() interface{} {
	c.vmx.Lock()
	defer c.vmx.Unlock()
	return c.val
}

// The SetValue attach any value to the connection
func (c *Conn) SetValue(val interface{}) {
	c.vmx.Lock()
	defer c.vmx.Unlock()
	c.val = val
}

// ========================================================================== //
//                               last access                                  //
// ========================================================================== //

// LastRead from underlying net.Conn
func (c *Conn) LastRead() time.Time {
	c.lrmx.Lock()
	defer c.lrmx.Unlock()
	return c.lastRead
}

// LastWrite to underlying net.Conn
func (c *Conn) LastWrite() time.Time {
	c.lwmx.Lock()
	defer c.lwmx.Unlock()
	return c.lastWrite
}

// ========================================================================== //
//                          send/receive queues                               //
// ========================================================================== //

// SendQueue returns channel for sending to the connection
func (c *Conn) SendQueue() chan<- []byte {
	return c.writeq
}

// ReceiveQueue returns receiving channel of the connection
func (c *Conn) ReceiveQueue() <-chan []byte {
	return c.readq
}

// ========================================================================== //
//                              information                                   //
// ========================================================================== //

// Address of remote node. The address will be address passed
// to (*Pool).Dial(), or remote address of underlying net.Conn
// if the connections accepted by listener
func (c *Conn) Address() string {
	c.cmx.Lock()
	defer c.cmx.Unlock()
	if c.address != "" {
		return c.address
	}
	return c.conn.RemoteAddr().String()
}

// IsIncoming reports true if the Conn accepted by listener
// and false if the Conn created using (*Pool).Dial()
func (c *Conn) IsIncoming() bool {
	return c.incoming
}

// State returns current state of the connection
func (c *Conn) State() ConnState {
	c.cmx.Lock()
	defer c.cmx.Unlock()
	return c.state
}

// ========================================================================== //
//                                  close                                     //
// ========================================================================== //

func (c *Conn) Close() (err error) {
	c.closeo.Do(func() {
		c.p.Debugf("closing %s...", c.address)
		defer c.p.Debugf("%s was closed", c.address)

		close(c.closed)
		c.closeConnection()
		c.p.delete(c.Address())
		if dh := c.p.conf.OnCloseConnection; dh != nil {
			dh(c)
		}
	})
	return
}

// Closed returns closing channel that sends
// when the connection is closed
func (c *Conn) Closed() <-chan struct{} {
	return c.closed
}

// ========================================================================== //
//                       last used and deadlines                              //
// ========================================================================== //

type timedReadWriter struct {
	c    *Conn
	conn net.Conn
}

func (t *timedReadWriter) Read(p []byte) (n int, err error) {
	if t.c.p.conf.ReadTimeout > 0 {
		err = t.conn.SetReadDeadline(time.Now().Add(t.c.p.conf.ReadTimeout))
		if err != nil {
			return
		}
	}
	if n, err = t.conn.Read(p); n > 0 {
		t.c.lrmx.Lock()
		defer t.c.lrmx.Unlock()
		t.c.lastRead = time.Now()
	}
	return
}

func (t *timedReadWriter) Write(p []byte) (n int, err error) {
	if t.c.p.conf.WriteTimeout > 0 {
		err = t.conn.SetWriteDeadline(time.Now().Add(t.c.p.conf.WriteTimeout))
		if err != nil {
			return
		}
	}
	if n, err = t.conn.Write(p); n > 0 {
		t.c.lwmx.Lock()
		defer t.c.lwmx.Unlock()
		t.c.lastWrite = time.Now()
	}
	return
}
