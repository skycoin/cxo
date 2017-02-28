package node

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"
)

type Sender interface {
	Send(msg interface{}) (err error)
}

type sender struct {
	node Node
	*gnet.Connection
}

func (r *sender) Send(msg interface{}) (err error) {
	var m *Msg
	if m, err = r.node.encode(msg); err != nil {
		return
	}
	r.ConnectionPool.SendMessage(r.Connection, m)
	return
}

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

func (o outgoing) List() <-chan Connection {
	reply := make(chan Connection, o.conf.MaxOutgoingConnections)
	err := o.enqueueEvent(listEvent{
		outgoing: true,
		reply:    reply,
	})
	if err != nil {
		o.Print("[ERR] enqueue event error: ", err)
		close(reply)
	}
	return reply
}

func (o outgoing) Terminate(pub cipher.PubKey) {
	err := o.enqueueEvent(terminateEvent{
		outgoing: true,
		pub:      pub,
	})
	if err != nil {
		o.Print("[ERR] enqueue event error: ", err)
	}
}

func (o outgoing) TerminateByAddress(address string) {
	err := o.enqueueEvent(terminateByAddressEvent{
		outgoing: true,
		address:  address,
	})
	if err != nil {
		o.Print("[ERR] enqueue event error: ", err)
	}
}

// we need to dial from separate goroutine, because of dial timeout
func (o outgoing) Connect(address string, desired cipher.PubKey) (err error) {
	var gc *gnet.Connection
	if gc, err = o.pool.Connect(address); err != nil {
		return
	}
	err = o.enqueueEvent(outgoingConnection{
		gc:      gc,
		start:   time.Now(),
		desired: desired,
	})
	return // nil
}
