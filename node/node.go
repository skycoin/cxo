package node

import (
	"errors"
	"reflect"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"

	"github.com/logrusorgru/cxo/typereg"
)

var (
	//
	// disconnect reasons
	//
	ErrPendingLimit gnet.DisconnectReason = errors.New(
		"penging connections limit")
	ErrIncomingLimit gnet.DisconnectReason = errors.New(
		"incoming connections limit")
	ErrOutgoingLimit gnet.DisconnectReason = errors.New(
		"outgoing connections limit")

	ErrHandshakeTimeout gnet.DisconnectReason = errors.New(
		"handshake timeout")
	ErrUnexpectedHandshake gnet.DisconnectReason = errors.New(
		"unexpected handshake")
	ErrMalformedHandshake gnet.DisconnectReason = errors.New(
		"malformed handshake")
	ErrPubKeyMismatch gnet.DisconnectReason = errors.New(
		"public key mismatch")
	ErrAlreadyConnected gnet.DisconnectReason = errors.New(
		"already connected")

	// ErrUnexpectedMessage mans that message received on
	// but handshake of connections is not performed yet
	ErrUnexpectedMessage gnet.DisconnectReason = errors.New(
		"unexpected message")

	ErrInternalError gnet.DisconnectReason = errors.New(
		"internal error")
	ErrManualDisconnect gnet.DisconnectReason = errors.New(
		"manual disconnect")

	ErrIncomingMessageInterface gnet.DisconnectReason = errors.New(
		"got mesage that doesn't implements IncomingHandler")
	ErrOutgoingMessageInterface gnet.DisconnectReason = errors.New(
		"got mesage that doesn't implements Outgoing")
)

type Node interface {
	Logger

	Incoming() Incoming // manage incoming connections
	Outgoing() Outgoing // manage outgoing connections

	PubKey() cipher.PubKey
	Sign(hash cipher.SHA256) cipher.Sig

	// Start launches node and block current goroutine.
	// It returns when node is closed or if some error occured
	Start() error
	// Close stops node
	Close()

	// Register registers type of mesages that conections can handle.
	// The method panics if given msg is invalid or given twice.
	// It's not safe to use the methos async (its designed to
	// register all types before start). Also, there's no way
	// to clean up registry
	Register(msg interface{})

	//
	// internals
	//

	pend(*gnet.Connection) (*pendingConnection, bool)

	// remote public key, is incoming and error
	handleMessage(*gnet.Connection) (pub cipher.PubKey,
		incoming bool,
		err error)

	addOutgoing(gc *gnet.Connection, pub cipher.PubKey) error
	addIncoming(gc *gnet.Connection, pub cipher.PubKey) error

	encode(msg interface{}) (m *Msg, err error)
	decode(body []byte) (msg interface{}, err error)
}

type connectEvent struct {
	gc       *gnet.Connection
	outgoing bool
}

type pendingConnection struct {
	remotePub cipher.PubKey
	quest     uuid.UUID
	outgoing  bool
	start     time.Time
	step      int
}

type node struct {
	Logger

	conf Config // local copy

	registry typereg.Typereg

	incomingTypes map[typereg.Type]struct{}
	outgoingTypes map[typereg.Type]struct{}

	//
	pub cipher.PubKey
	sec cipher.SecKey

	pool *gnet.ConnectionPool

	pending            map[*gnet.Connection]*pendingConnection
	onConnectEventChan chan connectEvent

	incoming map[*gnet.Connection]cipher.PubKey // incoming connections (feed)
	outgoing map[*gnet.Connection]cipher.PubKey // outgoing connections

	// manage
	lockChan   chan struct{}
	unlockChan chan struct{}

	quit chan struct{}
	done chan struct{}
}

func (n *node) onGnetConnect(gc *gnet.Connection, outgoing bool) {
	n.onConnectEventChan <- connectEvent{gc, outgoing}
}

func (n *node) onGnetDisconnect(gc *gnet.Connection,
	reason gnet.DisconnectReason) {

	var a string = gc.Addr()

	n.Print("[INF] disconnect %s, reason %v", a, reason)

	delete(n.incoming, gc)
	delete(n.outgoing, gc)

	n.Lock()
	defer n.Unlock()
	delete(n.pending, gc)
}

