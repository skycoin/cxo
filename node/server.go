package node

import (
	"fmt"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"

	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"

	"github.com/boltdb/bolt"
	"github.com/skycoin/skycoin/src/util"
	"io/ioutil"
	"os"
	"path/filepath"
)

type fillRoot struct {
	root  *skyobject.Root     // filling the root to send forward
	c     *gnet.Conn          // from which the root received
	await skyobject.Reference // waiting for
}

//
// TODO: GC for skyobject.Container
//

// A Server represents CXO server
// that includes RPC server if enabled
// by configs
type Server struct {
	// logger of the server
	log.Logger

	// configuratios
	conf ServerConfig

	// skyobject
	so *skyobject.Container

	// feeds
	fmx   sync.RWMutex
	feeds map[cipher.PubKey]map[*gnet.Conn]struct{}

	rmx   sync.RWMutex
	roots []*fillRoot // filling up

	// connections
	pool *gnet.Pool
	rpc  *RPC // rpc server

	// db
	db *bolt.DB

	// closing
	quit  chan struct{}
	quito sync.Once

	done  chan struct{} // when quit done
	doneo sync.Once

	await sync.WaitGroup
}

// NewServer creates new Server instnace using given
// configurations. The functions creates database and
// Container of skyobject instances internally
func NewServer(sc ServerConfig) (s *Server, err error) {
	s, err = NewServerSoDB(sc, skyobject.NewContainer(nil))
	return
}

// NewServerSoDB creates new Server instance using given
// configurations and Container
func NewServerSoDB(sc ServerConfig, so *skyobject.Container) (s *Server,
	err error) {

	if so == nil {
		panic("nil db")
	}

	s = new(Server)

	s.Logger = log.NewLogger(sc.Log.Prefix, sc.Log.Debug)
	s.conf = sc

	s.so = so
	s.feeds = make(map[cipher.PubKey]map[*gnet.Conn]struct{})

	sc.Config.Logger = s.Logger // use the same logger
	sc.Config.ConnectionHandler = s.connectHandler
	sc.Config.DisconnectHandler = s.disconnectHandler
	if s.pool, err = gnet.NewPool(sc.Config); err != nil {
		s = nil
		return
	}

	if sc.EnableRPC {
		s.rpc = newRPC(s)
	}

	util.InitDataDir(".skycoin")
	if sc.EnableBlockDB {
		s.openDB()
	}

	s.quit = make(chan struct{})
	s.done = make(chan struct{})

	return
}

