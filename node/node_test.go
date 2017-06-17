package node

import (
	"os"
	"path/filepath"
	//"strings"
	"io"
	"io/ioutil"
	"net"
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
		s, err := NewNode(newNodeConfig(false))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		if s.Container().CoreRegistry() != nil {
			t.Error("unexpected core registry")
		}
	})

	defer clean()

	// if database in memeory then Node doesn't
	// creates DataDir and database
	t.Run("memory db", func(t *testing.T) {
		clean()

		conf := newNodeConfig(false) // in-memory

		s, err := NewNode(conf)
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

		s, err := NewNode(conf)
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

		s, err := NewNode(conf)
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
	// loading feeds from database tested below in
	//    TestNewNode_loadingFeeds
	// ----

}

func TestNewNodeReg(t *testing.T) {
	// NewNodeReg(sc NodeConfig, reg *skyobject.Registry) (s *Node, err error)

	// registry must be the same
	t.Run("registry", func(t *testing.T) {
		reg := skyobject.NewRegistry()

		s, err := NewNodeReg(newNodeConfig(false), reg)
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
		if s, err := NewNode(conf); err == nil {
			t.Error("missing error")
			s.Close()
		}
	})

}

func TestNode_starting(t *testing.T) {

	t.Run("disable listener", func(t *testing.T) {
		s, err := NewNode(newNodeConfig(false))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		if s.pool.Address() != "" {
			t.Error("listens on")
		}
	})

	t.Run("enable listener", func(t *testing.T) {
		s, err := NewNode(newNodeConfig(true))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		if s.pool.Address() == "" {
			t.Error("doesn't listen on")
		}
	})

	t.Run("disable RPC listener", func(t *testing.T) {
		s, err := NewNode(newNodeConfig(false))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		if s.rpc != nil {
			t.Error("RPC was created")
		}
	})

	t.Run("enable RPC listener", func(t *testing.T) {
		conf := newNodeConfig(false)
		conf.EnableRPC = true

		s, err := NewNode(conf)
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		if s.rpc == nil {
			t.Error("RPC wasn't created")
		}
	})
}

func TestNode_Close(t *testing.T) {
	// Close() (err error)

	t.Run("close listener", func(t *testing.T) {
		s, err := NewNode(newNodeConfig(true))
		if err != nil {
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

		s, err := NewNode(conf)
		if err != nil {
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

		s, err := NewNode(conf)
		if err != nil {
			t.Fatal(err)
		}

		// close many times
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

	s, err := NewNode(newNodeConfig(false))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.Pool() == nil {
		t.Fatal("mising pool")
	}
}

func (s *Node) lenPending() int {
	s.pmx.Lock()
	defer s.pmx.Unlock()

	return len(s.pending)
}

func testNodeSubscribe_local(t *testing.T) {
	s, err := NewNode(newNodeConfig(false))
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

}

func testNodeSubscribe_remoteReject(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()

	aconf := newNodeConfig(false)
	bconf := newNodeConfig(false)

	reject := make(chan *gnet.Conn, 1)

	aconf.OnSubscriptionAccepted = func(_ *gnet.Conn, _ cipher.PubKey) {
		t.Error("accepted") // must not be accepted
	}
	aconf.OnSubscriptionRejected = func(c *gnet.Conn, feed cipher.PubKey) {
		if feed != pk {
			t.Error("wrong feed rejected")
		}
		reject <- c // to test connection
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
			t.Error("rejected by wrong connection")
		}
	case <-time.After(TM):
		t.Error("slow")
	}

	if a.lenPending() != 0 {
		t.Error("has pending subscriptions")
	}

}

func testNodeSubscribe_remoteAccept(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()

	aconf := newNodeConfig(false)
	bconf := newNodeConfig(false)

	accept := make(chan *gnet.Conn, 1)

	aconf.OnSubscriptionAccepted = func(c *gnet.Conn, feed cipher.PubKey) {
		if feed != pk {
			t.Error("wrong feed accepted")
		}
		accept <- c // to test connection
	}
	aconf.OnSubscriptionRejected = func(_ *gnet.Conn, _ cipher.PubKey) {
		t.Error("rejected")
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
			t.Error("rejected by wrong connection")
		}
	case <-time.After(TM):
		t.Error("slow")
	}

	if a.lenPending() != 0 {
		t.Error("has pending subscriptions")
	}

}

func testNodeSubscribe_remoteAcceptTwice(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()

	aconf := newNodeConfig(false)
	bconf := newNodeConfig(false)

	accept := make(chan *gnet.Conn, 1)

	aconf.OnSubscriptionAccepted = func(c *gnet.Conn, feed cipher.PubKey) {
		if feed != pk {
			t.Error("wrong feed accepted")
		}
		accept <- c // to test connection
	}
	aconf.OnSubscriptionRejected = func(_ *gnet.Conn, _ cipher.PubKey) {
		t.Error("rejected")
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
			t.Error("accepted by wrong connection")
		}
	case <-time.After(TM):
		t.Error("slow")
	}

	if t.Failed() {
		return // don't need to continue
	}

	// second subscription must not send SusbcribeMsg twice
	a.Subscribe(ac, pk)

	// the test is not strong, but it's fast;
	// if first part of this test passes then
	// we can trust this part; increase timeout
	// if you want strong result
	select {
	case <-accept:
		t.Error("SubscribeMsg sent twice for the same connection")
	case <-time.After(TM):
	}

}

func TestNode_Subscribe(t *testing.T) {
	// Subscribe(c *gnet.Conn, feed cipher.PubKey)

	t.Run("local", testNodeSubscribe_local)
	t.Run("remote reject", testNodeSubscribe_remoteReject)
	t.Run("remote accept", testNodeSubscribe_remoteAccept)
	t.Run("remote accept twice", testNodeSubscribe_remoteAcceptTwice)

}

func testNodeUnsubscribe_local(t *testing.T) {
	s, err := NewNode(newNodeConfig(false))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	pk, _ := cipher.GenerateKeyPair()

	s.Subscribe(nil, pk)

	// susbcriptions of the Node
	if feeds := s.Feeds(); len(feeds) != 1 && feeds[0] != pk {
		t.Error("somethig wrong with Subscribe")
		return
	}

	s.Unsubscribe(nil, pk)

	if len(s.Feeds()) != 0 {
		t.Error("don't unsusbcribes")
	}

}

func testNodeUnsubscribe_remoteSingle(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()

	aconf := newNodeConfig(false)
	bconf := newNodeConfig(false)

	accept := make(chan *gnet.Conn, 1)

	aconf.OnSubscriptionAccepted = func(c *gnet.Conn, feed cipher.PubKey) {
		if feed != pk {
			t.Error("wrong feed accepted")
		}
		accept <- c // to test connection
	}
	aconf.OnSubscriptionRejected = func(_ *gnet.Conn, _ cipher.PubKey) {
		t.Error("rejected")
	}

	unsub := make(chan *gnet.Conn, 1)
	bconf.OnUnsubscribeRemote = func(c *gnet.Conn, feed cipher.PubKey) {
		if feed != pk {
			t.Error("unsubscribe from wrong feed")
		}
		unsub <- c
	}

	a, b, ac, bc, err := newConnectedNodes(aconf, bconf)
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
			t.Error("accepted by wrong connection")
			return
		}
	case <-time.After(TM):
		t.Error("slow")
		return
	}

	a.Unsubscribe(ac, pk)

	select {
	case c := <-unsub:
		if c != bc {
			t.Error("got UnsubscribeMsg from wrong connection")
		}
	case <-time.After(TM):
		t.Error("misisng UnsubscribeMsg")
	}

}

