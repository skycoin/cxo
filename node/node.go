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
	gnet.DebugPrint = false //  conf.Debug
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
//     type SecondObject struct {
//     	Name string
//     	Value uint32
//     }
//
//     so.Register("FirstObject", FirstObject{})
//     so.Register("SecondObject", SecondObject{})
//
//     pub, sec := cipher.GenerateKeyPair()
//
//     // create root object using public key
//     root := so.NewRoot(pub)
//     root.Inject(FirstObject{
//     	Name:  "Old Uncle Tom Cobley",
//     	Value: 411,
//     })
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
//     so.AddRoot(root, sec)
//
//     // share the root again
//     n.Share(pub)
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
	n.Debugf("[DBG] handleShareEvent(): share %s", se.Hex())
	if _, ok := n.subs[se]; !ok {
		n.Print("[ERR] handleShareEvent(): " +
			"share root the node doesn't subscribed to")
		return // don't share
	}
	root := n.so.Root(se)
	if root == nil {
		n.Debug("[DBG] handleShareEvent(): share root the node hasn't got")
		return // hasn't got
	}
	if nw := n.newWantedOfRoot(want, root); len(nw) > 0 {
		n.Debugf("[DBG] handleShareEvent(): %d new wanted objects", len(nw))
		n.pool.BroadcastMessage(&Request{setToSlice(nw)})
	}
	n.Debug("[DBG] handleShareEvent(): " +
		"send announce about the Root the node share")
	n.pool.BroadcastMessage(&AnnounceRoot{
		[]RootHead{
			{root.Pub, root.Time},
		},
	})
}

// new connection
// 1) request roots the node subscribed to but hasn't got
// 2) request newer roots the node subscribed to and already has
func (n *Node) handleConnectEvent(ce connectEvent, want skyobject.Set) {
	n.Debug("[DBG] handleConnectEvent(): got new connection: ", ce.Address)
	if len(n.subs) == 0 {
		n.Debug("[DBG] handleConnectEvent(): no subscriptions")
		return
	}
	requests := make([]RootHead, 0, len(n.subs))
	for pub := range n.subs {
		if root := n.so.Root(pub); root != nil {
			requests = append(requests, RootHead{pub, root.Time}) // newer
			continue
		}
		requests = append(requests, RootHead{pub, 0}) // any
	}
	n.Debugf("[DBG] handleConnectEvent(): request %d root objects", len(n.subs))
	n.pool.SendMessage(ce.Address, &RequestRoot{requests})
}

func (n *Node) enqueueMsgEvent(msg gnet.Message, address string) {
	select {
	case n.msge <- msgEvent{Address: address, Msg: msg}:
	case <-n.quit:
	}
}

