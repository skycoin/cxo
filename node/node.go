package node

import (
	"errors"
	"io"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/skyobject"
)

var (
	ErrClosed     = errors.New("use of closed node")
	ErrNotFound   = errors.New("not found")
	ErrNotAllowed = errors.New("not allowed")

	ErrEmptySecret = errors.New("empty secret key")

	ErrManualDisconnect = errors.New("manual disconnect")
	ErrMalformedMessage = errors.New("malformed message")
)

//
// TODO: use logger instad of configs for node and node/gnet
//

// A Config represents configurations of Node
type Config struct {
	gnet.Config

	// Known is a list of known addresses (public key -> addresses)
	Known map[cipher.PubKey][]string
	Debug bool   //show debug logs
	Name  string // name of the node, that used as log prefix

	Listen string // listening address

	RPCEvents int // rpc events chan

	Subscribe []cipher.PubKey // subscribe to on launch

	// RemoteClose is used to deny or allow close the Node using RPC
	RemoteClose bool

	// Out for ligs. If the Out is nil then default (os.Stderr) used
	Out io.Writer
}

// NewConfig creates Config filled down with default values
func NewConfig() Config {
	return Config{
		Config:    gnet.NewConfig("", nil),
		RPCEvents: 10,
	}
}

// A Node represents P2P connections pool with RPC and list of known
// hosts to connect to. It automatically fetch latest root object,
// accept connections and send/receive objects
type Node struct {
	log.Logger
	conf Config

	db *data.DB
	so *skyobject.Container

	pool *gnet.Pool

	rpce     chan rpcEvent      // RPC events
	connecte chan *gnet.Conn    // new connection
	share    chan cipher.PubKey // share updated root

	subs map[cipher.PubKey]struct{} // subscriptions

	once sync.Once
	quit chan struct{}
	done chan struct{}
}

// NewNode creates Node with given config and DB. If given database is nil
// then it panics
func NewNode(conf Config, db *data.DB, so *skyobject.Container) (n *Node) {
	if db == nil {
		panic("NewNode: given database is nil")
	}
	if so == nil {
		panic("NewNode: given container is nil")
	}
	if conf.Name == "" {
		conf.Name = "node"
	}
	conf.Config.Debug = conf.Debug
	conf.Config.Name = "p:" + conf.Name
	n = &Node{
		Logger: log.NewLogger("["+conf.Name+"] ", conf.Debug),
		conf:   conf,
		db:     db,
		so:     so,
		subs:   make(map[cipher.PubKey]struct{}),
	}
	if conf.Out != nil {
		n.SetOutput(conf.Out)
	}
	return
}

