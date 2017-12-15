package node

import (
	"errors"
	"path/filepath"
	"sync"
	"time"

	"github.com/skycoin/net/skycoin-messenger/factory"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/data/cxds"
	"github.com/skycoin/cxo/data/idxdb"

	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/cxo/skyobject/statutil"

	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/node/msg"
)

// common errors
var (
	// ErrTimeout occurs when a request that waits response tooks too long
	ErrTimeout = errors.New("timeout")
	// ErrSubscriptionRejected means that remote peer rejects our subscription
	ErrSubscriptionRejected = errors.New("subscription rejected by remote peer")
	// ErrNilConnection means that you tries to subscribe or request list of
	// feeds from a nil-connection
	ErrNilConnection = errors.New("subscribe to nil connection")
	// ErrUnexpectedResponse occurs if a remote peer sends any unexpected
	// response for our request
	ErrUnexpectedResponse = errors.New("unexpected response")
	// ErrNonPublicPeer occurs if a remote peer can't give us list of
	// feeds because it is not public
	ErrNonPublicPeer = errors.New(
		"request list of feeds from non-public peer")
	// ErrConnClosed occurs if connection closed but an action requested
	ErrConnClosed = errors.New("connection closed")
	// ErrUnsubscribed is a reason of dropping a filling Root
	ErrUnsubscribed = errors.New("unsubscribed")
	// ErrInvalidPubKeyLength occurs during decoding cipher.PubKey from hex
	ErrInvalidPubKeyLength = errors.New("invalid PubKey length")
)

// A Node represents CXO P2P node
// that includes RPC server if enabled
// by configs
type Node struct {
	log.Logger // logger of the server

	seed *factory.SeedConfig // stub

	src msg.Src // msg source

	conf Config // configurations

	db *data.DB             // database
	so *skyobject.Container // skyobject

	// feeds
	fmx    sync.RWMutex
	feeds  map[cipher.PubKey]map[*Conn]struct{}
	feedsl []cipher.PubKey // cow

	// connections
	cmx    sync.RWMutex
	conns  []*Conn
	connsl []*Conn // cow

	// waiting/wanted objects
	wmx sync.Mutex
	wos map[cipher.SHA256]map[*Conn]struct{}

	// connections
	pool *gnet.Pool
	rpc  *rpcServer // rpc server

	// discovery
	discovery *factory.MessengerFactory

	// stat
	fillavg statutil.RollAvg // avg filling time of filled roots
	dropavg statutil.RollAvg // avg filling time of dropped roots

	// closing
	quit  chan struct{}
	quito sync.Once

	done  chan struct{} // when quit done
	doneo sync.Once

	await sync.WaitGroup
}

