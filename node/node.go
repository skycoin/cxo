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
)

// A NodeID represents randomly generated
// unique id that consist of cipher.PubKey
// and related cipher.SecKey. The NodeID
// used to protect nodes against cross
// connections (simultaneous connections,
// A to B and B to A). Because, a connection
// is a duplex
type NodeID struct {
	Pk cipher.PubKey
	Sk cipher.SecKey
}

func (n *NodeID) generate() {
	n.Pk, n.Sk = cipher.GenerateKeyPair()
}

// A Node represents network P2P transport
// for CX objects. The node used to receive
// send and retransmit CX obejcts
type Node struct {

	// logger
	log.Logger

	// unique random identifier of the Node
	id NodeID

	// lock
	mx sync.Mutex

	// related Container
	c *skyobject.Container

	// feeds -> []*Conn
	fc map[cipher.PubKey]map[*Conn]struct{}
	fl []cipher.PubKey // clear-on-write

	// node id -> connection
	ic map[cipher.PubKey]*Conn

	// listen and connect
	tcp *TCP
	udp *UDP

	// discovery
	discovery *discovery.MessengerFactory

	// keep config
	config *Config

	// pending connections
	pc map[*Conn]struct{}

	//
	// closing
	//

	// quiting
	quiting chan struct{}

	// clsoed
	clsoeq chan struct{}

	// wait for goroutines
	await sync.WaitGroup

	// close once
	closeo sync.Once
}

// ID retursn ID of the Node. See NodeID
// for details
func (n *Node) ID() (id NodeID) {
	return n.id
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

// Connections returns list of connections of given feed.
// Use blank public key to get all connections that not share
// a feed
func (n *Node) Connections(pk cipher.PubKey) (cf []*Conn) {

	n.mx.Lock()
	defer n.mx.Unlock()

	var cs = n.fc[pk]

	cf = make([]*Conn, 0, len(cs))

	for c := range cs {
		cf = append(cf, c)
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

	if _, ok := n.ic[c.peerID.Pk]; ok == true {
		return ErrAlreadyHaveConnection
	}

	delete(n.pc, c) // remove from pending

	n.fc[cipher.PubKey{}][c] = struct{}{}
	n.ic[c.peerID.Pk] = c

	return

}

// under lock
func (n *Node) delConnection(c *Conn) {

	delete(n.ic, c.peerID.Pk)

	// use range over the map instead of
	// c.Feeds() to avoid slice building
	for pk := range c.feeds {
		delete(n.fc[pk], c)
	}

	delete(n.fc[cipher.PubKey{}], c)

}

// under lock
func (n *Node) addConnFeed(c *Conn, pk cipher.PubKey) {

	delete(n.fc[cipher.PubKey{}], c)

	var cf, ok = n.fc[pk]

	if ok == false {
		cf = make(map[*Conn]struct{})
		n.fc[pk] = cf
	}

	cf[c] = struct{}{}

}

// under lock
func (n *Node) delConnFeed(c *Conn, pk cipher.PubKey) {

	delete(n.fc[pk], c)

	if len(c.feeds) == 0 {
		n.fc[cipher.PubKey{}][c] = struct{}{}
	}

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
		c.closeWithError(terminate)
	}

}

func (n *Node) onDisconenct(c *Conn, reason error) {

	var odc = n.config.OnDisconnect

	if odc == nil {
		return
	}

	odc(c, reason)

}

// callback
func (n *Node) acceptTCPConnection(fc *factory.Connection) {

	var c, err = n.wrapConnection(fc, true, true)

	if err != nil {

		n.Printf("[ERR] [%s] error: %v",
			connString(true, true, fc.GetRemoteAddr().String()),
			err)

	} else if n.Pins()&NewConnPin != 0 {

		n.Debug(NewConnPin, "[%s] accept", c.String())

	}

}

// callback
func (n *Node) acceptUDPConnection(fc *factory.Connection) {

	var c, err = n.wrapConnection(fc, true, false)

	if err != nil {

		n.Printf("[ERR] [%s] error: %v",
			connString(true, true, fc.GetRemoteAddr().String()),
			err)

	} else if n.Pins()&NewConnPin != 0 {

		n.Debug(NewConnPin, "[%s] accept", c.String())

	}

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

	c = n.newConnection(fc, true, true)

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

	// alrady have

	if _, ok := n.fc[pk]; ok == true {
		return
	}

	// add to the Container

	if err = n.c.AddFeed(feed); err != nil {
		return
	}

	// add

	n.fc[feed] = make(map[*Conn]struct{})
	n.fl = nil // clear

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

	n.mx.Lock()
	defer n.mx.Unlock()

	var cs, ok = n.fc[feed]

	if ok == false {
		return
	}

	for c := range cs {
		if err = c.Unsubscribe(feed); err != nil {
			return
		}
	}

	delete(n.fc, feed)
	n.fl = nil // clear

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
