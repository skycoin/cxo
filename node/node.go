package node

import (
	"errors"
	"fmt"
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

	fs *nodeFeeds // feeds

	pc map[string]*Conn        // peer addr -> pending connection
	ac map[string]*Conn        // peer addr -> connection
	ic map[cipher.PubKey]*Conn // neer ID (pubkey) -> connection

	ss map[cipher.PubKey]*Swarm // swarms

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

	// Generate random secret key or use one from config
	var defSK cipher.SecKey
	if conf.SecKey == defSK {
		n.id = discovery.NewSeedConfig()
	} else {
		if err := conf.SecKey.Verify(); err != nil {
			return nil, fmt.Errorf("invalid secret key - %s", err)
		}
		n.id = discovery.SecKeyToSeedConfig(conf.SecKey)
	}

	n.idpk, _ = cipher.PubKeyFromHex(n.id.PublicKey)

	n.c = c
	n.fs = newNodeFeeds(n)
	n.ic = make(map[cipher.PubKey]*Conn)
	n.pc = make(map[string]*Conn)
	n.ss = make(map[cipher.PubKey]*Swarm)

	n.config = conf
	n.config.Config = c.Config() // actual

	n.fillavg = statutil.NewDuration(conf.Config.RollAvgSamples)

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
		if err := n.TCP().ConnectToDiscoveryServer(address); err != nil {
			n.Debugln(DiscoveryPin,
				"error connecting to TCP discovery server:",
				err)
		}
	}

	for _, address := range conf.UDP.Discovery {
		if err := n.UDP().ConnectToDiscoveryServer(address); err != nil {
			n.Debugln(DiscoveryPin,
				"error connecting to UDP discovery server:",
				err)
		}
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

func (n *Node) onConnect(c *Conn) error {
	if occ := n.config.OnConnect; occ != nil {
		if err := occ(c); err != nil {
			return err
		}
	}

	n.Debugf(ConnEstPin, "[%s] established", c.Address())

	return nil
}

func (n *Node) onDisconenct(c *Conn, reason error) {
	if odc := n.config.OnDisconnect; odc != nil {
		odc(c, reason)
	}

	if reason != nil {
		n.Debugf(CloseConnPin, "[%s] closed: %v", c.Address(), reason)
	} else {
		n.Debugf(CloseConnPin, "[%s] closed", c.Address())
	}
}

func (n *Node) connCap() int {
	n.mx.Lock()
	defer n.mx.Unlock()

	if len(n.ic) >= n.config.MaxConnections {
		return 0
	}

	return n.config.MaxConnections - len(n.ic)
}

func (n *Node) pendingConnCap() int {
	n.mx.Lock()
	defer n.mx.Unlock()

	if len(n.pc) >= n.config.MaxPendingConnections {
		return 0
	}

	return n.config.MaxPendingConnections - len(n.pc)
}

// initConn initializes new connection.
func (n *Node) initConn(
	fc *factory.Connection, isIncoming bool) (*Conn, error) {

	var (
		addr    = fc.GetRemoteAddr().String()
		connStr = connString(isIncoming, fc.IsTCP(), addr)
	)

	n.Debugf(ConnHskPin, "[%s] init connection", connStr)

	// Get existing connection or create new one.
	c, isNew, isPending, err := n.onNewConn(fc, isIncoming)
	if err != nil {
		return nil, err
	}

	switch {
	// In case of exisitng connection return it.
	case !isNew && !isPending:
		return c, nil

	// In case of existing pending connnection wait for another initConn to finish.
	case !isNew && isPending:
		if err = c.waitForInit(); err != nil {
			n.onConnInitErr(c, err)
			return nil, fmt.Errorf("init of existing pending connection failed : %s", err)
		}

	// In case of new connection perform/accept handshake.
	case isNew:
		if err = c.handshake(); err != nil {
			n.onConnInitErr(c, err)
			return nil, err
		}
		if err = n.onConnInit(c); err != nil {
			return nil, err
		}

		go c.run()

		return c, nil
	}

	panic("onNewConn return invalid values")
}

// onNewConn atomically checks for existing connection
// to/form address and creates new one if neccessary.
func (n *Node) onNewConn(
	fc *factory.Connection, isIncoming bool) (c *Conn, isNew, isPending bool, err error) {

	n.mx.Lock()
	defer n.mx.Unlock()

	// Check limits for number of open connections.
	if len(n.ic) >= n.config.MaxConnections {
		return nil, false, false, ErrConnLimit
	}
	if len(n.pc) >= n.config.MaxPendingConnections {
		return nil, false, false, ErrPendConnLimit
	}

	addr := fc.GetRemoteAddr().String()

	// Check if connection to/from address already exists.
	if c, ok := n.ac[addr]; ok {
		return c, false, false, nil
	}
	if c, ok := n.pc[addr]; ok {
		return c, false, true, nil
	}

	// Create new pending connection.
	c = n.newConnection(fc, isIncoming)
	n.pc[addr] = c

	return c, true, false, nil
}

// onConnInitErr atomically removes pending
// connection and signals about initConn failure.
func (n *Node) onConnInitErr(c *Conn, initErr error) {
	n.mx.Lock()
	defer n.mx.Unlock()

	delete(n.pc, c.Address())

	c.initErr = initErr
	close(c.init)
}

// onConnInit atomically moves pending connection to cache
// and singals about initConn success. It also chekcs if
// peer's pubkey, receieved during hanshake is duplicate.
func (n *Node) onConnInit(c *Conn) error {
	n.mx.Lock()
	defer n.mx.Unlock()

	delete(n.pc, c.Address())

	if _, ok := n.ic[c.peerID]; ok {
		return ErrAlreadyHaveConnection
	}

	n.ac[c.Address()] = c
	n.ic[c.peerID] = c

	close(c.init)

	return nil
}

// removeConn reomoves connection from cache.
func (n *Node) removeConn(c *Conn) {
	n.mx.Lock()
	defer n.mx.Unlock()

	delete(c.n.ic, c.peerID)
	delete(c.n.ac, c.Address())
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
		n.mx.Lock()
		defer n.mx.Unlock()

		// Shutdown peer exchange.
		for _, s := range n.ss {
			s.shutdown()
		}

		// Close all connections.
		for _, c := range n.ic {
			c.Close()
		}
		for _, c := range n.pc {
			c.Close()
		}

		// Shutdown all transports.
		if n.tcp != nil {
			n.tcp.Close()
		}
		if n.udp != nil {
			n.udp.Close()
		}
		if n.rpc != nil {
			n.rpc.Close()
		}

		// Close database.
		err = n.c.Close()

		n.await.Wait()

	})

	return
}

func (n *Node) JoinSwarm(
	feed cipher.PubKey, cfg SwarmConfig) (*Swarm, error) {

	n.mx.Lock()
	defer n.mx.Unlock()

	if s, ok := n.ss[feed]; ok {
		return s, nil
	}

	s := newSwarm(n, feed, cfg)

	go s.run()

	n.ss[feed] = s

	return s, nil
}

func (n *Node) LeaveSwarm(feed cipher.PubKey) error {
	n.mx.Lock()
	defer n.mx.Unlock()

	s, ok := n.ss[feed]
	if !ok {
		return errors.New("node is not in swarm")
	}

	s.shutdown()

	delete(n.ss, feed)

	return nil
}

func (n *Node) InSwarm(feed cipher.PubKey) (*Swarm, bool) {
	n.mx.Lock()
	defer n.mx.Unlock()

	s, ok := n.ss[feed]

	return s, ok
}
