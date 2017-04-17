package pool

import (
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/node/log"
)

// prefix related constants
const (
	// PrefixLength is length of a Prefix
	PrefixLength int = 4
	// PrefixSpecial is reserved character, not allowed for
	// using in Prefixes. Beside the rune any space character or
	// an unprintabel characters are not allowed too
	PrefixSpecial rune = '-'
)

// Prefix represents message prefix, that
// is unique identifier of type of a message
type Prefix [PrefixLength]byte

// prefix related errors
var (
	ErrSpecialCharacterInPrefix     = errors.New("special character in prefix")
	ErrSpaceInPrefix                = errors.New("space in prefix")
	ErrUnprintableCharacterInPrefix = errors.New(
		"unprintable character in preix")
)

// Validate returns error if the Prefix is invalid
func (p Prefix) Validate() error {
	var r rune
	for _, r = range p.String() {
		if r == PrefixSpecial {
			return ErrSpecialCharacterInPrefix
		}
		if r == ' ' { // ASCII space
			return ErrSpaceInPrefix
		}
		if !unicode.IsPrint(r) {
			return ErrUnprintableCharacterInPrefix
		}
	}
	return
}

// String implements fmt.Stringer interface
func (p Prefix) String() string {
	return string(p)
}

// PrefixFilterFunc is an allow-filter
type PrefixFilterFunc func(Prefix) bool

// ErrFiltered represents errors that occurs when some
// prefix rejected by a `PrefixFilterFunc`s
type ErrFiltered struct {
	send   bool
	prefix Prefix
}

// Sending returns true if the error occurs during sending a message.
// Otherwise this error emitted by receive-filters
func (e *ErrFiltered) Sending() bool {
	return e.send
}

// Error implements error interface
func (e *ErrFiltered) Error() string {
	if e.send {
		return fmt.Sprintf("[%s] rejected by send-filter", e.prefix.String())
	}
	return fmt.Sprintf("[%s] rejected by receive-filter", e.prefix.String())
}

// set of prefix filters
type filter []PrefixFilterFunc

func (f filter) allow(p Prefix) (allow bool) {
	if len(f) == 0 {
		return true // allow
	}
	for _, filter := range f {
		if filter(p) {
			return true // allow
		}
	}
	return // deny
}

// A Message repreents any type that can be registered, encoded and
// decoded
type Message interface{}

// A Registry represents registry of types
// to send/receive. The Registry is not thread safe.
// I.e. you need to call all Register/AddSendFilter/AddReceiveFilter
// before using the Registry, before share reference to it to other
// goroutines
type Registry struct {
	reg map[Prefix]reflect.Type
	inv map[reflect.Type]Prefix

	sendf filter
	recvf filter
}

func (*Registry) typeOf(i Message) reflect.Type {
	return reflect.Indirect(reflect.ValueOf(i)).Type()
}

// Register type of given value with given prefix. The method
// panics if prefix is malformed or already registered, and
// if the type already registered. The method also panics if
// the type can't be encoded
func (r *Registry) Register(prefix Prefix, i Message) {
	if err := prefix.Validate(); err != nil {
		panic(err)
	}
	if _, ok := r.reg[prefix]; ok {
		panic("register prefix twice")
	}
	typ := r.typeOf(i)
	if _, ok := r.inv[typ]; ok {
		panic("register type twice")
	}
	encoder.Serialize(i) // can panic if the type is invalid for the encoder
	r.reg[prefix] = typ
	r.reg[typ] = prefix
}

// AddSendFilter to filter sending. By default all registered types allowed
// to sending. The filter should returns true for prefixes allowed to send
func (r *Registry) AddSendFilter(filter PrefixFilterFunc) {
	if filter == nil {
		panic("nil filter")
	}
	r.sendf = append(r.sendf, filter)
}

// AddReceiveFilter to filter receiving. By default all registered types allowed
// to receive. The filter should returns true for prefixes allowed to receive
func (r *Registry) AddReceiveFilter(filter PrefixFilterFunc) {
	if filter == nil {
		panic("nil filter")
	}
	r.recvf = append(r.recvf, filter)
}

