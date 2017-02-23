package node

import (
	"github.com/skycoin/skycoin/src/cipher"
)

type Event interface{}

type terminateEvent struct {
	outgoing bool
	pub      cipher.PubKey
}

type terminateByAddressEvent struct {
	outgoing bool
	address  string
}

type listEvent struct {
	outgoing bool
	reply    chan<- Connection
}

type broadcastEvent struct {
	msg *Msg
}

type anyEvent func(*node)
