package gnet

import (
	"errors"
	"net"
	"reflect"
	"sync"

	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/node/log"
)

// pool related errors
var (
	// ErrClosed occurs when the Pool is closed
	ErrClosed = errors.New("the pool was closed")
	// ErrConnectionsLimit occurs when ... (TODO)
	ErrConnectionsLimit = errors.New("connections limit reached")
)

type Pool struct {
	sync.RWMutex // conns and listener mutex

	log.Logger

	conf Config // configurations

	// registery
	reg map[reflect.Type]Prefix // registery
	rev map[Prefix]reflect.Type // inverse registery

	// pool
	l       net.Listener     // listener
	conns   map[string]*Conn // remote address -> connection
	receive chan Message     // receive messages
	sem     chan struct{}    // max connections

	wg        sync.WaitGroup // close
	closeOnce sync.Once
	quit      chan struct{} // quit
}

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

// acquire returns true if number of connections
// less then p.conf.MaxConnections
func (p *Pool) acquire() (ok bool) {
	if p.conf.MaxConnections == 0 {
		return true // no limit for connections
	}
	select {
	case p.sem <- struct{}{}:
		ok = true // acquire
	default:
	}
	return
}

// blocking acquire (for l.Accept())
func (p *Pool) acquireBlock() {
	if p.conf.MaxConnections == 0 {
		return // no limit for connections
	}
	select {
	case p.sem <- struct{}{}: // acquire
	case <-p.quit:
	}
}

// release connection
func (p *Pool) release() {
	if p.conf.MaxConnections == 0 {
		return // no limit for connections
	}
	select {
	case <-p.sem: // release
	case <-p.quit:
	}
}

// BroadcastExcept given message to all connections of the Pool
// except enumerated by 'except' argument
func (p *Pool) BroadcastExcept(m interface{}, except ...string) {
	var (
		em []byte = p.encodeMessage(m)

		address, e string
		c          *Conn

		err error
	)
	p.RLock() // map lock
	defer p.RUnlock()
Loop:
	for address, c = range p.conns {
		for _, e = range except {
			if address == e {
				continue Loop // except
			}
		}
		if err = c.sendEncodedMessage(em); err != nil {
			p.Printf("[ERR] %s error sending message: %v", c.Addr(), err)
			c.close(closeDontRemove) // terminate connection
			delete(p.conns, address) // and remove from the pool
		}
	}
}

// Broadcast given message to all connections of the Pool
func (p *Pool) Broadcast(m interface{}) {
	p.BroadcastExcept(m)
}

// Listen start listening on given address
func (p *Pool) Listen(address string) (err error) {
	p.Lock()
	defer p.Unlock()
	if p.l, err = net.Listen("tcp", address); err != nil {
		return
	}
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
		p.acquireBlock() // don't block for unlimited connections
		if c, err = l.Accept(); err != nil {
			select {
			case <-p.quit:
				err = nil // drop "use of closed network connection" error
				return
			default:
			}
			p.Print("[ERR] accept error: ", err)
			return
		}
		if err = p.handleConnection(c); err != nil {
			p.Print("[ERR] error handling connection: ", err)
		}
	}
}

func (p *Pool) handleConnection(c net.Conn) (err error) {
	p.Debug("got new connection: ", c.RemoteAddr().String())
	p.Lock()
	defer p.Unlock()
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
	x = newConn(c, p)
	p.conns[address] = x // add the connection to the pool
	x.handle()           // async non-blocking method
	if p.conf.ConnectionHandler != nil {
		p.conf.ConnectionHandler(x) // invoke the callback
	}
	return
}

// remove closed connection form the Pool
func (p *Pool) removeConnection(address string) {
	p.Lock()
	defer p.Unlock()
	delete(p.conns, address)
}

// Connect to given address. The Connect is blocking method and should
// not be called from main processing thread
func (p *Pool) Connect(address string) (err error) {
	var (
		ok bool
		c  net.Conn
	)
	// don't connect if the pool was closed
	select {
	case <-p.quit:
		err = ErrClosed
		return
	default:
	}
	// check out limit of connections
	if ok = p.acquire(); !ok {
		err = ErrConnectionsLimit
		return
	}
	// preliminary check
	p.RLock()
	if _, ok = p.conns[address]; ok {
		p.RUnlock()
		err = ErrConnAlreadyExists
		return
	}
	p.RUnlock()
	// dial
	if p.conf.DialTimeout > 0 {
		c, err = net.DialTimeout("tcp", address, p.conf.DialTimeout)
		if err != nil {
			return
		}
	} else {
		if c, err = net.Dial("tcp", address); err != nil {
			return
		}
	}
	// with check of p.conns
	err = p.handleConnection(c)
	return
}

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

// Disconnect connection with given address
func (p *Pool) Disconnect(address string) {
	p.Lock()
	defer p.Unlock()
	if c, ok := p.conns[address]; ok {
		c.close(closeDontRemove) // don't remove
		delete(p.conns, address) // and remove
	}
}

// Address returns listening address or empty string
func (p *Pool) Address() (address string) {
	p.RLock()
	defer p.RUnlock()
	if p.l != nil {
		address = p.l.Addr().String()
	}
	return
}

// The Close close listener and all connections related to the Pool
func (p *Pool) Close() {
	p.closeOnce.Do(func() {
		close(p.quit)
	})
	p.Lock()
	defer p.Unlock()
	if p.l != nil {
		p.l.Close()
	}
	for a, c := range p.conns {
		c.close(closeDontRemove) // don't remove
		delete(p.conns, a)       // and remove
	}
	p.wg.Wait() // await all goroutines
	return
}

// Connections returns list of all connections
func (p *Pool) Connections() (cs []string) {
	p.RLock()
	defer p.RUnlock()
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

// Receive returns channel of recieved data with
// connections from which the dat come from
func (p *Pool) Receive() <-chan Message {
	return p.receive
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
	p.Printf(`[INF] create pool:
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

func (p *Pool) IsConnExist(address string) (yep bool) {
	p.RLock()
	defer p.RUnlock()
	_, yep = p.conns[address]
	return
}