func receiveChanTimeout(c <-chan struct{}, tm time.Duration) (ok bool) {
	select {
	case <-c:
		ok = true
	case <-time.After(TM):
	}
	return
}

// TODO: DRY

func testNodeUnsubscribe_remoteMany(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()

	// create

	aconf := newNodeConfig(false) // client subscribes
	sconf := newNodeConfig(true)  // b and c (servers)

	accept := make(chan struct{}, 1)
	aconf.OnSubscriptionAccepted = func(_ *gnet.Conn, feed cipher.PubKey) {
		if feed != pk {
			t.Error("got AcceptSubscriptionMsg with wrong feed")
		}
		accept <- struct{}{}
	}

	a, err := NewNode(aconf)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	connect := make(chan struct{}, 1)
	sconf.Config.OnCreateConnection = func(*gnet.Conn) {
		connect <- struct{}{}
	}

	unsub := make(chan struct{}, 2)
	sconf.OnUnsubscribeRemote = func(_ *gnet.Conn, feed cipher.PubKey) {
		if feed != pk {
			t.Error("got UnsubscribeMsg with wrong feed")
		}
		unsub <- struct{}{}
	}

	b, err := NewNode(sconf)
	if err != nil {
		t.Error(err)
		return
	}
	defer b.Close()

	c, err := NewNode(sconf)
	if err != nil {
		t.Error(err)
		return
	}
	defer c.Close()

	// connect

	var abc, acc *gnet.Conn

	if abc, err = a.Pool().Dial(b.Pool().Address()); err != nil {
		t.Error(err)
		return
	}

	if !receiveChanTimeout(connect, TM) {
		t.Error("slow")
		return
	}

	if acc, err = a.Pool().Dial(c.Pool().Address()); err != nil {
		t.Error(err)
		return
	}

	if !receiveChanTimeout(connect, TM) {
		t.Error("slow")
		return
	}

	// subscribe servers

	b.Subscribe(nil, pk)
	c.Subscribe(nil, pk)

	// subscribe to b and c

	a.Subscribe(abc, pk)

	if !receiveChanTimeout(accept, TM) {
		t.Error("slow")
		return
	}

	a.Subscribe(acc, pk)

	if !receiveChanTimeout(accept, TM) {
		t.Error("slow")
		return
	}

	// total unsubscribe

	a.Unsubscribe(nil, pk)

	for i := 0; i < 2; i++ {
		if !receiveChanTimeout(unsub, TM) {
			t.Error("slow")
			return
		}
	}

}