// Start is used to launch the Node. The Start is non-blocking
func (n *Node) Start() {
	n.Debug("starting node")
	// be sure that the map is not nil
	if n.conf.Known == nil {
		n.conf.Known = make(map[cipher.PubKey][]string)
	}
	n.once = sync.Once{} // refresh once
	n.quit, n.done = make(chan struct{}), make(chan struct{})
	n.connecte = make(chan *gnet.Conn, n.conf.MaxConnections)
	n.share = make(chan cipher.PubKey, 10)
	n.conf.ConnectionHandler = n.onConnect
	n.pool = gnet.NewPool(n.conf.Config)
	registerMessages(n.pool)
	if err := n.pool.Listen(n.conf.Listen); err != nil {
		n.Panic(err)
	}
	var address string
	if address = n.pool.Address(); address == "" {
		n.Panic("[CRITICAL] can't obtain lisening address")
	} else {
		n.Print("[INF] listening on ", address)
	}
	n.rpce = make(chan rpcEvent, n.conf.RPCEvents)
	go n.handle(n.quit, n.done, n.connecte, n.pool.Receive(), n.rpce, n.share)
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
// The call of the Share is non-blocking. This
// way it's safe to call it from main thread
func (n *Node) Share(pub cipher.PubKey) {
	// first, try to send without creating goroutine
	select {
	case n.share <- pub:
		return
	case <-n.quit:
		return
	default:
	}
	// second, create a goroutine and try to
	// send from the goroutine asynchronously
	go func() {
		select {
		case n.share <- pub:
		case <-n.quit:
		}
	}()
}

func (n *Node) subscribe() {
	for _, pub := range n.conf.Subscribe {
		n.Subscribe(pub)
	}
}

// gnet callback
func (n *Node) onConnect(c *gnet.Conn) {
	n.Debug("got new connection ", c.Addr())
	select {
	case n.connecte <- c:
	case <-n.quit:
	}
}

// handling loop
func (n *Node) handle(quit, done chan struct{},
	connecte chan *gnet.Conn, msge <-chan gnet.Message, rpce chan rpcEvent,
	share chan cipher.PubKey) {

	n.Debug("start handling events")

	defer close(done)
	defer n.pool.Close()

	var want skyobject.Set = make(skyobject.Set)

	for {
		select {
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
		}
	}

}

func (n *Node) handleShareEvent(se cipher.PubKey, want skyobject.Set) {
	if _, ok := n.subs[se]; !ok {
		n.Print("[ERR] share root the node doesn't subscribed to")
		return // don't share
	}
	root := n.so.Root(se)
	if root == nil {
		return // hasn't got
	}
	for k := range n.newWantedOfRoot(want, root) {
		n.pool.Broadcast(&Request{cipher.SHA256(k)})
	}
	n.pool.Broadcast(&AnnounceRoot{root.Pub, root.Time})
}

// new connection
// 1) request roots the node subscribed to but hasn't got
// 2) request newer roots the node subscribed to and already has
func (n *Node) handleConnectEvent(ce *gnet.Conn, want skyobject.Set) {
	n.Debug("handle new connection: ", ce.Addr())
	if len(n.subs) == 0 {
		return
	}
	for pub := range n.subs {
		if root := n.so.Root(pub); root != nil {
			ce.Send(&RequestRoot{pub, root.Time})
			continue // request newer
		}
		ce.Send(&RequestRoot{pub, 0}) // request any
	}
}

func (n *Node) handleMsgEvent(me gnet.Message, want skyobject.Set) {
	n.Debugf("handle message: %T, from %s", me.Value, me.Conn.Addr())
	switch x := me.Value.(type) {
	case *Announce:
		if _, ok := want[skyobject.Reference(x.Hash)]; ok {
			me.Conn.Send(&Request{x.Hash})
		}
	case *Request:
		if data, ok := n.db.Get(x.Hash); ok {
			me.Conn.Send(&Data{data})
		}
	case *Data:
		hash := cipher.SumSHA256(x.Data)
		if _, ok := want[skyobject.Reference(hash)]; !ok {
			return
		}
		n.db.Set(hash, x.Data)
		delete(want, skyobject.Reference(hash))
		for k := range n.newWanted(want) {
			me.Conn.Send(&Request{cipher.SHA256(k)})
		}
		me.Conn.Broadcast(&Announce{hash}) // broadcast except
	case *AnnounceRoot:
		if _, ok := n.subs[x.Pub]; !ok {
			return // don't subscribed to the feed
		}
		if root := n.so.Root(x.Pub); root != nil && root.Time >= x.Time {
			return // already have (the same or newer)
		}
		me.Conn.Send(&RequestRoot{x.Pub, 0})
	case *RequestRoot:
		if _, ok := n.subs[x.Pub]; !ok {
			return
		}
		root := n.so.Root(x.Pub)
		if root == nil {
			return
		}
		if root.Time <= x.Time {
			return
		}
		me.Conn.Send(&DataRoot{
			root.Pub,
			root.Sig,
			root.Encode(),
		})
	case *DataRoot:
		if _, ok := n.subs[x.Pub]; !ok {
			return
		}
		ok, err := n.so.AddEncodedRoot(x.Root, x.Pub, x.Sig)
		if err != nil {
			n.Printf("[ERR] AddEncodedRoot (%s): %v", x.Pub.Hex(), err)
			return
		}
		if !ok {
			n.Debug("older root")
			return
		}
		root := n.so.Root(x.Pub)
		for k := range n.newWantedOfRoot(want, root) {
			me.Conn.Send(&Request{cipher.SHA256(k)})
		}
		// broadcast except
		me.Conn.Broadcast(&AnnounceRoot{root.Pub, root.Time})
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

func (n *Node) close() {
	n.once.Do(func() {
		n.Debug("closing node...")
		close(n.quit)
	})
}

// Close is used to shutdown the Node. It's safe to call
// the Close many times
func (n *Node) Close() {
	n.close()
	<-n.done
	n.Debug("node was closed")
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
