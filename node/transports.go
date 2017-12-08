package node

import (
	"sync"

	"github.com/skycoin/net/factory"
)

type acceptConnectionCallback func(c *factory.Connection)

// A TCP represents TCP transport
// of the Node. The TCP used to
// listen and connect
type TCP struct {
	// the factory
	*factory.TCPFactory

	// back reference
	n *Node

	//
	mx          sync.Mutex
	isListening bool
}

func newTCP(n *Node) (t *TCP) {

	t = new(TCP)

	t.TCPFactory = factory.NewTCPFactory()

	t.n = n
	t.AcceptedCallback = n.acceptTCPConnection

	return
}

// Listen on given address. It's possible to listen
// only once
func (t *TCP) Listen(address string) (err error) {

	t.mx.Lock()
	defer t.mx.Unlock()

	if t.isListening == true {
		return ErrAlreadyListen
	}

	return t.TCPFactory.Listen(address)

}

// Conecti to given TCP address. The method blocks
func (t *TCP) Connect(address string) (c *Conn, err error) {

	var fc *factory.Connection

	if fc, err = t.TCPFactory.Connect(address); err != nil {
		return
	}

	c, err = t.n.wrapConnection(fc, false, true)
	return
}

// A UDP represents UDP transport
// of the Node. The UDP used to
// listen and connect
type UDP struct {
	// the factory
	*factory.UDPFactory

	// back reference
	n *Node

	//
	mx          sync.Mutex
	isListening bool
}

func newUDP(n *Node) (u *UDP) {

	u = new(UDP)

	u.UDPFactory = factory.NewUDPFactory()

	u.n = n
	u.AcceptedCallback = n.acceptUDPConnection

	return
}

// Listen on given address. It's possible to listen
// only once
func (u *UDP) Listen(address string) (err error) {

	u.mx.Lock()
	defer u.mx.Unlock()

	if u.isListening == true {
		return ErrAlreadyListen
	}

	return u.UDPFactory.Listen(address)

}

// Connect to given UDP address
func (u *UDP) Connect(address string) (c *Conn, err error) {

	var fc *factory.Connection

	if fc, err = u.UDPFactory.Connect(address); err != nil {
		return
	}

	c, err = u.n.wrapConnection(fc, false, false)
	return
}
