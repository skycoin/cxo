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

	// cmx locks conn, and state fields
	cmx   sync.Mutex // connection lock for redialing
	conn  net.Conn
	state ConnState

	incoming bool

	readq  chan []byte
	writeq chan []byte

	dialtr chan error    // trigger dialing
	dialrl chan net.Conn // redialing (lock read)
	dialwl chan net.Conn // redialing (lock write)

	readd  chan struct{} // reading loop waits for redialing
	writed chan struct{} // write loop waits for redialing

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

	cn.incoming = true

	cn.readq = make(chan []byte, p.conf.ReadQueueLen)
	cn.writeq = make(chan []byte, p.conf.WriteQueueLen)

	cn.dialrl = make(chan net.Conn)
	cn.dialwl = make(chan net.Conn)

	// don't use readd and writed for incoming connections

	cn.closed = make(chan struct{})

	p.await.Add(2)
	go cn.read()
	go cn.write()

	// update connection and start read and write loops
	cn.triggerReadWrite(c)

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

	cn.dialtr = make(chan error)
	cn.dialrl = make(chan net.Conn) // not buffered
	cn.dialwl = make(chan net.Conn) // not buffered

	cn.readd = make(chan struct{})
	cn.writed = make(chan struct{})

	cn.closed = make(chan struct{})

	p.await.Add(3)
	go cn.read()
	go cn.write()
	go cn.dial(p.conf.DialsLimit)

	cn.triggerDialing(nil)

	return
}

// ========================================================================== //
//                             dial/read/write                                //
// ========================================================================== //

func (c *Conn) triggerDialing(err error) {
	c.p.Debugf("trigger dialing of %s by %v", c.address, err)

	c.cmx.Lock()
	defer c.cmx.Unlock()

	c.state = ConnStateDialing

	select {
	case c.dialtr <- err:
	case <-c.closed:
	}
}

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

// triggered by reading loop
func (c *Conn) triggerDialingRead(err error, conn net.Conn) {
	// close connection; don't need to lock because net.Conn
	// is thread safe and we aren't using c.conn field
	conn.Close()

	if c.incoming {
		c.Close() // terminate
		return    // don't redial if conenction is incoming
	}

	c.p.Debugf("triggerDialingRead of %s: %v", conn.RemoteAddr().String(), err)

	// may be writing loop already waits
	// reading loop to fail, thus we use
	// error of writing loop (first error)
	select {
	case <-c.writed:
		c.p.Debug("dialing deligated to writing trigger (1)")
		return // leave redialing for triggerRedialWrite
	default:
	}

	// wait until write loop triggers dialing
	select {
	case c.readd <- struct{}{}:
		c.p.Debug("dialing triggered by reading triggers")
		c.triggerDialing(err)
	case <-c.writed:
		c.p.Debug("dialing deligated to writing trigger (2)")
	case <-c.closed:
		return
	}

}

// triggered by writing loop
func (c *Conn) triggerDialingWrite(err error, conn net.Conn) {

	conn.Close()

	if c.incoming {
		c.Close() // terminate
		return
	}

	c.p.Debugf("triggerDialingWrite of %s: %v", conn.RemoteAddr().String(), err)

	select {
	case <-c.readd:
		c.p.Debug("dialing deligated to reading trigger (1)")
		return
	default:
	}

	select {
	case c.writed <- struct{}{}:
		c.p.Debug("dialing triggered by writing trigger")
		c.triggerDialing(err)
	case <-c.readd:
		c.p.Debug("dialing deligated to reading trigger (2)")
	case <-c.closed:
		return
	}
}

// create io.Reader from net.Conn, that can be buffered (if configured)
// and keeps last reading time
func (c *Conn) connectionReader(conn net.Conn) (r io.Reader) {
	if c.p.conf.ReadBufferSize > 0 { // buffered
		r = bufio.NewReaderSize(&timedReadWriter{c, conn},
			c.p.conf.ReadBufferSize)
	} else { // unbuffered
		r = &timedReadWriter{c, conn}
	}
	return
}

