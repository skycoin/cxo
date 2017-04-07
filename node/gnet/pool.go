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
	l       net.Listener         // listener
	conns   map[string]*Conn     // remote address -> connection
	receive chan receivedMessage // receive messages
	sem     chan struct{}        // max connections

	// any interface{} provided to be used
	// in Handle method of Message
	user interface{}

	closeOnce sync.Once
	quit      chan struct{} // quit
}

func (p *Pool) encodeMessage(m Message) (data []byte) {
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
func (p *Pool) BroadcastExcept(m Message, except ...string) {
	var (
		em []byte = p.encodeMessage(m)

		address, e string
		c          *Conn

		err error
	)
	p.RLock() // map lock
	defer p.RUnlock()
	for address, c = range p.conns {
		for _, e = range except {
			if address == e {
				continue // except
			}
		}
		if err = c.sendEncodedMessage(em); err != nil {
			p.Printf("[ERR] %s error sending message: %v", c.Addr(), err)
		}
	}
}

// Broadcast given message to all connections of the Pool
func (p *Pool) Broadcast(m Message) {
	p.BroadcastExcept(m)
}

// Listen start listening on given address
func (p *Pool) Listen(address string) (err error) {
	p.Lock()
	defer p.Unlock()
	if p.l, err = net.Listen("tcp", address); err != nil {
		return
	}
	go p.listen(p.l)
	return
}

func (p *Pool) listen(l net.Listener) {
	var (
		c   net.Conn
		err error
	)
	defer l.Close()
	for {
		p.acquireBlock()
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

// remove closed connection form the pool
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
func (p *Pool) Register(prefix Prefix, msg Message) {
	var (
		ok  bool
		typ reflect.Type = reflect.Indirect(reflect.ValueOf(msg)).Type()
		err error
	)
	if err = prefix.Validate(); err != nil {
		p.Panicf("%s: %v", prefix.String(), err)
	}
	if _, ok = p.reg[typ]; ok {
		p.Panicf("attemt to register type twice: %s", typ.String())
	}
	if _, ok = p.rev[prefix]; ok {
		p.Panicf("attempt to register prefix twice: %s", prefix.String())
	}
	p.reg[typ] = prefix
	p.rev[prefix] = typ
}

func (p *Pool) Disconnect(address string) {
	p.Lock()
	defer p.Unlock()
	if c, ok := p.conns[address]; ok {
		c.close(false)
		delete(p.conns, address)
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
		c.close(false)
		delete(p.conns, a)
	}
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

func (p *Pool) HandleMessages() {
	var (
		rc   receivedMessage
		term error
	)
	for len(p.receive) > 0 {
		select {
		case rc = <-p.receive:
		case <-p.quit:
			return
		}
		if term = rc.msg.Handle(rc, p.user); term != nil {
			p.Printf("closing connection %s by Handle error: %v",
				rc.Addr(), term)
			rc.Close()
		}
	}
}

func NewPool(c Config, user interface{}) (p *Pool) {
	c.applyDefaults()
	p = new(Pool)
	p.Logger = log.NewLogger("["+c.Name+"] ", c.Debug)
	p.conf = c
	p.reg = make(map[reflect.Type]Prefix)
	p.rev = make(map[Prefix]reflect.Type)
	p.conns = make(map[string]*Conn, c.MaxConnections)
	p.receive = make(chan receivedMessage, c.ReadQueueSize)
	if c.MaxConnections != 0 {
		p.sem = make(chan struct{}, c.MaxConnections)
	}
	p.user = user
	p.quit = make(chan struct{})
	return
}
