package node

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"
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

type outgoingConnection struct {
	gc      *gnet.Connection
	desired cipher.PubKey
	start   time.Time
}

type incomingConnection struct {
	gc    *gnet.Connection
	start time.Time
}
