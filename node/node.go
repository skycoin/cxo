package node

import (
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/net/factory"
	discovery "github.com/skycoin/net/skycoin-messenger/factory"

	"github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/node/msg"
	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/cxo/skyobject/registry"
	"github.com/skycoin/cxo/skyobject/statutil"
)

// A Node represents network P2P transport
// for CX objects. The node used to receive
// send and retransmit CX obejcts
type Node struct {
	mx sync.Mutex // lock

	log.Logger                       // logger
	id         *discovery.SeedConfig // unique random identifier
	c          *skyobject.Container  // related Container

	//
	// feeds and connections
	//

	fs *nodeFeeds              // feeds
	ic map[cipher.PubKey]*Conn // node id (pk) -> connection
	pc map[*Conn]struct{}      // pending connections

	//
	// transports
	//

	// listen and connect
	tcp *TCP
	udp *UDP

	//
	// other
	//

	// discovery
	discovery *discovery.MessengerFactory

	// keep config
	config             *Config
	maxFillingParallel int // copy of c.Config().MaxFillingParallel
	rollAvgSamples     int // copy of c.Config().RollAvgSamples

	//
	// stat
	//

	fillavg *statutil.Duration // filling average

	//
	// quiting and closing
	//

	quiting chan struct{} // quiting

	await  sync.WaitGroup // wait for goroutines
	closeo sync.Once      // close once
	clsoeq chan struct{}  // clsoed
}

// NewNode creates new Node instance using provided
// Config. If the Config is nil, then default is used
// (see NewConfig for defaults)
func NewNode(conf *Config) (n *Node, err error) {

	if conf == nil {
		conf = NewConfig()
	}

	if err = conf.Validate(); err != nil {
		return
	}

	// TODO

	return
}

// NewNodeContainer use provided Container.
// Configurations related to Container ignored.
func NewNodeContainer(
	conf *Config,
	c skyobject.Container,
) (
	n *Node,
	err error,
) {

	if conf == nil {
		conf = NewConfig()
	}

	if err = conf.Validate(); err != nil {
		return
	}

	// TOOD

	return
}

// ID retursn identifier of the Node. The identifier
// is unique random identifier that used by to avoid
// cross-conenctions
func (n *Node) ID() (id cipher.PubKey) {
	return n.id.PublicKey
}

// Config returns Config with which the
// Node was created
func (n *Node) Config() (conf *Config) {

	conf = new(Config)
	*conf = *n.config // copy

	return
}

// Container returns related Container instance
func (n *Node) Container() (c *skyobject.Container) {
	return n.c
}

// Discovery returns related discovery factory. That can be nil
// if this feature is turned off
func (n *Node) Discovery() (discovery *discovery.MessengerFactory) {
	return n.discovery
}

// Publish sends given Root obejct to peers that
// subscribed to feed of the Root. The Publish used
// to publish new Root obejcts
func (n *Node) Publish(r *registry.Root) {

	// TODO

}

// ConnectionsOfFeed returns list of connections of given
// feed. Use blank public key to get all connections that
// does not share a feed
func (n *Node) ConnectionsOfFeed(pk cipher.PubKey) (cs []*Conn) {
	return n.fs.connectionsOfFeed(pk)
}

// Connections returns all established connections
func (n *Node) Connections() (cs []*Conn) {

	n.mx.Lock()
	defer n.mx.Unlock()

	cs = make([]*Conn, 0, len(n.ic))

	for _, c := range n.ic {
		cs = append(cs, c)
	}

	return
}

// TCP returns TCP transport of the Node
func (n *Node) TCP() (tcp *TCP) {

	n.mx.Lock()
	defer n.mx.Unlock()

	n.createTCP()

	return n.tcp
}

// UDP returns UDP transport of the Node
func (n *Node) UDP() (tcp *UDP) {

	n.mx.Lock()
	defer n.mx.Unlock()

	n.createUDP()

	return n.udp
}

// add to pending
func (n *Node) addPendingConn(c *Conn) {
	n.mx.Lock()
	defer n.mx.Unlock()

	n.pc[c] = struct{}{}
}

// without lock
func (n *Node) addConnection(c *Conn) (err error) {

	n.mx.Lock()
	defer n.mx.Unlock()

	if _, ok := n.ic[c.peerID]; ok == true {
		return ErrAlreadyHaveConnection
	}

	n.ic[c.peerID] = c                   // add
	delete(n.pc, c)                      // remove from pending
	n.fs.addConnFeed(c, cipher.PubKey{}) // add to blank feed

	return
}

// under lock
func (n *Node) delConnection(c *Conn) {

	delete(n.ic, c.peerID)
	n.fs.delConn(c)

}

// call under lock of the mx
func (n *Node) createTCP() {

	if n.tcp != nil {
		return // already created
	}

	n.tcp = newTCP(n)

}

// call under lock of the mx
func (n *Node) createUDP() {

	if n.udp != nil {
		return // alrady created
	}

	n.udp = newUDP(n)

}

func (n *Node) onConnect(c *Conn) {

	var occ = n.config.OnConnect

	if occ == nil {
		return
	}

	var terminate error

	if terminate = occ(c); terminate != nil {
		c.close(terminate)
	}

}

