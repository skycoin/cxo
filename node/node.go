package node

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"

	"github.com/skycoin/cxo/data"
)

var (
	ErrClosed     = errors.New("use of closed node")
	ErrNotFound   = errors.New("not found")
	ErrNotAllowed = errors.New("not allowed")

	ErrManualDisconnect gnet.DisconnectReason = errors.New(
		"manual disconnect")
)

// A Config represents configurations of Node
type Config struct {
	gnet.Config

	// Known is a list of known addresses to subscribe to
	Known  []string
	Debug  bool          //show debug logs
	Name   string        // name of the node, that used as log prefix
	Ping   time.Duration // ping interval
	Events int           // rpc events chan

	// RemoteClose is used to deny or allow close the Node using RPC
	RemoteClose bool
}

// NewConfig creates Config filled down with default values
func NewConfig() Config {
	return Config{
		Config: gnet.NewConfig(),
		Ping:   25 * time.Second,
		Events: 10,
	}
}

// A Node represents P2P connections pool with RPC and list of known
// hosts to connect to. It automatically fetch latest root object,
// accept connections and send/receive objects
type Node struct {
	Logger
	conf Config

	db *data.DB

	pool *gnet.ConnectionPool

	rpce chan Event

	once sync.Once
	quit chan struct{}
	done chan struct{}
}

// NewNode creates Node with given config and DB. I given database is nil
// then it panics
func NewNode(conf Config, db *data.DB) *Node {
	if db == nil {
		panic("NewNode has got nil database")
	}
	if conf.Name == "" {
		conf.Name = "node"
	}
	if conf.Debug == false {
		gnet.DebugPrint = false
	}
	return &Node{
		Logger: NewLogger("["+conf.Name+"] ", conf.Debug),
		conf:   conf,
		db:     db,
	}
}

// Start is used to launch the Node. The Start is non-blocking
func (n *Node) Start() (err error) {
	n.Debug("[DBG] starting node")
	n.once = sync.Once{} // refresh once
	n.quit, n.done = make(chan struct{}), make(chan struct{})
	n.conf.ConnectCallback = func(gc *gnet.Connection, outgoing bool) {
		n.Debug("[DBG] got new connection ", gc.Addr())
		if !outgoing {
			n.sendEverythingWeHave(gc)
		}
	}
	n.pool = gnet.NewConnectionPool(n.conf.Config, n)
	if err = n.pool.StartListen(); err != nil {
		return
	}
	var addr net.Addr
	if addr, err = n.pool.ListeningAddress(); err != nil {
		n.Panic("[CRITICAL] can't obtain lisening address: ", err)
		return // never happens
	} else {
		n.Print("[INF] listening on ", addr)
	}
	n.rpce = make(chan Event, n.conf.Events)
	go n.pool.AcceptConnections()
	go n.handle(n.quit, n.done)
	go n.connectToKnown(n.quit)
	return
}

// sned all hashes from db we have to given connection
func (n *Node) sendEverythingWeHave(gc *gnet.Connection) {
	n.Debug("[DBG] send everything we have to ", gc.Addr())
	n.db.Range(func(k cipher.SHA256) {
		gc.ConnectionPool.SendMessage(gc, &Announce{k})
	})
}

// handling loop
func (n *Node) handle(quit, done chan struct{}) {
	n.Debug("[DBG] start handling events")
	var (
		tk   *time.Ticker
		ping <-chan time.Time
	)
	if n.conf.Ping > 0 {
		tk = time.NewTicker(n.conf.Ping)
		ping = tk.C
		defer tk.Stop()
	}
	defer close(done)
	defer n.pool.StopListen()
	for {
		select {
		case sr := <-n.pool.SendResults:
			if sr.Error != nil {
				n.Printf("[ERR] error sending %T to %s: %v",
					sr.Message,
					sr.Connection.Addr(),
					sr.Error)
			}
		case de := <-n.pool.DisconnectQueue:
			n.Debug("[DBG] disconnet %s because: ",
				n.pool.Pool[de.ConnId].Addr(),
				de.Reason)
			n.pool.HandleDisconnectEvent(de)
		case <-ping:
			n.Debug("[DBG] send pings")
			n.pool.BroadcastMessage(&Ping{})
		case rpce := <-n.rpce:
			rpce()
		case <-quit:
			return
		default:
			n.pool.HandleMessages()
		}
	}
}

// connect to list of known hosts
func (n *Node) connectToKnown(quit chan struct{}) {
	n.Debug("[DBG] connecting to know hosts")
	for _, a := range n.conf.Known {
		n.Debug("[DBG] connecting to ", a)
		if _, err := n.pool.Connect(a); err != nil {
			n.Printf("[ERR] error connecting to known host %s: %v", a, err)
		}
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
