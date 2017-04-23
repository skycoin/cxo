package node

import (
	"errors"
	"strconv"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"

	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"
)

// A Server represents CXO server
// that includes RPC server if enabled
// by configs
type Server struct {
	sync.Mutex // concurant access to the so and the feeds
	log.Logger

	conf ServerConfig

	db *data.DB
	so *skyobject.Container

	pool  *gnet.Pool
	feeds []cipher.PubKey

	rpc *RPC

	quito sync.Once
	quit  chan struct{}
}

// NewServer creates new Server instnace using given
// configurations. The functions creates database and
// Container of skyobject instances internally
func NewServer(sc ServerConfig) (s *Server) {
	var db *data.DB = data.NewDB()
	s = NewServerSoDB(sc, db, skyobject.NewContainer(db))
	return
}

// NewServerSoDB creates new Server instance using given
// configurations, database and Container of skyobject
// instances. Th functions panics if database of Contaner
// are nil
func NewServerSoDB(sc ServerConfig, db *data.DB,
	so *skyobject.Container) (s *Server) {

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

	sc.Config.Logger = s.Logger // use the same logger
	sc.Config.ConnectionHandler = s.connectHandler
	sc.Config.DisconnectHandler = s.disconnectHandler
	s.pool = gnet.NewPool(sc.Config)

	s.feeds = nil

	if sc.EnableRPC == true {
		s.rpc = newRPC(s, sc.RPCAddress)
	}

	s.quit = make(chan struct{})

	return
}

func zeroString(x int) string {
	if x == 0 {
		return "no limit"
	}
	return strconv.Itoa(x)
}

// Start the server
func (s *Server) Start() (err error) {
	s.Debugf(`strting server:
    max connections:      %s
    max message size:     %s

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
		zeroString(s.conf.MaxConnections),
		zeroString(s.conf.MaxMessageSize),
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
	s.quito.Do(func() {
		close(s.quit)
	})
	err = s.pool.Close()
	if s.conf.EnableRPC == true {
		s.rpc.Close()
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
		c.Addr())
	go s.handle(c)
}

func (s *Server) disconnectHandler(c *gnet.Conn) {
	s.Debugf("closed connection %s", c.Addr())
}

// handle connection
func (s *Server) handle(c *gnet.Conn) {
	var (
		quit    <-chan struct{} = s.quit
		receive <-chan []byte   = c.ReceiveQueue()
		closed  <-chan struct{} = c.Closed()

		p   []byte
		msg Msg

		err error
	)

	for {
		select {
		case p = <-receive:
			if msg, err = Decode(p); err != nil {
				s.Print("[ERR] error decoding message:", err)
				c.Close()
			}
			s.handleMessage(c, msg)
		case <-closed:
			s.Printf("[INF] %s connection was closed", c.Address())
			return
		case <-quit:
			return
		}
	}
}

// blocking
func sendToConn(c *gnet.Conn, msg Msg) {
	select {
	case <-c.Cloed():
	case c.SendQueue() <- msg:
	}
}

func contains(f cipher.PubKey, feeds []cipher.PubKey) bool {
	for _, x := range feeds {
		if x == f {
			return true
		}
	}
	return false
}

func (s *Server) getRoot(feed cipher.PubKey) *skyobject.Root {
	s.Lock()
	defer s.Unlock()
	return s.so.Root(feed)
}

func (s *Server) handleMessage(c *gnet.Conn, msg Msg) {
	s.Debug("got message %T", msg)
	switch x := msg.(type) {
	case *AddFeedMsg:
		var feeds []cipher.PubKey
		if val := c.Value(); val != nil {
			feeds = val.([]cipher.PubKey)
		}
		if contains(x.Feed, feeds) {
			return // already
		}
		feeds = append(feeds, x.Feed)
		c.SetValue(feeds)
		root := s.getRoot(x.Feed)
		if root == nil {
			return // haven't got
		}
		sendToConn(c, &RootMsg{x.Feed, root.Sig, root.Encode()}) // can block
	case *DelFeedMsg:
		var feeds []cipher.PubKey
		if val := c.Value(); val != nil {
			feeds = val.([]cipher.PubKey)
		}
		for i, f := range feeds {
			if f == x.Feed {
				feeds = append(feeds[:i], feeds[i+1:]...) // delete
				c.SetValue(feeds)
				return
			}
		}
	case *RootMsg:
		//
	case *RequestMsg:
		for _, hash := range x.Hash {
			if data, ok := s.db.Get(hash); ok {
				sendToConn(c, &DataMsg{data})
			}
		}
	case *DataMsg:
		//
	}
}
