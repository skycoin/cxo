package node

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject"
)

func TestNewClient(t *testing.T) {
	defer clean()
	t.Run("memory db", func(t *testing.T) {
		clean()
		conf := newClientConfig() // in-memory
		c, err := NewClient(conf, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
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
		conf := newClientConfig()
		conf.InMemoryDB = false
		c, err := NewClient(conf, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
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
		conf := newClientConfig()
		conf.InMemoryDB = false
		conf.DBPath = filepath.Join(testDataDir, "another.db")
		c, err := NewClient(conf, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if _, err := os.Stat(conf.DataDir); err != nil {
			t.Error(err)
		} else {
			if _, err := os.Stat(conf.DBPath); err != nil {
				t.Error(err)
			}
		}
	})
}

func TestClient_Start(t *testing.T) {
	// Start(address string) (err error) {
	t.Run("invalid address", func(t *testing.T) {
		c, err := newClient(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if err = c.Start("domain dot tld"); err == nil {
			t.Error("mising error")
		}
	})
	t.Run("connect", func(t *testing.T) {
		s, err := newRunninServer(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		c, err := newClient(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if err = c.Start(s.pool.Address()); err != nil {
			t.Error(err)
		}
	})
}

func TestClient_Close(t *testing.T) {
	// Close() (err error) {

	// TODO: low priority
}

func TestClient_IsConnected(t *testing.T) {
	// IsConnected() bool

	// TODO: low priority
}

func TestClient_Subscribe(t *testing.T) {
	// Subscribe(feed cipher.PubKey) (ok bool) {

	pk, sk := cipher.GenerateKeyPair()

	// 1. sending AddFeed
	// 2. sending last full root
	// 3. already subscribed

	t.Run("AddFeed sending", func(t *testing.T) {
		conf := newClientConfig()
		c, s, err := newRunningClient(conf, nil, []cipher.PubKey{pk})
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		defer c.Close()
		if !c.Subscribe(pk) {
			t.Fatal("can't subscribe")
		}

		time.Sleep(TM) // sleep....

		s.fmx.Lock()
		defer s.fmx.Unlock()
		if cs, ok := s.feeds[pk]; !ok {
			t.Error("misisng or slow")
		} else {
			if len(cs) != 1 {
				t.Error("misisng or slow")
			}
		}

	})

	t.Run("last full sending", func(t *testing.T) {

		type User struct{ Name string }

		reg := skyobject.NewRegistry()
		reg.Register("User", User{})

		c, err := newClient(reg)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		s, err := newRunninServer([]cipher.PubKey{pk})
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		root, err := c.so.NewRoot(pk, sk)
		if err != nil {
			t.Fatal(err)
		}

		if _, _, err := root.Inject("User", User{"Alice"}); err != nil {
			t.Fatal(err)
		}

		if err := c.Start(s.pool.Address()); err != nil {
			t.Fatal(err)
		}

		time.Sleep(TM) // sleep....

		if s.so.LastFullRoot(pk) == nil {
			t.Error("misisng or slow")
		}

	})

	t.Run("already subscribed", func(t *testing.T) {
		c, err := newClient(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		if !c.Subscribe(pk) {
			t.Error("can't subscribe")
		} else if c.Subscribe(pk) {
			t.Error("subscribed twice")
		}
	})

}

func TestClient_Unsubscribe(t *testing.T) {
	// Unsubscribe(feed cipher.PubKey, del bool) (ok bool) {

}

func TestClient_Feeds(t *testing.T) {
	// Feeds() (feeds []cipher.PubKey) {

}

func TestClient_Container(t *testing.T) {
	// Container() *Container {

}
