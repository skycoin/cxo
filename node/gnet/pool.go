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

// pool related errors
var (
	// ErrClosed occurs when the Pool is closed
	ErrClosed = errors.New("the pool was closed")
	// ErrConnectionsLimit occurs when ... (TODO)
	ErrConnectionsLimit = errors.New("connections limit reached")
)

type Pool struct {
	sync.RWMutex

	log.Logger

	conf Config // configurations

	// registery
	reg map[reflect.Type]Prefix // registery
	rev map[Prefix]reflect.Type // inverse registery

	// pool
	l       net.Listener         // listener
	conns   map[string]*conn     // remote address -> connection
	receive chan receivedMessage // receive messages
	sem     chan struct{}        // max connections

	newc chan *conn // new connections

	// any interface{} provided to be used
	// in Handle method of Message
	user interface{}

	quit chan struct{} // quit
}

func (p *Pool) encodeMessage(m Message) (data []byte) {
	var (
		val    reflect.Value = reflect.Indirect(reflect.ValueOf(m))
		prefix string
		ok     bool

		em []byte
	)
	if prefix, ok = p.reg[val.Type()]; !ok {
		p.Panicf("send unregistered message: %T", m)
	}
	em = encoder.Serialize(m)
	data = make([]byte, 0, 4+PrefixLength+len(em))
	data = append(data, prefix...)
	data = append(data, encoder.SerializeAtomic(uint32(len(em)))...)
	data = append(data, em...)
	return
}

// acquire returns true if number of connections
// later then p.conf.MaxConnections
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
		c          *conn

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
		if err = conn.sendEncodedMessage(em); err != nil {
			// TODO: handle error
		}
	}
}

// Broadcast given message to all connections of the Pool
func (p *Pool) Broadcast(m Message) {
	p.BroadcastExcept(m)
}

// Listen start listening on given address
func (p *Pool) Listen(address string) (err error) {
	if p.l, err = net.Listen("tcp", address); err != nil {
		return
	}
	go p.listen(p.l)
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

		x *conn
	)
	if _, ok = p.conns[address]; ok {
		err = ErrConnAlreadyExists
		c.Close()
		return
	}
	x = newConn(c, p)
	p.conns[address] = x // add the connection to the pool
	x.handle()           // asyn non-blocking method
	select {
	case p.newc <- x:
	case <-p.quit:
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
	err = p.handleConnection(c)
	return
}

func (p *Pool) Register(prefix string, i interface{}) {
	if len(prefix) != PrefixLength {
		p.Panicf("wrong prefix length: ", len(prefix))
	}
}