// AllowSend returns true if all send-filters allows messages with
// given prefix to send
func (r *Registry) AllowSend(p Prefix) (allow bool) {
	allow = r.sendf.allow(p)
	return
}

// AllowReceive returns true if all send-filters allows messages with
// given prefix to be received. The method can be used to break connection
// with unwanted messages skipping reading the whole message
func (r *Registry) AllowReceive(p Prefix) (allow bool) {
	allow = r.recvf.allow(p)
	return
}

// Type returns registered reflect.Type by given prefix and true or
// nil and false if the Prefix was not registered
func (r *Registry) Type(p Prefix) (typ reflect.Type, ok bool) {
	typ, ok = r.reg[p]
	return
}

// Encode given message. The method panics if type of given value
// is not registered. It can return filtering error (see ErrFiltered)
func (r *Registry) Enocde(m Message) (p []byte, err error) {
	typ := r.typeOf(m)
	prefix, ok := r.reg[typ]
	if !ok {
		panic("encoding unregistered type: " + typ.String())
	}
	if !r.sendf.allow(prefix) {
		err = &ErrFiltered{true, prefix} // sending
		return
	}
	p = encoder.Serialize(m)
	return
}

// Registry related errors
var (
	ErrNotRegistered      = errors.New("not registered prefix")
	ErrIncompliteDecoding = errors.New("incomlite decoding")
)

// Decode body of a message type of which described by given prefix. The
// method can returns decoding or filtering error (see ErrFiltered). The
// method can also returns ErrNotRegistered
func (r *Registry) Decode(prefix Prefix, p []byte) (v Message, err error) {
	typ, ok := r.Type(prefix)
	if !ok {
		err = ErrNotRegistered
		return
	}
	if !r.recvf.allow(prefix) {
		err = &ErrFiltered{false, prefix} // receiving
		return
	}
	val := reflect.New(typ)
	if err = r.DecodeValue(p, val); err != nil {
		return
	}
	v = val.Interface()
	return
}

// DecodeValue skips all receive-filters and uses reflect.Value instead of
// Prefix. The method is short-hand for encoder.DeserializeRawToValue
func (*Registry) DecodeValue(p []byte, val reflect.Value) (err error) {
	var n int
	n, err = encoder.DeserializeRawToValue(p, val)
	if err == nil && n < len(p) {
		err = ErrIncompliteDecoding
	}
	return
}

// A Schema is connection schema such as "tcp", "tls+tcp", etc
type Schema string

// MessageContext represents received message with
// conection from which the message received
type MessageContext struct {
	Message Message // received message
	Conn    Conn    // connection from wich the message received
}

// A Transport represents transporting interface
type Transport interface {
	// Schema of the Transport
	Schema() Schema

	// Initialize the transport, providing
	// registry with filters and shared receiver
	// channel for incoming messages
	Initialize(reg *Registry, receiver chan<- MessageContext)

	// Dial to given address
	Dial(address string) (c Conn, err error)
	// Listen on given address
	Listen(address string) (l Listener, err error)
}

// A Listener represents listener interface.
// All methods of the listener are thread safe
type Listener interface {
	// Accept connection
	Accept() (c Conn, err error)
	// Address the Listener listening on
	// with schema
	Address() string
	// Close the Listener
	Close() (err error)
}

// A Conn represents connection interface.
// All methods of the connection are thread-safe
type Conn interface {
	// Address returns address of remote node
	// with schema. For example tcp://127.0.0.1:9987
	Address() string
	// Send given message to remote node. The method
	// returns immediately
	Send(m Message) (err error)
	// SendRaw sends encoded message with
	// length and prefix to remote node. The
	// method returns immediately
	SendRaw(p []byte) (err error)
	// Close the connection
	Close() (err error)

	// IsIncoming returns true if the connection
	// accepted by a listener. If the connection
	// created using Dial method then the
	// IsIncoming method returns false
	IsIncoming() bool

	//
	// Attach any user-provided value to the connection
	//

	Value() (value interface{}) // get the value
	SetValue(value interface{}) // set the value
}