// NewNode creates new Node instance using given
// configurations. The functions creates database and
// Container of skyobject instances internally. Use
// Config.Skyobject to provide appropriate configuration
// for skyobject.Container such as skyobject.Registry,
// etc. For example
//
//     conf := NewConfig()
//     conf.Skyobject.Registry = skyobject.NewRegistry(blah)
//
//     node, err := NewNode(conf)
//
// The recommended way to use NewConfig and changing
// result, instead of "empty config" with some fields:
//
//     // use this if you really know what you do
//     s, err := node.NewNode(node.Config{Listen: "[::]:8878"})
//
//     // I'd recommend
//     conf := node.NewConfig()
//     conf.Listen = "[::]:8887"
//
//     s, err := node.NewNode(conf)
//
func NewNode(sc Config) (s *Node, err error) {

	// data dir

	if sc.DataDir != "" {
		if err = initDataDir(sc.DataDir); err != nil {
			return
		}
	}

	// database

	var db *data.DB
	var cxPath, idxPath string

	if sc.DB != nil {
		cxPath, idxPath = "<used provided DB>", "<used provided DB>"
		db = sc.DB
	} else if sc.InMemoryDB {
		cxPath, idxPath = "<in memory>", "<in memory>"
		db = data.NewDB(cxds.NewMemoryCXDS(), idxdb.NewMemeoryDB())
	} else {
		if sc.DBPath == "" {
			cxPath = filepath.Join(sc.DataDir, CXDS)
			idxPath = filepath.Join(sc.DataDir, IdxDB)
		} else {
			cxPath = sc.DBPath + ".cxds"
			idxPath = sc.DBPath + ".idx"
		}
		var cx data.CXDS
		var idx data.IdxDB
		if cx, err = cxds.NewDriveCXDS(cxPath); err != nil {
			return
		}
		if idx, err = idxdb.NewDriveIdxDB(idxPath); err != nil {
			cx.Close()
			return
		}
		db = data.NewDB(cx, idx)
	}

	// container

	var so *skyobject.Container
	if so, err = skyobject.NewContainer(db, sc.Skyobject); err != nil {
		db.Close()
		return
	}

	// node instance

	s = new(Node)

	// TOOD (kostyarin): unfortunately, with the seed the nodes
	//                   doesn't connects, may be one constant seed
	//                   for al nodes required?

	s.seed = factory.NewSeedConfig() // seed
	s.seed = nil

	s.Logger = log.NewLogger(sc.Log)
	s.conf = sc

	s.db = db

	s.so = so
	s.feeds = make(map[cipher.PubKey]map[*Conn]struct{})

	s.wos = make(map[cipher.SHA256]map[*Conn]struct{})

	// fill up feeds from database
	err = s.db.IdxDB().Tx(func(feeds data.Feeds) (err error) {
		return feeds.Iterate(func(pk cipher.PubKey) (err error) {
			s.feeds[pk] = make(map[*Conn]struct{})
			return
		})
	})
	if err != nil {
		db.Close() // close DB
		s = nil    // GC
		return
	}

	if sc.Config.Logger == nil {
		sc.Config.Logger = s.Logger // use the same logger
	}

	// gnet related callbacks
	if ch := sc.Config.OnCreateConnection; ch == nil {
		sc.Config.OnCreateConnection = s.onConnect
	} else {
		sc.Config.OnCreateConnection = func(c *gnet.Conn) {
			s.onConnect(c)
			ch(c)
		}
	}
	if dh := sc.Config.OnCloseConnection; dh == nil {
		sc.Config.OnCloseConnection = s.onDisconnect
	} else {
		sc.Config.OnCloseConnection = func(c *gnet.Conn) {
			s.onDisconnect(c)
			dh(c)
		}
	}
	if dc := sc.Config.OnDial; dc == nil {
		sc.Config.OnDial = s.onDial
	} else {
		sc.Config.OnDial = func(c *gnet.Conn, err error) error {
			if err = dc(c, err); err != nil {
				return err
			}
			return s.onDial(c, err)
		}
	}

	if s.pool, err = gnet.NewPool(sc.Config); err != nil {
		db.Close() // close DB
		s = nil
		return
	}

	if sc.EnableRPC {
		s.rpc = newRPC(s)
	}

	s.quit = make(chan struct{})
	s.done = make(chan struct{})

	// stat
	s.fillavg = statutil.NewRollAvg(5) // TODO (kostyarin): make configurable
	s.dropavg = statutil.NewRollAvg(5) // TODO (kostyarin): make configurable

	if err = s.start(cxPath, idxPath); err != nil {
		s.Close()
		s = nil
	}
	return
}

