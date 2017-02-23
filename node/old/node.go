package node

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"

	"github.com/skycoin/cxo/db"
	"github.com/skycoin/cxo/enc"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrNotListening = errors.New("not listening")

	ErrManualDisconnect gnet.DisconnectReason = errors.New(
		"manual disconnect")
	ErrUnexpectedHandshake gnet.DisconnectReason = errors.New(
		"unexpected handshake messge")
	ErrMalformedHandshake gnet.DisconnectReason = errors.New(
		"malformed handshake messge")
	ErrHandshakeTimeout gnet.DisconnectReason = errors.New(
		"handshake timeout was exceeded")
	ErrAlreadyConnected gnet.DisconnectReason = errors.New(
		"node already connected")
	ErrMalformedMessage gnet.DisconnectReason = errors.New(
		"malformed message")
	ErrPubKeyMismatch gnet.DisconnectReason = errors.New(
		"publick key mismatch")
)

// SetDebugLog for underlying gnet logger
func SetDebugLog(debug bool) {
	gnet.DebugPrint = debug
}

// A ListFunc is used for (Feed).List and (Inflow).List
// functions. Return true to stop itteration
type ListFunc func(cipher.PubKey, net.Addr) (stop bool)

// ReplyFunc is part of ReceiveCallback
type ReplyFunc func([]byte)

// ReceiveCallback called when new message was received.
// Call ReplyFunc to reply for received message. If the
// callback returns non-nil error then connectio will be
// terminated
type ReceiveCallback func([]byte, Node, ReplyFunc) error

// A Node represents ...
type Node interface {
	Logger

	// Feed returns subscribers
	Feed() Feed
	// Inflow returns subscriptions
	Inflow() Inflow

	// DB returns DB of the Node
	DB() db.DB
	// Encoder return encoder of the Node
	Encoder() enc.Encoder

	// PubKey returns public key of Node
	PubKey() cipher.PubKey
	// Sign is used to sign given hash using secret key of Node
	Sign(cipher.SHA256) cipher.Sig

	// Start the Node
	Start() error
	// Shutdown the Node
	Close()

	//
	// internals
	//

	hsQuestion(*hsQuestion, *gnet.MessageContext) error
	hsQuestionAnswer(*hsQuestionAnswer, *gnet.MessageContext) error
	hsAnswer(*hsAnswer, *gnet.MessageContext) error
	hsSuccess(*hsSuccess, *gnet.MessageContext) error
}

type node struct {
	sync.RWMutex

	Logger

	conf Config

	sec cipher.SecKey
	pub cipher.PubKey

	pool   *gnet.ConnectionPool
	poolmx sync.RWMutex

	back   map[*gnet.Connection]*conn
	backmx sync.RWMutex

	quitq   chan struct{}
	handleq chan struct{}
}

// NewNode creates new node using provided configs. It panics
// if c is nil, because Node requires secret key. Config must
// contain valid secret key. NewNode makes copy of given Config.
// If conf.DB is nil then db.NewDB is used. If conf.Encoder is nil
// then enc.NewEncoder is used. If conf.ReceiveCallback is nil
// NewNode panics
func NewNode(conf *Config) Node {
	if conf == nil {
		panic("NewNode config is nil")
	}
	var (
		n   *node = new(node)
		err error

		gc gnet.Config
	)
	if n.sec, err = cipher.SecKeyFromHex(conf.SecretKey); err != nil {
		panic(err)
	}
	n.pub = cipher.PubKeyFromSecKey(n.sec)

	if conf.ReceiveCallback == nil {
		panic("NewNode config doesn't contain ReceiveCallback")
	}

	n.conf = *conf // make copy

	if n.conf.DB == nil {
		n.conf.DB = db.NewDB()
	}
	if n.conf.Encoder == nil {
		n.conf.Encoder = enc.NewEncoder()
	}
	n.Logger = NewLogger("["+n.conf.Name+"] ", n.conf.Debug)

	n.Debug("create node")
	//
	gc = n.conf.gnetConfig()
	gc.ConnectCallback = func(gc *gnet.Connection, dial bool) {
		if dial {
			return
		}
		n.Debug("new subscriber: ", gc.Addr())
		var (
			c *conn = &conn{
				done: make(chan struct{}),
				feed: true,
			}
		)
		n.backmx.Lock()
		defer n.backmx.Unlock()
		n.back[gc] = c
		if n.conf.HandshakeTimeout > 0 {
			go n.countDownHandshakeTimeout(gc, c.done)
		}
	}
	gc.DisconnectCallback = func(gc *gnet.Connection,
		err gnet.DisconnectReason) {

		n.Printf("disconnect %v, reason %v", gc.Addr(), err)

		n.backmx.Lock()
		defer n.backmx.Unlock()
		delete(n.back, gc)
	}
	//
	n.pool = gnet.NewConnectionPool(gc, n)
	//
	n.back = make(map[*gnet.Connection]*conn)

	n.Debug(n.conf.HumanString())

	return n
}

func (n *node) DB() db.DB             { return n.conf.DB }
func (n *node) Encoder() enc.Encoder  { return n.conf.Encoder }
func (n *node) PubKey() cipher.PubKey { return n.pub }

func (n *node) Sign(hash cipher.SHA256) cipher.Sig {
	return cipher.SignHash(hash, n.sec)
}

func (n *node) Inflow() Inflow { return inflow{n} }
func (n *node) Feed() Feed     { return feed{n} }

func (n *node) Start() (err error) {
	n.Lock()
	defer n.Unlock()

	n.poolmx.Lock()
	defer n.poolmx.Unlock()

	if err = n.pool.StartListen(); err != nil {
		return
	}

	n.quitq = make(chan struct{})
	n.handleq = make(chan struct{})

	go n.pool.AcceptConnections()
	go n.handleEvents()

	return
}

func (n *node) Close() {
	n.Lock()
	defer n.Unlock()
	if n.quitq == nil {
		return
	}
	n.poolmx.Lock()
	{
		n.pool.StopListen()
	}
	n.poolmx.Unlock()
	close(n.quitq)
	<-n.handleq
	n.quitq = nil
	n.handleq = nil
}

func (n *node) handleEvents() {
	n.Debug("start handling events")
	var (
		sr gnet.SendResult
		de gnet.DisconnectEvent
	)
	for {
		select {
		case <-n.quitq:
			n.Debug("quit handling events")
			close(n.handleq)
			return
		case sr = <-n.pool.SendResults:
			if sr.Error != nil {
				n.Print("error sending to %s: %v",
					sr.Connection.Addr(),
					sr.Error)
			}
		case de = <-n.pool.DisconnectQueue:
			n.Print("disconnect event: ", de)
			n.poolmx.Lock()
			{
				n.pool.HandleDisconnectEvent(de)
			}
			n.poolmx.Unlock()
		default:
			n.poolmx.Lock()
			{
				n.pool.HandleMessages()
			}
			n.poolmx.Unlock()
		}
	}
}

func (n *node) countDownHandshakeTimeout(gc *gnet.Connection,
	done <-chan struct{}) {

	n.Debug("counting down handshake timeout for ", gc.Addr())

	var tick *time.Ticker = time.NewTicker(n.conf.HandshakeTimeout)
	defer tick.Stop()

	select {
	case <-done:
		n.Debug("handshake was done: ", gc.Addr())
		return
	case <-n.quitq:
		n.Debug("handshake timeout quiting: ", gc.Addr())
		return
	case <-tick.C:
		n.Debug("handshake timeout was exceeded: ", gc.Addr())
		n.pool.Disconnect(gc, ErrHandshakeTimeout)
	}
}