// create io.Writer (and *bufio.Writer if configured), that
// keeps last writing time
func (c *Conn) connectionWriter(conn net.Conn) (w io.Writer, bw *bufio.Writer) {
	if c.p.conf.WriteBufferSize > 0 {
		bw = bufio.NewWriterSize(&timedReadWriter{c, conn},
			c.p.conf.WriteBufferSize)
		w = bw
	} else {
		w = &timedReadWriter{c, conn}
	}
	return
}

func (cn *Conn) updateConnection(c net.Conn) {
	cn.p.Debug("update connection of ", cn.address)

	cn.cmx.Lock()
	defer cn.cmx.Unlock()

	cn.conn = c
	cn.state = ConnStateConnected
}

// update connection and trigger read and
// write loops after successful dialing
func (c *Conn) triggerReadWrite(conn net.Conn) {
	c.p.Debug("trigger read/write loops of ", c.address)

	c.updateConnection(conn)
	select {
	case c.dialrl <- conn:
		select {
		case c.dialwl <- conn:
		case <-c.closed:
			return
		}
	case c.dialwl <- conn:
		select {
		case c.dialrl <- conn:
		case <-c.closed:
			return
		}
	case <-c.closed:
		return
	}
}

func (c *Conn) isClosed() (closed bool) {
	select {
	case <-c.closed:
		closed = true
	default:
	}
	return
}

func (c *Conn) dial(diallm int) {
	c.p.Debug("start dial loop ", c.address)
	defer c.p.Debug("stop dial loop ", c.address)

	defer c.p.await.Done()
	defer c.Close()

	var (
		conn net.Conn
		err  error

		tm time.Duration // redial timeout
	)

	// for infinity redials
	if diallm == 0 {
		diallm-- // = -1
	}

TriggerLoop: // -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -
	for {

		select {
		case err = <-c.dialtr: // trigger
			c.p.Debug("redialing ", c.address)
			tm = c.p.conf.RedialTimeout // set/reset

		DialLoop: // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
			for {

				if diallm == 0 { // check out dialing limit
					c.p.Debug("dials limit exceeded: ", c.address)
					return // close
				}
				diallm--

				// perform dialing callback
				if callback := c.p.conf.OnDial; callback != nil {
					if err = callback(c, err); err != nil {
						c.p.Debug("dialing terminanted by OnDial callback: ",
							err)
						return // we don't want to redial anymore (close)
					}
				}

				if conn, err = c.dialing(); err != nil {
					// -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -
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
						if c.isClosed() {
							return
						}
						continue DialLoop
					}
					// -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -
				}

				// success
				c.triggerReadWrite(conn) // and update connection
				continue TriggerLoop     // (break DialLoop)

			} // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

		case <-c.closed:
			return
		}

	} // -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --

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

		r    io.Reader
		conn net.Conn
	)

DialLoop: // -------------------------------------------------------------------
	for {
		c.p.Debug("read: DialLoop")
		select {
		case conn = <-c.dialrl: // waiting for dialing
			r = c.connectionReader(conn)
		case <-c.closed:
			return
		}
		c.p.Debug("start reading in loop ", c.address)

	ReadLoop: // -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --
		for {
			c.p.Debug("read: ReadLoop")
			c.p.Debug("read message ", c.address)
			if _, err = io.ReadFull(r, head); err != nil {
				if c.isClosed() {
					return
				}
				c.p.Printf("[ERR] %s reading error: %v", c.address, err)
				c.p.Debug("triggerDialingRead")
				c.triggerDialingRead(err, conn)
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
				if c.isClosed() {
					return
				}
				c.p.Printf("[ERR] %s reading error: %v",
					c.address,
					err)
				c.p.Debug("triggerDialingRead")
				c.triggerDialingRead(err, conn)
				continue DialLoop // waiting for redialing
			}
			select {
			case c.readq <- body: // receive
				c.p.Debug("msg enqueued to ReceiveQueue ", c.address)
			case <-c.closed:
				return
			}
			continue ReadLoop // semantic and code readablility
		} // -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -

	} // -----------------------------------------------------------------------

}

