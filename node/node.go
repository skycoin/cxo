// Package node implemnts core of cxo daemon including
// connection pool, subscriptions, etc
package node

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/satori/go.uuid"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"

	"github.com/skycoin/cxo/db"
	"github.com/skycoin/cxo/enc"
)

// SetDebugLog for underlying gnet logger
func SetDebugLog(debug bool) {
	gnet.DebugPrint = debug
}

// A ListFunc represents function that called by (Node).ListSubscriptions
// and (Node).ListSubscribers for every connection it has. Return true to
// stop itteration
type ListFunc func(pub cipher.PubKey, remote net.Addr) (stop bool)

// A Node represents ...
type Node interface {
	Logger

	// Start starts node
	Start() error
	// Close shuts node down
	Close()

	// PubKey returns public key of Node
	PubKey() cipher.PubKey
	// Sign is used to sign given hash using secret key of Node
	Sign(cipher.SHA256) cipher.Sig

	// Broadcast sends announce to subscribers
	Broadcast(cipher.SHA256)

	// Database access (Has/Get/Set/Stat)
	DB() db.DB
	// Encoder access (Register/Encode/Decode)
	Encoder() enc.Encoder

	//
	// Information and control
	//

	// Address retuns listening address (address of feed)
	Address() net.Addr

	// Subscribe subscribes to remote feed with given address
	// and, optional, public key. Pass cipher.PubKey{} to
	// subscribe to feed with any public key
	Subscribe(address string, pub cipher.PubKey) error
	// Unsubscribe unsubscribes from feed with given public key
	Unsubscribe(pub cipher.PubKey) error
	// ListSubscriptions is used to see all feeds this node subscribed to
	ListSubscriptions(ListFunc)

	// ListSubscribers is used to see all nodes subscribed to this node
	ListSubscribers(ListFunc)

	//
	// internals
	//

	// connection side

	isFeed(*gnet.MessageContext) bool
	isInflow(*gnet.MessageContext) bool

	// handshake processing

	// handled by feed
	hsQuestion(*hsQuestion, *gnet.MessageContext) error
	// handled by inflow
	hsQuestionAnswer(*hsQuestionAnswer, *gnet.MessageContext) error
	// handled by feed
	hsAnswer(*hsAnswer, *gnet.MessageContext) error
	// handled by inflow
	hsSuccess(*hsSuccess, *gnet.MessageContext) error
}

// NewNode creates new Node with given secret key and configs.
// If conf is nil then default configs are used (see NewConfig).
// You can provide DB and Encoder interfaces. If they are nil
// then they wil be created automatically. Don't forget to
// register types you want to use
func NewNode(sec cipher.SecKey, conf *Config, d db.DB, e enc.Encoder) Node {
	if conf == nil {
		conf = NewConfig()
	}

	// =========================================================================
	// '''''''''''''''''''''''''''''''''''''''''''''''''''''''''''''''''''''''''

	// Unfotunately, it's impossible to determine listening address
	// and port, because of _gnet_ limitations. Thus, we must provide
	// real address and real port (instead of empty values for
	// arbitrary assignment)

	// PR: https://github.com/skycoin/skycoin/pull/290

	if conf.Address == "" {
		conf.Address = "127.0.0.1"
	}

	if conf.Port == 0 {
		conf.Port = 7899
	}

	//
	// TODO: modify gnet and modify Address() method of Node
	//       to return listening address from listener
	//

	// ,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,
	// =========================================================================

	if d == nil {
		d = db.NewDB()
	}

	if e == nil {
		e = enc.NewEncoder()
	}

	n := &node{
		conf:     *conf, // make copy of given configs
		sec:      sec,
		pub:      cipher.PubKeyFromSecKey(sec),
		pending:  make(map[*gnet.Connection]*hsp),
		feedpk:   make(map[cipher.PubKey]*gnet.Connection),
		inflowpk: make(map[cipher.PubKey]*gnet.Connection),
		desired:  make(map[*gnet.Connection]cipher.PubKey),

		db:  d,
		enc: e,

		Logger: NewLogger(LOG_PREFIX, DEBUG),
	}

	n.Debug("create node")
	n.Debug(conf.HumanString())
	return n
}

