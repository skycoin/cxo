package node

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

var (
	ErrClosed     = errors.New("use of closed node")
	ErrNotFound   = errors.New("not found")
	ErrNotAllowed = errors.New("not allowed")

	ErrEmptySecret = errors.New("empty secret key")

	ErrManualDisconnect gnet.DisconnectReason = errors.New(
		"manual disconnect")
	ErrMalformedMessage gnet.DisconnectReason = errors.New(
		"malformed message")
)

// A Config represents configurations of Node
type Config struct {
	gnet.Config

	// Known is a list of known addresses (public key -> list of nodes)
	Known     map[cipher.PubKey][]string
	Debug     bool          //show debug logs
	Name      string        // name of the node, that used as log prefix
	Ping      time.Duration // ping interval
	RPCEvents int           // rpc events chan

	// RemoteClose is used to deny or allow close the Node using RPC
	RemoteClose bool
}

// NewConfig creates Config filled down with default values
func NewConfig() Config {
	return Config{
		Config:    gnet.NewConfig(),
		Ping:      25 * time.Second,
		RPCEvents: 10,
	}
}

// new connection
type connectEvent struct {
	Address string
}

// new message received
type msgEvent struct {
	Address string
	Msg     gnet.Message
}

// A Node represents P2P connections pool with RPC and list of known
// hosts to connect to. It automatically fetch latest root object,
// accept connections and send/receive objects
type Node struct {
	Logger
	conf Config

	db *data.DB
	so *skyobject.Container

	pool *gnet.ConnectionPool

	rpce     chan rpcEvent     // RPC events
	connecte chan connectEvent // new connection
	msge     chan msgEvent     // new message received

	once sync.Once
	quit chan struct{}
	done chan struct{}
}

// NewNode creates Node with given config and DB. If given database is nil
// then it panics
func NewNode(conf Config, db *data.DB, so *skyobject.Container) *Node {
	if db == nil {
		panic("NewNode: given database is nil")
	}
	if so == nil {
		panic("NewNode: given container is nil")
	}
	if conf.Name == "" {
		conf.Name = "node"
	}
	// gnet debugging messages and debug messages of node
	gnet.DebugPrint = conf.Debug
	return &Node{
		Logger: NewLogger("["+conf.Name+"] ", conf.Debug),
		conf:   conf,
		db:     db,
		so:     so,
	}
}

// Start is used to launch the Node. The Start is non-blocking
func (n *Node) Start() {
	n.Debug("[DBG] starting node")
	// be sure that the map is not nil
	if n.conf.Known == nil {
		n.conf.Known = make(map[cipher.PubKey][]string)
	}
	n.once = sync.Once{} // refresh once
	n.quit, n.done = make(chan struct{}), make(chan struct{})
	n.connecte = make(chan connectEvent, n.conf.MaxConnections)
	n.msge = make(chan msgEvent, n.conf.BroadcastResultSize)
	n.conf.ConnectCallback = n.onConnect
	n.pool = gnet.NewConnectionPool(n.conf.Config, n)
	n.pool.Run()
	var addr net.Addr
	var err error
	if addr, err = n.pool.ListeningAddress(); err != nil {
		n.Panic("[CRITICAL] can't obtain lisening address: ", err)
		return // never happens
	} else {
		n.Print("[INF] listening on ", addr)
	}
	n.rpce = make(chan rpcEvent, n.conf.RPCEvents)
	go n.handle(n.quit, n.done, n.connecte, n.msge, n.rpce)
	return
}

// Share sends root object to all connected nodes. The method is used to
// publish new root object manually. If there is no root object, it does
// nothing. A call of the Share is useful only if you update the root
// using skyobject API and want to publish it. For example
//
//     db := data.NewDB()
//     so := skyobject.NewContainer(db)
//
//     conf := node.NewConfig()
//     conf.Name = "example node"
//     conf.Debug = true
//
//     n := node.NewNode(conf, db, so)
//
//     n.Start()
//     defer n.Close()
//
//     // example root object
//
//     type Root struct {
//     	Name  string
//     	Value int64
//     }
//
//     pub, sec := cipher.GenerateKeyPair()
//
//     root := so.NewRoot(pub)
//     root.Set(Root{
//     	Name:  "Old Uncle Tom Cobley",
//     	Value: 411,
//     })
//     root.Touch()
//     so.AddRoot(root, sec)
//     n.Share(pub)
//
//     //
//     // stuff
//     //
//
//     return
//
func (n *Node) Share(pub cipher.PubKey) {
	n.enqueueRpcEvent(func() {
		if root := n.so.Root(pub); root != nil {
			n.Debug("[DBG] broadcast root object: ", root.Pub.Hex())
			r := &Root{
				Pub:  root.Pub,
				Sig:  root.Sig,
				Root: root.Encode(),
			}
			n.pool.BroadcastMessage(r)
		}
	})
}