func (n *node) onConnectEvent(ce connectEvent) {
	var (
		gc *gnet.Connection = ce.gc
		a  string           = gc.Addr()
	)
	if ce.outgoing {
		n.Debug("[DBG] connected to ", a)
	} else {
		n.Debug("[DBG] got incoming connection from:", a)
	}
	// lock n.pending
	n.Lock()
	defer n.Unlock()

	if len(n.pending) == n.conf.MaxPendingConnections {
		n.Debug("[DBG] pending limit disconnecting: ", a)
		gc.ConnectionPool.Disconnect(gc, ErrPendingLimit)
		return
	}
	if ce.outgoing {
		if len(n.outgoing) == n.conf.MaxOutgoingConnections {
			n.Debug("[DBG] outgoing limit disconnecting: ", a)
			gc.ConnectionPool.Disconnect(gc, ErrOutgoingLimit)
			return
		}
		// n.pending filled down using Connect for outgoing connections
		// send handshake question
		pend, ok := n.pending[gc]
		if !ok {
			n.Print("[BUG] missing pending connection")
			gc.ConnectionPool.Disconnect(gc, ErrInternalError)
			return
		}
		pend.quest = uuid.NewV4()
		pend.step++
		gc.ConnectionPool.SendMessage(gc, &hsQuest{
			Quest: pend.quest,
			Pub:   n.PubKey(),
		})
	} else {
		if len(n.incoming) == n.conf.MaxIncomingConnections {
			n.Debug("[DBG] incoming limit disconnecting: ", ce.gc.Addr())
			ce.gc.ConnectionPool.Disconnect(ce.gc, ErrIncomingLimit)
			return
		}
		n.pending[ce.gc] = &pendingConnection{
			outgoing: false,
			start:    time.Now(),
		}
	}
}

func (n *node) lookupHandshakeTime() {
	n.Lock()
	defer n.Unlock()
	var now time.Time = time.Now()
	for gc, pc := range n.pending {
		if now.Sub(pc.start) >= n.conf.HandshakeTimeout {
			n.Debug("[DBG] handshake timeout was exceeded: ", gc.Addr())
			gc.ConnectionPool.Disconnect(gc, ErrHandshakeTimeout)
		}
	}
}

// NewNode creates new Node using given Configs
// and secret key
func NewNode(sec cipher.SecKey, conf Config) (n Node, err error) {
	var (
		nd *node
		gc gnet.Config
	)

	if err = conf.Validate(); err != nil {
		return
	}

	if sec == (cipher.SecKey{}) {
		err = errors.New("empty secret key")
		return
	}

	nd = new(node)

	nd.Logger = NewLogger("["+conf.Name+"] ", conf.Debug)
	nd.conf = conf

	nd.pub = cipher.PubKeyFromSecKey(sec)
	nd.sec = sec

	nd.registry = typereg.NewTypereg()

	nd.incomingTypes = make(map[typereg.Type]struct{})
	nd.outgoingTypes = make(map[typereg.Type]struct{})

	gc = conf.gnetConfig()

	gc.ConnectCallback = nd.onGnetConnect
	gc.DisconnectCallback = nd.onGnetDisconnect

	nd.pool = gnet.NewConnectionPool(conf.gnetConfig(), nd)

	nd.pending = make(map[*gnet.Connection]*pendingConnection,
		conf.MaxPendingConnections)
	nd.onConnectEventChan = make(chan connectEvent, conf.MaxPendingConnections)

	nd.incoming = make(map[*gnet.Connection]cipher.PubKey,
		conf.MaxIncomingConnections)
	nd.outgoing = make(map[*gnet.Connection]cipher.PubKey,
		conf.MaxOutgoingConnections)

	nd.lockChan = make(chan struct{})
	nd.unlockChan = make(chan struct{})

	n = nd
	return
}

func (n *node) PubKey() cipher.PubKey {
	return n.pub
}

func (n *node) Sign(hash cipher.SHA256) cipher.Sig {
	return cipher.SignHash(hash, n.sec)
}

func (n *node) pend(gc *gnet.Connection) (p *pendingConnection, ok bool) {
	n.Lock()
	defer n.Unlock()
	p, ok = n.pending[gc]
	return
}

func (n *node) hasOutgoing(pub cipher.PubKey) (ok bool) {
	for _, pk := range n.outgoing {
		if pk == pub {
			return true
		}
	}
	return
}

func (n *node) hasIncoming(pub cipher.PubKey) (ok bool) {
	for _, pk := range n.incoming {
		if pk == pub {
			return true
		}
	}
	return
}

