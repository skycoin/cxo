package pool

import (
	"errors"
	"fmt"
	"sync"

	"github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/node/pool/registry"
	"github.com/skycoin/cxo/node/pool/transport"
)

type Config struct{}

// A Pool represents pool of listeners and connections
type Pool struct {
	conf Config

	log.Logger // logger of the Pool

	// registry
	reg *registry.Registry

	// registered transports
	transports map[transport.Schema]transport.Transport

	lmx       sync.RWMutex
	listeners map[string]transport.Listener
	cmx       sync.RWMutex
	conns     map[string]transport.Conn

	receive chan transport.MessageContext // inflow

	newc    chan transport.Conn
	closedc chan transport.Conn

	newl    chan transport.Listener
	closedl chan transport.Listener

	sem chan struct{} // connections limit (if set)

	await sync.WaitGroup
	quito sync.Once
	quit  chan struct{}
}

// AddTransport to the pool. The method panics if transport
// with the same schema already registered. The method is not
// thread safe. All transports should be added before using
// the pool. If Initialize method of given transport returns an
// error the AddTransport method panics with the error
func (p *Pool) AddTransport(t transport.Transport) {
	if _, ok := p.transports[t.Schema()]; ok {
		panic("transport already registered: " + string(t.Schema()))
	}
	t.Initialize(p.reg, p.receive)
	p.transports[t.Schema()] = t
}

// Receive all messages using the channel
func (p *Pool) Receive() <-chan transport.MessageContext {
	return p.receive
}

// NewConnections retusn channel to
// receive all new connections
func (p *Pool) NewConnections() <-chan transport.Conn {
	return p.newc
}

// ClosedConnections returns channel
// to receive all closed connections
// until the Pool closed
func (p *Pool) ClosedConnections() <-chan transport.Conn {
	return p.closedc
}

// NewConnections retusn channel to
// receive all new listeners
func (p *Pool) NewListeners() <-chan transport.Listener {
	return p.newl
}

// ClosedListeners returns channel
// to receive all closed listeners
// until the Pool closed
func (p *Pool) ClosedListeners() <-chan transport.Listener {
	return p.closedl
}

// Connections returns list of remote
// addresses of all connections
func (p *Pool) Connections() (cs []string) {
	p.cmx.RLock()
	defer p.cmx.RUnlock()
	if len(p.conns) == 0 {
		return
	}
	cs = make([]string, 0, len(p.conns))
	for address := range p.conns {
		cs = append(cs, address)
	}
	return
}

// Connection returns connection by remote address or nil
// if the connection doesn't exists
func (p *Pool) Connection(address string) (c transport.Conn) {
	p.cmx.RLock()
	defer p.cmx.RUnlock()
	c = p.conns[address]
	return
}

// Listeners returns list of listening
// addresses of all listeners
func (p *Pool) Listeners() (ls []string) {
	p.lmx.RLock()
	defer p.lmx.RUnlock()
	if len(p.listeners) == 0 {
		return
	}
	ls = make([]string, 0, len(p.listeners))
	for address := range p.listeners {
		ls = append(ls, address)
	}
	return
}

// Listener returns listener by listening address or nil
// if the listener doesn't exists
func (p *Pool) Listener(address string) (l transport.Listener) {
	p.lmx.RLock()
	defer p.lmx.RUnlock()
	l = p.listeners[address]
	return
}

func (p *Pool) enqueueNewConnection(c transport.Conn) {
	select {
	case p.newc <- c:
	case <-p.quit:
	}
}

func (p *Pool) enqueueClosedConn(c transport.Conn) {
	select {
	case <-p.quit:
	default:
		select {
		case p.closedc <- c:
		case <-p.quit:
		}
	}
}

func (p *Pool) enqueueNewListener(l transport.Listener) {
	select {
	case p.newl <- l:
	case <-p.quit:
	}
}

func (p *Pool) enqueueClosedListener(l transport.Listener) {
	select {
	case <-p.quit:
	default:
		select {
		case p.closedl <- l:
		case <-p.quit:
		}
	}
}

func (p *Pool) closeListeners() {
	p.lmx.RLock()
	defer p.lmx.RUnlock()
	for _, l := range p.listeners {
		p.lmx.RUnlock()
		{
			l.Close()
		}
		p.lmx.RLock()
	}
}

func (p *Pool) closeConnections() {
	p.cmx.RLock()
	defer p.cmx.RUnlock()
	for _, c := range p.conns {
		p.cmx.RUnlock()
		{
			c.Close()
		}
		p.cmx.RLock()
	}
}

// Close the pool, all related listeners,
// connections, and close receiving channel.
// When the Close called the ClosedConnections
// and ClosedListeners don't more send
// closed connections and listeners, and
// NewConnections can skip new connections
// created during closing the Pool (but
// who cares)
func (p *Pool) Close() {
	p.quito.Do(func() {
		close(p.quit)
		p.closeListeners()
		p.closeConnections()
		p.await.Wait()
		close(p.receive)
	})
	return
}