type node struct {
	// sync.RWMutex

	Logger

	// keep confiurations
	conf Config

	// keys
	pub cipher.PubKey
	sec cipher.SecKey

	// db
	db  db.DB
	enc enc.Encoder

	// connections

	feed   *gnet.ConnectionPool // braodcast data
	inflow *gnet.ConnectionPool // receive data (connections to remote feeds)

	// pending connections (handshake processing)
	pending   map[*gnet.Connection]*hsp
	pendinglk sync.RWMutex

	// desired pubkey of remote node to connect to
	desired   map[*gnet.Connection]cipher.PubKey
	desiredlk sync.Mutex

	// pubkey -> connection (after handshake)
	feedpk     map[cipher.PubKey]*gnet.Connection
	inflowpk   map[cipher.PubKey]*gnet.Connection
	feedpklk   sync.RWMutex
	inflowpklk sync.RWMutex

	// quiting
	quito sync.Once
	quitq chan struct{}

	// done report
	feedd   chan struct{} // feed is done
	inflowd chan struct{} // inflow is done
}

func (n *node) Start() (err error) {
	n.Debug("start node")
	if err = n.startFeed(); err != nil {
		return
	}
	n.startInflow()
	// create channels for Close
	n.quitq = make(chan struct{})
	n.feedd = make(chan struct{})
	n.inflowd = make(chan struct{})
	return
}

// count handshake timeout and close connection if it was exceeded
func (n *node) countHandshakeTimeout(c *gnet.Connection) {
	n.Debug("start counting handshake timeout: ", c.Conn.RemoteAddr())
	select {
	case <-n.quitq:
		n.Debug("quit counting handshake timeout: ", c.Conn.RemoteAddr())
		return
	case <-time.After(n.conf.HandshakeTimeout):
		n.pendinglk.RLock()
		defer n.pendinglk.RUnlock()
		// if !ok then handshake already performed
		if _, ok := n.pending[c]; !ok {
			return
		}
		n.Debug("handshake timeout was exceeded: ", c.Conn.RemoteAddr())
		// DisconnectCallback will delete(n.pending, c)
		c.ConnectionPool.Disconnect(c, ErrHandshakeTimeout)
	}
}

// startFeed creates outgoign connections pool and starts it
func (n *node) startFeed() (err error) {
	n.Debug("start feed")

	var gc gnet.Config = n.conf.gnetConfigFeed()

	// set up disconnect callback
	gc.DisconnectCallback = func(c *gnet.Connection,
		reason gnet.DisconnectReason) {

		n.Printf("disconnect from feed: %v, reason: %v",
			c.Conn.RemoteAddr(),
			reason)

		n.pendinglk.Lock()
		defer n.pendinglk.Unlock()

		// is it pending?
		if _, ok := n.pending[c]; ok {
			delete(n.pending, c)
		} else {
			// established
			n.feedpklk.Lock()
			defer n.feedpklk.Unlock()

			for k, v := range n.feedpk {
				if v == c {
					n.Debug("remove subscriber: ", k.Hex())
					delete(n.feedpk, k)
				}
			}
		}

	}

	// set up connect callback
	gc.ConnectCallback = func(c *gnet.Connection, solicited bool) {

		n.Printf("new incoming connection: %v",
			c.Conn.RemoteAddr())

		n.pendinglk.Lock()
		defer n.pendinglk.Unlock()

		n.pending[c] = new(hsp)

		// is there any handshake timeout?
		if n.conf.HandshakeTimeout != 0 {
			go n.countHandshakeTimeout(c)
		}

	}

	n.feed = gnet.NewConnectionPool(gc, n)

	if err = n.feed.StartListen(); err != nil {
		return
	}

	go n.feed.AcceptConnections()
	go n.handleFeed()

	return
}