func (n *Node) handleMsgEvent(me msgEvent, want skyobject.Set) {
	n.Debug("[DBG] handleMsgEvent()")
	switch x := me.Msg.(type) {
	case *Announce:
		n.Debug("[DBG] handleMsgEvent() Announce ", len(x.Hashes))
		if len(x.Hashes) == 0 {
			return // avoid make
		}
		// request messages if the node want it
		requests := make([]cipher.SHA256, 0, len(x.Hashes)) // avoid relocation
		for _, hash := range x.Hashes {
			if _, ok := want[skyobject.Reference(hash)]; ok {
				requests = append(requests, hash) // want it
			}
		}
		if len(requests) == 0 {
			return
		}
		n.pool.SendMessage(me.Address, &Request{requests})
	case *Request:
		n.Debug("[DBG] handleMsgEvent() Request ", len(x.Hashes))
		if len(x.Hashes) == 0 {
			return // avoid make
		}
		// send data if the node has got it
		data := make([][]byte, 0, len(x.Hashes)) // avoid relocations
		for _, hash := range x.Hashes {
			if d, ok := n.db.Get(hash); ok {
				data = append(data, d) // got it
			}
		}
		if len(data) == 0 {
			return
		}
		n.pool.SendMessage(me.Address, &Data{data})
	case *Data:
		n.Debug("[DBG] handleMsgEvent() Data ", len(x.Data))
		if len(x.Data) == 0 {
			return // avoid make
		}
		// add the data to db if the node want it
		announces := make([]cipher.SHA256, 0, len(x.Data)) // avoid relocations
		for _, data := range x.Data {
			hash := cipher.SumSHA256(data)
			if _, ok := want[skyobject.Reference(hash)]; !ok {
				continue
			}
			n.db.Set(hash, data)
			delete(want, skyobject.Reference(hash))
			announces = append(announces, hash)
		}
		if len(announces) == 0 { // no new data given
			return
		}
		nw := n.newWanted(want)
		if len(nw) > 0 { // new wanted objects
			n.pool.BroadcastMessage(&Request{setToSlice(nw)})
		}
		n.pool.BroadcastMessage(&Announce{announces})
	case *AnnounceRoot:
		n.Debug("[DBG] handleMsgEvent() AnnounceRoot ", len(x.Roots))
		if len(x.Roots) == 0 {
			return // avoid make
		}
		// request root if the root is newer
		requests := make([]RootHead, 0, len(x.Roots))
		for _, head := range x.Roots {
			if _, ok := n.subs[head.Pub]; !ok {
				continue // don't subscribed to the feed
			}
			root := n.so.Root(head.Pub)
			if root == nil {
				continue
			}
			if root.Time >= head.Time {
				continue // already have (the same or newer)
			}
			requests = append(requests, RootHead{head.Pub, 0})
		}
		if len(requests) == 0 {
			return
		}
		n.pool.SendMessage(me.Address, &RequestRoot{requests})
	case *RequestRoot:
		n.Debug("[DBG] handleMsgEvent() RequestRoots ", len(x.Roots))
		// send roots if the node has them got
		if len(x.Roots) == 0 {
			return // avoid make
		}
		data := make([]RootBody, 0, len(x.Roots)) // avoid relocations
		for _, head := range x.Roots {
			if _, ok := n.subs[head.Pub]; !ok {
				continue
			}
			root := n.so.Root(head.Pub)
			if root == nil {
				continue
			}
			if root.Time <= head.Time {
				continue
			}
			data = append(data, RootBody{
				root.Pub,
				root.Sig,
				root.Encode(),
			})
		}
		if len(data) == 0 {
			return
		}
		n.pool.SendMessage(me.Address, &DataRoot{data})
	case *DataRoot:
		n.Debug("[DBG] handleMsgEvent() DataRoot ", len(x.Roots))
		// replace existing root with received one
		// if its possible if:
		// 1) the node subbscibed to the root
		// 2) the node can decode given root
		// 3) the given root is newer
		if len(x.Roots) == 0 {
			return // avoid make
		}
		var (
			announces = make([]RootHead, 0, len(x.Roots))
			set       = make(skyobject.Set) // new wanted
		)
		for _, dr := range x.Roots { // range over []DataRoot
			if _, ok := n.subs[dr.Pub]; !ok {
				continue
			}
			ok, err := n.so.AddEncodedRoot(dr.Root, dr.Pub, dr.Sig)
			if err != nil {
				n.Printf("[ERR] AddEncodedRoot (%s): %v", dr.Pub.Hex(), err)
				continue
			}
			if !ok {
				n.Debug("[DBG] older root")
				continue
			}
			root := n.so.Root(dr.Pub)
			announces = append(announces, RootHead{root.Pub, root.Time})
			for k := range n.newWantedOfRoot(want, root) {
				set[k] = struct{}{} // merge
			}
		}
		if len(set) > 0 {
			n.pool.BroadcastMessage(&Request{setToSlice(set)})
		}
		if len(announces) == 0 {
			return
		}
		n.pool.BroadcastMessage(&AnnounceRoot{announces})
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

func setToSlice(set skyobject.Set) (ary []cipher.SHA256) {
	if len(set) == 0 {
		return // avoid make
	}
	ary = make([]cipher.SHA256, 0, len(set))
	for k := range set {
		ary = append(ary, cipher.SHA256(k))
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
