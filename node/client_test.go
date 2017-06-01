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
	// Start(address string) (err error)

	// 1. invalid address
	// 2. valid address

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
	// Close() (err error)

	// TODO: low priority
}

func TestClient_IsConnected(t *testing.T) {
	// IsConnected() bool

	// TODO: low priority
}

func TestClient_Subscribe(t *testing.T) {
	// Subscribe(feed cipher.PubKey) (ok bool)

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

		root, err := c.so.NewRoot(pk, sk)
		if err != nil {
			t.Fatal(err)
		}

		if _, _, err := root.Inject("User", User{"Alice"}); err != nil {
			t.Fatal(err)
		}

		s, err := newRunninServer([]cipher.PubKey{pk})
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

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
	// Unsubscribe(feed cipher.PubKey, del bool) (ok bool)

	pk, sk := cipher.GenerateKeyPair()

	// 1. sending DelFeed
	// 2. deleting feed
	// 3. not subscribed

	t.Run("DelFeed sending", func(t *testing.T) {
		conf := newClientConfig()
		c, s, err := newRunningClient(conf, nil, []cipher.PubKey{pk})
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		defer c.Close()

		// subscribe first to be sure that DelFeed recived by server
		if !c.Subscribe(pk) {
			t.Fatal("can't subscribe")
		}

		time.Sleep(TM) // sleep....

		s.fmx.Lock()
		{
			if cs, ok := s.feeds[pk]; !ok {
				t.Fatal("misisng or slow")
			} else {
				if len(cs) != 1 {
					t.Fatal("misisng or slow")
				}
			}
		}
		s.fmx.Unlock()

		if !c.Unsubscribe(pk, false) {
			t.Fatal("can't unsubscribe")
		}

		time.Sleep(TM) // sleep....

		s.fmx.Lock()
		defer s.fmx.Unlock()
		if cs, ok := s.feeds[pk]; ok {
			if len(cs) != 0 {
				t.Error("unexpected or slow")
			}
		}
	})

	t.Run("deleting feed", func(t *testing.T) {

		type User struct{ Name string }

		reg := skyobject.NewRegistry()
		reg.Register("User", User{})

		c, err := newClient(reg)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		// subscribe first
		if !c.Subscribe(pk) {
			t.Fatal("can't subscribe")
		}

		root, err := c.so.NewRoot(pk, sk)
		if err != nil {
			t.Fatal(err)
		}

		if _, _, err := root.Inject("User", User{"Alice"}); err != nil {
			t.Fatal(err)
		}

		if !c.Unsubscribe(pk, true) {
			t.Error("can't unsubscribe")
		} else if c.so.LastRoot(pk) != nil {
			t.Error("feed not deleted")
		}

	})

	t.Run("not subscribed", func(t *testing.T) {
		c, err := newClient(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		if c.Unsubscribe(pk, false) {
			t.Error("unsubscribed while not subscribed")
		}
	})

}

func TestClient_Feeds(t *testing.T) {
	// Feeds() (feeds []cipher.PubKey)

	f1, _ := cipher.GenerateKeyPair()
	f2, _ := cipher.GenerateKeyPair()

	c, err := newClient(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	if c.Feeds() != nil {
		t.Error("non nil for empty list of feeds")
	}

	c.Subscribe(f1)
	if feeds := c.Feeds(); len(feeds) != 1 {
		t.Error("unexpected amount of feeds subscribed to")
	} else if feeds[0] != f1 {
		t.Error("wrong feed")
	}

	c.Subscribe(f2)
	if feeds := c.Feeds(); len(feeds) != 2 {
		t.Error("unexpected amount of feeds subscribed to")
	} else {
		if !((feeds[0] == f1 && feeds[1] == f2) ||
			(feeds[1] == f1 && feeds[0] == f2)) {
			t.Error("wrong feeds")
		}
	}

}

func TestClient_Container(t *testing.T) {
	// Container() *Container

}
