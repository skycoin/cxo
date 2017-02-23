package node

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

func secretKey() (sec cipher.SecKey) {
	_, sec = cipher.GenerateKeyPair()
	return
}

func TestNewNode(t *testing.T) {
	var (
		n   Node
		err error
	)
	n, err = NewNode(secretKey(), NewConfig())
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}
	if n == nil {
		t.Fatal("NewNode returns nil")
	}
}

func TestNode_Incoming(t *testing.T) {
	//
}

func TestNode_Outgoing(t *testing.T) {
	//
}

func TestNode_PubKey(t *testing.T) {
	//
}

func TestNode_Sign(t *testing.T) {
	//
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

func TestNode_pend(t *testing.T) {
	//
}

func TestNode_handleMessage(t *testing.T) {
	//
}

func TestNode_addOutgoing(t *testing.T) {
	//
}

func TestNode_addIncoming(t *testing.T) {
	//
}

func TestNode_encode(t *testing.T) {
	//
}

func TestNode_decode(t *testing.T) {
	//
}
