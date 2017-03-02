package node

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"

	"github.com/skycoin/cxo/data"
)

// A Config represents configurations of Node
type Config struct {
	gnet.Config

	// Known is a list of known addresses to subscribe to
	Known []string
	Debug bool
	Name  string
	Ping  time.Duration
}

type Node struct {
	Logger
	conf Config
	db   *data.DB
	pool *gnet.ConnectionPool
	quit chan struct{}
	done chan struct{}
}

func NewNode(conf Config, db *data.DB) *Node {
	if db == nil {
		panic("NewNode has got nil database")
	}
	if conf.Name == "" {
		conf.Name = "node"
	}
	return &Node{
		Logger: NewLogger("["+conf.Name+"] ", conf.Debug),
		conf:   conf,
		db:     db,
	}
}

func (n *Node) Start() (err error) {
	n.Debug("[DBG] starting node")
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
	go n.pool.AcceptConnections()
	go n.handle(n.quit)
}

func (n *Node) sendEverythingWeHave(gc *gnet.Connection) {
	n.Debug("[DBG] send everything we have to ", gc.Addr())
	for k := range n.db {
		gc.ConnectionPool.SendMessage(gc, Announce{k})
	}
}

func (n *Node) handle(quit, done chan struct{}) {
	var (
		tk   time.Ticker
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
			n.pool.HandleDisconnectEvent(de)
		case <-ping:
			n.pool.BroadcastMessage(&Ping{})
		case <-quit:
			return
		default:
			n.pool.HandleMessages()
		}
	}
}

func (n *Node) Close() {
	close(n.quit)
	<-n.done
}