func (s *Node) start(cxPath, idxPath string) (err error) {
	s.Debugf(log.All, `starting node:
    data dir:             %s

    max connections:      %d
    max message size:     %d

    dial timeout:         %v
    read timeout:         %v
    write timeout:        %v

    ping interval:        %v

    read queue:           %d
    write queue:          %d

    redial timeout:       %d
    max redial timeout:   %d
    dials limit:          %d

    read buffer:          %d
    write buffer:         %d

    TLS:                  %v

    enable RPC:           %v
    RPC address:          %s
    listening address:    %s
    enable listening:     %v
    remote close:         %t

    in-memory DB:         %v
    CXDS path:            %s
    index DB path:        %s

    discovery:            %s

    debug:                %#v
`,
		s.conf.DataDir,
		s.conf.MaxConnections,
		s.conf.MaxMessageSize,

		s.conf.DialTimeout,
		s.conf.ReadTimeout,
		s.conf.WriteTimeout,

		s.conf.PingInterval,

		s.conf.ReadQueueLen,
		s.conf.WriteQueueLen,

		s.conf.RedialTimeout,
		s.conf.MaxRedialTimeout,
		s.conf.DialsLimit,

		s.conf.ReadBufferSize,
		s.conf.WriteBufferSize,

		s.conf.TLSConfig != nil,

		s.conf.EnableRPC,
		s.conf.RPCAddress,
		s.conf.Listen,
		s.conf.EnableListener,
		s.conf.RemoteClose,

		s.conf.InMemoryDB,
		cxPath,
		idxPath,

		s.conf.DiscoveryAddresses.String(),

		s.conf.Log.Debug,
	)

	if len(s.conf.DiscoveryAddresses) > 0 {

		f := factory.NewMessengerFactory()

		for _, addr := range s.conf.DiscoveryAddresses {

			f.ConnectWithConfig(addr, &factory.ConnConfig{
				SeedConfig:                     s.seed,
				Reconnect:                      true,
				ReconnectWait:                  time.Second * 30,
				FindServiceNodesByKeysCallback: s.findServiceNodesCallback,
				OnConnected: s.
					updateServiceDiscoveryCallback,
			})

		}

		s.discovery = f

		if s.conf.Log.Debug == true && s.conf.Log.Pins&DiscoveryPin != 0 {
			f.SetLoggerLevel(factory.DebugLevel)
		} else {
			f.SetLoggerLevel(factory.ErrorLevel)
		}

	}

	// start listener
	if s.conf.EnableListener == true {
		if err = s.pool.Listen(s.conf.Listen); err != nil {
			return
		}
		s.Print("listen on ", s.pool.Address())
	}

	// start rpc listener if need
	if s.conf.EnableRPC == true {
		if err = s.rpc.Start(s.conf.RPCAddress); err != nil {
			s.pool.Close()
			return
		}
		s.Print("rpc listen on ", s.rpc.Address())
	}

	if s.conf.PingInterval > 0 {
		s.await.Add(1)
		go s.pingsLoop()
	}

	return
}

func (s *Node) addConn(c *Conn) {
	s.cmx.Lock()
	defer s.cmx.Unlock()

	c.gc.SetValue(c) // for resubscriptions

	s.conns = append(s.conns, c)
	s.connsl = nil // clear cow copy
}

func (s *Node) delConn(c *Conn) {
	s.cmx.Lock()
	defer s.cmx.Unlock()

	s.connsl = nil // clear cow copy

	for i, x := range s.conns {
		if x == c {
			s.conns[i] = s.conns[len(s.conns)-1]
			s.conns[len(s.conns)-1] = nil
			s.conns = s.conns[:len(s.conns)-1]
			return
		}
	}

}

func (s *Node) updateServiceDiscoveryCallback(conn *factory.Connection) {
	s.Debugln(DiscoveryPin, "updateServiceDiscoveryCallback")

	feeds := s.Feeds()
	services := make([]*factory.Service, len(feeds))
	for i, feed := range feeds {
		services[i] = &factory.Service{Key: feed}
	}
	s.updateServiceDiscovery(conn, feeds, services)
}

func (s *Node) updateServiceDiscovery(conn *factory.Connection,
	feeds []cipher.PubKey, services []*factory.Service) {

	s.Debugln(DiscoveryPin, "updateServiceDiscovery", feeds)

	if err := conn.FindServiceNodesByKeys(feeds); err != nil {
		s.Debug(DiscoveryPin, "finding error: ", err)
	}

	if s.conf.PublicServer == true && s.conf.EnableListener == true {

		var address = s.Pool().Address()

		if address == "" {
			return // the node doesn't listen
		}

		s.Debug(DiscoveryPin, "UpdateServices adress: ", address)

		err := conn.UpdateServices(&factory.NodeServices{
			ServiceAddress: address,
			Services:       services,
		})

		if err != nil {
			s.Debug(DiscoveryPin, "updating error: ", err)
		}

	}

}

func (s *Node) findServiceNodesCallback(resp *factory.QueryResp) {

	s.Debug(DiscoveryPin, "findServiceNodesCallback:", len(resp.Result))

	if len(resp.Result) < 1 {
		return
	}

	for _, si := range resp.Result {

		if si == nil {
			continue // happens, TODO (kostyarin): ask about it
		}

		// TODO (kostyarin): filter by the seed (pk, that is node id)

		for _, ni := range si.Nodes {

			if ni == nil {
				continue // never happens
			}

			// ignore ni.PubKey for now

			c, err := s.ConnectOrGet(ni.Address)

			if err != nil {
				s.Debugf(DiscoveryPin, "can't ConnectOrGet %q: %v",
					ni.Address,
					err)
				continue
			}

			if err = c.Subscribe(si.PubKey); err != nil {
				s.Debugln(DiscoveryPin, "can't Subscribe:", err)
			}

		}

	}

}

