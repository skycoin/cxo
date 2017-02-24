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

func newNode(c ...Config) (Node, error) {
	var conf Config
	if len(c) > 0 {
		conf = c[0]
	} else {
		conf = newConfig()
	}
	n, err := NewNode(secretKey(), conf)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func TestNewNode(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		n, err := NewNode(secretKey(), NewConfig())
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
		if _, err := NewNode(secretKey(), c); err == nil {
			t.Error("missing error")
		}
	})
	t.Run("invalid secret key", func(t *testing.T) {
		if _, err := NewNode(cipher.SecKey{}, newConfig()); err == nil {
			t.Error("missing error")
		}
	})
}

func TestNode_onConnect(t *testing.T) {
	// TODO
}

func TestNode_onGnetConnect(t *testing.T) {
	// TODO
}

func TestNode_onGnetDisconnect(t *testing.T) {
	// TODO
}

func TestNode_lookupHandshakeTime(t *testing.T) {
	//
}

func nodeAddress(n Node, t *testing.T) string {
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
	n, err := newNode(c)
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
	addr := nodeAddress(n, t)
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

func mustNode(name ...string) Node {
	c := newConfig()
	if len(name) > 0 {
		c.Name = name[0]
	}
	n, err := newNode(c)
	if err != nil {
		panic(err)
	}
	return n
}

func TestNode_PubKey(t *testing.T) {
	n := mustNode()
	pub := n.PubKey()
	if err := pub.Verify(); err != nil {
		t.Error("empty public key: ", err)
	}
}

func TestNode_Sign(t *testing.T) {
	//
}

func TestNode_pend(t *testing.T) {
	//
}

func TestNode_hasOutgoing(t *testing.T) {
	//
}

func TestNode_hasIncoming(t *testing.T) {
	//
}

func TestNode_addOutgoing(t *testing.T) {
	//
}

func TestNode_addIncoming(t *testing.T) {
	//
}

func TestNode_Start(t *testing.T) {
	//
}

func TestNode_start(t *testing.T) {
	//
}

func TestNode_drainEvents(t *testing.T) {
	//
}

func TestNode_Close(t *testing.T) {
	//
}

func TestNode_Incoming(t *testing.T) {
	//
}

func TestNode_Outgoing(t *testing.T) {
	//
}

func TestNode_terminate(t *testing.T) {
	//
}

func TestNode_terminateByAddress(t *testing.T) {
	//
}

func TestNode_list(t *testing.T) {
	//
}

func TestNode_handleEvents(t *testing.T) {
	//
}

func TestNode_handleMessage(t *testing.T) {
	//
}

func TestNode_encode(t *testing.T) {
	//
}

func TestNode_decode(t *testing.T) {
	//
}

func TestNode_Register(t *testing.T) {
	//
}
