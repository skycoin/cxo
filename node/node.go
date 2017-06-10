// A node package implements P2P transport for sharing CX objects
package node

import (
	"errors"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"

	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"
)

var (
	ErrTimeout              = errors.New("timeout")
	ErrSubscriptionRejected = errors.New("subscription rejected by remote peer")
	ErrNilConnection        = errors.New("subscribe to nil connection")
	ErrUnexpectedResponse   = errors.New("unexpected response")
)

type fillRoot struct {
	root  *skyobject.Root     // filling the root to send forward
	c     *gnet.Conn          // from which the root received
	await skyobject.Reference // waiting for
}

// A Node represents CXO P2P node
// that includes RPC server if enabled
// by configs
type Node struct {
	// logger of the server
	log.Logger

	src msgSource

	// configuratios
	conf NodeConfig

	// database
	db data.DB

	// skyobject
	so *skyobject.Container

	// feeds
	fmx   sync.RWMutex
	feeds map[cipher.PubKey]map[*gnet.Conn]struct{}

	// pending subscriptions
	// (while a node subscribes to feed of another node
	// the first node sends SubscrieMsg and waits for
	// accept or deny (reject))
	pmx     sync.Mutex
	pending map[*gnet.Conn]map[cipher.PubKey]struct{}

	rmx   sync.RWMutex
	roots []*fillRoot // filling up

	// request/response replies
	rpmx      sync.Mutex
	responses map[uint32]chan Msg

	// connections
	pool *gnet.Pool
	rpc  *RPC // rpc server

	// closing
	quit  chan struct{}
	quito sync.Once

	done  chan struct{} // when quit done
	doneo sync.Once

	await sync.WaitGroup
}

// NewNode creates new Node instnace using given
// configurations. The functions creates database and
// Container of skyobject instances internally
func NewNode(sc NodeConfig) (s *Node, err error) {
	s, err = NewNodeReg(sc, nil)
	return
}

// NewNodeReg creates new Node instance using given
// skyobject.Registry to create container
func NewNodeReg(sc NodeConfig, reg *skyobject.Registry) (s *Node,
	err error) {

	// database

	var db data.DB
	if sc.InMemoryDB {
		db = data.NewMemoryDB()
	} else {
		if sc.DataDir != "" {
			if err = initDataDir(sc.DataDir); err != nil {
				return
			}
		}
		if db, err = data.NewDriveDB(sc.DBPath); err != nil {
			return
		}
	}

	// container

	var so *skyobject.Container
	so = skyobject.NewContainer(db, reg)

	// node instance

	s = new(Node)

	s.Logger = log.NewLogger(sc.Log.Prefix, sc.Log.Debug)
	s.conf = sc

	s.db = db

	s.so = so
	s.feeds = make(map[cipher.PubKey]map[*gnet.Conn]struct{})

	s.pending = make(map[*gnet.Conn]map[cipher.PubKey]struct{})

	s.responses = make(map[uint32]chan Msg)

	// fill up feeds from database
	for _, pk := range s.db.Feeds() {
		s.feeds[pk] = make(map[*gnet.Conn]struct{})
	}

	if sc.Config.Logger == nil {
		sc.Config.Logger = s.Logger // use the same logger
	}

	// gnet related callbacks
	if ch := sc.Config.OnCreateConnection; ch == nil {
		sc.Config.OnCreateConnection = s.connectHandler
	} else {
		sc.Config.OnCreateConnection = func(c *gnet.Conn) {
			s.connectHandler(c)
			ch(c)
		}
	}
	if dh := sc.Config.OnCloseConnection; dh == nil {
		sc.Config.OnCloseConnection = s.disconnectHandler
	} else {
		sc.Config.OnCloseConnection = func(c *gnet.Conn) {
			s.disconnectHandler(c)
			dh(c)
		}
	}

	if s.pool, err = gnet.NewPool(sc.Config); err != nil {
		s = nil
		return
	}

	if sc.EnableRPC {
		s.rpc = newRPC(s)
	}

	s.quit = make(chan struct{})
	s.done = make(chan struct{})

	return
}