func (s *Node) gotObject(key cipher.SHA256, obj *msg.Object) {
	s.wmx.Lock()
	defer s.wmx.Unlock()

	for c := range s.wos[key] {
		c.Send(obj)
	}
	delete(s.wos, key)
}

func (s *Node) wantObject(key cipher.SHA256, c *Conn) {
	s.wmx.Lock()
	defer s.wmx.Unlock()

	if cs, ok := s.wos[key]; ok {
		cs[c] = struct{}{}
		return
	}
	s.wos[key] = map[*Conn]struct{}{c: {}}
}

func (s *Node) delConnFromWantedObjects(c *Conn) {
	s.wmx.Lock()
	defer s.wmx.Unlock()

	for _, cs := range s.wos {
		delete(cs, c)
	}
}

// Discovery returns *factory.MessengerFactory of the Node.
// It can be nil if this feature disabled by configs
func (s *Node) Discovery() *factory.MessengerFactory {
	return s.discovery
}

// ConnectToMessenger connects to a messenger server
func (s *Node) ConnectToMessenger(address string) (*factory.Connection, error) {
	if s.discovery == nil {
		return nil, errors.New("messenger factory not initialised")
	}
	return s.discovery.ConnectWithConfig(address, &factory.ConnConfig{
		SeedConfig:                     s.seed,
		Reconnect:                      true,
		ReconnectWait:                  time.Second * 30,
		FindServiceNodesByKeysCallback: s.findServiceNodesCallback,
		OnConnected:                    s.updateServiceDiscoveryCallback,
	})
}

// Connections of the Node. It returns shared
// slice and you must not modify it
func (s *Node) Connections() []*Conn {
	s.cmx.RLock()
	defer s.cmx.RUnlock()

	if s.connsl != nil {
		return s.connsl
	}

	s.connsl = make([]*Conn, len(s.conns))
	copy(s.connsl, s.conns)

	return s.connsl
}

// Connection by address. It returns nil if
// connection not found or not established yet
func (s *Node) Connection(address string) (c *Conn) {
	if gc := s.pool.Connection(address); gc != nil {
		c, _ = gc.Value().(*Conn)
	}
	return
}

// Close the Node. A Node can't be used after closing
func (s *Node) Close() (err error) {

	// release the s.Quiting() channel
	s.quito.Do(func() {
		close(s.quit)
	})

	// close discovery
	if s.discovery != nil {
		s.discovery.Close() // TODO (kostyarin): error
	}

	// close listener and all connections
	err = s.pool.Close()

	// close RPC server
	if s.conf.EnableRPC {
		s.rpc.Close() // TODO (kostyarin): error
	}

	// wait all connections and other goroutines
	s.await.Wait()

	// close Container
	s.so.Close() // TODO (kostyarin): error

	// close database after all
	s.db.Close() // TODO (kostyarin): error

	// release the s.Closed() channel
	s.doneo.Do(func() {
		close(s.done)
	})
	return
}

// DB of the Node
func (s *Node) DB() *data.DB { return s.db }

// Container of the Node
func (s *Node) Container() *skyobject.Container {
	return s.so
}

//
// Public methods of the Node
//

// Pool returns underlying *gnet.Pool.
// It returns nil if the Node is not started
// yet. Use methods of this Pool to manipulate
// connections: Dial, Connection, Connections,
// Address, etc
func (s *Node) Pool() *gnet.Pool {
	return s.pool
}

// Feeds the server share. It returns shared
// slice and you must not modify it
func (s *Node) Feeds() []cipher.PubKey {

	// locks: s.fmx RLock/RUnlock

	s.fmx.RLock()
	defer s.fmx.RUnlock()

	if s.feedsl != nil {
		return s.feedsl
	}

	s.feedsl = make([]cipher.PubKey, 0, len(s.feeds))
	for f := range s.feeds {
		s.feedsl = append(s.feedsl, f)
	}

	return s.feedsl
}

// HasFeed or has not
func (s *Node) HasFeed(pk cipher.PubKey) (ok bool) {
	s.fmx.RLock()
	defer s.fmx.RUnlock()
	_, ok = s.feeds[pk]
	return
}