// startInflow creates incoming connections pool and starts it
func (n *node) startInflow() {
	n.Debug("start inflow")

	var gc gnet.Config = n.conf.gnetConfigInflow()

	// set up DisconnectCallback
	gc.DisconnectCallback = func(c *gnet.Connection,
		reason gnet.DisconnectReason) {

		n.Printf("disconnect from remote feed: %v, reason: %v",
			c.Conn.RemoteAddr(),
			reason)

		n.pendinglk.Lock()
		defer n.pendinglk.Unlock()

		// is it pending?
		if _, ok := n.pending[c]; ok {
			delete(n.pending, c)
			// may be this conenction has desired public key
			n.desiredlk.Lock()
			{
				delete(n.desired, c)
			}
			n.desiredlk.Unlock()
		} else {
			// established
			n.inflowpklk.Lock()
			defer n.inflowpklk.Unlock()

			for k, v := range n.inflowpk {
				if v == c {
					n.Debug("remove subscription: ", k.Hex())
					delete(n.inflowpk, k)
					break
				}
			}
		}

	}

	// set up ConnectCallback
	// ok, we need to send hsQuestion and start
	// handshake timeout counting
	gc.ConnectCallback = func(c *gnet.Connection, solicited bool) {

		n.Printf("connecting to remote feed: %v",
			c.Conn.RemoteAddr())

		var h *hsp = new(hsp)

		n.pendinglk.Lock()
		{
			n.pending[c] = h
		}
		n.pendinglk.Unlock()

		// is there any handshake timeout?
		if n.conf.HandshakeTimeout != 0 {
			go n.countHandshakeTimeout(c)
		}

		// send initial handshake message
		n.Debug("send handshake question")
		n.inflow.SendMessage(c, h.hsQuestion(n))
	}

	n.inflow = gnet.NewConnectionPool(gc, n)

	go n.handleInflow()
}

func (n *node) handleFeed() {
	n.Debug("handle feed events")
	for {
		select {
		case <-n.quitq:
			n.Debug("stop handling feed events")
			close(n.feedd)
			return
		default:
		}
		n.feed.HandleMessages()
	}
}

func (n *node) handleInflow() {
	n.Debug("handle inflow events")
	for {
		select {
		case <-n.quitq:
			n.Debug("stop handling inflow events")
			close(n.inflowd)
			return
		default:
		}
		n.inflow.HandleMessages()
	}
}

// Close closes node and make it invaid for further using
func (n *node) Close() {
	n.Debug("closing node...")
	if n.inflow != nil {
		n.inflow.StopListen()
	}
	if n.feed != nil {
		n.feed.StopListen()
	}
	if n.quitq != nil {
		n.quito.Do(func() {
			close(n.quitq)
		})
		// TODO: close without start block
		<-n.inflowd // inflow is done
		<-n.feedd   // feed is done
	}
	n.Debug("node was closed")
}

//
// ...
//

func (n *node) PubKey() cipher.PubKey {
	return n.pub
}

func (n *node) Sign(hash cipher.SHA256) cipher.Sig {
	return cipher.SignHash(hash, n.sec)
}

func (n *node) Broadcast(hash cipher.SHA256) {
	n.Debug("Broadcast: ", hash.Hex())
	n.feed.BroadcastMessage(&Announce{
		Hash: hash,
	})
}

func (n *node) DB() db.DB {
	return n.db
}

func (n *node) Encoder() enc.Encoder {
	return n.enc
}

//
// Information and control
//

func (n *node) Address() (addr net.Addr) {
	addr, _ = net.ResolveTCPAddr(
		"tcp",
		fmt.Sprintf("%s:%d", n.conf.Address, n.conf.Port))
	return
}

func (n *node) Subscribe(address string, pub cipher.PubKey) error {
	if pub == (cipher.PubKey{}) {
		n.Debug("subscribe to ", address)
	} else {
		n.Debug("subscribe to %s, with desired pub key: %s",
			address, pub.Hex())
	}
	conn, err := n.inflow.Connect(address)
	if err != nil {
		return err
	}
	if pub != (cipher.PubKey{}) {
		n.desiredlk.Lock()
		{
			n.desired[conn] = pub
		}
		n.desiredlk.Unlock()
	}
	return nil
}

func (n *node) Unsubscribe(pub cipher.PubKey) (err error) {
	n.Debug("unsubscribe ", pub.Hex())
	var (
		conn *gnet.Connection
		ok   bool
	)
	n.inflowpklk.RLock()
	{
		conn, ok = n.inflowpk[pub]
	}
	n.inflowpklk.RUnlock()
	if !ok {
		err = errors.New("not found")
		return
	}
	conn.ConnectionPool.Disconnect(
		conn,
		gnet.DisconnectReason(errors.New("unsibscribe")))
	return
}

