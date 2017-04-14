package gnet

import (
	"errors"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/node/log"
)

// pool related errors
var (
	// ErrClosed occurs when the Pool is closed
	ErrClosed = errors.New("the pool was closed")
	// ErrConnectionsLimit occurs when ... (TODO)
	ErrConnectionsLimit = errors.New("connections limit reached")
	// ErrNotFound ...(TODO)
	ErrNotFound = errors.New("not found")
)

type Pool struct {
	sync.RWMutex // conns and listener mutex

	log.Logger

	conf Config // configurations

	// registery
	reg map[reflect.Type]Prefix // registery
	rev map[Prefix]reflect.Type // inverse registery

	// pool --------------------------------------------------+
	lmx sync.RWMutex  // listener mutex                       |
	l   net.Listener  // listener                             |
	sem chan struct{} // max connections                      |
	//                                                        |
	cmx   sync.RWMutex     // connections mutex               |
	conns map[string]*Conn // remote address -> connection    |
	// -------------------------------------------------------+

	receive chan Message // receive messages

	wg        sync.WaitGroup // close
	closeOnce sync.Once
	quit      chan struct{} // quit
}

// NewPool creates new Pool instance
// with given config
func NewPool(c Config) (p *Pool) {
	c.applyDefaults()
	p = new(Pool)
	if c.Logger == nil {
		p.Logger = log.NewLogger("", false)
	} else {
		p.Logger = c.Logger
	}
	p.conf = c
	p.reg = make(map[reflect.Type]Prefix)
	p.rev = make(map[Prefix]reflect.Type)
	p.conns = make(map[string]*Conn, c.MaxConnections)
	p.receive = make(chan Message, c.ReadQueueSize)
	if c.MaxConnections != 0 {
		p.sem = make(chan struct{}, c.MaxConnections)
	}
	p.quit = make(chan struct{})
	p.Debugf(`create pool:
    max connections   %d
    max messageSize   %d
    dial timeout      %v
    read timeout      %v
    write timeout     %v
    read buffer size  %d
    write buffer size %d
    read queue size   %d
    write queue size  %d
    ping interval     %v
`,
		c.MaxConnections,
		c.MaxMessageSize,
		c.DialTimeout,
		c.ReadTimeout,
		c.WriteTimeout,
		c.ReadBufferSize,
		c.WriteBufferSize,
		c.ReadQueueSize,
		c.WriteQueueSize,
		c.PingInterval)
	return
}

// -------------------------------------------------------------------------- //
//                                 helpers                                    //
// -------------------------------------------------------------------------- //

func (p *Pool) isClosed() (closed bool) {
	select {
	case <-p.quit:
		closed = true
	default:
	}
	return
}

// -------------------------------------------------------------------------- //
//                            listen and dial                                 //
// -------------------------------------------------------------------------- //

// -------------------------------- limit ----------------------------------- //

// blocking acquire
func (p *Pool) acquireBlock() (err error) {
	if p.sem != nil {
		select {
		case p.sem <- struct{}{}: // blocking acquire
		case <-p.quit:
			err = ErrClosed
		}
	}
	return
}

// try acquire
func (p *Pool) acquire() (got bool) {
	if got = (p.sem == nil); !got {
		select {
		case p.sem <- struct{}{}:
			got = true
		default:
		}
	}
	return
}

func (p *Pool) release() {
	if p.sem != nil {
		<-p.sem
	}
}

// ------------------------- limited connection ----------------------------- //

type limitedConnection struct {
	net.Conn
	releaseOnce sync.Once
	release     func()
}

func (l *limitedConnection) Close() (err error) {
	err = l.Conn.Close()
	l.releaseOnce.Do(l.release)
	return
}

// ------------------------------- accept ----------------------------------- //

func (p *Pool) accept(l net.Listener) (c net.Conn, err error) {
	if err = p.acquireBlock(); err != nil {
		return
	}
	if c, err = l.Accept(); err != nil {
		p.release()
		return
	}
	if p.sem == nil {
		return
	}
	c = &limitedConnection{
		Conn:    c,
		release: p.release,
	}
	return
}

// ------------------------------- listen ----------------------------------- //

// Listen start listening on given address
func (p *Pool) Listen(address string) (err error) {
	// don't listen if the pool was closed
	if p.isClosed() {
		err = ErrClosed
		return
	}
	p.lmx.Lock()
	defer p.lmx.Unlock()
	if p.l, err = net.Listen("tcp", address); err != nil {
		return
	}
	p.Print("[INF] listening on ", p.l.Addr().String())
	p.wg.Add(1)
	go p.listen(p.l)
	return
}

