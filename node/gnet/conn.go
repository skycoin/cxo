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

// there are three possible states of a conenction
const (
	ConnStateConnected ConnState = iota // conncetion works
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

	cmx   sync.Mutex // connection lock for redialing
	conn  net.Conn
	state ConnState

	r io.Reader
	w io.Writer

	bw *bufio.Writer // if writing is buffered

	incoming bool

	readq  chan []byte
	writeq chan []byte

	dialo  *sync.Once    // trigger dialing once per connection fail
	dialtr chan struct{} // trigger dialing
	dialrl chan struct{} // redialing (lock read)
	dialwl chan struct{} // redialing (lock write)

	lrmx      sync.Mutex
	lastRead  time.Time
	lwmx      sync.Mutex
	lastWrite time.Time

	p *Pool // logger and configs

	vmx sync.Mutex
	val interface{}

	closeo sync.Once
	closed chan struct{}
}

// accept connection by listener
func (p *Pool) acceptConnection(c net.Conn) (cn *Conn, err error) {
	p.cmx.Lock()
	defer p.cmx.Unlock()
	var got bool
	if _, got = p.conns[c.RemoteAddr().String()]; got {
		err = fmt.Errorf("connection already exists %s",
			c.RemoteAddr().String())
		return
	}
	cn = new(Conn)

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
	cn = new(Conn)

	p.conns[address] = cn // save
	cn.p = p

	cn.incoming = false

	cn.readq = make(chan []byte, p.conf.ReadQueueLen)
	cn.writeq = make(chan []byte, p.conf.WriteQueueLen)

	cn.dialrl = make(chan struct{})
	cn.dialwl = make(chan struct{})

	cn.closed = make(chan struct{})

	p.await.Add(3)
	go cn.dial()
	go cn.read()
	go cn.write()

	cn.triggerDialing()

	return
}

// ========================================================================== //
//                             dial/read/write                                //
// ========================================================================== //

func (c *Conn) closeConnection() {
	c.cmx.Lock()
	defer c.cmx.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.state = ConnStateClosed
	}
}

func (cn *Conn) dialing() (c net.Conn, err error) {
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

func (c *Conn) triggerDialing() {
	c.cmx.Lock()
	defer c.cmx.Unlock()
	c.dialo.Do(func() {
		if c.conn != nil {
			c.conn.Close() // make sure that conn is closed
		}
		c.state = ConnStateDialing
		select {
		case c.dialtr <- struct{}{}:
		case <-c.closed:
		}
	})
}

func (c *Conn) changeState(state ConnState) {
	c.cmx.Lock()
	defer c.cmx.Unlock()
	c.state = state
}

func (cn *Conn) updateConnection(c net.Conn) {
	cn.cmx.Lock()
	defer cn.cmx.Unlock()

	cn.dialo = new(sync.Once) // refresh

	cn.conn = c
	cn.state = ConnStateConnected

	// r io.Reader
	if cn.p.conf.ReadBufferSize > 0 { // buffered
		cn.r = bufio.NewReaderSize(&timedReadWriter{cn},
			cn.p.conf.ReadBufferSize)
	} else { // unbuffered
		cn.r = &timedReadWriter{cn}
	}
	// w io.Writer
	if cn.p.conf.WriteBufferSize > 0 {
		cn.bw = bufio.NewWriterSize(&timedReadWriter{cn},
			cn.p.conf.WriteBufferSize)
		cn.w = cn.bw
	} else {
		cn.w = &timedReadWriter{cn}
	}
}

// update connection and trigger read and
// write loops after successfull dialing
func (c *Conn) triggerReadWrite(conn net.Conn) {
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
	ReadLoop:
		for {
			if _, err = io.ReadFull(r, head); err != nil {
				select {
				case <-c.closed:
					return
				default:
				}
				c.p.Printf("[ERR] %s reading error: %v",
					c.conn.RemoteAddr().String(),
					err)
				c.triggerDialing()
				continue DialLoop // waiting for redialing
			}
			// the head contains message length
			ln = binary.LittleEndian.Uint32(head)
			l = int(ln)
			if l < 0 { // negative length (32-bit CPU)
				c.p.Printf("[ERR] %s negative messge length %d",
					c.conn.RemoteAddr().String(),
					l)
				return // fatal
			}
			if c.p.conf.MaxMessageSize > 0 && l > c.p.conf.MaxMessageSize {
				c.p.Printf("[ERR] %s got messge exceeds max size allowed %d",
					c.conn.RemoteAddr().String(),
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
					c.conn.RemoteAddr().String(),
					err)
				c.triggerDialing()
				continue DialLoop // waiting for redialing
			}
			select {
			case c.readq <- body: // receive
			case <-c.closed:
				return
			}
			continue ReadLoop // semantic and code readablility
		}
	}
}

