package node

import (
	"net"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

func dummyReceiveCallback([]byte, Node, ReplyFunc) error {
	return nil
}

func missingPanic(t *testing.T) {
	if recover() == nil {
		t.Error("missing panic")
	}
}

func unexpectedPanic(t *testing.T) {
	if recover() != nil {
		t.Error("unexpected panic")
	}
}

func newConfig() *Config {
	c := NewConfig()
	c.Debug = testing.Verbose()
	return c
}

func newNode() Node {
	c := newConfig()
	_, sec := cipher.GenerateKeyPair()
	c.SecretKey = sec.Hex()
	c.ReceiveCallback = dummyReceiveCallback
	return NewNode(c)
}

func TestNewNode(t *testing.T) {
	t.Run("config nil panic", func(t *testing.T) {
		defer missingPanic(t)
		NewNode(nil)
	})
	t.Run("empty secret key panic", func(t *testing.T) {
		defer missingPanic(t)
		c := newConfig()
		c.ReceiveCallback = dummyReceiveCallback
		NewNode(c)
	})
	t.Run("invalid secret key panic", func(t *testing.T) {
		defer missingPanic(t)
		c := newConfig()
		c.ReceiveCallback = dummyReceiveCallback
		c.SecretKey = "[secret]"
		NewNode(c)
	})
	t.Run("nil receive callback panic", func(t *testing.T) {
		defer missingPanic(t)
		c := newConfig()
		_, sec := cipher.GenerateKeyPair()
		c.SecretKey = sec.Hex()
		NewNode(c)
	})
	t.Run("should not panic", func(t *testing.T) {
		defer unexpectedPanic(t)
		c := newConfig()
		_, sec := cipher.GenerateKeyPair()
		c.SecretKey = sec.Hex()
		c.ReceiveCallback = dummyReceiveCallback
		NewNode(c)
	})
	t.Run("not nil", func(t *testing.T) {
		if newNode() == nil {
			t.Error("NewNode returns nil")
		}
	})
}

func TestNode_DB(t *testing.T) {
	n := newNode()
	if n.DB() == nil {
		t.Error("(Node).DB returns nil")
	}
}

func TestNode_Encoder(t *testing.T) {
	n := newNode()
	if n.Encoder() == nil {
		t.Error("(Node).Encoder returns nil")
	}
}

func TestNode_PubKey(t *testing.T) {
	n := newNode()
	if n.PubKey() == (cipher.PubKey{}) {
		t.Error("(Node).PubKey returns blank public key")
	}
}

func TestNode_Sign(t *testing.T) {
	n := newNode()
	if n.Sign(cipher.SumSHA256([]byte("data"))) == (cipher.Sig{}) {
		t.Error("(Node).Sign return blank Sig")
	}
}

func TestNode_Feed(t *testing.T) {
	if newNode().Feed() == nil {
		t.Error("(Node).Feed returns nil")
	}
}

func TestNode_Inflow(t *testing.T) {
	if newNode().Inflow() == nil {
		t.Error("(Node).Inflow returns nil")
	}
}

type dummyListener struct {
	l net.Listener
}

func startDummyListener(t *testing.T) *dummyListener {
	var err error
	dl := new(dummyListener)
	if dl.l, err = net.Listen("tcp", ""); err != nil {
		t.Fatal("error starting dummy listener: ", err)
	}
	go func() {
		_, err := dl.l.Accept()
		if err != nil {
			t.Error("accept error: ", err)
			return
		}
	}()
	return dl
}

func (d *dummyListener) address() string {
	return d.l.Addr().String()
}

func TestNode_stop(t *testing.T) {
	n := newNode()
	dl := startDummyListener(t)
	if err := n.Start(); err != nil {
		t.Fatal(err)
	}
	if err := n.Inflow().Subscribe(dl.address(), cipher.PubKey{}); err != nil {
		t.Fatal(err)
	}
	n.Close()
}

//	Start() error
//	Close()