func (n *node) ListSubscriptions(fn ListFunc) {
	if fn == nil {
		n.Panic("call ListSubscriptions with nil")
	}
	n.inflowpklk.RLock()
	defer n.inflowpklk.RUnlock()
	for pk, conn := range n.inflowpk {
		fn(pk, conn.Conn.RemoteAddr())
	}
}

func (n *node) ListSubscribers(fn ListFunc) {
	if fn == nil {
		n.Panic("call ListSubscribers with nil")
	}
	n.feedpklk.RLock()
	defer n.feedpklk.RUnlock()
	for pk, conn := range n.feedpk {
		fn(pk, conn.Conn.RemoteAddr())
	}
}

//
// Internals
//

// side of connection

func (n *node) isFeed(ctx *gnet.MessageContext) bool {
	return ctx.Conn.ConnectionPool == n.feed
}

func (n *node) isInflow(ctx *gnet.MessageContext) bool {
	return ctx.Conn.ConnectionPool == n.inflow
}

// handshake processing

var (
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

// handled by feed
func (n *node) hsQuestion(q *hsQuestion, ctx *gnet.MessageContext) error {
	n.Debugf("got handshake question from %v, public key of wich is %s",
		ctx.Conn.Conn.RemoteAddr(),
		q.PubKey.Hex())
	// hsQuestion must be handled by feed
	if !n.isFeed(ctx) {
		return ErrUnexpectedHandshake // terminate connection
	}
	// persistence check
	var (
		h  *hsp // h is a pointer, we are free to modify it
		ok bool
	)
	// reading lock/unlock
	n.pendinglk.RLock()
	{
		h, ok = n.pending[ctx.Conn]
	}
	n.pendinglk.RUnlock()
	// doesn't exists
	if !ok {
		return ErrUnexpectedHandshake // terminate connection
	}
	// check handshake step
	if h.step != 0 {
		return ErrUnexpectedHandshake // terminate connection
	}
	// check handshake message
	if q.PubKey == (cipher.PubKey{}) || q.Question == (uuid.UUID{}) {
		return ErrMalformedHandshake // terminate connection
	}
	// answer and question
	ctx.Conn.ConnectionPool.SendMessage(ctx.Conn, h.hsQuestionAnswer(q, n))

	return nil
}

// handled by inflow
func (n *node) hsQuestionAnswer(qa *hsQuestionAnswer,
	ctx *gnet.MessageContext) error {

	n.Debugf(
		"got handshake question-answer from %v, public key of wich is %s",
		ctx.Conn.Conn.RemoteAddr(),
		qa.PubKey.Hex())

	// hsQuestionAnswer must be handled by inflow
	if !n.isInflow(ctx) {
		return ErrUnexpectedHandshake // terminate connection
	}
	// persistence check
	var (
		h   *hsp // h is a pointer, we are free to modify it
		ok  bool
		err error
	)
	// reading lock/unlock
	n.pendinglk.RLock()
	{
		h, ok = n.pending[ctx.Conn]
	}
	n.pendinglk.RUnlock()
	// doesn't exists
	if !ok {
		return ErrUnexpectedHandshake // terminate connection
	}
	// check handshake step (0 - send question, 1 -receive qa)
	if h.step != 1 {
		return ErrUnexpectedHandshake // terminate connection
	}
	// check handshake message
	if qa.PubKey == (cipher.PubKey{}) ||
		qa.Question == (uuid.UUID{}) ||
		qa.Answer == (cipher.Sig{}) {
		return ErrMalformedHandshake // terminate connection
	}
	// desired public key
	n.desiredlk.Lock()
	{
		var dpub cipher.PubKey
		// do we have desired public key for this connection
		if dpub, ok = n.desired[ctx.Conn]; ok {
			// missmatch
			if h.pub != dpub {
				// n.desired will be cleaned up by DisconnectCallback
				n.desiredlk.Unlock()
				return ErrPubKeyMismatch
			}
		}

		// ok, we don't have desired public key or
		// received (remote) key is what we want
	}
	n.desiredlk.Unlock()
	//
	// verify answer
	err = cipher.VerifySignature(
		qa.PubKey,
		qa.Answer,
		cipher.SumSHA256(h.question.Bytes()))
	if err != nil {
		return gnet.DisconnectReason(err) // terminate connection
	}
	// answer and question
	ctx.Conn.ConnectionPool.SendMessage(ctx.Conn, h.hsAnswer(qa, n))

	return nil
}

// handled by feed
func (n *node) hsAnswer(a *hsAnswer, ctx *gnet.MessageContext) error {

	n.Debugf("got handshake answer from %v",
		ctx.Conn.Conn.RemoteAddr())

	// hsAnswer must be handled by feed
	if !n.isFeed(ctx) {
		return ErrUnexpectedHandshake // terminate connection
	}
	// persistence check
	var (
		h   *hsp // h is a pointer, we are free to modify it
		ok  bool
		err error
	)
	// reading lock/unlock
	n.pendinglk.RLock()
	{
		h, ok = n.pending[ctx.Conn]
	}
	n.pendinglk.RUnlock()
	// doesn't exists
	if !ok {
		return ErrUnexpectedHandshake // terminate connection
	}
	// check handshake step (0 - send qa, 1 -receive answer)
	if h.step != 1 {
		return ErrUnexpectedHandshake // terminate connection
	}
	// check handshake message
	if a.Answer == (cipher.Sig{}) {
		return ErrMalformedHandshake // terminate connection
	}
	// verify answer
	err = cipher.VerifySignature(
		h.pub,    // remote public key
		a.Answer, // received answer
		cipher.SumSHA256(h.question.Bytes())) // my question hash
	if err != nil {
		return gnet.DisconnectReason(err) // terminate connection
	}
	// remove connection from pending and add it to pubKey->connection mapping
	// (not sure that deferred unlock is frendly with SendMessage)
	n.feedpklk.Lock()
	{
		// firstly we need to check n.feedpk; may be we already have
		// connection with given public key
		if _, ok = n.feedpk[h.pub]; ok {
			n.feedpklk.Unlock()
			return ErrAlreadyConnected
		}
		// map public key -> connection
		n.feedpk[h.pub] = ctx.Conn
		// delete from pending
		n.pendinglk.Lock()
		{
			delete(n.pending, ctx.Conn)
		}
		n.pendinglk.Unlock()
		n.Debug("feed got new established connection from %v,"+
			" public key of wich is %s",
			ctx.Conn.Conn.RemoteAddr(),
			h.pub.Hex())
	}
	n.feedpklk.Unlock()
	// success, yay!
	ctx.Conn.ConnectionPool.SendMessage(ctx.Conn, h.hsSuccess())

	return nil
}

// handled by inflow
func (n *node) hsSuccess(s *hsSuccess, ctx *gnet.MessageContext) error {

	n.Debugf("got handshake success from %v",
		ctx.Conn.Conn.RemoteAddr())

	// hsSuccess must be handled by inflow
	if !n.isInflow(ctx) {
		return ErrUnexpectedHandshake // terminate connection
	}
	// persistence check
	var (
		h  *hsp // h is a pointer, we are free to modify it
		ok bool
	)
	// reading lock/unlock
	n.pendinglk.RLock()
	{
		h, ok = n.pending[ctx.Conn]
	}
	n.pendinglk.RUnlock()
	// doesn't exists
	if !ok {
		return ErrUnexpectedHandshake // terminate connection
	}
	// check handshake step
	if h.step != 2 {
		return ErrUnexpectedHandshake // terminate connection
	}
	// remove connection from pending and add it to pubKey->connection mapping
	// (not sure that deferred unlock is frendly with SendMessage)
	n.inflowpklk.Lock()
	{
		// firstly we need to check n.inflowpk; may be we already have
		// connection with given public key
		if _, ok = n.inflowpk[h.pub]; ok {
			n.inflowpklk.Unlock()
			return ErrAlreadyConnected
		}
		// map public key -> connection
		n.inflowpk[h.pub] = ctx.Conn
		// delete from pending
		n.pendinglk.Lock()
		{
			delete(n.pending, ctx.Conn)
		}
		n.pendinglk.Unlock()
		n.Debug(
			"new connection was established to %v, public key of wich is %s",
			ctx.Conn.Conn.RemoteAddr(),
			h.pub.Hex())
	}
	n.inflowpklk.Unlock()
	// success, yay!
	// no more handshake messages

	return nil
}
