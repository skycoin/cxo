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
	List() []Connection          // list all outgoing connections
	Terminate(pub cipher.PubKey) // terminate outgoing connection by public key
}

// namespace isolation for node->incoming and node->outgoing
type outgoing struct {
	*node
}

func (i outgoing) List() (cs []Connection) {
	i.lock()
	defer i.unlock()
	cs = make([]Connection, 0, len(i.outgoing))
	for gc, pk := range i.outgoing {
		cs = append(cs, Connection{
			Pub:  pk,
			Addr: gc.Addr(),
		})
	}
	return
}

func (i outgoing) Terminate(pub cipher.PubKey) {
	i.lock()
	defer i.unlock()
	for gc, pk := range i.outgoing {
		if pk == pub {
			gc.ConnectionPool.Disconnect(gc, ErrManualDisconnect)
			return
		}
	}
	return
}

func (o outgoing) Connect(address string, desired cipher.PubKey) (err error) {
	o.Lock()
	defer o.Unlock()
	var gc *gnet.Connection
	if gc, err = o.pool.Connect(address); err != nil {
		return
	}
	// create pending connection with desired public key
	o.pending[gc] = &pendingConnection{
		outgoing:  true,
		remotePub: desired,
		start:     time.Now(),
	}
	return
}