func (n *node) addOutgoing(gc *gnet.Connection, pub cipher.PubKey) error {
	n.Lock()
	defer n.Unlock()
	if n.hasOutgoing(pub) {
		return ErrAlreadyConnected
	}
	n.outgoing[gc] = pub
	delete(n.pending, gc)
	n.Print("[INF] new established outgoing connection %s, %s",
		gc.Addr(),
		pub.Hex())
	return nil
}

func (n *node) addIncoming(gc *gnet.Connection, pub cipher.PubKey) error {
	n.Lock()
	defer n.Unlock()
	if n.hasIncoming(pub) {
		return ErrAlreadyConnected
	}
	n.incoming[gc] = pub
	delete(n.pending, gc)
	n.Print("[INF] new established incoming connection %s, %s",
		gc.Addr(),
		pub.Hex())
	return nil
}

// start blocks
func (n *node) Start() (err error) {
	n.Print("[INF] start node")
	n.Debug("[DBG] ", n.conf.HumanString())

	n.quit = make(chan struct{})
	n.done = make(chan struct{})

	var (
		handleMsgTicker        *time.Ticker
		handshakeTimeoutTicker *time.Ticker

		handshakeTimeoutChan <-chan time.Time

		de gnet.DisconnectEvent
		sr gnet.SendResult

		ce connectEvent

		// manage

		// suppress race detecton panic
		quit chan struct{} = n.quit
		done chan struct{} = n.done
	)

	handleMsgTicker = time.NewTicker(n.conf.MessageHandlingRate)
	defer handleMsgTicker.Stop()

	if n.conf.HandshakeTimeout > 0 {
		handshakeTimeoutTicker = time.NewTicker(
			n.conf.HandshakeTimeout)
		defer handshakeTimeoutTicker.Stop()
		handshakeTimeoutChan = handshakeTimeoutTicker.C
	}

	if n.conf.MaxIncomingConnections > 0 {
		n.Debugf("[DBG] start listener on: %s:%d", n.conf.Address, n.conf.Port)
		if err = n.pool.StartListen(); err != nil {
			n.Debug("[DBG] error starting listener: ", err)
			return
		}
		go n.pool.AcceptConnections()
	}

	defer close(done)
	defer n.pool.StopListen()

	n.Debug("[DBG] start event loop")
	for {
		select {
		case de = <-n.pool.DisconnectQueue:
			n.Debug("[DBG] disconnect event")
			n.pool.HandleDisconnectEvent(de)
		case sr = <-n.pool.SendResults:
			n.Debug("[DBG] send result event")
			if sr.Error != nil {
				n.Printf("[ERR] error sending message %s to %s: %v",
					reflect.TypeOf(sr.Message).Name(),
					sr.Connection.Addr(),
					sr.Error)
			}
		case <-handleMsgTicker.C:
			n.pool.HandleMessages()
		case <-handshakeTimeoutChan:
			n.Debug("[DBG] lookup handshake tick")
			n.lookupHandshakeTime()
		case ce = <-n.onConnectEventChan:
			n.Debug("[DBG] connect event")
			n.onConnectEvent(ce)
		case <-n.lockChan:
			n.Debug("[DBG] lock chan")
			<-n.unlockChan
			n.Debug("[DBG] unlock chan")
		case <-quit:
			n.Debug("[DBG] quiting start loop")
			return
		}
	}

}

func (n *node) Close() {
	n.Debug("[DBG] closing node")
	close(n.quit)
	<-n.done
	n.Debug("[DBG] node was closed")
}

func (n *node) lock() {
	n.lockChan <- struct{}{}
}

func (n *node) unlock() {
	n.unlockChan <- struct{}{}
}

func (n *node) Incoming() Incoming {
	return incoming{n}
}

func (n *node) Outgoing() Outgoing {
	return outgoing{n}
}

func (n *node) handleMessage(gc *gnet.Connection) (pub cipher.PubKey,
	incoming bool,
	err error) {

	if pub, incoming = n.incoming[gc]; incoming {
		return
	}
	if pub, incoming = n.outgoing[gc]; incoming {
		incoming = false // outgoing
		return
	}
	err = ErrUnexpectedMessage
	return
}

func (n *node) encode(msg interface{}) (m *Msg, err error) {
	var body []byte
	body, err = n.registry.Encode(msg)
	if err != nil {
		return
	}
	m = &Msg{
		Body: body,
	}
	return
}

func (n *node) decode(body []byte) (msg interface{}, err error) {
	return n.registry.Decode(body)
}

func (n *node) Register(msg interface{}) {
	n.registry.Register(msg)
}
