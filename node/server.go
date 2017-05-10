package node

import (
	"fmt"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"

	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"
)

type feed struct {
	conns []*gnet.Conn // connections of the feed
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

	// database
	db *data.DB

	// skyobject
	so *skyobject.Container

	// feeds
	fmx   sync.RWMutex
	feeds map[cipher.PubKey]*feed

	// connections
	pool *gnet.Pool
	rpc  *RPC // rpc server

	// closing
	quit  chan struct{}
	quito sync.Once
	await sync.WaitGroup
}

// NewServer creates new Server instnace using given
// configurations. The functions creates database and
// Container of skyobject instances internally
func NewServer(sc ServerConfig) (s *Server, err error) {
	var db *data.DB = data.NewDB()
	s, err = NewServerSoDB(sc, db, skyobject.NewContainer(nil))
	return
}

// NewServerSoDB creates new Server instance using given
// configurations, database and Container of skyobject
// instances. Th functions panics if database of Contaner
// are nil
func NewServerSoDB(sc ServerConfig, db *data.DB,
	so *skyobject.Container) (s *Server, err error) {

	if db == nil {
		panic("nil db")
	}
	if so == nil {
		panic("nil db")
	}

	s = new(Server)

	s.Logger = log.NewLogger(sc.Log.Prefix, sc.Log.Debug)
	s.conf = sc

	s.db = db
	s.so = so
	s.feeds = make(map[cipher.PubKey]*feed)

	sc.Config.Logger = s.Logger // use the same logger
	sc.Config.ConnectionHandler = s.connectHandler
	sc.Config.DisconnectHandler = s.disconnectHandler
	if s.pool, err = gnet.NewPool(sc.Config); err != nil {
		s = nil
		return
	}

	if sc.EnableRPC == true {
		s.rpc = newRPC(s)
	}

	s.quit = make(chan struct{})

	return
}

