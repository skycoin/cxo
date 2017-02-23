package node

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"
)

type Incoming interface {
	Broadcast(msg interface{}) (err error) // broadcast msg to all subscribers
	List() []Connection                    // list all incoming connections
	Terminate(pub cipher.PubKey)           // terminate incoming connection
}

// namespace isolation for node->incoming and node->outgoing
type incoming struct {
	*node
}

type Connection struct {
	Pub  cipher.PubKey
	Addr string
}

func (i incoming) List() (cs []Connection) {
	i.lock()
	defer i.unlock()
	cs = make([]Connection, 0, len(i.incoming))
	for gc, pk := range i.incoming {
		cs = append(cs, Connection{
			Pub:  pk,
			Addr: gc.Addr(),
		})
	}
	return
}

func (i incoming) Terminate(pub cipher.PubKey) {
	i.lock()
	defer i.unlock()
	for gc, pk := range i.incoming {
		if pk == pub {
			gc.ConnectionPool.Disconnect(gc, ErrManualDisconnect)
			return
		}
	}
	return
}

func (i incoming) Broadcast(msg interface{}) (err error) {
	var (
		d  *Msg
		gc *gnet.Connection
	)
	if d, err = i.encode(msg); err != nil {
		i.Printf("[ERR] error encoding message: %q, %T, %v,",
			err.Error(),
			msg,
			msg)
		return
	}
	for gc = range i.incoming {
		gc.ConnectionPool.SendMessage(gc, d)
	}
	return
}