// send Root to subscribers
func (s *Node) broadcastRoot(r *skyobject.Root, e *Conn) {
	s.fmx.RLock()
	defer s.fmx.RUnlock()

	for c := range s.feeds[r.Pub] {
		if c == e {
			continue // except
		}
		c.SendRoot(r)
	}
}

func (s *Node) addConnToFeed(c *Conn, pk cipher.PubKey) (added bool) {
	s.fmx.Lock()
	defer s.fmx.Unlock()

	if cs, ok := s.feeds[pk]; ok {
		cs[c], added = struct{}{}, true
	}
	return
}

func (s *Node) delConnFromFeed(c *Conn, pk cipher.PubKey) (deleted bool) {
	s.fmx.Lock()
	defer s.fmx.Unlock()

	if cs, ok := s.feeds[pk]; ok {
		if _, deleted = cs[c]; deleted {
			delete(cs, c)
		}
	}
	return
}

func (s *Node) onConnect(gc *gnet.Conn) {
	s.Debugf(ConnPin, "[%s] new connection %t", gc.Address(), gc.IsIncoming())

	if gc.IsIncoming() {

		c := s.newConn(gc)

		s.await.Add(1)
		go c.handle(nil)

	}

	// all outgoing connections processed by s.Connect()
}

func (s *Node) onDisconnect(gc *gnet.Conn) {
	s.Debugf(ConnPin, "[%s] close connection %t", gc.Address(), gc.IsIncoming())
}

func (s *Node) onDial(gc *gnet.Conn, _ error) (_ error) {
	if c, ok := gc.Value().(*Conn); ok {
		c.enqueueEvent(resubscribeEvent{})
	}
	return
}

// Quiting returns channel that closed
// when the Node performs closing and
// after that. This channel can be used to
// stop Root objects handling and
// publishing, because connections are closed
// of closing. To determine the moment while
// node closed use Closed() method
func (s *Node) Quiting() <-chan struct{} {
	return s.quit // when quit done
}

// Closed returns channel that closed when
// the Node has been closed. This channel
// can be used to close application or
// remove the Node instance. If the channel
// closed then all Node internals closed
func (s *Node) Closed() <-chan struct{} {
	return s.done
}

// RPCAddress returns address of RPC listener
// or an empty string if disabled
func (s *Node) RPCAddress() (address string) {
	if s.rpc != nil {
		address = s.rpc.Address()
	}
	return
}

// Publish given Root (send to feed). Given Root
// must be held and not changed during this call
// (held during this call only)
func (s *Node) Publish(r *skyobject.Root) {

	// make sterile copy first

	root := new(skyobject.Root)

	root.Reg = r.Reg
	root.Pub = r.Pub
	root.Seq = r.Seq
	root.Time = r.Time
	root.Sig = r.Sig
	root.Hash = r.Hash
	root.Prev = r.Prev
	root.IsFull = r.IsFull

	root.Refs = make([]skyobject.Dynamic, 0, len(r.Refs))

	for _, dr := range r.Refs {
		root.Refs = append(root.Refs, skyobject.Dynamic{
			SchemaRef: dr.SchemaRef,
			Object:    dr.Object,
		})
	}

	s.broadcastRoot(root, nil)
}

// Connect to peer. This call blocks until connection created
// and established (after successful handshake). The method
// returns error if connection already exists, connections
// limit reached, given address malformed or handhsake can't
// be performed for some reason
func (s *Node) Connect(address string) (c *Conn, err error) {

	var gc *gnet.Conn
	if gc, err = s.pool.Dial(address); err != nil {
		return
	}

	return s.createConnection(gc)
}

