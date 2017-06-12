package node

import (
	"os"
	"path/filepath"
	//"strings"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	// "github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/skyobject"
)

func TestNewNode(t *testing.T) {
	// NewNode(sc NodeConfig) (s *Node, err error)

	// registry must be nil
	t.Run("registry", func(t *testing.T) {
		s, err := newNode(newNodeConfig(false))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		if s.Container().CoreRegistry() != nil {
			t.Error("unexpected core registry")
		}
	})

	defer clean()

	// if datbase in memeory then Node doesn't
	// creates DataDir and datbase
	t.Run("memory db", func(t *testing.T) {
		clean()

		conf := newNodeConfig(false) // in-memory

		s, err := newNode(conf)
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

	// If database is not in memory, then
	// DataDir must be created even if DBPath
	// points to another directory
	t.Run("data dir", func(t *testing.T) {
		clean()

		conf := newNodeConfig(false)
		conf.InMemoryDB = false

		s, err := newNode(conf)
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

		conf := newNodeConfig(false)
		conf.InMemoryDB = false
		conf.DBPath = filepath.Join(testDataDir, "another.db")

		s, err := newNode(conf)
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

	// ----
	// loading feeds from datbase tested below in
	//    TestNewNode_loadingFeeds
	// ----

}

func TestNewNodeReg(t *testing.T) {
	// NewNodeReg(sc NodeConfig, reg *skyobject.Registry) (s *Node, err error)

	// registry must be the same
	t.Run("registry", func(t *testing.T) {
		reg := skyobject.NewRegistry()

		s, err := newNodeReg(newNodeConfig(false), reg)
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		if s.Container().CoreRegistry() != reg {
			t.Error("wrong or missing registry")
		}
	})

	t.Run("invalid gnet.Config", func(t *testing.T) {
		conf := newNodeConfig(false)
		conf.Config.DialTimeout = -1000
		if s, err := newNode(conf); err == nil {
			t.Error("missing error")
			s.Close()
		}
	})

}

func TestNode_Start(t *testing.T) {
	// Start() (err error)

	t.Run("disable listener", func(t *testing.T) {
		s, err := newNode(newNodeConfig(false))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		if err = s.Start(); err != nil {
			t.Fatal(err)
		}

		if s.pool.Address() != "" {
			t.Error("listens on")
		}
	})

	t.Run("enable listener", func(t *testing.T) {
		s, err := newNode(newNodeConfig(true))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		if err = s.Start(); err != nil {
			t.Fatal(err)
		}

		if s.pool.Address() == "" {
			t.Error("doesn't listen on")
		}
	})

	t.Run("disable RPC listener", func(t *testing.T) {
		s, err := newNode(newNodeConfig(false))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		if err = s.Start(); err != nil {
			t.Fatal(err)
		}

		if s.rpc != nil {
			t.Error("RPC was created")
		}
	})

	t.Run("enable RPC listener", func(t *testing.T) {
		conf := newNodeConfig(false)
		conf.EnableRPC = true

		s, err := newNode(conf)
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		if err = s.Start(); err != nil {
			t.Fatal(err)
		}

		if s.rpc == nil {
			t.Error("RPC wasn't created")
		}
	})
}

func TestNode_Close(t *testing.T) {
	// Close() (err error)

	t.Run("close listener", func(t *testing.T) {
		s, err := newNode(newNodeConfig(true))
		if err != nil {
			t.Fatal(err)
		}
		if err = s.Start(); err != nil {
			t.Fatal(err)
		}
		s.Close()

		if err = s.pool.Listen(""); err == nil {
			t.Error("missing error")
		} else if err != gnet.ErrClosed {
			t.Error("unexpected error:", err)
		}
	})

	t.Run("close RPC listener", func(t *testing.T) {
		conf := newNodeConfig(false)
		conf.EnableRPC = true

		s, err := newNode(conf)
		if err != nil {
			t.Fatal(err)
		}
		if err = s.Start(); err != nil {
			t.Fatal(err)
		}
		s.Close()

		// TODO: can block if not closed
		if c, err := s.rpc.l.Accept(); err == nil {
			t.Error("misisng error")
			c.Close()
		}
	})

	t.Run("twice", func(t *testing.T) {
		defer clean()

		conf := newNodeConfig(true)
		conf.EnableRPC = true
		conf.InMemoryDB = false // force to use BoltDBs

		s, err := newNode(conf)
		if err != nil {
			t.Fatal(err)
		}
		if err = s.Start(); err != nil {
			t.Fatal(err)
		}
		if err = s.Close(); err != nil {
			t.Error(err)
		}
		if err = s.Close(); err != nil {
			t.Error(err)
		}
	})

}

func TestNode_Pool(t *testing.T) {
	// Pool() (pool *gnet.Pool)

	s, err := newNode(newNodeConfig(false))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.Pool() == nil {
		t.Fatal("mising pool")
	}
}

func TestNode_Subscribe(t *testing.T) {
	// Subscribe(c *gnet.Conn, feed cipher.PubKey)

	t.Run("local", func(t *testing.T) {
		s, err := newNode(newNodeConfig(false))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		pk, _ := cipher.GenerateKeyPair()

		s.Subscribe(nil, pk)

		// susbcriptions of the Node

		if feeds := s.Feeds(); len(feeds) != 1 {
			t.Error("misisng feed")
		} else if feeds[0] != pk {
			t.Error("wrong feed subscribed to")
		}

		// database

		if feeds := s.Container().DB().Feeds(); len(feeds) != 1 {
			t.Error("misisng feed")
		} else if feeds[0] != pk {
			t.Error("wrong feed subscribed to")
		}

	})

	t.Run("remote reject", func(t *testing.T) {
		pk, _ := cipher.GenerateKeyPair()

		aconf := newNodeConfig(false)
		bconf := newNodeConfig(false)

		reject := make(chan *gnet.Conn, 1)

		aconf.OnSubscriptionAccepted = func(_ *gnet.Conn, _ cipher.PubKey) {
			t.Error("accepted")
		}
		aconf.OnSubscriptionDenied = func(c *gnet.Conn, feed cipher.PubKey) {
			if feed != pk {
				t.Error("wrong feed rejected")
			}
			reject <- c // to test conenction
		}

		a, b, ac, _, err := newConnectedNodes(aconf, bconf)
		if err != nil {
			t.Fatal(err)
		}
		defer a.Close()
		defer b.Close()

		a.Subscribe(ac, pk)

		// remote peer must reject the subscription because the
		// remote peer doesn't share pk feed

		select {
		case c := <-reject:
			if c != ac {
				t.Error("rejected from wrong connection")
			}
		case <-time.After(TM):
			t.Error("slow")
		}

	})

	t.Run("remote accept", func(t *testing.T) {
		pk, _ := cipher.GenerateKeyPair()

		aconf := newNodeConfig(false)
		bconf := newNodeConfig(false)

		accept := make(chan *gnet.Conn, 1)

		aconf.OnSubscriptionAccepted = func(c *gnet.Conn, feed cipher.PubKey) {
			if feed != pk {
				t.Error("wrong feed accepted")
			}
			accept <- c // to test conenction
		}
		aconf.OnSubscriptionDenied = func(_ *gnet.Conn, _ cipher.PubKey) {
			t.Error("accepted")
		}

		a, b, ac, _, err := newConnectedNodes(aconf, bconf)
		if err != nil {
			t.Fatal(err)
		}
		defer a.Close()
		defer b.Close()

		b.Subscribe(nil, pk)
		a.Subscribe(ac, pk)

		// remote peer must accept the subscription because the
		// remote peer shares pk feed

		select {
		case c := <-accept:
			if c != ac {
				t.Error("rejected from wrong connection")
			}
		case <-time.After(TM):
			t.Error("slow")
		}

	})

	t.Run("remote accept twice", func(t *testing.T) {
		pk, _ := cipher.GenerateKeyPair()

		aconf := newNodeConfig(false)
		bconf := newNodeConfig(false)

		accept := make(chan *gnet.Conn, 1)

		aconf.OnSubscriptionAccepted = func(c *gnet.Conn, feed cipher.PubKey) {
			if feed != pk {
				t.Error("wrong feed accepted")
			}
			accept <- c // to test conenction
		}
		aconf.OnSubscriptionDenied = func(_ *gnet.Conn, _ cipher.PubKey) {
			t.Error("accepted")
		}

		a, b, ac, _, err := newConnectedNodes(aconf, bconf)
		if err != nil {
			t.Fatal(err)
		}
		defer a.Close()
		defer b.Close()

		b.Subscribe(nil, pk)
		a.Subscribe(ac, pk)

		// remote peer must accept the subscription because the
		// remote peer shares pk feed

		select {
		case c := <-accept:
			if c != ac {
				t.Error("rejected from wrong connection")
			}
		case <-time.After(TM):
			t.Error("slow")
		}

		if t.Failed() {
			return // don't need to continue
		}

		// second subscription must not send SusbcribeMsg twice
		a.Subscribe(ac, pk)
		select {
		case c := <-accept:
			if c != ac {
				t.Error("rejected from wrong connection")
			}
		case <-time.After(TM):
			// the test is not strong, but it's fast,
			// if first part of the test passes then this
			// part should work in many cases
		}

	})

}

func TestNode_Unsusbcribe(t *testing.T) {
	// Unsubscribe(c *gnet,.Conn, feed cipher.PubKey)

	t.Run("local", func(t *testing.T) {
		s, err := newNode(newNodeConfig(false))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		pk, _ := cipher.GenerateKeyPair()

		s.Subscribe(nil, pk)

		// susbcriptions of the Node
		if feeds := s.Feeds(); len(feeds) != 1 && feeds[0] != pk {
			t.Error("somethig wrong with Susbcribe")
			return
		}

		s.Unsubscribe(nil, pk)

		if len(s.Feeds()) != 0 {
			t.Error("don't unsusbcribes")
		}
	})
}

func TestNewNode_loadingFeeds(t *testing.T) {
	// TODO: high priority

	//
}