func testNodeUnsubscribe_removeFromPending(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()

	a, err := NewNode(newNodeConfig(false))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	// create stub remote peer

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Error(err)
		return
	}
	defer l.Close()

	go func() {
		c, err := l.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		io.Copy(ioutil.Discard, c) // redirect input to /dev/null
	}()

	ac, err := a.Pool().Dial(l.Addr().String())
	if err != nil {
		t.Error(err)
		return
	}

	// subscribe (make pending subscription)

	a.Subscribe(ac, pk)
	a.Unsubscribe(ac, pk)

	if a.lenPending() != 0 {
		t.Error("not removed from pending during Unsubscribe")
	}

}

func TestNode_Unsubscribe(t *testing.T) {
	// Unsubscribe(c *gnet,.Conn, feed cipher.PubKey)

	t.Run("local", testNodeUnsubscribe_local)
	t.Run("remote single", testNodeUnsubscribe_remoteSingle)
	t.Run("remote many", testNodeUnsubscribe_remoteMany)
	t.Run("remove from pending", testNodeUnsubscribe_removeFromPending)
}

func TestNewNode_loadingFeeds(t *testing.T) {

	defer clean()

	conf := newNodeConfig(false)
	conf.InMemoryDB = false

	s, err := NewNode(conf)
	if err != nil {
		t.Fatal(err)
	}

	f1, _ := cipher.GenerateKeyPair()
	f2, _ := cipher.GenerateKeyPair()

	s.Subscribe(nil, f1)
	s.Subscribe(nil, f2)

	s.Close()

	// recreate

	s, err = NewNode(conf)
	if err != nil {
		t.Error(err)
		return
	}
	defer s.Close()

	if feeds := s.Feeds(); len(feeds) != 2 {
		t.Error("feeds not loaded from database")
	} else {
		if !((feeds[0] == f1 && feeds[1] == f2) ||
			(feeds[1] == f1 && feeds[0] == f2)) {
			t.Error("wrong feeds loaded")
		}
	}

}

