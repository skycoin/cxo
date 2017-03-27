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

	// Known is a list of known addresses (public key -> addresses)
	Known     map[cipher.PubKey][]string
	Debug     bool            //show debug logs
	Name      string          // name of the node, that used as log prefix
	Ping      time.Duration   // ping interval
	RPCEvents int             // rpc events chan
	Subscribe []cipher.PubKey // subscribe to on launch

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

	rpce     chan rpcEvent      // RPC events
	connecte chan connectEvent  // new connection
	msge     chan msgEvent      // new message received
	share    chan cipher.PubKey // share updated root

	subs map[cipher.PubKey]struct{} // subscriptions

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
		subs:   make(map[cipher.PubKey]struct{}),
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
	n.share = make(chan cipher.PubKey, 10)
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
	go n.handle(n.quit, n.done, n.connecte, n.msge, n.rpce, n.share)
	go n.subscribe() // subscribe to n.conf.Subscribe
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
//     type FirstObject struct {
//     	Name  string
//     	Value int64
//     }
//
//     type SecontObject struct {
//     	Name string
//     	Value uint32
//     }
//
//     pub, sec := cipher.GenerateKeyPair()
//
//     // create root object using public key
//     root := so.NewRoot(pub)
//     root.Inject(FirstObject{
//     	Name:  "Old Uncle Tom Cobley",
//     	Value: 411,
//     })
//     root.Touch()
//     so.AddRoot(root, sec)
//
//     // share the root
//     n.Share(pub)
//
//     //
//     // stuff
//     //
//
//     // get the root from container by public key
//     root := so.Root(pub)
//     root.Inject(SecondObject{
//     	Name: "Billy Kid",
//     	Value: 16,
//     })
//     root.Touch()
//     so.AddRoot(root, sec)
//
//     return
//
func (n *Node) Share(pub cipher.PubKey) {
	select {
	case n.share <- pub:
	case <-n.quit:
	}
}

func (n *Node) subscribe() {
	for _, pub := range n.conf.Subscribe {
		n.Subscribe(pub)
	}
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
	connecte chan connectEvent, msge chan msgEvent, rpce chan rpcEvent,
	share chan cipher.PubKey) {

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

	var want skyobject.Set = make(skyobject.Set)

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
			n.handleConnectEvent(ce, want)
		case me := <-msge:
			n.handleMsgEvent(me, want)
		case se := <-share:
			n.handleShareEvent(se, want)
		case rpce := <-rpce:
			rpce()
		case <-quit:
			return
		case <-pings:
			n.pool.SendPings(n.conf.Ping/2, &Ping{})
		}
	}

}

func (n *Node) handleShareEvent(se cipher.PubKey, want skyobject.Set) {
	if _, ok := n.subs[se]; !ok {
		return // don't share
	}
	root := n.so.Root(se)
	if root == nil {
		return // haven't got
	}
	for k := range n.newWantedOfRoot(want, root) {
		n.pool.BroadcastMessage(&Request{cipher.SHA256(k)})
	}
	n.pool.BroadcastMessage(&AnnounceRoot{
		root.Pub,
		root.Time,
	})
}

// new connection
func (n *Node) handleConnectEvent(ce connectEvent, want skyobject.Set) {
	for pub := range n.subs {
		root := n.so.Root(pub)
		if root == nil {
			n.pool.SendMessage(ce.Address, &RequestRoot{Pub: pub}) // any
		} else {
			n.pool.SendMessage(ce.Address, &RequestRoot{
				Pub:  pub,
				Time: root.Time, // newer
			})
		}
	}
}

func (n *Node) enqueueMsgEvent(msg gnet.Message, address string) {
	select {
	case n.msge <- msgEvent{Address: address, Msg: msg}:
	case <-n.quit:
	}
}

func (n *Node) handleMsgEvent(me msgEvent, want skyobject.Set) {
	switch x := me.Msg.(type) {
	case *Announce:
		if _, ok := want[skyobject.Reference(x.Hash)]; ok {
			n.pool.SendMessage(me.Address, &Request{x.Hash})
		}
	case *Request:
		if data, ok := n.db.Get(x.Hash); ok {
			n.pool.SendMessage(me.Address, &Data{data})
		}
	case *Data:
		hash := cipher.SumSHA256(x.Data)
		if _, ok := want[skyobject.Reference(hash)]; ok {
			n.db.Set(hash, x.Data)
			delete(want, skyobject.Reference(hash))
			for k := range n.newWanted(want) {
				n.pool.BroadcastMessage(&Request{cipher.SHA256(k)})
			}
			n.pool.BroadcastMessage(&Announce{hash})
		}
	case *AnnounceRoot:
		if _, ok := n.subs[x.Pub]; !ok {
			return
		}
		root := n.so.Root(x.Pub)
		if root != nil {
			if root.Time >= x.Time {
				return // already have
			}
		}
		n.pool.SendMessage(me.Address, &RequestRoot{x.Pub, 0})
	case *RequestRoot:
		if _, ok := n.subs[x.Pub]; !ok {
			return
		}
		root := n.so.Root(x.Pub)
		if root != nil {
			if root.Time > x.Time {
				n.pool.SendMessage(me.Address, &DataRoot{
					root.Pub,
					root.Sig,
					root.Encode(),
				})
			}
		}
	case *DataRoot:
		if _, ok := n.subs[x.Pub]; !ok {
			return
		}
		ok, err := n.so.AddEncodedRoot(x.Root, x.Pub, x.Sig)
		if err != nil {
			n.Print("[ERR] AddEncodedRoot: ", err)
			return
		}
		if !ok {
			n.Debug("[DBG] older root")
			return
		}
		root := n.so.Root(x.Pub)
		for k := range n.newWantedOfRoot(want, root) {
			n.pool.BroadcastMessage(&Request{cipher.SHA256(k)})
		}
		n.pool.BroadcastMessage(&AnnounceRoot{
			root.Pub,
			root.Time,
		})
	}
}

func (n *Node) newWantedOfRoot(want skyobject.Set,
	root *skyobject.Root) (nwr skyobject.Set) {

	nwr = make(skyobject.Set)
	set, _ := root.Want()
	for k := range set {
		if _, ok := want[k]; !ok {
			want[k] = struct{}{}
			nwr[k] = struct{}{}
		}
	}
	return
}

// nww containe only new objects that doesn't requested yet
// and want map updated
func (n *Node) newWanted(want skyobject.Set) (nww skyobject.Set) {
	nww = make(skyobject.Set)
	for pub := range n.subs {
		root := n.so.Root(pub)
		if root == nil {
			continue
		}
		set, _ := root.Want()
		for k := range set {
			if _, ok := want[k]; !ok {
				want[k] = struct{}{}
				nww[k] = struct{}{}
			}
		}
	}
	return
}

func mergeRootWanted(root *skyobject.Root, want skyobject.Set) {
	rw, _ := root.Want()
	for k := range rw {
		want[k] = struct{}{}
	}
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