// A Pool represents pool of listeners and connections
type Pool struct {
	conf Config

	log.Logger // logger of the Pool

	// registry
	reg *Registry

	// registered transports
	transports map[Schema]Transport

	lmx       sync.RWMutex
	listeners map[string]Listener
	cmx       sync.RWMutex
	conns     map[string]Conn

	receive chan MessageContext // inflow

	newc    chan Conn
	closedc chan Conn

	newl    chan Listener
	closedl chan Listener

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
func (p *Pool) AddTransport(t Transport) {
	if _, ok := p.transports[t.Schema()]; ok {
		panic("transport already registered: " + string(t.Schema()))
	}
	t.Initialize(p.reg, p.receive)
	p.transports[t.Schema()] = t
}

// Receive all messages using the channel
func (p *Pool) Receive() <-chan MessageContext {
	return p.receive
}

// NewConnections retusn channel to
// receive all new connections
func (p *Pool) NewConnections() <-chan Conn {
	return p.newc
}

// ClosedConnections returns channel
// to receive all closed connections
// until the Pool closed
func (p *Pool) ClosedConnections() <-chan Conn {
	return p.closedc
}

// NewConnections retusn channel to
// receive all new listeners
func (p *Pool) NewListeners() <-chan Listener {
	return p.newl
}

// ClosedListeners returns channel
// to receive all closed listeners
// until the Pool closed
func (p *Pool) ClosedListeners() <-chan Listener {
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
func (p *Pool) Connection(address string) (c Conn) {
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
func (p *Pool) Listener(address string) (l Listener) {
	p.lmx.RLock()
	defer p.lmx.RUnlock()
	c = p.listeners[address]
	return
}

func (p *Pool) enqueueNewConnection(c Conn) {
	select {
	case p.newc <- c:
	case <-p.quit:
	}
}

func (p *Pool) enqueueClosedConn(c Conn) {
	select {
	case <-p.quit:
	default:
		select {
		case p.closedc <- c:
		case <-p.quit:
		}
	}
}

func (p *Pool) enqueueNewListener(l Listener) {
	select {
	case p.newl <- l:
	case <-p.quit:
	}
}

func (p *Pool) enqueueClosedListener(l Listener) {
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

func (p *Pool) release(c Conn) {
	if p.sem != nil {
		<-p.sem
	}
	if c != nil {
		p.enqueueClosedConn(c)
	}
}

// ------------------------- connection wrapper ----------------------------- //

type connection struct {
	Conn
	releaseOnce sync.Once
	release     func()
}

func (l *connection) Close() (err error) {
	err = l.Conn.Close()
	l.releaseOnce.Do(l.release)
	return
}

// ------------------------------- accept ----------------------------------- //

func (p *Pool) accept(l net.Listener) (c Conn, err error) {
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

// SplitSchemaAddress split given string by "://"
func SplitSchemaAddress(address string) (schema Schema, addr string) {
	if ss := strings.Split(address, "://"); len(ss) == 2 {
		schema, addr = Schema(ss[0]), ss[1]
	} else if len(ss) == 1 {
		schema = Schema(ss[0])
	}
	return
}

// Listen on given address
func (p *Pool) Listen(address string) (err error) {
	// don't listen if the pool was closed
	if p.isClosed() {
		err = ErrClosed
		return
	}
	// take a look the schema
	schema, addr := SplitSchemaAddress(address)
	if schema == "" {
		err = fmt.Errorf("missing schema in %q", address)
		return
	}
	// lock p.listeners
	p.lmx.Lock()
	defer p.lmx.Unlock()
	// take a look transports
	transport, ok := p.transports[schema]
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
	var l Listener
	if l, err = transport.Listen(address); err != nil {
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
	go p.listen(p.l)
	return
}

func (p *Pool) listen(l Listener) {
	var (
		c   Conn
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
func (p *Pool) Dial(address string) (c Conn, err error) {
	var ok bool
	// don't connect if the pool was closed
	if p.isClosed() {
		err = ErrClosed
		return
	}
	// schema and address
	schema, addr := SplitSchemaAddress(address)
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
	c = &limitedConnection{
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

func (p *Pool) handleConnection(c Conn) (err error) {
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
		p.release(nil) // close
		c.Conn.Close() // skipping p.release(c)
		return
	}
	// add
	p.conns[address] = c      // add the connection to the pool
	p.enqueueNewConnection(c) // emit new connection
	return
}
