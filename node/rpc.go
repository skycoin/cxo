package node

import (
	"net"

	"github.com/skycoin/cxo/data"
)

// An rpcEvent represent RPC event
type rpcEvent func()

// enqueue rpc event
func (n *Node) enqueueEvent(done <-chan struct{}, evt rpcEvent) (err error) {
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
func (n *Node) Connect(address string) error {
	return n.pool.Connect(address)
}

// Disconnect should be called from RPC server. It trying to
// disconnect from given address
func (n *Node) Disconnect(address string) (err error) {
	if !n.pool.IsConnExist(address) {
		err = ErrNotFound
	} else {
		n.pool.Disconnect(address, ErrManualDisconnect)
	}
	return
}

// List should be called from RPC server. The List returns all
// connections
func (n *Node) List() (list []string, err error) {
	cc := n.pool.GetConnections()
	list = make([]string, 0, len(cc))
	for _, c := range cc {
		list = append(list, c.Addr())
	}
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