func TestNode_Want(t *testing.T) {
	// Want(feed cipher.PubKey) (wn []cipher.SHA256)

	// TODO: mid. priority
}

func TestNode_Got(t *testing.T) {
	// Got(feed cipher.PubKey) (gt []cipher.SHA256)

	// TODO: mid. priority
}

func TestNode_Feeds(t *testing.T) {
	// Feeds() (fs []cipher.PubKey)

	// TODO: mid. priority
}

func TestNode_Quiting(t *testing.T) {
	// Quiting() <-chan struct{}

	// TODO: mid. priority
}

func TestNode_SubscribeResponse(t *testing.T) {
	// SubscribeResponse(c *gnet.Conn, feed cipher.PubKey) error

	t.Run("accept", func(t *testing.T) {
		conf := newNodeConfig(false)

		a, b, ac, _, err := newConnectedNodes(conf, conf)
		if err != nil {
			t.Fatal(err)
		}
		defer a.Close()
		defer b.Close()

		pk, _ := cipher.GenerateKeyPair()

		b.Subscribe(nil, pk)

		// subscribe response

		if err := a.SubscribeResponse(ac, pk); err != nil {
			t.Error(err)
		}

	})

	t.Run("reject", func(t *testing.T) {
		conf := newNodeConfig(false)

		a, b, ac, _, err := newConnectedNodes(conf, conf)
		if err != nil {
			t.Fatal(err)
		}
		defer a.Close()
		defer b.Close()

		pk, _ := cipher.GenerateKeyPair()

		// subscribe response

		if err := a.SubscribeResponse(ac, pk); err == nil {
			t.Error("misisng error")
		} else if err != ErrSubscriptionRejected {
			t.Error("wrong error:", err)
		}

	})

	t.Run("timeout", func(t *testing.T) {
		conf := newNodeConfig(false)
		conf.ResponseTimeout = TM

		a, err := NewNode(conf)
		if err != nil {
			t.Fatal(err)
		}
		defer a.Close()

		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Error(err)
			return
		}
		defer l.Close()

		go func() {
			c, err := l.Accept()
			if err != nil {
				t.Error(err)
				return
			}
			defer c.Close()
			io.Copy(ioutil.Discard, c)
		}()

		ac, err := a.Pool().Dial(l.Addr().String())
		if err != nil {
			t.Error(err)
			return
		}

		pk, _ := cipher.GenerateKeyPair()

		// subscribe response

		if err := a.SubscribeResponse(ac, pk); err == nil {
			t.Error("misisng error")
		} else if err != ErrTimeout {
			t.Error("wrong error:", err)
		}

	})

}

func TestNode_SubscribeResponseTimeout(t *testing.T) {
	// SubscribeResponseTimeout(c *gnet.Conn, feed cipher.PubKey,
	//     timeout time.Duration) (err error)

	// TODO: mid. priority
}

