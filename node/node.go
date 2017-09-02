package node

import (
	"errors"
	"path/filepath"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/data/cxds"
	"github.com/skycoin/cxo/data/idxdb"
	"github.com/skycoin/cxo/skyobject"

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
	ErrConnClsoed = errors.New("connection closed")
)

// A Node represents CXO P2P node
// that includes RPC server if enabled
// by configs
type Node struct {
	log.Logger                      // logger of the server
	src        msg.Src              // msg source
	conf       Config               // configuratios
	db         *data.DB             // database
	so         *skyobject.Container // skyobject

	// feeds
	fmx   sync.RWMutex
	feeds map[cipher.PubKey]map[*Conn]struct{}

	// connections
	cmx   sync.RWMutex
	conns []*Conn

	// waiting/wanted obejcts
	wmx sync.Mutex
	wos map[cipher.SHA256]map[*Conn]struct{}

	// connections
	pool *gnet.Pool
	rpc  *rpcServer // rpc server

	// closing
	quit  chan struct{}
	quito sync.Once

	done  chan struct{} // when quit done
	doneo sync.Once

	await sync.WaitGroup
}

// NewNode creates new Node instnace using given
// configurations. The functions creates database and
// Container of skyobject instances internally. Use
// Config.Skyobject to provide appropriate configuration
// for skyobject.Container such as skyobject.Regsitry,
// etc. For example
//
//     conf := NewConfig()
//     conf.Skyobject.Regsitry = skyobject.NewRegistry(blah)
//
//     node, err := NewNode(conf)
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

	if sc.DB != nil {
		db = sc.DB
	} else if sc.InMemoryDB {
		db = data.NewDB(cxds.NewMemoryCXDS(), idxdb.NewMemeoryDB())
	} else {
		var cxPath, idxPath string
		if sc.DBPath == "" {
			cxPath = filepath.Join(sc.DataDir, "cxds.db")
			idxPath = filepath.Join(sc.DataDir, "idx.db")
		} else {
			cxPath = sc.DBPath + ".cxds"
			cxPath = sc.DBPath + ".idx"
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
	so = skyobject.NewContainer(db, sc.Skyobject)

	// node instance

	s = new(Node)

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

	if err = s.start(); err != nil {
		s.Close()
		s = nil
	}
	return
}

func (s *Node) start() (err error) {
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
    DB path:              %s

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
		s.conf.DBPath,

		s.conf.Log.Debug,
	)

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

func (n *Node) addConn(c *Conn) {
	n.cmx.Lock()
	defer n.cmx.Unlock()

	n.conns = append(n.conns, c)
}

func (n *Node) delConn(c *Conn) {
	n.cmx.Lock()
	defer n.cmx.Unlock()

	for i, x := range n.conns {
		if x == c {
			n.conns[i] = n.conns[len(n.conns)-1]
			n.conns[len(n.conns)-1] = nil
			n.conns = n.conns[:len(n.conns)-1]
			return
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

func (s *Node) wantObejct(key cipher.SHA256, c *Conn) {
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

// Connections of the Node
func (s *Node) Connections() (cs []*Conn) {
	s.cmx.RLock()
	defer s.cmx.RUnlock()

	cs = make([]*Conn, len(s.conns))
	copy(cs, s.conns)
	return
}

// Connections by address. Itreturns nil if conenction not
// found or not established yet
func (s *Node) Connection(address string) (c *Conn) {
	if gc := s.pool.Connection(address); gc != nil {
		c, _ = gc.Value().(*Conn)
	}
	return
}

// Close the Node
func (s *Node) Close() (err error) {
	s.quito.Do(func() {
		close(s.quit)
	})
	err = s.pool.Close()
	if s.conf.EnableRPC {
		s.rpc.Close()
	}
	s.await.Wait()
	// we have to close boltdb once
	s.doneo.Do(func() {
		// close Container
		s.so.Close()
		// close database after all, otherwise, it panics
		s.db.Close()
		// close the Quiting channel
		close(s.done)
	})

	return
}

// DB of the Node
func (n *Node) DB() *data.DB { return n.db }

// Container of the Node
func (n *Node) Container() *skyobject.Container {
	return n.so
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

// Feeds the server share
func (s *Node) Feeds() (fs []cipher.PubKey) {

	// locks: s.fmx RLock/RUnlock

	s.fmx.RLock()
	defer s.fmx.RUnlock()

	if len(s.feeds) == 0 {
		return
	}
	fs = make([]cipher.PubKey, 0, len(s.feeds))
	for f := range s.feeds {
		fs = append(fs, f)
	}
	return
}

// HasFeed or has not
func (n *Node) HasFeed(pk cipher.PubKey) (ok bool) {
	n.fmx.RLock()
	defer n.fmx.RUnlock()
	_, ok = n.feeds[pk]
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

	c := s.newConn(gc)

	s.await.Add(1)
	go c.handle()
}

func (s *Node) onDisconnect(gc *gnet.Conn) {
	s.Debugf(ConnPin, "[%s] close conenction %t", gc.Address(), gc.IsIncoming())
}

func (s *Node) onDial(gc *gnet.Conn, _ error) (_ error) {
	if c, ok := gc.Value().(*Conn); ok {
		c.enqueueEvent(resubscribeEvent{})
	}
	return
}

// Quiting returns cahnnel that closed
// when the Node closed
func (s *Node) Quiting() <-chan struct{} {
	return s.done // when quit done
}

// RPCAddress returns address of RPC listener or an empty
// stirng if disabled
func (s *Node) RPCAddress() (address string) {
	if s.rpc != nil {
		address = s.rpc.Address()
	}
	return
}

// Publish given Root (send to feed). Given Root
// must be holded and not chagned during this call
// (holded during this call only)
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

// Connect to peer. Use callback to handle the Conn
func (n *Node) Connect(address string) (err error) {
	_, err = n.pool.Dial(address)
	return
}

// AddFeed to list of feed the Node shares.
// This method adds feed to undrlying skyobject.Container
// and database. But it doesn't starts exchanging
// the feed with peers. Use following code to
// subscribe al connections to the feed
//
//     if err := n.AddFeed(pk); err != nil {
//         // database failure
//     }
//     for _, c := range n.Connections() {
//         // blocking call
//         if err := c.Subscribe(pk); err != nil {
//             // handle the err
//         }
//     }
//
func (n *Node) AddFeed(pk cipher.PubKey) (err error) {
	n.fmx.Lock()
	defer n.fmx.Unlock()

	if _, ok := n.feeds[pk]; !ok {
		if err = n.so.AddFeed(pk); err != nil {
			return
		}
		n.feeds[pk] = make(map[*Conn]struct{})
	}
	return
}

// del feed from share-list
func (n *Node) delFeed(pk cipher.PubKey) (ok bool) {
	n.fmx.Lock()
	defer n.fmx.Unlock()

	if _, ok = n.feeds[pk]; ok {
		delete(n.feeds, pk)
	}
	return
}

// del feed from conenctions, every conenction must
// reply when it done, because we have to know
// the moment after which our DB doesn't contains
// non-full Root object; thus, every conenctions
// terminates fillers of the feed and removes non-full
// root objects
func (n *Node) delFeedConns(pk cipher.PubKey) (dones []delFeedConnsReply) {
	n.cmx.RLock()
	defer n.cmx.RUnlock()

	dones = make([]delFeedConnsReply, 0, len(n.conns))

	for _, c := range n.conns {

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
	closed <-chan struct{} // conenctions closed and done
}

// DelFed stops sharing given feed. It unsubscribes
// from all conenctions
func (n *Node) DelFeed(pk cipher.PubKey) (err error) {

	if false == n.delFeed(pk) {
		return // not deleted (we haven't the feed)
	}

	dones := n.delFeedConns(pk)

	// wait
	for _, dfcr := range dones {
		select {
		case <-dfcr.done:
		case <-dfcr.closed: // connection's done
		}
	}

	// now, we can remove the feed if there
	// are not holded Root objects
	err = n.so.DelFeed(pk)
	return
}

/*
// Stat of underlying DB and Container
func (s *Node) Stat() (st Stat) {
	st.Data = s.DB().Stat()
	st.CXO = s.Container().Stat()
	return
}
*/

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
