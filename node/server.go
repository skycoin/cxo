package node

import (
	"errors"
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

// Start the server
func (s *Server) Start() (err error) {
	s.Debugf(`strting server:
    max connections:      %d
    max message size:     %d
    dial timeout:         %v
    read timeout:         %v
    write timeout:        %v
    read buffer size:     %d
    write buffer size:    %d
    read queue size:      %d
    write queue size:     %d
    ping interval:        %d
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
		s.conf.ReadBufferSize,
		s.conf.WriteBufferSize,
		s.conf.ReadQueueSize,
		s.conf.WriteQueueSize,
		s.conf.PingInterval,
		s.conf.TLSConfig != nil,

		s.conf.EnableRPC,
		s.conf.RPCAddress,
		s.conf.Listen,
		s.conf.RemoteClose,

		s.conf.Log.Debug,
	)
	// start litener
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
	go s.handle()
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
}

func (s *Server) disconnectHandler(c *gnet.Conn) {
	s.Debugf("closed connection %s", c.Addr())
	/// TODO: reconnect by reason
}

func (s *Server) handle() {
	var (
		quit    <-chan struct{}     = s.quit
		receive <-chan gnet.Message = s.pool.Receive()

		m gnet.Message
	)

	for {
		select {
		case m = <-receive:
			s.handleMessage(m)
		case <-quit:
			return
		}
	}
}

func (s *Server) handleMessage(m gnet.Message) {
	s.Debug("got message %T", m.Value)
	select {
	//
	}
}
