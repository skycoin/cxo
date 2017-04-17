package node

import (
	"flag"
	"net"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/skyobject"
)

// defaults
const (
	LISTEN       string = "127.0.0.1:9987"
	REMOTE_CLOSE bool   = false
)

type Config struct {
	gnet.Config // pool configurations

	Listen string          // listen on
	Known  []string        // known hosts
	Feeds  []cipher.PubKey // feeds to share

	RemoteClose bool // allow closing using CLI
}

func NewConfig() (c Config) {
	c.Config = gnet.NewConfig()
	c.Listen = LISTEN
	c.RemoteClose = REMOTE_CLOSE
	return
}

func (c *Config) FromFlags() {
	c.Config.FromFlags()
	flag.StringVar(&c.Listen,
		"listen",
		c.Listen,
		"listening address")
	flag.BoolVar(&c.RemoteClose,
		"remote-close",
		c.RemoteClose,
		"allow closing using CLI")
}

type Server struct {
	log.Logger

	conf Config

	// skyobjects
	db *data.DB
	so *skyobject.Container

	// network
	pool *gnet.Pool

	// feeds the server share and subscribed to
	share []cipher.PubKey

	// address of server -> feeds the server share
	servers map[string][]cipher.PubKey
	// address of client -> feeds the client subscribed to
	clients map[string][]cipher.PubKey

	pending []string // pending connections

	// new connections
	connect chan *gnet.Conn

	// administration
	rpc chan rpcEvent

	await sync.WaitGroup // await closing
	quito sync.Once      // close once
	quit  chan struct{}  // shutdown
}

func NewServer(c Config, db *data.DB, so *skyobject.Container) (s *Server) {
	if db == nil {
		panic("nil db")
	}
	if so == nil {
		panic("nil skyobject")
	}
	s = new(Server)
	s.conf = Config
	s.db = db
	s.so = so
	s.servers = make(map[string][]cipher.PubKey)
	s.clients = make(map[string][]cipher.PubKey)
	s.connect = make(chan *gnet.Conn, 64) // todo: configure
	s.rpc = make(chan rpcEvent, 10)       // todo: configure
	c.Config.ConnectionHandler = s.onConnect
	c.Config.DisconnectHandler = s.onDisconnect
	s.pool = gnet.NewPool(c.Config)
	// register messages to send-receive
	s.pool.Register(gnet.NewPrefix("SRCN"), ServerConnect{})
	s.pool.Register(gnet.NewPrefix("SRSC"), ServerSync{})
	s.pool.Register(gnet.NewPrefix("SRQT"), ServerQuit{})
	s.pool.Register(gnet.NewPrefix("CLSC"), ClientSync{})
	s.pool.Register(gnet.NewPrefix("ROOT"), Root{})
	s.pool.Register(gnet.NewPrefix("RQST"), Request{})
	s.pool.Register(gnet.NewPrefix("DATA"), Data{})
	//
	s.share = s.conf.Feeds // feeds to share
	// register known as servers
	for _, address := range s.conf.Known {
		s.servers[address] = nil
	}
	return
}

func (s *Server) Start() (err error) {
	if err = s.pool.Listen(s.conf.Listen); err != nil {
		return
	}
	s.await.Add(2)
	go s.handle(s.quit, s.done, s.pool.Receive(), s.connect, s.rpc)
	go s.connectToKnown()
	return
}

func (s *Server) onConnect(c *gnet.Conn) {
	select {
	case s.connect <- c:
	case <-s.quit:
	}
}

func (s *Server) onDisconnect(c *gnet.Conn, err error) {
	// remove from pending
	ca := c.Addr()
	for i, a := range s.pending {
		if a == ca {
			s.pending = append(s.pending[:i], s.pending[i+1:]...)
			break
		}
	}
	if c.IsIncoming() {
		return // can't redial
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		// TODO: try redial N-times
	}
}

func (s *Server) handle(quit <-chan struct{}, inflow <-chan gnet.Message,
	connect <-chan *gnet.Conn, rpc <-chan rpcEvent) {

	var (
		m gnet.Message
		r rpcEvent
		c *gnet.Conn // new connection
	)

	defer s.await.Done()
	defer s.pool.Close()

	for {
		select {
		case c = <-connect:
			if c.IsOutgoing() { // connection to another server
				c.Send(&ServerConnect{Feeds: s.share})
			} else {
				s.pending = append(s.pending, c.Addr())
			}
		case r = <-rpc:
			r()
		case m = <-inflow:
			s.handleMessage(m)
		case <-quit:
			return
		}
	}

}

func (s *Server) handleMessage(m gnet.Message) {
	switch x := m.Value.(type) {
	case *ServerConnect:
		if _, ok := s.servers[m.Conn.Addr()]; !ok {
			// probably the connection is not a server
			s.Print("[ERR] unknown server connects: ", m.Conn.Addr())
			m.Conn.Close()
			return
		}
		// okay, the server is registered as a server
		s.servers[m.Conn.Addr()] = x.Feeds // store its feeds
		m.Conn.Send(&ServerSync{s.share})  // reply with own feeds
		// and send root objects we have and remote server want
		for _, pub := range s.share {
			if !contains(x.Feeds, pub) {
				continue
			}
			root := s.so.Root(pub)
			if root == nil {
				continue
			}
			m.Conn.Send(&Root{
				Feed: pub,
				Sig:  root.Sig,
				Root: root.Encode(),
			})
		}
	case *ServerSync:
		//
	case *ServerQuit:
		//
	case *ClientSync:
		//
	case *Root:
		//
	case *Request:
		//
	case *Data:
		//
	}
}

func contains(haystack []cipher.PubKey, needle cipher.PubKey) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