// Start the server
func (s *Server) Start() (err error) {
	s.Debugf(`strting server:
    max connections:      %d
    max message size:     %d

    dial timeout:         %v
    read timeout:         %v
    write timeout:        %v

    read queue:           %d
    write queue:          %d

    redial timeout:       %d
    max redial timeout:   %d
    redials limit:        %d

    read buffer:          %d
    write buffer:         %d

    TLS:                  %t

    enable RPC:           %t
    RPC address:          %s
    lListening address:   %s
    remote close:         %t

    debug:                %t
`,
		s.conf.MaxConnections,
		s.conf.MaxMessageSize,
		s.conf.DialTimeout,
		s.conf.ReadTimeout,
		s.conf.WriteTimeout,
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

	return
}

// Close the server
func (s *Server) Close() (err error) {
	err = s.pool.Close()
	if s.conf.EnableRPC == true {
		s.rpc.Close()
	}
	s.await.Wait()
	s.quito.Do(func() {
		close(s.quit)
	})
	return
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
FeedsLoop:
	for _, f := range s.feeds {
		for i, cx := range f.conns {
			if cx == c {
				f.conns = append(f.conns[:i], f.conns[i+1:]...) // del
				continue FeedsLoop
			}
		}
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

func (s *Server) handleMsg(c *gnet.Conn, msg Msg) {
	s.Debugf("handle message %T from %s", msg, c.Address())

	switch x := msg.(type) {
	case *AddFeedMsg:
		ca := c.Address() // address of the connection (for debug logs)
		s.fmx.Lock()
		defer s.fmx.Unlock()
		if f, ok := s.feeds[x.Feed]; ok {
			// add to feeds
			for _, cx := range f.conns {
				if cx == c {
					s.Debug("already have the connection ", ca)
					return // already have the connection
				}
			}
			s.Debug("add connection to requested feed list ", ca)
			f.conns = append(f.conns, c)
			// send root to the connectiosn if we have the root
			root := s.so.LastRoot(x.Feed)
			if root == nil {
				s.Debug("don't have a root of the feed, ", ca)
				return
			}
			s.Debug("send root of the feed, ", ca)
			p, sig := root.Encode()
			s.sendMessage(c, &RootMsg{x.Feed, sig, p})
			return
		}
		s.Debug("don't share the feed connection request for, ", ca)
	case *DelFeedMsg:
		ca := c.Address() // for debug logs
		s.fmx.Lock()
		defer s.fmx.Unlock()
		if f, ok := s.feeds[x.Feed]; ok {
			for i, cx := range f.conns {
				if cx == c {
					s.Debug("delete connection from the feed, ", ca)
					f.conns = append(f.conns[:i], f.conns[i+1:]...) // delete
					return
				}
			}
		}
		s.Debug("don't share the feed to delete ", ca)
	case *RootMsg:
		ca := c.Address()
		s.fmx.RLock()
		defer s.fmx.RUnlock()
		if f, ok := s.feeds[x.Feed]; ok {
			root, err := s.so.AddEncodedRoot(x.Root, x.Sig)
			if err != nil {
				s.Print("[ERR] %s error decoding root: %v", ca, err)
				c.Close() // fatal
				return
			}
			if root == nil {
				s.Debug("older root received, ", ca)
				return // older root object received
			}
			if !root.HasRegistry() {
				// request registry
				s.sendMessage(c, &RequestRegistryMsg{
					Ref: root.RegistryReference(),
				})
			}
			s.Debugf("root of [%s] was updated by %s",
				shortHex(x.Feed.Hex()),
				ca)
			// send the new root to subscribers
			for _, cx := range f.conns {
				if cx == c {
					continue // skip connection from which the root received
				}
				s.sendMessage(cx, x)
			}
			return
		}
		s.Debug("don't share the feed root received for, ", ca)
	case *RequestRegistryMsg:
		if reg, _ := s.so.Registry(x.Ref); reg != nil {
			s.sendMessage(c, &RegistryMsg{
				Ref: x.Ref,
				Reg: reg.Encode(),
			})
		}
	case *RegistryMsg:
		if s.so.WantRegistry(x.Ref) {
			if reg, err := skyobject.DecodeRegistry(x.Reg); err != nil {
				s.Print("[ERR] error decoding registry: ", err)
				return
			} else if reg.Reference() != x.Ref {
				// reference not match registry
				s.Print("[ERR] received registry key-body missmatch")
				return
			} else {
				s.so.AddRegistry(reg)
			}
			// TODO: optimisation?
			// broadcast the Registry to all peers
			s.fmx.RLock()
			defer s.fmx.RUnlock()
			for _, f := range s.feeds {
				for _, cx := range f.conns {
					if cx == c {
						continue // skip this connection
					}
					s.sendMessage(cx, x)
				}
			}
		}
	case *RequestDataMsg:
		data, ok := c.so.Get(x.Ref)
		if !ok {
			return
		}
		c.sendMessage(&DataMsg{
			Feed: x.Feed,
			Data: data,
		})
	case *DataMsg:
		ca := c.Address() // for debug logs
		s.fmx.RLock()
		defer s.fmx.RUnlock()
		if f, ok := s.feeds[x.Feed]; ok {
			hash := skyobject.Reference(cipher.SumSHA256(x.Data))
			sent := false
			err := s.so.WantFeed(x.Feed, func(k skyobject.Reference) error {
				if k == hash {
					s.Debugf("add data [%s] %s",
						shortHex(hash.String()),
						ca)
					s.db.Set(cipher.SHA256(hash), x.Data)
					sent = true
					return skyobject.ErrStopRange
				}
				return nil
			})
			if err != nil {
				s.Print("[ERR] CRITICAL ERROR: ", err)
			}
			if !sent {
				s.Debugf("don't want received data [%s] %s",
					shortHex(hash.String()),
					ca)
			} else {
				// send the new data to subscribers
				for _, cx := range f.conns {
					if cx == c {
						continue // skip connection from which the data received
					}
					s.sendMessage(cx, x)
				}
			}
			return
		}
		s.Debug("don't share the feed data received for, ", ca)
	default:
		s.Printf("[CRIT] unlandled message type %T", msg)
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

func (s *Server) Connections() []string {
	return s.pool.Connections()
}

func (s *Server) Connection(address string) *gnet.Conn {
	return s.pool.Connection(address)
}

// AddFeed adds the feed to list of feeds, the Server share, and
// sends root object of the feed to subscribers
func (s *Server) AddFeed(f cipher.PubKey) (added bool) {
	s.fmx.Lock()
	defer s.fmx.Unlock()
	if _, ok := s.feeds[f]; !ok {
		s.feeds[f], added = &feed{}, true
	}
	return
}

// DelFeed stops sharing given feed
func (s *Server) DelFeed(f cipher.PubKey) (deleted bool) {
	s.fmx.Lock()
	defer s.fmx.Unlock()
	if _, ok := s.feeds[f]; ok {
		delete(s.feeds, f)
		deleted = true
	}
	s.so.DelFeed(f) // delete from skyobject
	return
}

// Want returns lits of objects related to given
// feed that the server hasn't but knows about
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

// database satatistic
func (s *Server) Stat() data.Stat {
	return s.db.Stat()
}

func (s *Server) Quiting() <-chan struct{} {
	return s.quit
}