// pool related errors
var (
	ErrClosed           = errors.New("pool was closed")
	ErrConnectionsLimit = errors.New("connections limit reached")
)

func (p *Pool) isClosed() (yep bool) {
	select {
	case <-p.quit:
		yep = true
	default:
	}
	return
}

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

func (p *Pool) release(c transport.Conn) {
	if p.sem != nil {
		<-p.sem
	}
	if c != nil {
		p.enqueueClosedConn(c)
	}
}

// ------------------------- connection wrapper ----------------------------- //

type connection struct {
	transport.Conn
	releaseOnce sync.Once
	release     func()
}

func (l *connection) Close() (err error) {
	err = l.Conn.Close()
	l.releaseOnce.Do(l.release)
	return
}

// ------------------------------- accept ----------------------------------- //

func (p *Pool) accept(l transport.Listener) (c transport.Conn, err error) {
	if err = p.acquireBlock(); err != nil {
		return
	}
	if c, err = l.Accept(); err != nil {
		p.release(nil)
		return
	}
	c = &connection{
		Conn:    c,
		release: func() { p.release(c) },
	}
	return
}

// ------------------------------- listen ----------------------------------- //

// Listen on given address
func (p *Pool) Listen(address string) (err error) {
	// don't listen if the pool was closed
	if p.isClosed() {
		err = ErrClosed
		return
	}
	// take a look the schema
	schema, addr := transport.SplitSchemaAddress(address)
	if schema == "" {
		err = fmt.Errorf("missing schema in %q", address)
		return
	}
	// lock p.listeners
	p.lmx.Lock()
	defer p.lmx.Unlock()
	// take a look transports
	t, ok := p.transports[schema]
	if !ok {
		err = fmt.Errorf("unknown schema: %q", schema)
		return
	}
	// check map of listeners preliminary
	if addr != "" {
		if _, ok := p.listeners[address]; ok {
			err = fmt.Errorf("already listening on %s", address)
			return
		}
	}
	var l transport.Listener
	if l, err = t.Listen(address); err != nil {
		return
	}
	// strict check map of listeners (impossible check)
	address = l.Address()
	if _, ok := p.listeners[address]; ok {
		l.Close()
		err = fmt.Errorf("already listening on %s", address)
		return
	}
	// add and start accept loop
	p.listeners[address] = l
	p.enqueueNewListener(l)
	p.await.Add(1)
	go p.listen(l)
	return
}

func (p *Pool) listen(l transport.Listener) {
	var (
		c   transport.Conn
		err error
	)
	// closing
	defer p.await.Done()
	defer p.enqueueClosedListener(l)
	defer l.Close()
	defer p.Debugf("%s stop accept loop", l.Address())
	// accept loop
	p.Debugf("%s start accept loop", l.Address())
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
		go p.handleConnection(c) // ignore error
	}
}

// --------------------------------- dial ----------------------------------- //

// Dial to given address
func (p *Pool) Dial(address string) (c transport.Conn, err error) {
	var ok bool
	// don't connect if the pool was closed
	if p.isClosed() {
		err = ErrClosed
		return
	}
	// schema and address
	schema, addr := transport.SplitSchemaAddress(address)
	if schema == "" {
		err = fmt.Errorf("missing schema in %q", address)
		return
	}
	if addr == "" {
		err = fmt.Errorf("invalid address %q", address)
		return
	}
	// transport
	transport, ok := p.transports[schema]
	if !ok {
		err = fmt.Errorf("unknown schema %q", schema)
		return
	}
	// preliminary check
	p.cmx.RLock()
	if _, ok = p.conns[address]; ok {
		p.cmx.RUnlock()
		err = fmt.Errorf("connection to %q already exists", address)
		return
	}
	p.cmx.RUnlock()
	// check out limit of connections
	if !p.acquire() {
		err = ErrConnectionsLimit
		return
	}
	// dial
	if c, err = transport.Dial(address); err != nil {
		p.release(nil)
		return
	}
	c = &connection{
		Conn:    c,
		release: func() { p.release(c) },
	}
	// strict check of p.conns
	if err = p.handleConnection(c); err != nil {
		c = nil // clear
	}
	return
}

// -------------------------------- handle ---------------------------------- //

func (p *Pool) handleConnection(c transport.Conn) (err error) {
	p.Debug("got new connection: ", c.Address())
	p.cmx.Lock()
	defer p.cmx.Unlock()
	var (
		address = c.Address()
		ok      bool
	)
	// strict check
	if _, ok = p.conns[address]; ok {
		err = fmt.Errorf("connection to %s already exists", address)
		p.release(nil)               // close
		c.(*connection).Conn.Close() // skipping p.release(c)
		return
	}
	// add
	p.conns[address] = c      // add the connection to the pool
	p.enqueueNewConnection(c) // emit new connection
	return
}