func TestNode_ListOfFeedsResponse(t *testing.T) {
	// ListOfFeedsResponse(c *gnet.Conn) ([]cipher.PubKey, error)

	t.Run("success", func(t *testing.T) {
		aconf := newNodeConfig(false)
		bconf := newNodeConfig(true)
		bconf.PublicServer = true

		a, b, ac, _, err := newConnectedNodes(aconf, bconf)
		if err != nil {
			t.Fatal(err)
		}
		defer a.Close()
		defer b.Close()

		pk, _ := cipher.GenerateKeyPair()

		b.Subscribe(nil, pk)

		// subscribe response

		if list, err := a.ListOfFeedsResponse(ac); err != nil {
			t.Error(err)
		} else if len(list) != 1 {
			t.Error("wrong list length")
		} else if list[0] != pk {
			t.Error("wrong content of the list")
		}

	})

	t.Run("non-public", func(t *testing.T) {
		conf := newNodeConfig(false)

		a, b, ac, _, err := newConnectedNodes(conf, conf)
		if err != nil {
			t.Fatal(err)
		}
		defer a.Close()
		defer b.Close()

		// subscribe response

		if _, err := a.ListOfFeedsResponse(ac); err == nil {
			t.Error("misisng error")
		} else if err != ErrNonPublicPeer {
			t.Error("wrong error:", err)
		}

	})

	t.Run("timeout", func(t *testing.T) {
		conf := newNodeConfig(false)
		conf.ResponseTimeout = TM

		a, err := NewNode(conf)
		if err != nil {
			t.Fatal(err)
		}
		defer a.Close()

		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Error(err)
			return
		}
		defer l.Close()

		go func() {
			c, err := l.Accept()
			if err != nil {
				t.Error(err)
				return
			}
			defer c.Close()
			io.Copy(ioutil.Discard, c)
		}()

		ac, err := a.Pool().Dial(l.Addr().String())
		if err != nil {
			t.Error(err)
			return
		}

		// subscribe response

		if _, err := a.ListOfFeedsResponse(ac); err == nil {
			t.Error("misisng error")
		} else if err != ErrTimeout {
			t.Error("wrong error:", err)
		}

	})

}

func TestNode_ListOfFeedsResponseTimeout(t *testing.T) {
	// ListOfFeedsResponseTimeout(c *gnet.Conn,
	//     timeout time.Duration) (list []cipher.PubKey, err error)

	// TODO: mid. priority

}

// A Node that connects to another node recreates
// connection automatically. But for other side
// this connection will be another connection. Thus,
// we need to send all SubscribeMsg that we sent before
func TestNode_resubscribe(t *testing.T) {

	if testing.Short() {
		t.Skip("short")
	}

	aconf := newNodeConfig(false)
	aconf.Log.Prefix = "[A CLIENT]"
	aconf.Config.RedialTimeout = 0 // redial immediately
	aconf.Config.OnDial = nil      // clear any default redialing filters
	accept := make(chan struct{}, 1)
	aconf.OnSubscriptionAccepted = func(*gnet.Conn, cipher.PubKey) {
		accept <- struct{}{}
	}

	// we can't use OnSubscriptionAccepted because it never called
	// if remote peer already subscribed to a feed by this (local)
	// node; but we can use OnSubscribeRemote of b
	bconf := newNodeConfig(true)
	bconf.Log.Prefix = "[B SERVER]"
	subs := make(chan cipher.PubKey, 1)
	bconf.OnSubscribeRemote = func(_ *gnet.Conn, feed cipher.PubKey) (_ error) {
		subs <- feed
		return
	}

	conn := make(chan struct{}, 1)
	// we need to handle connections to be sure that remote peer redials
	bconf.Config.OnCreateConnection = func(*gnet.Conn) {
		conn <- struct{}{}
	}

	a, b, ac, bc, err := newConnectedNodes(aconf, bconf)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	defer b.Close()

	<-conn // newConnectedNodes creates connection

	pk, _ := cipher.GenerateKeyPair()

	b.Subscribe(nil, pk)
	a.Subscribe(ac, pk)

	select {
	case <-subs:
	case <-time.After(TM):
		t.Error("slow")
		return
	}

	select {
	case <-accept:
		// be sure that this susbcription was accepted by remote peer
	case <-time.After(TM):
		t.Error("slow")
		return
	}

	// close connection from b side
	bc.Close()

	select {
	case <-conn:
		// we got renewed connection
	case <-time.After(TM):
		t.Error("slow redialing")
		return
	}

	select {
	case feed := <-subs:
		if feed != pk {
			t.Error("wrong feed resubscribing")
		}
	case <-time.After(TM * 2):
		// increase TM because of Redial+resusbcribe
		t.Error("slow or missing resubscribing")
	}

}