// Start the Node
func (s *Node) Start() (err error) {
	s.Debugf(`starting node:
    data dir:             %s

    max connections:      %d
    max message size:     %d

    dial timeout:         %v
    read timeout:         %v
    write timeout:        %v

    ping interval:        %v

    read queue:           %d
    write queue:          %d

    redial timeout:       %d
    max redial timeout:   %d
    dials limit:          %d

    read buffer:          %d
    write buffer:         %d

    TLS:                  %v

    enable RPC:           %v
    RPC address:          %s
    listening address:    %s
    enable listening:     %v
    remote close:         %t

    in-memory DB:         %v
    DB path:              %s

    gc interval:          %v

    debug:                %#v
`,
		s.conf.DataDir,
		s.conf.MaxConnections,
		s.conf.MaxMessageSize,

		s.conf.DialTimeout,
		s.conf.ReadTimeout,
		s.conf.WriteTimeout,

		s.conf.PingInterval,

		s.conf.ReadQueueLen,
		s.conf.WriteQueueLen,

		s.conf.RedialTimeout,
		s.conf.MaxRedialTimeout,
		s.conf.DialsLimit,

		s.conf.ReadBufferSize,
		s.conf.WriteBufferSize,

		s.conf.TLSConfig != nil,

		s.conf.EnableRPC,
		s.conf.RPCAddress,
		s.conf.Listen,
		s.conf.EnableListener,
		s.conf.RemoteClose,

		s.conf.InMemoryDB,
		s.conf.DBPath,

		s.conf.GCInterval,

		s.conf.Log.Debug,
	)

	// start listener
	if s.conf.EnableListener == false {
		if err = s.pool.Listen(s.conf.Listen); err != nil {
			return
		}
		s.Print("listen on ", s.pool.Address())
	}

	// start rpc listener if need
	if s.conf.EnableRPC == true {
		if err = s.rpc.Start(s.conf.RPCAddress); err != nil {
			s.pool.Close()
			return
		}
		s.Print("rpc listen on ", s.rpc.Address())
	}

	if s.conf.PingInterval > 0 {
		s.await.Add(1)
		go s.pingsLoop()
	}

	if s.conf.GCInterval > 0 {
		s.await.Add(1)
		go s.gcLoop()
	}

	return
}

