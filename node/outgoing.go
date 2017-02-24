package node

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"
)

type Outgoing interface {
	// Connect creates new outgoing connection to given address.
	// The 'desired' argument is desired public key of
	// remote node we connect to. If the desired is empty
	// then any public key allowed
	Connect(address string, desired cipher.PubKey) error
	List() <-chan Connection     // list all outgoing connections
	Terminate(pub cipher.PubKey) // terminate outgoing connection by public key
	TerminateByAddress(address string)
}

// namespace isolation for node->incoming and node->outgoing
type outgoing struct {
	*node
}

func (i outgoing) List() <-chan Connection {
	reply := make(chan Connection, i.conf.MaxOutgoingConnections)
	i.events <- listEvent{
		outgoing: true,
		reply:    reply,
	}
	return reply
}

func (i outgoing) Terminate(pub cipher.PubKey) {
	i.events <- terminateEvent{
		outgoing: true,
		pub:      pub,
	}
}

func (i outgoing) TerminateByAddress(address string) {
	i.events <- terminateByAddressEvent{
		outgoing: true,
		address:  address,
	}
}

// we need to dial from separate goroutine, because of dial timeout
func (o outgoing) Connect(address string, desired cipher.PubKey) (err error) {
	var gc *gnet.Connection
	if gc, err = o.pool.Connect(address); err != nil {
		return
	}
	o.events <- outgoingConnection{
		gc:      gc,
		start:   time.Now(),
		desired: desired,
	}
	return // nil
}
