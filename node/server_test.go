package node

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

func TestNewServer(t *testing.T) {
	// NewServer(sc ServerConfig) (s *Server, err error)

	// 1. memory db
	// 2. data dir
	// 3. database path

	defer clean()

	t.Run("memory db", func(t *testing.T) {
		clean()

		conf := newServerConfig() // in-memory

		s, err := NewServer(conf)
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		if _, err := os.Stat(conf.DataDir); err == nil {
			t.Error("unexpected data dir:", conf.DataDir)
			if _, err = os.Stat(conf.DBPath); err == nil {
				t.Error("unexpected db file:", conf.DBPath)
			} else if !os.IsNotExist(err) {
				t.Error("unexpected error")
			}
		} else if !os.IsNotExist(err) {
			t.Error("unexpected error:", err)
		}

	})

	t.Run("data dir", func(t *testing.T) {
		clean()

		conf := newServerConfig()
		conf.InMemoryDB = false

		s, err := NewServer(conf)
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		if _, err := os.Stat(conf.DataDir); err != nil {
			t.Error(err)
		} else {
			if _, err := os.Stat(conf.DBPath); err != nil {
				t.Error(err)
			}
		}

	})

	t.Run("dbpath", func(t *testing.T) {
		clean()

		conf := newServerConfig()
		conf.InMemoryDB = false
		conf.DBPath = filepath.Join(testDataDir, "another.db")

		s, err := NewServer(conf)
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		if _, err := os.Stat(conf.DataDir); err != nil {
			t.Error(err)
		} else {
			if _, err := os.Stat(conf.DBPath); err != nil {
				t.Error(err)
			}
		}

	})

}

func TestNewServerSoDB(t *testing.T) {
	// NewServerSoDB(sc ServerConfig, db data.DB,
	//     so *skyobject.Container) (s *Server, err error)

	// 1. db != nil
	// 2. so != nil
	// 3. invalig gnet.Config

	t.Run("nil db", func(t *testing.T) {
		defer shouldPanic(t)
		NewServerSoDB(ServerConfig{}, nil, nil)
	})

	t.Run("nil so", func(t *testing.T) {
		defer shouldPanic(t)
		NewServerSoDB(ServerConfig{}, data.NewMemoryDB(), nil)
	})

	t.Run("invalid gnet.Config", func(t *testing.T) {
		conf := newServerConfig()
		conf.Config.DialTimeout = -1000
		db := data.NewMemoryDB()
		so := skyobject.NewContainer(db, nil)
		_, err := NewServerSoDB(conf, db, so)
		if err == nil {
			t.Error("missing error")
		}
	})

}

func TestServer_Start(t *testing.T) {
	// Start() (err error)

	// 1. arbitrary address
	// 2. provided address

	t.Run("arbitrary", func(t *testing.T) {
		conf := newServerConfig()
		conf.Listen = ""

		s, err := newServerConf(conf)
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		if err := s.Start(); err != nil {
			t.Error(err)
		}
	})

	t.Run("provided", func(t *testing.T) {
		conf := newServerConfig()
		conf.Listen = "[::]:8987"

		s, err := newServerConf(conf)
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		if err := s.Start(); err != nil {
			if !strings.Contains(err.Error(), "address already in use") {
				t.Error(err)
			}
			return
		}

		if a := s.pool.Address(); a != "[::]:8987" {
			t.Error("wrong listening address:", a)
		}
	})

}

func TestServer_Close(t *testing.T) {
	// Close() (err error)

	//

}

func TestServer_Connect(t *testing.T) {
	// Connect(address string) (err error)

	//

}

func TestServer_Disconnect(t *testing.T) {
	// Disconnect(address string) (err error)

	//

}

func TestServer_Connections(t *testing.T) {
	// Connections() []*gnet.Conn

	//

}

func TestServer_Connection(t *testing.T) {
	// Connection(address string) *gnet.Conn

	//

}

func TestServer_AddFeed(t *testing.T) {
	// AddFeed(f cipher.PubKey) (added bool)

	//

}

func TestServer_DelFeed(t *testing.T) {
	// DelFeed(f cipher.PubKey) (deleted bool)

	//

}

func TestServer_Want(t *testing.T) {
	// Want(feed cipher.PubKey) (wn []cipher.SHA256)

	//

}

func TestServer_Got(t *testing.T) {
	// Got(feed cipher.PubKey) (gt []cipher.SHA256)

	//

}

func TestServer_Feeds(t *testing.T) {
	// Feeds() (fs []cipher.PubKey)

	//

}

func TestServer_Stat(t *testing.T) {
	// Stat() stat.Stat

	//

}

func TestServer_Quiting(t *testing.T) {
	// Quiting() <-chan struct{}

	//

}
