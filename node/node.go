package node

import (
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/net/factory"
	discovery "github.com/skycoin/net/skycoin-messenger/factory"

	"github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/cxo/skyobject/registry"
	"github.com/skycoin/cxo/skyobject/statutil"
)

// A Node represents network P2P transport
// for CX objects. The node used to receive,
// send and retransmit CX objects. The Node
// cares about last Root object of active
// head (see skyobject.Index.ActiveHead) of
// a feed. E.g. the Node never replicates an
// old Root if there is a newer one. The Node
// uses TCP and UDP transports.
type Node struct {
	mx sync.Mutex // lock

	log.Logger                       // logger
	id         *discovery.SeedConfig // unique random identifier
	c          *skyobject.Container  // related Container

	idpk cipher.PubKey // id.PublicKey (string -> pk)

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

	config             *Config // keep config
	maxFillingParallel int     // copy of c.Config().MaxFillingParallel
	rollAvgSamples     int     // copy of c.Config().RollAvgSamples

	//
	// stat
	//

	fillavg *statutil.Duration // filling average

	//
	// rpc
	//

	rpc *rpcServer

	//
	//  closing
	//

	await  sync.WaitGroup // wait for goroutines
	closeo sync.Once      // close once
	closeq chan struct{}  // closed
}

// NewNode creates new Node instance using provided
// Config. If the Config is nil, then default is used
// (see NewConfig for defaults)
func NewNode(conf *Config) (n *Node, err error) {

	if conf == nil {
		conf = NewConfig() // use defaults
	}

	if conf.Config == nil {
		conf.Config = skyobject.NewConfig() // use defaults
	}

	if err = conf.Validate(); err != nil {
		return // invalid
	}

	var c *skyobject.Container

	if c, err = skyobject.NewContainer(conf.Config); err != nil {
		return
	}

	return NewNodeContainer(conf, c)
}

// NewNodeContainer use provided Container.
// Configurations related to Container ignored.
func NewNodeContainer(
	conf *Config,
	c *skyobject.Container,
) (
	n *Node,
	err error,
) {

	if conf == nil {
		conf = NewConfig() // use defaults
	}

	if err = conf.Validate(); err != nil {
		return // invalid
	}

	n = new(Node)

	n.id = discovery.NewSeedConfig()
	n.idpk, _ = cipher.PubKeyFromHex(n.id.PublicKey)
	n.c = c
	n.fs = newNodeFeeds(n)
	n.ic = make(map[cipher.PubKey]*Conn)
	n.pc = make(map[*Conn]struct{})

	n.config = conf
	n.config.Config = c.Config() // actual

	n.fillavg = statutil.NewDuration(conf.Config.RollAvgSamples)
	n.closeq = make(chan struct{})

	//
	// create
	//

	// logger

	n.Logger = log.NewLogger(conf.Logger) // logger

	// listen

	if conf.TCP.Listen != "" {
		if err = n.TCP().Listen(conf.TCP.Listen); err != nil {
			n.Close()
			return
		}
	}

	if conf.UDP.Listen != "" {
		if err = n.UDP().Listen(conf.UDP.Listen); err != nil {
			n.Close()
			return
		}
	}

	// rpc

	if conf.RPC != "" {

		n.rpc = n.newRPC()

		if err = n.rpc.Listen(conf.RPC); err != nil {
			n.Close()
			return
		}

	}

	// discoveries

	for _, address := range conf.TCP.Discovery {
		n.TCP().ConnectToDiscoveryServer(address)
	}

	for _, address := range conf.UDP.Discovery {
		n.UDP().ConnectToDiscoveryServer(address)
	}

	// TODO (kostyarin): pings (move to connection)

	return
}

// ID retursn identifier of the Node. The identifier
// is unique random identifier that used to avoid
// cross-connections
func (n *Node) ID() (id cipher.PubKey) {
	return n.idpk
}

// Config returns Config with which the
// Node was created. The Config must not
// be modified. If the Node created using
// NewNodeContainer, then Config field
// replaced with config of given Container
func (n *Node) Config() (conf *Config) {
	return n.config // copy
}

// Container returns related Container instance
func (n *Node) Container() (c *skyobject.Container) {
	return n.c
}

// Publish sends given Root object to peers that
// subscribed to feed of the Root. The Publish used
// to publish new Root objects. E.g. the Node sends
// last Root object of a feed to subsctibers. But
// the Node knows nothing about new Root objects.
// And to share an updated Root, call the Publish.
// And don't call the publish for Root objects that
// alredy saved (that saved before subscription)
func (n *Node) Publish(r *registry.Root) {
	n.fs.broadcastRoot(connRoot{nil, r})
}