// gnet callback
func (n *Node) onConnect(address string, outgoing bool) {
	n.Debug("[DBG] got new connection ", address)
	select {
	case n.connecte <- connectEvent{address}:
	case <-n.quit:
	}
}

// handling loop
func (n *Node) handle(quit, done chan struct{},
	connecte chan connectEvent, msge chan msgEvent, rpce chan rpcEvent) {
	n.Debug("[DBG] start handling events")

	var (
		pingsTicker *time.Ticker
		pings       <-chan time.Time
	)

	if n.conf.Ping > 0 {
		pingsTicker = time.NewTicker(n.conf.Ping / 2)
		defer pingsTicker.Stop()
		pings = pingsTicker.C
	}

	defer close(done)
	defer n.pool.Shutdown()

	// subscriptions and objects we want from them
	var subs map[cipher.PubKey]struct{} = make(map[cipher.PubKey]struct{})
	var want skyobject.Set = make(skyobject.Set)

	//

	for {
		select {
		case sr := <-n.pool.SendResults:
			if sr.Error != nil {
				n.Printf("[ERR] error sending %T to %s: %v",
					sr.Message,
					sr.Addr,
					sr.Error)
			}
		case ce := <-connecte:
			n.handleConnectEvent(ce, subs, want)
		case me := <-msge:
			n.handleMsgEvent(me, subs, &want)
		case rpce := <-rpce:
			n.handleRpcEvt(rpce, subs)
		case <-quit:
			return
		case <-pings:
			n.pool.SendPings(n.conf.Ping/2, &Ping{})
		}
	}
}

func (n *Node) handleRpcEvt(evt rpcEvent, subs map[cipher.PubKey]struct{}) {
	switch x := evt.(type) {
	case func():
		x()
	case func(map[cipher.PubKey]struct{}):
		x(subs)
	default:
		n.Panicf("[CRITICAL] Invalid type of rpc event %T", evt)
	}
}

func (n *Node) handleConnectEvent(ce connectEvent,
	subs map[cipher.PubKey]struct{},
	want skyobject.Set) {
	// send all root objects we have and request all we want
	for pub := range subs {
		if root := n.so.Root(pub); root != nil {
			n.Debugf("[DBG] send root %s object to %s",
				pub.Hex(), ce.Address)
			n.pool.SendMessage(ce.Address, &Root{
				Sig:  root.Sig,
				Pub:  root.Pub,
				Root: root.Encode(),
			})
		}
	}
	// request wanted objects of the root
	for k := range want {
		n.pool.SendMessage(ce.Address, &Request{cipher.SHA256(k)})
	}
}

func (n *Node) enqueueMsgEvent(msg gnet.Message, address string) {
	select {
	case n.msge <- msgEvent{Address: address, Msg: msg}:
	case <-n.quit:
	}
}