func (p *Pool) listen(l net.Listener) {
	var (
		c   net.Conn
		err error
	)
	// closing
	defer p.wg.Done()
	defer l.Close()
	defer p.Debug("stop accept loop")
	// accept loop
	p.Debug("start accept loop")
	for {
		if c, err = p.accept(l); err != nil {
			select {
			case <-p.quit:
				err = nil // drop "use of closed network connection" error
				return
			default:
			}
			p.Print("[ERR] accept error: ", err)
			return
		}
		if err = p.handleConnection(c, false); err != nil {
			p.Print("[ERR] error handling connection: ", err)
		}
	}
}

// --------------------------------- dial ----------------------------------- //

func (p *Pool) dial(address string) (c net.Conn, err error) {
	// check out limit of connections
	if !p.acquire() {
		err = ErrConnectionsLimit
		return
	}
	if c, err = net.Dial("tcp", address); err != nil {
		p.release()
		return
	}
	if p.sem == nil {
		return
	}
	c = &limitedConnection{
		Conn:    c,
		release: p.release,
	}
	return
}

func (p *Pool) dialTimeout(address string,
	timeout time.Duration) (c net.Conn, err error) {

	// check out limit of connections
	if !p.acquire() {
		err = ErrConnectionsLimit
		return
	}
	if c, err = net.DialTimeout("tcp", address, timeout); err != nil {
		p.release()
		return
	}
	if p.sem == nil {
		return
	}
	c = &limitedConnection{
		Conn:    c,
		release: p.release,
	}
	return
}

// ------------------------------- connect ---------------------------------- //

// Connect to given address. The Connect is blocking method and should
// not be called from main processing thread
func (p *Pool) Connect(address string) (err error) {
	var (
		ok bool
		c  net.Conn
	)
	// don't connect if the pool was closed
	if p.isClosed() {
		err = ErrClosed
		return
	}
	// preliminary check
	p.cmx.RLock()                     // >--------------------+
	if _, ok = p.conns[address]; ok { //                      |
		p.cmx.RUnlock()            // <-----------------------+
		err = ErrConnAlreadyExists //                         |
		return                     //                         |
	} //                                                      |
	p.cmx.RUnlock() // <--------------------------------------+
	// dial
	if p.conf.DialTimeout > 0 {
		c, err = p.dialTimeout(address, p.conf.DialTimeout)
	} else {
		c, err = p.dial(address)
	}
	if err != nil {
		return
	}
	// with check of p.conns
	err = p.handleConnection(c, true)
	return
}

// -------------------------------- handle ---------------------------------- //

func (p *Pool) handleConnection(c net.Conn, outgoing bool) (err error) {
	p.Debug("got new connection: ", c.RemoteAddr().String())
	p.cmx.Lock()
	defer p.cmx.Unlock()
	var (
		address = c.RemoteAddr().String()
		ok      bool

		x *Conn
	)
	if _, ok = p.conns[address]; ok {
		err = ErrConnAlreadyExists
		c.Close()
		return
	}
	x = newConn(c, p, outgoing)
	p.conns[address] = x // add the connection to the pool
	x.handle()           // async non-blocking method
	if p.conf.ConnectionHandler != nil {
		p.conf.ConnectionHandler(x) // invoke the callback
	}
	return
}

// ------------------------------ disconnect -------------------------------- //

func (p *Pool) get(address string) (c *Conn) {
	p.cmx.RLock()
	defer p.cmx.RUnlock()
	return p.conns[address]
}

// Disconnect connection with given address. It returns
// closing error if any or ErrNotFound if connection
// doesn't exist
func (p *Pool) Disconnect(address string) (err error) {
	if c := p.get(address); c != nil {
		c.setDisconnectReason(ErrManualDisconnect)
		err = c.Close()
	} else {
		err = ErrNotFound
	}
	return
}

// -------------------------------------------------------------------------- //
//                                 encode                                     //
// -------------------------------------------------------------------------- //

func (p *Pool) encodeMessage(m interface{}) (data []byte) {
	var (
		val    reflect.Value = reflect.Indirect(reflect.ValueOf(m))
		prefix Prefix
		ok     bool

		em []byte
	)
	if prefix, ok = p.reg[val.Type()]; !ok {
		p.Panicf("send unregistered message: %T", m)
	}
	em = encoder.Serialize(m)
	if p.conf.MaxMessageSize > 0 && len(em) > p.conf.MaxMessageSize {
		p.Panicf(
			"try to send message greater than size limit %T, %d (%d)",
			m, len(em), p.conf.MaxMessageSize,
		)
	}
	data = make([]byte, 0, 4+PrefixLength+len(em))
	data = append(data, prefix[:]...)
	data = append(data, encoder.SerializeAtomic(uint32(len(em)))...)
	data = append(data, em...)
	return
}

