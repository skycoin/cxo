package node

import (
	"net"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

func newConfig() (c Config) {
	c = NewConfig()
	c.Debug = testing.Verbose()
	return
}

func secretKey() (sec cipher.SecKey) {
	_, sec = cipher.GenerateKeyPair()
	return
}

func newNode(c Config, user interface{}) (Node, error) {
	n, err := NewNode(secretKey(), c, user)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func TestNewNode(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		n, err := NewNode(secretKey(), NewConfig(), nil)
		if err != nil {
			t.Error(err)
			return
		}
		if n == nil {
			t.Error("NewNode returns nil")
		}
	})
	t.Run("invalid config", func(t *testing.T) {
		c := NewConfig()
		c.MaxPendingConnections = 0
		if _, err := NewNode(secretKey(), c, nil); err == nil {
			t.Error("missing error")
		}
	})
	t.Run("invalid secret key", func(t *testing.T) {
		if _, err := NewNode(cipher.SecKey{}, newConfig(), nil); err == nil {
			t.Error("missing error")
		}
	})
}

func nodeMustGetAddress(n Node, t *testing.T) string {
	a, err := n.Incoming().Address()
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestNode_handshakeTimeoutRemoving(t *testing.T) {
	th := newTestHook()
	c := newConfig()
	c.HandshakeTimeout = 200 * time.Millisecond
	// after c.HandshakeTimeout x 2 + aroung 100ms for concurrency reason
	n, err := newNode(c, nil)
	if err != nil {
		t.Error(err)
		return
	}
	th.Set(n)
	nd := n.(*node)
	if err = nd.Start(); err != nil {
		t.Error(err)
		return
	}
	defer nd.Close()
	addr := nodeMustGetAddress(n, t)
	n.Debug("[TEST] listening address: ", addr)
	if _, err = net.Dial("tcp", addr); err != nil {
		t.Error("error dialing node: ", err)
		return
	}

	if !th.pendTimeout(1, time.Second, t) {
		return
	}

	time.Sleep(c.HandshakeTimeout*2 + 100*time.Millisecond)

	if len(nd.pending) != 0 {
		t.Error("handshake timeout doesn't work")
	}
}

// with optional name
func mustCreateNode(name string, user interface{}) Node {
	c := newConfig()
	c.Name = name
	n, err := newNode(c, user)
	if err != nil {
		panic(err)
	}
	return n
}

func TestNode_PubKey(t *testing.T) {
	n := mustCreateNode("node", nil)
	pub := n.PubKey()
	if err := pub.Verify(); err != nil {
		t.Error("empty public key: ", err)
	}
}

func TestNode_Sign(t *testing.T) {
	pub, sec := cipher.GenerateKeyPair()
	n, err := NewNode(sec, newConfig(), nil)
	if err != nil {
		t.Fatal(err)
	}
	hash := cipher.SumSHA256([]byte("some value"))
	sig := n.Sign(hash)
	if err := cipher.VerifySignature(pub, sig, hash); err != nil {
		t.Error("invalid Sign: ", err)
	}
}

func TestNode_hasOutgoing(t *testing.T) {
	tf, ts := newTestHook(), newTestHook()
	feed, subscr := mustCreateNode("feed", nil), mustCreateNode("subscr", nil)
	for _, x := range []struct {
		node Node
		hook *testHook
	}{
		{feed, tf},
		{subscr, ts},
	} {
		if err := x.node.Start(); err != nil {
			t.Error(err)
			return
		}
		defer x.node.Close()
		x.hook.Set(x.node)
	}
	// connect to feed
	err := subscr.Outgoing().Connect(
		nodeMustGetAddress(feed, t),
		cipher.PubKey{})
	if err != nil {
		t.Error(err)
		return
	}
	if !ts.estOutTimeout(1, time.Second, t) {
		t.Error("can't establish outgoing connection after second")
		return
	}
	if !tf.estIncTimeout(1, time.Second, t) {
		t.Error("can't establish incoming connecition after second")
		return
	}
	if !subscr.(*node).hasOutgoing(feed.(*node).pub) {
		t.Error("hasOutgoing() returns false for existed outgoign connection")
	}
}

func TestNode_hasIncoming(t *testing.T) {
	tf, ts := newTestHook(), newTestHook()
	feed, subscr := mustCreateNode("feed", nil), mustCreateNode("subscr", nil)
	for _, x := range []struct {
		node Node
		hook *testHook
	}{
		{feed, tf},
		{subscr, ts},
	} {
		x.hook.Set(x.node)
		if err := x.node.Start(); err != nil {
			t.Error(err)
			return
		}
		defer x.node.Close()
	}
	// connect to feed
	err := subscr.Outgoing().Connect(
		nodeMustGetAddress(feed, t),
		cipher.PubKey{})
	if err != nil {
		t.Error(err)
		return
	}
	if !ts.estOutTimeout(1, time.Second, t) {
		t.Error("can't establish outgoing connection after second")
		return
	}
	if !tf.estIncTimeout(1, time.Second, t) {
		t.Error("can't establish incoming connecition after second")
		return
	}
	if !feed.(*node).hasIncoming(subscr.(*node).pub) {
		t.Error("hasIncoming() returns false for existed incoming connection")
	}
}

func TestNode_Incoming(t *testing.T) {
	if mustCreateNode("node", nil).Incoming() == nil {
		t.Error("(Node).Incoming() returns nil")
	}
}

func TestNode_Outgoing(t *testing.T) {
	if mustCreateNode("node", nil).Outgoing() == nil {
		t.Error("(Node).Outgoign() returns nil")
	}
}

func TestNode_Start(t *testing.T) {
	//
}

func TestNode_Close(t *testing.T) {
	//
}

func TestNode_Register(t *testing.T) {
	//
}