func (c *Conn) writeMsg(w io.Writer, head, body []byte) (terminate,
	redial bool) {

	if c.p.conf.MaxMessageSize > 0 &&
		len(body) > c.p.conf.MaxMessageSize {
		c.p.Panicf(
			"[CRIT] attempt to send a message exceeds"+
				" configured max size %d", len(body))
		return // terminate everything
	}

	binary.LittleEndian.PutUint32(head, uint32(len(body)))

	var err error

	// write the head
	if _, err = w.Write(head); err != nil {
		select {
		case <-c.closed:
			terminate = true
			return
		default:
		}
		c.p.Printf("[ERR] %s writing error: %v",
			c.conn.RemoteAddr().String(),
			err)
		redial = true
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
			c.conn.RemoteAddr().String(),
			err)
		redial = true
	}

	return
}

func (c *Conn) write() {
	defer c.p.await.Done()
	defer c.Close()
	var (
		head, body []byte = make([]byte, 4), nil

		err error

		terminate, redial bool

		w  io.Writer
		bw *bufio.Writer
	)
DialLoop:
	for {
		select {
		case <-c.dialrl: // waiting for dialing
			c.cmx.Lock() // {
			w, bw = c.w, c.bw
			c.cmx.Unlock() // }
		case <-c.closed:
			return
		}
	WriteLoop:
		for {
			select {
			case body = <-c.writeq: // receive
			case <-c.closed:
				return
			}

			switch terminate, redial = c.writeMsg(w, head, body); {
			case terminate:
				return
			case redial:
				c.triggerDialing()
				continue DialLoop
			}

			// write all possible messages
			for {
				select {
				case body = <-c.writeq:
					switch terminate, redial = c.writeMsg(w, head, body); {
					case terminate:
						return
					case redial:
						c.triggerDialing()
						continue DialLoop
					}
				default:
					// flush the buffer if writing is buffered
					if bw != nil {
						if err = bw.Flush(); err != nil {
							select {
							case <-c.closed:
								return
							default:
							}
							c.p.Printf("[ERR] %s flushing buffer error: %v",
								c.conn.RemoteAddr().String(),
								err)
							c.triggerDialing()
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
		close(c.closed)
		c.closeConnection()
		c.p.delete(c.Address())
		if dh := c.p.conf.DisconnectHandler; dh != nil {
			dh(c)
		}
	})
	return
}

// ========================================================================== //
//                       last used and deadlines                              //
// ========================================================================== //

type timedReadWriter struct {
	c *Conn
}

func (t *timedReadWriter) Read(p []byte) (n int, err error) {
	if t.c.p.conf.ReadTimeout > 0 {
		err = t.c.conn.SetReadDeadline(time.Now().Add(t.c.p.conf.ReadTimeout))
		if err != nil {
			return
		}
	}
	if n, err = t.c.conn.Read(p); n > 0 {
		t.c.lrmx.Lock()
		defer t.c.lrmx.Unlock()
		t.c.lastRead = time.Now()
	}
	return
}

func (t *timedReadWriter) Write(p []byte) (n int, err error) {
	if t.c.p.conf.WriteTimeout > 0 {
		err = t.c.conn.SetWriteDeadline(time.Now().Add(t.c.p.conf.WriteTimeout))
		if err != nil {
			return
		}
	}
	if n, err = t.c.conn.Write(p); n > 0 {
		t.c.lwmx.Lock()
		defer t.c.lwmx.Unlock()
		t.c.lastWrite = time.Now()
	}
	return
}