// -------------------------------------------------------------------------- //
//                              broadcasting                                  //
// -------------------------------------------------------------------------- //

// BroadcastExcept given message to all connections of the Pool
// except enumerated by 'except' argument
func (p *Pool) BroadcastExcept(m interface{}, except ...string) {
	var (
		em []byte = p.encodeMessage(m)

		address, e string
		c          *Conn

		err error
	)
	p.cmx.RLock()         // map lock
	defer p.cmx.RUnlock() // and unlock
Loop:
	for address, c = range p.conns {
		for _, e = range except {
			if address == e {
				continue Loop // except
			}
		}
		p.cmx.RUnlock() // temporary unlock
		{
			if err = c.sendEncodedMessage(em); err != nil {
				c.setDisconnectReason(err)
				p.Printf("[ERR] %s error sending message: %v", c.Addr(), err)
				c.Close()
			}
		}
		p.cmx.RLock() // and lock back
	}
}

// Broadcast given message to all connections of the Pool
func (p *Pool) Broadcast(m interface{}) {
	p.BroadcastExcept(m)
}

// remove closed connection form the Pool
func (p *Pool) delete(address string) {
	p.cmx.Lock()
	defer p.cmx.Unlock()
	delete(p.conns, address)
}

// -------------------------------------------------------------------------- //
//                                 registry                                   //
// -------------------------------------------------------------------------- //

// Register given message with given prefix. The method panics
// if given prefix invalid or already registered. It also panics
// if given type alredy registered
func (p *Pool) Register(prefix Prefix, msg interface{}) {
	var (
		ok  bool
		typ reflect.Type = reflect.Indirect(reflect.ValueOf(msg)).Type()
		err error
	)
	if err = prefix.Validate(); err != nil {
		p.Panicf("%s: %v", prefix.String(), err)
	}
	encoder.Serialize(msg) // panic if the msg can't be serialized
	if _, ok = p.reg[typ]; ok {
		p.Panicf("attemt to register type twice: %s", typ.String())
	}
	if _, ok = p.rev[prefix]; ok {
		p.Panicf("attempt to register prefix twice: %s", prefix.String())
	}
	p.reg[typ] = prefix
	p.rev[prefix] = typ
}

// -------------------------------------------------------------------------- //
//                                information                                 //
// -------------------------------------------------------------------------- //

// Address returns listening address or empty string
func (p *Pool) Address() (address string) {
	p.lmx.RLock()
	defer p.lmx.RUnlock()
	if p.l != nil {
		address = p.l.Addr().String()
	}
	return
}

// Connections returns list of all connections
func (p *Pool) Connections() (cs []string) {
	p.cmx.RLock()
	defer p.cmx.RUnlock()
	if len(p.conns) == 0 {
		return
	}
	var a string
	cs = make([]string, 0, len(cs))
	for a = range p.conns {
		cs = append(cs, a)
	}
	return
}

// IsConnExists returns true if connections with
// given remote address exists in the Pool
func (p *Pool) IsConnExist(address string) (yep bool) {
	p.cmx.RLock()
	defer p.cmx.RUnlock()
	_, yep = p.conns[address]
	return
}

// -------------------------------------------------------------------------- //
//                                   inflow                                   //
// -------------------------------------------------------------------------- //

// Receive returns channel of recieved data with
// connections from which the data come from. When
// Pool closed the channel is closed too. I.e.
// it can be used in range reading loop
//
//     // handle incoming messages until Pool closed
//     go func(receive <-chan gnet.Message){
//         for msg :=<-receive {
//             // handle msg
//         }
//     }(pool.Receive())
//
func (p *Pool) Receive() <-chan Message {
	return p.receive
}

// -------------------------------------------------------------------------- //
//                                   close                                    //
// -------------------------------------------------------------------------- //

func (p *Pool) closeConnections() {
	p.cmx.RLock()         // map lock
	defer p.cmx.RUnlock() // and unlock
	for _, c := range p.conns {
		p.cmx.RUnlock() // temporary unlock
		{
			c.Close()
		}
		p.cmx.RLock() // and lock back
	}
}

func (p *Pool) closeListener() (err error) {
	p.lmx.RLock()
	defer p.lmx.RUnlock()
	if p.l != nil {
		err = p.l.Close()
	}
	return
}

// Close listener and all connections related to the Pool
// It can return listener closing error
func (p *Pool) Close() (err error) {
	// chan
	p.closeOnce.Do(func() {
		close(p.quit)
	})
	// listener
	err = p.closeListener()
	// connections
	p.closeConnections()
	// await all goroutines
	p.wg.Wait()
	// release receiving channel
	close(p.receive)
	return
}
