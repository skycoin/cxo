package client

import (
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/gui"
	"github.com/skycoin/cxo/node"
)

type nodeMessenger struct {
	node node.Node
}

func NodeMessanger(node node.Node) *gui.Messenger {
	return &gui.Messenger{Context: &nodeMessenger{node: node}}
}

func (m *nodeMessenger) send(msg interface{}) error {
	fmt.Println("BroadcastToSubscribers", msg)
	return m.node.Incoming().Broadcast(msg)
}

func (m *nodeMessenger) Announce(hash cipher.SHA256) error {
	return m.send(AnnounceMessage{Hash: hash})
}

func (m *nodeMessenger) Request(hash cipher.SHA256) error {
	return m.send(RequestMessage{Hash: hash})
}

//type RootMessage struct {
//	Hash cipher.SHA256
//}

// upstream -> downstream (HandleFromUpstream)
// AnnounceMessage is a message that announces to subscribers
// that it has new data
type AnnounceMessage struct {
	Hash cipher.SHA256
}

// downstream -> upstream (HandleFromDownstream)
// RequestMessage is a request from a subscriber for data
type RequestMessage struct {
	Hash cipher.SHA256
}

// upstream -> downstream (HandleFromUpstream)
// DataMessage is the response to a RequestMessage, with the data requested
type DataMessage struct {
	Hash cipher.SHA256
	Data []byte
}