func (n *Node) handleMsgEvent(me msgEvent,
	subs map[cipher.PubKey]struct{},
	want *skyobject.Set) {
	switch x := me.Msg.(type) {
	case *Announce:
		if len(*want) == 0 {
			return // we don't want anything
		}
		if _, ok := (*want)[skyobject.Reference(x.Hash)]; ok { // do we want it?
			n.pool.SendMessage(me.Address, &Request{x.Hash}) // requst it
		}
	case *Request:
		if data, ok := n.db.Get(x.Hash); ok {
			n.pool.SendMessage(me.Address, &Data{data})
		}
	case *Data:
		if len(*want) == 0 {
			return // we don't want anything
		}
		hash := cipher.SumSHA256(x.Data)
		if _, ok := (*want)[skyobject.Reference(hash)]; !ok { // do we want it?
			return
		}
		n.db.Set(hash, x.Data)
		// send announce
		n.pool.BroadcastMessage(&Announce{hash})
		// refresh list of wanted objects and request them
		nw := n.wanted(subs)
		for k := range addRemove(*want, nw) {
			n.pool.BroadcastMessage(&Request{cipher.SHA256(k)})
		}
	case *Root:
		n.Debug("[DBG] got Root meesage from: ", me.Address)
		// add address of the subscription to known
		n.addToKnown(x.Pub, me.Address)
		// do we subscribed to the feed?
		if _, ok := subs[x.Pub]; !ok {
			return // we don't subscribed to the feed
		}
		ok, terminate := n.so.SetEncodedRoot(x.Root, x.Pub, x.Sig)
		if terminate != nil {
			n.Print("[ERR] decoding root: ", terminate)
			// terminate connection that sends malformed messages
			n.pool.Disconnect(me.Address, ErrMalformedMessage)
			return
		} else if !ok { // older or the same
			n.Debug("[DBG] received root is older or the same we already have")
			return
		}
		// broadcast the root
		n.pool.BroadcastMessage(x)
		// refresh and request list of wanted objects
		ow := *want                                   // keep old wanted objects
		*want = n.wanted(subs)                        // refresh list of wanted
		for k := range wantButNotRequest(*want, ow) { // request new objs. only
			// broadcast request to connections of the feed
			n.broadcast(x.Pub, &Request{cipher.SHA256(k)})
		}
	}
}

func (n *Node) addToKnown(pub cipher.PubKey, address string) {
	list := n.conf.Known[pub]
	for _, address := range list {
		if address == address {
			return // already exists
		}
	}
	n.conf.Known[pub] = append(list, address) // add to known
}

func (n *Node) broadcast(pub cipher.PubKey, msg gnet.Message) {
	for _, address := range n.conf.Known[pub] {
		if n.pool.IsConnExist(address) {
			n.pool.SendMessage(address, msg)
		}
	}
}

// get single list of all wanted objects of subscriptions
func (n *Node) wanted(subs map[cipher.PubKey]struct{}) (set skyobject.Set) {
	set = make(skyobject.Set)
	for pub := range subs {
		if root := n.so.Root(pub); root != nil {
			if rs, err := root.Want(); err != nil {
				n.Print("[ERR] error requesting wanted objects: ", err)
			} else {
				for k := range rs {
					set[k] = struct{}{}
				}
			}
		}
	}
	return
}

func (n *Node) close() {
	n.once.Do(func() {
		n.Debug("[DBG] closing node...")
		close(n.quit)
	})
}

// Close is used to shutdown the Node. It's safe to call
// the Close many times
func (n *Node) Close() {
	n.close()
	<-n.done
	n.Debug("[DBG] node was closed")
}

// Quiting is used to detect when the node going down.
// This is useful for terminating node using RPC, when a
// node doesn't wait for SIGINT. For example
//
//     n := NewNode(blah, blahBlah)
//     if err := n.Start(); err != nil {
//         // handle error
//     }
//     defer n.Close()
//
//
//     // catch SIGINT for system administation
//     // catch n.Quiting for remote shutting down
//
//     sig := make(chan os.Signal, 1)
//     singal.Notify(sig, os.Interrupt)
//     select {
//     case <-sig:
//     case <-n.Quiting():
//     }
//
//     // shutdown
//
func (n *Node) Quiting() <-chan struct{} {
	return n.quit
}

// addRemove removes objects that is in the want from the nw,
// and appends objects that is in the nw only to want
func addRemove(want, nw skyobject.Set) skyobject.Set {
	for k := range nw {
		if _, ok := want[k]; ok {
			delete(nw, k) // delete objets we already requested
		} else {
			want[k] = struct{}{} // append new objects to list of wanted
		}
	}
	return nw
}

// want - new, fresh list of wanted objects based on new freshly received root
// ow   - objects we already requested
// x    - objects we want, but don't request yet
func wantButNotRequest(want, ow skyobject.Set) skyobject.Set {
	x := make(skyobject.Set)
	for k := range want {
		if _, ok := ow[k]; !ok {
			x[k] = struct{}{}
		}
	}
	return x
}