// ConnectOrGet connects to peer or returns
// connection if it already exist
func (s *Node) ConnectOrGet(address string) (c *Conn, err error) {

	var gc *gnet.Conn
	var fresh bool
	if gc, fresh, err = s.pool.DialOrGet(address); err != nil {
		return
	}

	if true == fresh {
		return s.createConnection(gc)
	}

	if cv := gc.Value(); cv != nil {
		c = cv.(*Conn) // already have
		return
	}

	// So, at this point the gc can be created by any other
	// goroutine, but not established yet. Thus, we can't create a
	// connection by it

	// TODO (kostyarin): rid out of spinning; so, gnet changes required,
	//                   like DialWithValue or something like this

	for {
		select {
		case <-gc.Closed():
			err = ErrConnClosed // TODO (kostyarin): recreate or not?
			return
		default:
			if cv := gc.Value(); cv != nil {
				c = cv.(*Conn) // got it
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

}

func (s *Node) createConnection(gc *gnet.Conn) (c *Conn, err error) {
	hs := make(chan error)

	c = s.newConn(gc)

	s.await.Add(1)
	go c.handle(hs)

	err = <-hs
	return
}

// AddFeed to list of feed the Node shares.
// This method adds feed to underlying skyobject.Container
// and database. But it doesn't starts exchanging
// the feed with peers. Use following code to
// subscribe al connections to the feed
//
//     if err := s.AddFeed(pk); err != nil {
//         // database failure
//     }
//     for _, c := range s.Connections() {
//         // blocking call
//         if err := c.Subscribe(pk); err != nil {
//             // handle the err
//         }
//     }
//
func (s *Node) AddFeed(pk cipher.PubKey) (err error) {
	s.fmx.Lock()
	defer s.fmx.Unlock()

	if _, ok := s.feeds[pk]; !ok {
		if err = s.so.AddFeed(pk); err != nil {
			return
		}
		s.feeds[pk] = make(map[*Conn]struct{})
		s.feedsl = nil // clear cow copy
		updateServiceDiscovery(s)
	}
	return
}

// del feed from share-list
func (s *Node) delFeed(pk cipher.PubKey) (ok bool) {
	s.fmx.Lock()
	defer s.fmx.Unlock()

	if _, ok = s.feeds[pk]; ok {
		delete(s.feeds, pk)
		s.feedsl = nil // clear cow copy
		updateServiceDiscovery(s)
	}
	return
}

// perform it under 'fmx' lock
func updateServiceDiscovery(n *Node) {

	n.Debug(DiscoveryPin, "updateServiceDiscovery function")

	if n.discovery != nil {
		feeds := make([]cipher.PubKey, 0, len(n.feeds))
		services := make([]*factory.Service, 0, len(feeds))

		for pk := range n.feeds {
			feeds = append(feeds, pk)
			services = append(services, &factory.Service{Key: pk})
		}
		go n.discovery.ForEachConn(func(connection *factory.Connection) {
			n.updateServiceDiscovery(connection, feeds, services)
		})
	}
}

// del feed from connections, every connection must
// reply when it done, because we have to know
// the moment after which our DB doesn't contains
// non-full Root object; thus, every connections
// terminates fillers of the feed and removes non-full
// root objects
func (s *Node) delFeedConns(pk cipher.PubKey) (dones []delFeedConnsReply) {
	s.cmx.RLock()
	defer s.cmx.RUnlock()

	dones = make([]delFeedConnsReply, 0, len(s.conns))

	for _, c := range s.conns {

		done := make(chan struct{})

		select {
		case c.events <- &unsubscribeFromDeletedFeedEvent{pk, done}:
		case <-c.gc.Closed():
		}

		dones = append(dones, delFeedConnsReply{done, c.done})
	}
	return
}

type delFeedConnsReply struct {
	done   <-chan struct{} // filler closed
	closed <-chan struct{} // connections closed and done
}

// DelFeed stops sharing given feed. It unsubscribes
// from all connections
func (s *Node) DelFeed(pk cipher.PubKey) (err error) {

	if false == s.delFeed(pk) {
		return // not deleted (we haven't the feed)
	}

	dones := s.delFeedConns(pk)

	// wait
	for _, dfcr := range dones {
		select {
		case <-dfcr.done:
		case <-dfcr.closed: // connection's done
		}
	}

	// now, we can remove the feed if there
	// are not held Root objects
	err = s.so.DelFeed(pk)
	return
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func (s *Node) pingsLoop() {
	defer s.await.Done()

	tk := time.NewTicker(s.conf.PingInterval)
	defer tk.Stop()

	for {
		select {
		case <-tk.C:
			now := time.Now()
			for _, c := range s.Connections() {
				md := maxDuration(now.Sub(c.gc.LastRead()),
					now.Sub(c.gc.LastWrite()))
				if md < s.conf.PingInterval {
					continue
				}
				c.SendPing()
			}
		case <-s.quit:
			return
		}
	}
}