// Start the server
func (s *Server) Start() (err error) {
	var dbFile string
	if s.db != nil {
		dbFile = s.db.Path()
	}
	s.Debugf(`starting server:
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
    redials limit:        %d

    read buffer:          %d
    write buffer:         %d

    TLS:                  %v

    enable RPC:           %v
    RPC address:          %s
    lListening address:   %s
    remote close:         %t

    enable DB:            %v
    DB location:          %s

    debug:                %#v
`,
		util.DataDir,
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
		s.conf.RedialsLimit,

		s.conf.ReadBufferSize,
		s.conf.WriteBufferSize,

		s.conf.TLSConfig != nil,

		s.conf.EnableRPC,
		s.conf.RPCAddress,
		s.conf.Listen,
		s.conf.RemoteClose,

		s.conf.EnableBlockDB,
		dbFile,

		s.conf.Log.Debug,
	)
	// start listener
	if err = s.pool.Listen(s.conf.Listen); err != nil {
		return
	}
	s.Print("listen on ", s.pool.Address())
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

func (s *Server) openDB() (err error) {
	dbFile := filepath.Join(util.DataDir, "cxod.db")
	if s.conf.RandomizeDBPath {
		var file *os.File
		if file, err = ioutil.TempFile("", "blockdb_"); err != nil {
			return
		}
		dbFile = file.Name()
		file.Close()
	}
	s.db, err = bolt.Open(dbFile, 0600, &bolt.Options{
		Timeout: 500 * time.Millisecond,
	})
	if err != nil {
		return fmt.Errorf("Open boltdb failed, err:%v", err)
	}
	return
}

func (s *Server) closeDB() (err error) {
	if !s.conf.EnableBlockDB {
		return
	}
	dbFile := s.db.Path()
	s.db.Close() // drop closing error
	if s.conf.RandomizeDBPath {
		err = os.Remove(dbFile)
	}
	return nil
}

// Close the server
func (s *Server) Close() (err error) {
	s.quito.Do(func() {
		close(s.quit)
	})
	err = s.pool.Close()

	s.closeDB()
	if s.conf.EnableRPC {
		s.rpc.Close()
	}
	s.await.Wait()
	s.doneo.Do(func() {
		close(s.done)
	})
	return
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func (s *Server) pingsLoop() {
	defer s.await.Done()

	var tk *time.Ticker = time.NewTicker(s.conf.PingInterval)
	defer tk.Stop()

	for {
		select {
		case <-tk.C:
			now := time.Now()
			for _, c := range s.pool.Connections() {
				md := maxDuration(now.Sub(c.LastRead()), now.Sub(c.LastWrite()))
				if md < s.conf.PingInterval {
					continue
				}
				s.sendMessage(c, &PingMsg{})
			}
		case <-s.quit:
			return
		}
	}
}

// send a message to given connection
func (s *Server) sendMessage(c *gnet.Conn, msg Msg) (ok bool) {
	s.Debugf("send message %T to %s", msg, c.Address())

	select {
	case c.SendQueue() <- Encode(msg):
		ok = true
	case <-c.Closed():
	default:
		s.Print("[ERR] %s send queue full", c.Address())
		c.Close()
	}
	return
}

func boolString(t bool, ts, fs string) string {
	if t {
		return ts
	}
	return fs
}

func (s *Server) connectHandler(c *gnet.Conn) {
	s.Debugf("got new %s connection %s %s",
		boolString(c.IsIncoming(), "incoming", "outgoing"),
		boolString(c.IsIncoming(), "from", "to"),
		c.Address())
	// handle
	s.await.Add(1)
	go s.handleConnection(c)
	// send feeds we are interesting in
	s.fmx.RLock()
	defer s.fmx.RUnlock()
	for f := range s.feeds {
		if !s.sendMessage(c, &AddFeedMsg{f}) {
			return
		}
	}
}

func (s *Server) close(c *gnet.Conn) {
	s.fmx.Lock()
	defer s.fmx.Unlock()
	c.Close()
	for _, cs := range s.feeds {
		delete(cs, c)
	}
}

func (s *Server) disconnectHandler(c *gnet.Conn) {
	s.Debugf("closed connection %s", c.Address())
}

func (s *Server) handleConnection(c *gnet.Conn) {
	s.Debug("handle connection ", c.Address())
	defer s.Debug("stop handling connection", c.Address())

	defer s.await.Done()
	defer s.close(c)

	var (
		closed  <-chan struct{} = c.Closed()
		receive <-chan []byte   = c.ReceiveQueue()

		data []byte
		msg  Msg

		err error
	)

	for {
		select {
		case <-closed:
			return
		case data = <-receive:
			if msg, err = Decode(data); err != nil {
				s.Printf("[ERR] %s decoding essage: %v", c.Address(), err)
				return
			}
			s.handleMsg(c, msg)
		}
	}

}

func shortHex(a string) string {
	return string([]byte(a)[:7])
}

func (s *Server) addFeedOfConn(c *gnet.Conn, feed cipher.PubKey) (added bool) {
	s.fmx.Lock()
	defer s.fmx.Unlock()
	if cs, ok := s.feeds[feed]; ok {
		if _, ok := cs[c]; ok {
			return // already
		}
		cs[c], added = struct{}{}, true
	}
	return
}

func (s *Server) handleAddFeedMsg(c *gnet.Conn, msg *AddFeedMsg) {
	if !s.addFeedOfConn(c, msg.Feed) {
		return
	}
	full := s.so.LastFullRoot(msg.Feed)
	if full == nil {
		return
	}
	s.sendMessage(c, &RootMsg{msg.Feed, full.Encode()})
}

func (s *Server) handleDelFeedMsg(c *gnet.Conn, msg *DelFeedMsg) {
	s.fmx.Lock()
	defer s.fmx.Unlock()
	if cs, ok := s.feeds[msg.Feed]; ok {
		delete(cs, c)
	}
	return
}

func (s *Server) hasFeed(pk cipher.PubKey) (yep bool) {
	s.fmx.RLock()
	defer s.fmx.RUnlock()
	_, yep = s.feeds[pk]
	return
}

func (s *Server) sendToFeed(feed cipher.PubKey, msg Msg, except *gnet.Conn) {
	s.fmx.RLock()
	defer s.fmx.RUnlock()
	cs, ok := s.feeds[feed]
	if !ok {
		return
	}
	for c := range cs {
		if c == except {
			continue
		}
		s.sendMessage(c, msg)
	}
}

func (s *Server) addNonFullRoot(root *skyobject.Root,
	c *gnet.Conn) (fl *fillRoot) {

	fl = &fillRoot{root, c, skyobject.Reference{}}
	s.roots = append(s.roots, fl)
	return
}

func (s *Server) delNonFullRoot(root *skyobject.Root) {
	for i, fl := range s.roots {
		if fl.root == root {
			copy(s.roots[i:], s.roots[i+1:])
			s.roots[len(s.roots)-1] = nil // set to nil for golang GC
			s.roots = s.roots[:len(s.roots)-1]
			return
		}
	}
	return
}

func (s *Server) handleRootMsg(c *gnet.Conn, msg *RootMsg) {
	if !s.hasFeed(msg.Feed) {
		return
	}
	root, err := s.so.AddRootPack(msg.RootPack)
	if err != nil {
		// TODO: high priority after database
		if err == skyobject.ErrAlreadyHaveThisRoot {
			s.Debug("reject root: alredy have this root")
			return
		}
		s.Print("[ERR] error decoding root: ", err)
		return
	}
	if root.IsFull() {
		s.sendToFeed(msg.Feed, msg, c)
		return
	}

	s.rmx.Lock()
	defer s.rmx.Unlock()

	fl := s.addNonFullRoot(root, c)
	if !root.HasRegistry() {
		if !s.sendMessage(c, &RequestRegistryMsg{root.RegistryReference()}) {
			s.delNonFullRoot(root) // sending error (connection closed)
		}
		return
	}
	err = root.WantFunc(func(ref skyobject.Reference) error {
		if !s.sendMessage(c, &RequestDataMsg{ref}) {
			s.delNonFullRoot(root) // sending error (connection closed)
		} else {
			fl.await = ref // keep last requested reference
		}
		return skyobject.ErrStopRange
	})
	if err != nil {
		s.Print("[ERR] unexpected error: ", err)
	}
}

func (s *Server) handleRequestRegistryMsg(c *gnet.Conn,
	msg *RequestRegistryMsg) {

	if reg, _ := s.so.Registry(msg.Ref); reg != nil {
		s.sendMessage(c, &RegistryMsg{reg.Encode()})
	}
}

func (s *Server) handleRegistryMsg(c *gnet.Conn, msg *RegistryMsg) {
	reg, err := skyobject.DecodeRegistry(msg.Reg)
	if err != nil {
		s.Print("[ERR] error decoding received registry:", err)
		return
	}

	if !s.so.WantRegistry(reg.Reference()) {
		return // don't want the registry
	}

	s.so.AddRegistry(reg)

	s.rmx.Lock()
	defer s.rmx.Unlock()
	var i int = 0 // index for deleting
	for _, fl := range s.roots {
		if fl.root.RegistryReference() == reg.Reference() {
			if fl.root.IsFull() {
				s.sendToFeed(fl.root.Pub(), &RootMsg{
					Feed:     fl.root.Pub(),
					RootPack: fl.root.Encode(),
				}, fl.c)
				continue // delete
			}
			var sent bool
			err = fl.root.WantFunc(func(ref skyobject.Reference) error {
				if sent = s.sendMessage(c, &RequestDataMsg{ref}); sent {
					fl.await = ref
				}
				return skyobject.ErrStopRange
			})
			if err != nil {
				s.Print("[ERR] unexpected error: ", err)
				continue // delete
			}
			if !sent {
				continue // delete
			}
		}
		s.roots[i] = fl
		i++
	}
	s.roots = s.roots[:i]
}

func (s *Server) handleRequestDataMsg(c *gnet.Conn, msg *RequestDataMsg) {
	if data, ok := s.so.Get(msg.Ref); ok {
		s.sendMessage(c, &DataMsg{data})
	}
}

func (s *Server) handleDataMsg(c *gnet.Conn, msg *DataMsg) {
	hash := skyobject.Reference(cipher.SumSHA256(msg.Data))

	s.rmx.Lock()
	defer s.rmx.Unlock()

	// does the Server really want the data
	var want bool
	for _, fl := range s.roots {
		if fl.await == hash {
			want = true
			break
		}
	}
	if !want {
		return // doesn't want the data
	}
	s.so.Set(hash, msg.Data) // save

	// check filling
	var i int = 0 // index for deleting
	for _, fl := range s.roots {
		if fl.await == hash {
			if fl.root.IsFull() {
				s.sendToFeed(fl.root.Pub(), &RootMsg{
					Feed:     fl.root.Pub(),
					RootPack: fl.root.Encode(),
				}, fl.c)
				continue // delete
			}
			var sent bool
			err := fl.root.WantFunc(func(ref skyobject.Reference) error {
				if sent = s.sendMessage(c, &RequestDataMsg{ref}); sent {
					fl.await = ref
				}
				return skyobject.ErrStopRange
			})
			if err != nil {
				s.Print("[ERR] unexpected error: ", err)
				continue // delete
			}
			if !sent {
				continue // delete
			}
		}
		s.roots[i] = fl
		i++
	}
	s.roots = s.roots[:i]
}

func (s *Server) handlePingMsg(c *gnet.Conn) {
	s.sendMessage(c, &PongMsg{})
}

func (s *Server) handleMsg(c *gnet.Conn, msg Msg) {
	s.Debugf("handle message %T from %s", msg, c.Address())

	switch x := msg.(type) {
	case *AddFeedMsg:
		s.handleAddFeedMsg(c, x)
	case *DelFeedMsg:
		s.handleDelFeedMsg(c, x)
	case *RootMsg:
		s.handleRootMsg(c, x)
	case *RequestRegistryMsg:
		s.handleRequestRegistryMsg(c, x)
	case *RegistryMsg:
		s.handleRegistryMsg(c, x)
	case *RequestDataMsg:
		s.handleRequestDataMsg(c, x)
	case *DataMsg:
		s.handleDataMsg(c, x)
	case *PingMsg:
		s.handlePingMsg(c)
	case *PongMsg:
		// do nothing
	default:
		s.Printf("[CRIT] unhandled message type %T", msg)
	}
}

//
// Public methods of the Server
//

func (s *Server) Connect(address string) (err error) {
	_, err = s.pool.Dial(address)
	return
}

func (s *Server) Disconnect(address string) (err error) {
	cx := s.pool.Connection(address)
	if cx == nil {
		err = fmt.Errorf("connection not found %q", address)
		return
	}
	err = cx.Close()
	return
}

func (s *Server) Connections() []*gnet.Conn {
	return s.pool.Connections()
}

func (s *Server) Connection(address string) *gnet.Conn {
	return s.pool.Connection(address)
}

func (s *Server) broadcast(msg Msg) {
	// todo: modify gnet.(*Pool).Connections() to return *gnet.Conn
	//       instead of list of addresses
	for _, c := range s.pool.Connections() {
		s.sendMessage(c, msg)
	}
}

// AddFeed adds the feed to list of feeds, the Server share, and
// sends root object of the feed to subscribers
func (s *Server) AddFeed(f cipher.PubKey) (added bool) {
	s.fmx.Lock()
	defer s.fmx.Unlock()
	if _, ok := s.feeds[f]; !ok {
		s.feeds[f], added = make(map[*gnet.Conn]struct{}), true
		s.broadcast(&AddFeedMsg{f})
	}
	return
}

// DelFeed stops sharing given feed
func (s *Server) DelFeed(f cipher.PubKey) (deleted bool) {
	s.fmx.Lock()
	defer s.fmx.Unlock()
	if _, ok := s.feeds[f]; ok {
		delete(s.feeds, f)
		s.broadcast(&DelFeedMsg{f})
		// delete from filling
		s.rmx.Lock()
		defer s.rmx.Unlock()
		var i int = 0
		for _, fl := range s.roots {
			if fl.root.Pub() == f {
				continue // delete
			}
			i++
			s.roots[i] = fl
		}
		s.roots = s.roots[:i]
		// delete from skyobject
		s.so.DelFeed(f)
		deleted = true
	}
	return
}

// TODO: + Want per root of a feed

// Want returns lits of objects related to given
// feed that the server hasn't got but knows about
func (s *Server) Want(feed cipher.PubKey) (wn []cipher.SHA256, err error) {
	set := make(map[skyobject.Reference]struct{})
	err = s.so.WantFeed(feed, func(k skyobject.Reference) error {
		set[k] = struct{}{}
		return nil
	})
	if err != nil {
		return
	}
	if len(set) == 0 {
		return
	}
	wn = make([]cipher.SHA256, 0, len(set))
	for k := range set {
		wn = append(wn, cipher.SHA256(k))
	}
	return
}

// TODO: + Got per root of a feed

// Got returns lits of objects related to given
// feed that the server has got
func (s *Server) Got(feed cipher.PubKey) (gt []cipher.SHA256, err error) {
	set := make(map[skyobject.Reference]struct{})
	err = s.so.GotFeed(feed, func(k skyobject.Reference) error {
		set[k] = struct{}{}
		return nil
	})
	if err != nil {
		return
	}
	if len(set) == 0 {
		return
	}
	gt = make([]cipher.SHA256, 0, len(set))
	for k := range set {
		gt = append(gt, cipher.SHA256(k))
	}
	return
}

// Feeds the server share
func (s *Server) Feeds() (fs []cipher.PubKey) {
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

// Stat returns satatistic of database
func (s *Server) Stat() data.Stat {
	return s.so.DB().Stat()
}

// Quititng returns cahnnel that closed
// when the Server closed
func (s *Server) Quiting() <-chan struct{} {
	return s.done // when quit done
}