// ConnectionsOfFeed returns list of connections of given
// feed. Use blank public key to get all connections that
// does not share a feed
func (n *Node) ConnectionsOfFeed(feed cipher.PubKey) (cs []*Conn) {
	return n.fs.connectionsOfFeed(feed)
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

// don't create TCP in background
// returning nil, if the TCP doesn't
// exist
func (n *Node) getTCP() (t *TCP) {
	n.mx.Lock()
	defer n.mx.Unlock()

	return n.tcp
}

// TCP returns TCP transport of the Node
func (n *Node) TCP() (tcp *TCP) {

	n.mx.Lock()
	defer n.mx.Unlock()

	n.createTCP()

	return n.tcp
}

// don't create TCP in background
// returning nil, if the TCP doesn't
// exist
func (n *Node) getUDP() (u *UDP) {
	n.mx.Lock()
	defer n.mx.Unlock()

	return n.udp
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

func (n *Node) delConnection(c *Conn) {
	n.mx.Lock()
	defer n.mx.Unlock()

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

	if occ := n.config.OnConnect; occ != nil {

		if terminate := occ(c); terminate != nil {
			c.close(terminate)
			return
		}

	}

	n.Debugf(ConnEstPin, "[%s] established", c.Address())

}

func (n *Node) onDisconenct(c *Conn, reason error) {

	if odc := n.config.OnDisconnect; odc != nil {
		odc(c, reason)
	}

	if reason != nil {
		n.Debugf(CloseConnPin, "[%s] closed by %v", c.Address(), reason)
	} else {
		n.Debugf(CloseConnPin, "[%s] closed", c.Address())
	}

}

func (n *Node) acceptConnection(fc *factory.Connection) {

	n.Debugf(NewInConnPin, "[%s] accept",
		connString(true, fc.IsTCP(), fc.GetRemoteAddr().String()))

	if _, err := n.wrapConnection(fc, true); err != nil {

		n.Printf("[ERR] [%s] handshake error: %v",
			connString(true, fc.IsTCP(), fc.GetRemoteAddr().String()),
			err)

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
) (
	c *Conn, //                :
	err error, //              :
) {

	n.Debugf(ConnHskPin, "[%s] wrapConnection",
		connString(isIncoming, fc.IsTCP(), fc.GetRemoteAddr().String()))

	c = n.newConnection(fc, isIncoming) // adds to pending

	// handshake
	if err = c.handshake(n.closeq); err != nil {
		n.delPendingConnClose(c)
		return
	}

	// check out peer id

	if err = n.addConnection(c); err != nil {
		n.delPendingConnClose(c)
		return
	}

	// add to transport if the connection is incoming
	if isIncoming == true {
		if c.IsTCP() == true {
			n.TCP().addAcceptedConnection(c)
		} else {
			n.UDP().addAcceptedConnection(c)
		}
	}

	c.run()
	n.onConnect(c)

	return

}

func (n *Node) updateServiceDiscovery() {

	n.mx.Lock()
	defer n.mx.Unlock()

	var notUsed = (n.tcp == nil || n.tcp.Discovery() == nil) &&
		(n.udp == nil || n.udp.Discovery() == nil)

	if notUsed == true {
		return
	}

	var feeds = n.fs.list()

	n.Debug(DiscoveryPin, "(Node) updateServiceDiscovery ", len(feeds))

	if n.tcp != nil && n.tcp.Discovery() != nil {
		n.tcp.updateServiceDiscovery(feeds)
	}

	if n.udp != nil && n.udp.Discovery() != nil {
		n.udp.updateServiceDiscovery(feeds)
	}

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
// the feed with a connection. You should to call
// (*Conn).Subscribe to do that.
func (n *Node) Share(feed cipher.PubKey) (err error) {

	if feed == (cipher.PubKey{}) {
		return ErrBlankFeed
	}

	// add to the Container
	if err = n.c.AddFeed(feed); err != nil {
		return
	}

	// TODO (kostyarin): Subscribe that calls the Share that
	//                   add a new feed then calls
	//                   updateServiceDiscovery that invoke
	//                   send new list to discovery and so on
	//
	// :: suppress the recursion ::

	if n.fs.addFeed(feed) == true {
		n.updateServiceDiscovery()
	}

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

	n.fs.delFeed(feed)
	n.updateServiceDiscovery()

	return
}

func (n *Node) onSubscribeRemote(c *Conn, feed cipher.PubKey) (reject error) {

	if osr := n.config.OnSubscribeRemote; osr != nil {
		reject = osr(c, feed)
	}

	return
}

func (n *Node) onUnsubscribeRemote(c *Conn, feed cipher.PubKey) {

	if ousr := n.config.OnUnsubscribeRemote; ousr != nil {
		ousr(c, feed)
	}

}

// Feeds the Node share. The reply is read-only
func (n *Node) Feeds() (feeds []cipher.PubKey) {
	return n.fs.list()
}

// IsSharing returns true if the Node is sharing given feed
func (n *Node) IsSharing(feed cipher.PubKey) (ok bool) {
	return n.fs.hasFeed(feed)
}

func (n *Node) onRootReceived(c *Conn, r *registry.Root) (err error) {

	if orr := n.config.OnRootReceived; orr != nil {
		err = orr(c, r)
	}

	return
}

func (n *Node) onRootFilled(r *registry.Root) {

	if orf := n.config.OnRootFilled; orf != nil {
		orf(n, r)
	}

}

func (n *Node) onFillingBreaks(r *registry.Root, reason error) {

	if brk := n.config.OnFillingBreaks; brk != nil {
		brk(n, r, reason)
	}

}

// has connection to peer with given id (pk)
func (n *Node) hasPeer(id cipher.PubKey) (c *Conn, yep bool) {
	n.mx.Lock()
	defer n.mx.Unlock()

	c, yep = n.ic[id]
	return
}

// A Stat represents Node stat
type Stat struct {
	*skyobject.Stat
	Fillavg time.Duration
}

// Stat returns statistic of the Node
func (n *Node) Stat() (s *Stat) {

	// TODO (kostyarin): improve the stat

	s = new(Stat)
	s.Stat = n.c.Stat()
	s.Fillavg = n.fillavg.Value()

	return
}

// Close the Node. The Close returns error
// of (skyobject.Container).Close once.
func (n *Node) Close() (err error) {
	n.closeo.Do(func() {

		close(n.closeq)

		n.mx.Lock()
		defer n.mx.Unlock()

		err = n.c.Close()

		if n.tcp != nil {
			n.tcp.Close()
		}

		if n.udp != nil {
			n.udp.Close()
		}

		if n.rpc != nil {
			n.rpc.Close()
		}

		n.await.Wait()

	})

	return
}