func (c *Conn) writeMsg(w io.Writer, body []byte) (terminate bool, err error) {

	c.p.Debug("write message to ", c.address)

	if c.p.conf.MaxMessageSize > 0 &&
		len(body) > c.p.conf.MaxMessageSize {
		c.p.Printf(
			"[CRIT] attempt to send a message exceeds"+
				" configured max size %d", len(body))
		terminate = true
		return // terminate everything
	}

	var head []byte = make([]byte, 4)

	binary.LittleEndian.PutUint32(head, uint32(len(body)))

	// write the head
	if _, err = w.Write(head); err != nil {
		if c.isClosed() {
			terminate = true
			return
		}
		c.p.Printf("[ERR] %s writing error: %v",
			c.address,
			err)
		return
	}

	// write the body
	if _, err = w.Write(body); err != nil {
		if c.isClosed() {
			terminate = true
			return
		}
		c.p.Printf("[ERR] %s writing error: %v",
			c.address,
			err)
		return
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

		terminate bool

		conn net.Conn
		w    io.Writer
		bw   *bufio.Writer
	)
DialLoop: // -------------------------------------------------------------------
	for {
		c.p.Debug("write: DialLoop")
		select {
		case conn = <-c.dialwl: // waiting for dialing
			w, bw = c.connectionWriter(conn)
		case <-c.closed:
			return
		}
		c.p.Debug("start writing in loop ", c.address)

	WriteLoop: // -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --
		for {
			c.p.Debug("write: WriteLoop")
			select {
			case body = <-c.writeq: // send
				c.p.Debug("msg was dequeued from SendQueue ", c.address)
			case <-c.readd:
				// redialing triggered by reading loop
				c.p.Debug("delegate dialing to reading trigger")
				continue DialLoop
			case <-c.closed:
				return
			}

			if terminate, err = c.writeMsg(w, body); terminate {
				return
			} else if err != nil {
				c.p.Debug("triggerDialingWrite")
				c.triggerDialingWrite(err, conn)
				continue DialLoop
			}

			// write all possible messages and then
			// flush writing buffer if there is
			for { // -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -
				select {
				case body = <-c.writeq:

					c.p.Debug("msg was dequeued from SendQueue ", c.address)
					if terminate, err = c.writeMsg(w, body); terminate {
						return
					} else if err != nil {
						c.p.Debug("triggerDialingWrite")
						c.triggerDialingWrite(err, conn)
						continue DialLoop
					}

				default:

					// flush the buffer if writing is buffered
					if bw != nil {
						c.p.Debug("flush write buffer ", c.address)
						if err = bw.Flush(); err != nil {
							if c.isClosed() {
								return
							}
							c.p.Printf("[ERR] %s flushing buffer error: %v",
								c.conn.RemoteAddr().String(),
								err)
							c.p.Debug("triggerDialingWrite")
							c.triggerDialingWrite(err, conn)
							continue DialLoop
						}
					}

					continue WriteLoop // break this small write+flush loop
				}
			} // -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -  -

			// continue WriteLoop
		} // -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -

	} // -----------------------------------------------------------------------

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
func (c *Conn) Address() (address string) {
	return c.address
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

// Conn returns underlying net.Conn. It can returns nil
// or closed connection. The method is useful if you want
// to get local/remote addresses of the Conn. Keep in mind
// that underlying net.Conn can be changed anytime
func (c *Conn) Conn() net.Conn {
	c.cmx.Lock()
	defer c.cmx.Unlock()

	return c.conn
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