func (n *Node) onDisconenct(c *Conn, reason error) {

	var odc = n.config.OnDisconnect

	if odc == nil {
		return
	}

	odc(c, reason)

}

func (n *Node) acceptConnection(fc *factory.Connection, isTCP bool) {

	var c, err = n.wrapConnection(fc, true, isTCP)

	if err != nil {

		n.Printf("[ERR] [%s] error: %v",
			connString(true, true, fc.GetRemoteAddr().String()),
			err)

	} else if n.Pins()&NewConnPin != 0 {

		n.Debug(NewConnPin, "[%s] accept", c.String())

	}

}

// callback
func (n *Node) acceptTCPConnection(fc *factory.Connection) {
	n.acceptConnection(fc, true)
}

// callback
func (n *Node) acceptUDPConnection(fc *factory.Connection) {
	n.acceptConnection(fc, false)
}

// delete from pending and close underlying
// factory.Connection
func (n *Node) delPendingConnClose(c *Conn) {

	n.mx.Lock()
	defer n.mx.Unlock()

	delete(n.pc, c)
	c.Connection.Close()

}

func (n *Node) wrapConnection(
	fc *factory.Connection, // :
	isIncoming bool, //        :
	isTCP bool, //             :
) (
	c *Conn, //                :
	err error, //              :
) {

	c = n.newConnection(fc, true, true) // adds to pending

	// handshake
	if err = c.handshake(n.quiting); err != nil {
		n.delPendingConnClose(c)
		return
	}

	// check out peer id

	if err = n.addConnection(c); err != nil {
		n.delPendingConnClose(c)
		return
	}

	c.run()
	n.onConnect(c)

	return

}

// Share given feed. By default the Node dousn't share
// a feed. Even if underlying Container have any. To
// start sharing a feed, call this method. Thus, you can
// keep some feeds locally. To share all feeds of the
// Container use following code
//
//     for _, pk := range n.Container().Feeds() {
//         n.Share(pk)
//      }
//
// The Share method adds given feed to underlying
// Container calling (*skyobjec.Container).AddFeed()
// method. The method never return an error if given
// feed is already shared. The share never associate
// the feed with a conenction. You should to call
// (*Conn).Subscribe to do that.
func (n *Node) Share(feed cipher.PubKey) (err error) {

	// add to the Container
	if err = n.c.AddFeed(feed); err != nil {
		return
	}

	n.fs.addFeed(pk)
	return
}

// DontShare given feed. The method does't remove the
// feed from underlying Container. The method calls
// Unsubscribe() for all connections that share the feed.
// The method does nothing if the Node doesn't share the
// feed The Node never sync with Container automatically,
// and before removing a feed from Container, call this
// method to prevent sharing feed that does not exist
func (n *Node) DontShare(feed cipher.PubKey) (err error) {
	n.fs.delFeed(pk)
	return
}

func (n *Node) onSubscribeRemote(c *Conn, feed cipher.PubKey) (reject error) {

	var osr = n.config.OnSubscribeRemote

	if osr == nil {
		return
	}

	return osr(c, feed)
}

func (n *Node) onUnsubscribeRemote(c *Conn, feed cipher.PubKey) {

	var ousr = n.config.OnUnsubscribeRemote

	if ousr == nil {
		return
	}

	ousr(c, feed)
}

// send to feed (call under lock)
func (n *Node) send(pk cipher.PubKey, m msg.Msg) {

	for c := range n.fc[pk] {
		c.sendMsg(c.nextSeq(), 0, m)
	}

}

// Feeds the Node share. The reply is read-only
func (n *Node) Feeds() (feeds []cipher.PubKey) {

	n.mx.Lock()
	defer n.mx.Unlock()

	if feeds = n.fl; feeds != nil {
		return
	}

	// one for cipher.PubKey{} (blank)
	feeds = make([]cipher.PubKey, 0, len(n.fc)-1)

	for pk := range n.fc {

		if pk == (cipher.PubKey{}) {
			continue
		}

		feeds = append(feeds, pk)
	}

	n.fl = feeds

	return
}

// HasFeed returns true if the Node share given feed
func (n *Node) HasFeed(feed cipher.PubKey) (ok bool) {

	if feed == (cipher.PubKey{}) {
		return
	}

	n.mx.Lock()
	defer n.mx.Unlock()

	_, ok = n.fc[feed]

	return
}

func (n *Node) onRootReceived(c *Conn, r *registry.Root) (err error) {

	var orr = n.config.OnRootReceived

	if orr == nil {
		return
	}

	return orr(c, r)

}

func (n *Node) OnFillingBreaks(c *Conn, r *registry.Root, reason error) {

	var brk = n.config.OnFillingBreaks

	if brk == nil {
		return
	}

	brk(c, r, reason)

}

func (n *Node) fill(r *registry.Root) {
	n.mx.Lock()
	defer n.mx.Unlock()

	if nf, ok = n.fc[r.Pub]; ok == false {
		return
	}

	nf.fillesr

}
