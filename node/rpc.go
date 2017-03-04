package node

import (
	"net"

	"github.com/skycoin/cxo/data"
)

// An Event represent RPC event
type Event func()

// enqueue rpc event
func (n *Node) enqueueEvent(done <-chan struct{}, evt Event) (err error) {
	// enquue
	select {
	case n.rpce <- evt:
	case <-n.quit:
		err = ErrClosed
		return
	}
	// wait
	select {
	case <-done:
	case <-n.quit:
		err = ErrClosed
	}
	return
}

// Connect should be called from RPC server. It trying
// to connect to given address
func (n *Node) Connect(address string) (err error) {
	// We need to dial manually to prevent concurent map
	// access
	var conn net.Conn
	conn, err = net.DialTimeout("tcp", address, n.conf.DialTimeout)
	if err != nil {
		return
	}
	done := make(chan struct{})
	err = n.enqueueEvent(done, func() {
		defer close(done)
		_, err = n.pool.Connect(address)
	})
	return
}

// Disconnect should be called from RPC server. It trying to
// disconnect from given address
func (n *Node) Disconnect(address string) (err error) {
	done := make(chan struct{})
	err = n.enqueueEvent(done, func() {
		defer close(done)
		if gc, ok := n.pool.Addresses[address]; ok {
			n.pool.Disconnect(gc, ErrManualDisconnect)
			return
		}
		err = ErrNotFound
	})
	return
}

// List should be called from RPC server. The List returns all
// connections
func (n *Node) List() (list []string, err error) {
	done := make(chan struct{})
	err = n.enqueueEvent(done, func() {
		close(done)
		list = make([]string, 0, len(n.pool.Addresses))
		for a := range n.pool.Addresses {
			list = append(list, a)
		}
	})
	return
}

// Info is for RPC. It returns all useful inforamtions about the node
// except statistic. I.e. it returns listening address
func (n *Node) Info() (address string, err error) {
	done := make(chan struct{})
	err = n.enqueueEvent(done, func() {
		defer close(done)
		var a net.Addr
		if a, err = n.pool.ListeningAddress(); err != nil {
			return
		}
		address = a.String()
	})
	return
}

// Stat is for RPC. It returns database statistic
func (n *Node) Stat() (stat data.Stat, err error) {
	done := make(chan struct{})
	err = n.enqueueEvent(done, func() {
		defer close(done)
		stat = n.db.Stat()
	})
	return
}

// Terminate is the same as Close but designed for RPC
func (n *Node) Terminate() (err error) {
	if !n.conf.RemoteClose {
		err = ErrNotAllowed
		return
	}
	done := make(chan struct{})
	err = n.enqueueEvent(done, func() {
		defer close(done)
		n.close()
	})
	// we will have got ErrClosed from enqueueEvent that
	// is not actual error
	if err == ErrClosed {
		err = nil
	}
	return
}
