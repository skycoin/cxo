package client

import (
	"github.com/skycoin/cxo/nodeManager"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/cxo/gui"
	"fmt"
)

type nodeMessenger struct {
	node *nodeManager.Node
}

func NodeMessanger(node *nodeManager.Node) *gui.Messenger {
	return &gui.Messenger{Context:&nodeMessenger{node:node}}
}

func (m *nodeMessenger) send(message interface{}) error {
	fmt.Println("BroadcastToSubscribers", message)
	return m.node.BroadcastToSubscribers(message)
}

func (m *nodeMessenger) Announce(hash cipher.SHA256) error {
	return m.send(AnnounceMessage{Hash:hash})
}

func (m *nodeMessenger) Request(hash cipher.SHA256) error {
	return m.send(RequestMessage{Hash:hash})
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
