package node

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"
)

// An Event is any event for pipeline of a Node
type Event interface{}

// terminate connection by public key
type terminateEvent struct {
	outgoing bool
	pub      cipher.PubKey
}

// teminate connection by address string
type terminateByAddressEvent struct {
	outgoing bool
	address  string
}

// get list of connections
type listEvent struct {
	outgoing bool
	reply    chan<- Connection
}

// broadcast message (internal)
type broadcastEvent struct {
	msg *Msg
}

// any event to perform in main goroutine
type anyEvent func(*node)

// an outgoing connectio with desired public key
type outgoingConnection struct {
	gc      *gnet.Connection
	desired cipher.PubKey
	start   time.Time
}

// an incomng connection we need to handle
type incomingConnection struct {
	gc    *gnet.Connection
	start time.Time
}
