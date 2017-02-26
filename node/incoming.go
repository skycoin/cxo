package node

import (
	"net"

	"github.com/skycoin/skycoin/src/cipher"
)

type Incoming interface {
	Address() (string, error)
	Broadcast(msg interface{}) (err error) // broadcast msg to all subscribers
	List() <-chan Connection               // list all incoming connections
	Terminate(pub cipher.PubKey)           // terminate incoming connection
	TerminateByAddress(address string)     // terminate by address
}

// namespace isolation for node->incoming and node->outgoing
type incoming struct {
	*node
}

func (i incoming) List() <-chan Connection {
	reply := make(chan Connection, i.conf.MaxIncomingConnections)
	err := i.enqueueEvent(listEvent{
		outgoing: false,
		reply:    reply,
	})
	if err != nil {
		i.Print("[ERR] enqueue event error: ", err)
		close(reply)
	}
	return reply
}

func (i incoming) Terminate(pub cipher.PubKey) {
	err := i.enqueueEvent(terminateEvent{
		outgoing: false,
		pub:      pub,
	})
	if err != nil {
		i.Print("[ERR] enqueue event error: ", err)
	}
}

func (i incoming) TerminateByAddress(address string) {
	err := i.enqueueEvent(terminateByAddressEvent{
		outgoing: false,
		address:  address,
	})
	if err != nil {
		i.Print("[ERR] enqueue event error: ", err)
	}
}

func (i incoming) Broadcast(msg interface{}) (err error) {
	i.Debug("[DBG] broadcast: ", msg)
	var d *Msg
	if d, err = i.encode(msg); err != nil {
		i.Printf("[ERR] error encoding message: %q, %T, %v,",
			err.Error(),
			msg,
			msg)
		return
	}
	err = i.enqueueEvent(broadcastEvent{d})
	return
}

func (i incoming) Address() (addr string, err error) {
	done := make(chan struct{})
	err = i.enqueueEvent(anyEvent(func(n *node) {
		var na net.Addr
		if na, err = i.pool.ListeningAddress(); err != nil {
			return
		}
		addr = na.String()
		close(done)
	}))
	if err != nil {
		return
	}
	<-done
	return
}
