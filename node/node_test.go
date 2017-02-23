package node

import (
	"testing"

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

func newNode() (Node, error) {
	return NewNode(secretKey(), newConfig())
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
	//
}

func TestNode_onGnetConnect(t *testing.T) {
	//
}

func TestNode_onGnetDisconnect(t *testing.T) {
	//
}

func TestNode_lookupHandshakeTime(t *testing.T) {
	//
}

func TestNode_PubKey(t *testing.T) {
	//
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
