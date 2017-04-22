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

	s.pool = gnet.NewPool(sc.Config)
	s.feeds = nil

	if sc.EnableRPC == true {
		s.rpc = newRPC(s, sc.RPCAddress)
	}

	s.quit = make(chan struct{})

	return
}