// Close the Node
func (s *Node) Close() (err error) {
	s.quito.Do(func() {
		close(s.quit)
	})
	err = s.pool.Close()

	if s.conf.EnableRPC {
		s.rpc.Close()
	}
	s.await.Wait()
	s.db.Close() // <- close database after all (otherwise, it causes panicing)
	s.doneo.Do(func() {
		close(s.done)
	})
	return
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func (s *Node) pingsLoop() {
	defer s.await.Done()

	var tk *time.Ticker = time.NewTicker(s.conf.PingInterval)
	defer tk.Stop()

	for {
		select {
		case <-tk.C:
			now := time.Now()
			for _, c := range s.pool.Connections() {
				md := maxDuration(now.Sub(c.LastRead()), now.Sub(c.LastWrite()))
				if md < s.conf.PingInterval {
					continue
				}
				s.sendPingMsg(c)
			}
		case <-s.quit:
			return
		}
	}
}

func (s *Node) gcLoop() {
	defer s.await.Done()

	var tk *time.Ticker = time.NewTicker(s.conf.GCInterval)
	defer tk.Stop()

	s.Debug("start GC loop ", s.conf.GCInterval)
	for {
		select {
		case <-tk.C:
			tp := time.Now()
			s.Debug("GC pause")
			s.so.GC(false)
			s.Debug("GC done ", time.Now().Sub(tp))
		case <-s.quit:
			return
		}
	}

}

// send a message to given connection
func (s *Node) sendMessage(c *gnet.Conn, msg Msg) (ok bool) {
	return s.sendEncodedMessage(c, Encode(msg))
}

func (s *Node) sendEncodedMessage(c *gnet.Conn, msg []byte) (ok bool) {
	s.Debugf("send message %T to %s", msg, c.Address())

	select {
	case c.SendQueue() <- msg:
		ok = true
	case <-c.Closed():
	default:
		s.Print("[ERR] %s send queue full", c.Address())
		c.Close()
	}
	return
}

func boolString(t bool, ts, fs string) string {
	if t {
		return ts
	}
	return fs
}

func (s *Node) connectHandler(c *gnet.Conn) {
	s.Debugf("got new %s connection %s %s",
		boolString(c.IsIncoming(), "incoming", "outgoing"),
		boolString(c.IsIncoming(), "from", "to"),
		c.Address())
	// handle
	s.await.Add(1)
	go s.handleConnection(c)
}

func (s *Node) disconnectHandler(c *gnet.Conn) {
	s.Debugf("closed connection %s", c.Address())
}

// delete connection from feeds
func (s *Node) deleteConnFromFeeds(c *gnet.Conn) {
	s.fmx.Lock()
	defer s.fmx.Unlock()

	for _, cs := range s.feeds {
		delete(cs, c)
	}
}

// delete connection from pendings
func (s *Node) deleteConnFromPending(c *gnet.Conn) {
	s.pmx.Lock()
	defer s.pmx.Unlock()

	delete(s.pending, c)
}

// close a connection removing associated resources
func (s *Node) close(c *gnet.Conn) {
	s.deleteConnFromFeeds(c)
	s.deleteConnFromPending(c)
	c.Close()
}

func (s *Node) handleConnection(c *gnet.Conn) {
	s.Debug("handle connection ", c.Address())
	defer s.Debug("stop handling connection", c.Address())

	defer s.await.Done()
	defer s.close(c)

	var (
		closed  <-chan struct{} = c.Closed()
		receive <-chan []byte   = c.ReceiveQueue()

		data []byte
		msg  Msg

		err error
	)

	for {
		select {
		case <-closed:
			return
		case data = <-receive:
			if msg, err = Decode(data); err != nil {
				s.Printf("[ERR] %s decoding message: %v", c.Address(), err)
				return
			}
			s.handleMsg(c, msg)
		}
	}

}

func shortHex(a string) string {
	return string([]byte(a)[:7])
}

func (s *Node) subscribeConn(c *gnet.Conn, feed cipher.PubKey) (accept,
	already bool) {

	s.fmx.Lock()
	defer s.fmx.Unlock()

	if cs, ok := s.feeds[feed]; ok {
		if _, already = cs[c]; already {
			return
		}
		cs[c], accept = struct{}{}, true
	}

	return // no such feed
}

func (s *Node) sendLastFullRoot(c *gnet.Conn, feed cipher.PubKey) (sent bool) {
	if full := s.so.LastFullRoot(feed); full != nil {
		sent = s.sendRootMsg(c, feed, full.Encode())
	}
	return
}

func (s *Node) handleSubscribeMsg(c *gnet.Conn, msg *SubscribeMsg) {
	// (1) subscribe if the Node shares feed and send AcceptSubscriptionMsg back
	//     and send latest full root of the feed if has
	// (2) send AcceptSubscriptionMsg back if the connection already
	//     subscibed to the feed
	// (3) send  DenySubscription if the Node doesn't share feed
	if accept, already := s.subscribeConn(c, msg.Feed); already == true {
		// (2)
		s.sendAcceptSubscriptionMsg(c, msg.Id(), msg.Feed)
		return
	} else if accept == true {
		// (1)
		if s.sendAcceptSubscriptionMsg(c, msg.Id(), msg.Feed) {
			s.sendLastFullRoot(c, msg.Feed)
		}
		return
	}
	s.sendDenySubscriptionMsg(c, msg.Id(), msg.Feed) // (3)
}

func (s *Node) handleUnsubscribeMsg(c *gnet.Conn, msg *UnsubscribeMsg) {
	// just unsubscribe if subscribed
	s.fmx.Lock()
	defer s.fmx.Unlock()

	if cs, ok := s.feeds[msg.Feed]; ok {
		delete(cs, c)
	}
}

// the function deletes given conn->feed from pendings
// and returns true if there was
func (s *Node) deleteConnFeedFromPending(c *gnet.Conn,
	feed cipher.PubKey) (ok bool) {

	s.pmx.Lock()
	defer s.pmx.Unlock()

	var cf map[cipher.PubKey]struct{}
	if cf, ok = s.pending[c]; !ok {
		return // no such conn->feed in pending
	}
	if _, ok = cf[feed]; !ok {
		return // no such conn->feed in pending
	}
	if len(cf) == 1 {
		delete(s.pending, c)
		return
	}
	delete(cf, feed)
	return
}

func (s *Node) handleAcceptSubscriptionMsg(c *gnet.Conn,
	msg *AcceptSubscriptionMsg) {

	// if subscription had been accepted then we
	// need to subscribe remote peer our side

	// But (!) we must not subscribe a remote peer if we
	// receive an AcceptSubscriptionMsg but we didn't send
	// SubscribeMsg to the remote peer before

	if !s.deleteConnFeedFromPending(c, msg.Feed) {
		s.Debug("unexpected AcceptSubscriptionMsg from ", c.Address())
		return
	}

	// subscribe the remote peer to the subscription
	if ok, _ := s.subscribeConn(c, msg.Feed); ok {
		if !s.sendLastFullRoot(c, msg.Feed) {
			return // sending error
		}

		// call OnSubscriptionAccepted callback
		if callback := s.conf.OnSubscriptionAccepted; callback != nil {
			callback(c, msg.Feed)
		}
	}

	// else -> seems the feed was removed from the node

}

func (s *Node) handleDenySubscriptionMsg(c *gnet.Conn,
	msg *DenySubscriptionMsg) {

	// remove from pending and call OnSubscriptionDenied callback

	if !s.deleteConnFeedFromPending(c, msg.Feed) {
		s.Debug("unexpected DenySubscriptionMsg from ", c.Address())
		return
	}

	if callback := s.conf.OnSubscriptionDenied; callback != nil {
		callback(c, msg.Feed)
	}
}

func (s *Node) sendToFeed(feed cipher.PubKey, msg Msg, except *gnet.Conn) {

	var data []byte = Encode(msg) // encode once

	s.fmx.RLock()
	defer s.fmx.RUnlock()

	for c := range s.feeds[feed] {
		if c == except {
			continue
		}
		s.sendEncodedMessage(c, data) // send many times the same slice
	}
}

func (s *Node) addNonFullRoot(root *skyobject.Root,
	c *gnet.Conn) (fl *fillRoot) {

	fl = &fillRoot{root, c, skyobject.Reference{}}
	s.roots = append(s.roots, fl)
	return
}

func (s *Node) delNonFullRoot(root *skyobject.Root) {
	for i, fl := range s.roots {
		if fl.root == root {
			copy(s.roots[i:], s.roots[i+1:])
			s.roots[len(s.roots)-1] = nil // set to nil for golang GC
			s.roots = s.roots[:len(s.roots)-1]
			return
		}
	}
	return
}

func (s *Node) hasFeed(pk cipher.PubKey) (yep bool) {
	s.fmx.RLock()
	defer s.fmx.RUnlock()

	_, yep = s.feeds[pk]
	return
}

func (s *Node) handleRootMsg(c *gnet.Conn, msg *RootMsg) {
	if !s.hasFeed(msg.Feed) {
		s.Debug("reject root: not subscribed")
		return
	}
	root, err := s.so.AddRootPack(&msg.RootPack)
	if err != nil {
		if err == data.ErrRootAlreadyExists {
			s.Debug("reject root: alredy have this root")
			return
		}
		s.Print("[ERR] error appending root: ", err)
		return
	}
	if root.IsFull() {
		// change remote Id to local to avoid collisions
		s.sendToFeed(msg.Feed, msg, c)
		return
	}

	s.rmx.Lock()
	defer s.rmx.Unlock()

	fl := s.addNonFullRoot(root, c)
	if !root.HasRegistry() {
		if !s.sendRequestRegistryMsg(c, root.RegistryReference()) {
			s.delNonFullRoot(root) // sending error (connection closed)
		}
		return
	}
	err = root.WantFunc(func(ref skyobject.Reference) error {
		if !s.sendRequestDataMsg(c, ref) {
			s.delNonFullRoot(root) // sending error (connection closed)
		} else {
			fl.await = ref // keep last requested reference
		}
		return skyobject.ErrStopRange
	})
	if err != nil {
		s.Print("[ERR] unexpected error: ", err)
	}
}

func (s *Node) handleRequestRegistryMsg(c *gnet.Conn,
	msg *RequestRegistryMsg) {

	if encReg, ok := s.db.Get(cipher.SHA256(msg.Ref)); ok {
		s.sendRegistryMsg(c, encReg)
	}
}

func (s *Node) handleRegistryMsg(c *gnet.Conn, msg *RegistryMsg) {
	reg, err := skyobject.DecodeRegistry(msg.Reg)
	if err != nil {
		s.Print("[ERR] error decoding received registry:", err)
		return
	}

	if !s.so.WantRegistry(reg.Reference()) {
		return // don't want the registry
	}

	s.so.AddRegistry(reg)

	s.rmx.Lock()
	defer s.rmx.Unlock()
	var i int = 0 // index for deleting
	for _, fl := range s.roots {
		if fl.root.RegistryReference() == reg.Reference() {
			if fl.root.IsFull() {
				s.sendToFeed(fl.root.Pub(), s.src.NewRootMsg(
					fl.root.Pub(),    // feed
					fl.root.Encode(), // root pack
				), fl.c)
				continue // delete
			}
			var sent bool
			err = fl.root.WantFunc(func(ref skyobject.Reference) error {
				if sent = s.sendRequestDataMsg(c, ref); sent {
					fl.await = ref
				}
				return skyobject.ErrStopRange
			})
			if err != nil {
				s.Print("[ERR] unexpected error: ", err)
				continue // delete
			}
			if !sent {
				continue // delete
			}
		}
		s.roots[i] = fl
		i++
	}
	s.roots = s.roots[:i]
}

func (s *Node) handleRequestDataMsg(c *gnet.Conn, msg *RequestDataMsg) {
	if data, ok := s.so.Get(msg.Ref); ok {
		s.sendDataMsg(c, data)
	}
}

func (s *Node) handleDataMsg(c *gnet.Conn, msg *DataMsg) {
	hash := skyobject.Reference(cipher.SumSHA256(msg.Data))

	s.rmx.Lock()
	defer s.rmx.Unlock()

	// does the Server really want the data
	var want bool
	for _, fl := range s.roots {
		if fl.await == hash {
			want = true
			break
		}
	}
	if !want {
		return // doesn't want the data
	}
	s.so.Set(hash, msg.Data) // save

	// check filling
	var i int = 0 // index for deleting
	for _, fl := range s.roots {
		if fl.await == hash {
			if fl.root.IsFull() {
				s.sendToFeed(fl.root.Pub(), s.src.NewRootMsg(
					fl.root.Pub(),    // feed
					fl.root.Encode(), // root pack
				), fl.c)
				continue // delete
			}
			var sent bool
			err := fl.root.WantFunc(func(ref skyobject.Reference) error {
				if sent = s.sendRequestDataMsg(c, ref); sent {
					fl.await = ref
				}
				return skyobject.ErrStopRange
			})
			if err != nil {
				s.Print("[ERR] unexpected error: ", err)
				continue // delete
			}
			if !sent {
				continue // delete
			}
		}
		s.roots[i] = fl
		i++
	}
	s.roots = s.roots[:i]
}

func (s *Node) handleRequestListOfFeedsMsg(c *gnet.Conn,
	x *RequestListOfFeedsMsg) {

	if s.conf.PublicServer == true {
		s.sendListOfFeedsMsg(c, x.Id(), s.Feeds())
	} else {
		s.sendNonPublicServerMsg(c, x.Id()) // reject
	}
}

func (s *Node) handlePingMsg(c *gnet.Conn) {
	s.sendPongMsg(c)
}

func (s *Node) handleMsg(c *gnet.Conn, msg Msg) {
	s.Debugf("handle message %T from %s", msg, c.Address())

	switch x := msg.(type) {

	//
	// subscribe/unsubscribe
	//

	// subscribe/unsubscribe
	case *SubscribeMsg:
		s.handleSubscribeMsg(c, x)
	case *UnsubscribeMsg:
		s.handleUnsubscribeMsg(c, x)

	// relies for subscribing
	case *AcceptSubscriptionMsg:
		s.handleAcceptSubscriptionMsg(c, x)
	case *DenySubscriptionMsg:
		s.handleDenySubscriptionMsg(c, x)

		//
		// root, data, registry, requests
		//

	// root
	case *RootMsg:
		s.handleRootMsg(c, x)

	// registry
	case *RequestRegistryMsg:
		s.handleRequestRegistryMsg(c, x)
	case *RegistryMsg:
		s.handleRegistryMsg(c, x)

	//data
	case *RequestDataMsg:
		s.handleRequestDataMsg(c, x)
	case *DataMsg:
		s.handleDataMsg(c, x)

		//
		// public servers
		//

	case *RequestListOfFeedsMsg:
		s.handleRequestListOfFeedsMsg(c, x)
	case *ListOfFeedsMsg:
		// do ntohing (handled at the bottom of this method)
	case *NonPublicServerMsg:
		// do ntohing (handled at the bottom of this method)

	//
	// ping / pong
	//

	// ping/pong
	case *PingMsg:
		s.handlePingMsg(c)
	case *PongMsg:
		// do nothing

	// critical
	default:
		s.Printf("[CRIT] unhandled message type %T", msg)
	}

	// the msg is not request that need identified response
	if msg.ResponseFor() == 0 {
		return
	}

	// process responses after handling

	var rc chan Msg
	var ok bool
	if rc, ok = s.takeWaitingForResponse(msg.Id()); ok {
		rc <- msg
	}
}

//
// Public methods of the Node
//

// A Pool returns underlying *gnet.Pool.
// It returns nil if the Node is not started
// yet. Use methods of this Pool to manipulate
// connections: Connect, Disconnect, Address, etc
func (s *Node) Pool() *gnet.Pool {
	return s.pool
}

func (s *Node) addFeed(feed cipher.PubKey) (already bool) {
	s.fmx.Lock()
	defer s.fmx.Unlock()

	if _, already = s.feeds[feed]; !already {
		s.so.AddFeed(feed)
		s.feeds[feed] = make(map[*gnet.Conn]struct{})
	}
	return
}

func (s *Node) addToPending(c *gnet.Conn, feed cipher.PubKey) {
	s.pmx.Lock()
	defer s.pmx.Unlock()

	var ps map[cipher.PubKey]struct{}
	var ok bool
	if ps, ok = s.pending[c]; !ok {
		ps = make(map[cipher.PubKey]struct{})
		s.pending[c] = ps
	}
	ps[feed] = struct{}{} // add anyway
}

// Subscribe to given feed. If given connection is nil, then this subscription
// is local. Otherwise, it subscribes to a remote peer. To handle result use
// (NodeConfig).OnAcceptSubsctiption and OnDeniedSubscription callbacks. The
// connection must be from the gnet.Pool of the Node. To subscribe to the same
// feed of many remote peers call the method many times for every connection
// you want. To make the server subscribed to a feed (even if it is not
// conencted to any remote peer) call this method with nil. To obtain
// *gnet.Conn use (*Node).Pool() methods like
// (*net.Pool).Connection(address string) (*gnet.Conn)
func (s *Node) Subscribe(c *gnet.Conn, feed cipher.PubKey) {
	// subscribe the Node to the feed, create feed in database if not exists
	s.addFeed(feed)
	// just return if we don't want to subscribe to feed of a remote peer
	if c == nil {
		return
	}
	// add conn->feed to pendings
	s.addToPending(c, feed)
	// send SubscribeMsg
	s.sendSubscribeMsg(c, feed)
	return
}

// delte (any connection)->feed from all pending subscriptions
func (s *Node) deleteFeedFromPending(feed cipher.PubKey) {
	s.pmx.Lock()
	defer s.pmx.Unlock()

	for c, ps := range s.pending {
		delete(ps, feed)
		if len(ps) == 0 {
			delete(s.pending, c)
		}
	}
}

// delte all filling root objects of a feed
func (s *Node) deleteFeedFromFilling(feed cipher.PubKey) {
	s.rmx.Lock()
	defer s.rmx.Unlock()

	var i int = 0
	for _, fl := range s.roots {
		if fl.root.Pub() == feed {
			continue // delete
		}
		i++
		s.roots[i] = fl
	}
	s.roots = s.roots[:i]
}

// delete a feed and all associated resources without sending UnsubscribeMsg
// to peers; the sending is not palced in the method to unlock fmx mutex
func (s *Node) deleteFeed(feed cipher.PubKey) (cs map[*gnet.Conn]struct{}) {
	s.fmx.Lock()
	defer s.fmx.Unlock()

	var ok bool
	if cs, ok = s.feeds[feed]; ok {
		delete(s.feeds, feed)
		s.deleteFeedFromPending(feed)
		s.deleteFeedFromFilling(feed)
		s.so.DelFeed(feed) // delete from database
	}
	return
}

// total unsubscribing; delete given feed and all associated resources,
// send UnsubscribeMsg to peers that share the feed
func (s *Node) unsubscribe(feed cipher.PubKey) {
	// we can't use sendToFeed here
	var unsub []byte = Encode(s.src.NewUnsubscribeMsg(feed))
	for peer := range s.deleteFeed(feed) {
		s.sendEncodedMessage(peer, unsub)
	}
}

func (s *Node) deleteConnFeedFromFeeds(c *gnet.Conn, feed cipher.PubKey) {
	s.fmx.Lock()
	defer s.fmx.Unlock()

	if cs, ok := s.feeds[feed]; ok {
		delete(cs, c)
	}
}

// Unsubscribe from a feed of a remote peer or from all remote peers and
// locally too if given gnet.Conn is nil. Given *gnet.Conn must be from
// *gnet.Pool of this Node. Unsubscribe with nil removes feed from
// underlying database and the Node stops sharing the feed
func (s *Node) Unsubscribe(c *gnet.Conn, feed cipher.PubKey) {
	if c == nil {
		s.unsubscribe(feed)
		return
	}
	// 1. remove the conn->feed from pendings
	s.deleteConnFeedFromPending(c, feed)
	// 2. remove the conn from s.feeds->feed
	s.deleteConnFeedFromFeeds(c, feed)
	// 3. send UnsubscribeMsg to peer
	s.sendUnsubscribeMsg(c, feed)
}

// TODO: + Want per root of a feed

// Want returns lits of objects related to given
// feed that the server hasn't got but knows about
func (s *Node) Want(feed cipher.PubKey) (wn []cipher.SHA256) {
	set := make(map[skyobject.Reference]struct{})
	s.so.WantFeed(feed, func(k skyobject.Reference) error {
		set[k] = struct{}{}
		return nil
	})
	if len(set) == 0 {
		return
	}
	wn = make([]cipher.SHA256, 0, len(set))
	for k := range set {
		wn = append(wn, cipher.SHA256(k))
	}
	return
}

// TODO: + Got per root of a feed

// Got returns lits of objects related to given
// feed that the server has got
func (s *Node) Got(feed cipher.PubKey) (gt []cipher.SHA256) {
	set := make(map[skyobject.Reference]struct{})
	s.so.GotFeed(feed, func(k skyobject.Reference) error {
		set[k] = struct{}{}
		return nil
	})
	if len(set) == 0 {
		return
	}
	gt = make([]cipher.SHA256, 0, len(set))
	for k := range set {
		gt = append(gt, cipher.SHA256(k))
	}
	return
}

// Feeds the server share
func (s *Node) Feeds() (fs []cipher.PubKey) {
	s.fmx.RLock()
	defer s.fmx.RUnlock()
	if len(s.feeds) == 0 {
		return
	}
	fs = make([]cipher.PubKey, 0, len(s.feeds))
	for f := range s.feeds {
		fs = append(fs, f)
	}
	return
}

// Quititng returns cahnnel that closed
// when the Server closed
func (s *Node) Quiting() <-chan struct{} {
	return s.done // when quit done
}

//
// request response
//

func (s *Node) addWaitingForResponse(id uint32, rc chan Msg) {
	s.rpmx.Lock()
	defer s.rpmx.Unlock()

	s.responses[id] = rc
}

func (s *Node) takeWaitingForResponse(id uint32) (rc chan Msg, ok bool) {
	s.rpmx.Lock()
	defer s.rpmx.Unlock()

	if rc, ok = s.responses[id]; ok {
		delete(s.responses, id)
	}
	return
}

func (s *Node) sendMsgAndWaitForResponse(c *gnet.Conn,
	msg Msg, timeout time.Duration) (response Msg, err error) {

	var (
		tm *time.Timer
		tc <-chan time.Time
		rc chan Msg = make(chan Msg, 1) // don't block sender
	)

	if timeout > 0 {
		tm = time.NewTimer(timeout)
		defer tm.Stop()
		tc = tm.C
	}

	s.addWaitingForResponse(msg.Id(), rc)
	defer s.takeWaitingForResponse(msg.Id())

	s.sendMessage(c, msg)

	select {
	case <-tc:
		err = ErrTimeout
	case response = <-rc:
	}
	return
}

func (s *Node) sendSubscribeMsgAndWaitForResponse(c *gnet.Conn,
	feed cipher.PubKey, timeout time.Duration) (response Msg, err error) {

	return s.sendMsgAndWaitForResponse(c, s.src.NewSubscribeMsg(feed), timeout)
}

// SubscribeResponse is similar to subscribe but it requires non-nil conenction
// and waits for reply from remote peer. It waits for response
// NodeConfig.ResponseTimeout
func (s *Node) SubscribeResponse(c *gnet.Conn, feed cipher.PubKey) error {
	return s.SubscribeResponseTimeout(c, feed, s.conf.ResponseTimeout)
}

// SubscribeResponseTimeout uses provided timeout instead of configured
func (s *Node) SubscribeResponseTimeout(c *gnet.Conn, feed cipher.PubKey,
	timeout time.Duration) (err error) {

	if c == nil {
		err = ErrNilConnection
		return
	}
	var response Msg
	response, err = s.sendSubscribeMsgAndWaitForResponse(c, feed, timeout)
	if err != nil {
		return
	}
	typ := response.MsgType()
	if typ == DenySubscriptionMsgType {
		err = ErrSubscriptionRejected
		return
	} else if typ == AcceptSubscriptionMsgType {
		return // nil
	}
	s.Debug("unexpected subscription response: ", typ.String())
	err = ErrUnexpectedResponse
	return
}
