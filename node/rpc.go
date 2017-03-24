package node

import (
	"net"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/rpc/comm"
)

// An rpcEvent represent RPC event
type rpcEvent interface{}

// enqueue rpc event
func (n *Node) enqueueRpcEvent(evt rpcEvent) (err error) {
	var done = make(chan struct{})
	// enquue
	switch e := evt.(type) {
	case func(): //
		select {
		case n.rpce <- func() {
			defer close(done)
			e()
		}:
		case <-n.quit:
			err = ErrClosed
			return
		}
	case func(map[cipher.PubKey]struct{}): //
		select {
		case n.rpce <- func(subs map[cipher.PubKey]struct{}) {
			defer close(done)
			e(subs)
		}:
		case <-n.quit:
			err = ErrClosed
			return
		}
	default: //
		n.Panicf("[CRITICAL] invalid type of rpc event %T", evt)
	}
	// wait
	select {
	case <-done:
	case <-n.quit:
		err = ErrClosed
	}
	return
}

// Subscribe to a feed by public key
func (n *Node) Subscribe(pub cipher.PubKey) {
	n.enqueueRpcEvent(func(subs map[cipher.PubKey]struct{}) {
		subs[pub] = struct{}{}
		exc := []string{} // existing connections
		for _, address := range n.conf.Known[pub] {
			if !n.pool.IsConnExist(address) {
				n.Debug("[DBG] connecting to ", address)
				n.pool.Connect(address)
			} else {
				exc = append(exc, address)
			}
		}
		if root := n.so.Root(pub); root != nil {
			// Broadcast root of the feed (if we have it)
			// to all existing connections. The root will be
			// send to new connections automatically
			msg := &Root{
				Pub:  root.Pub,
				Sig:  root.Sig,
				Root: root.Encode(),
			}
			for _, address := range exc {
				n.Debugf("[DBG] send root of %s to: %s",
					pub.Hex(),
					address)
				n.pool.SendMessage(address, msg)
			}
		}
	})
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

func (n *Node) Inject(hash cipher.SHA256, sec cipher.SecKey) (err error) {
	if hash == (cipher.SHA256{}) {
		err = ErrEmptyHash
		return
	}
	if sec == (cipher.SecKey{}) {
		err = ErrEmptySecret
		return
	}
	n.enqueueRpcEvent(func() {
		//
	})
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
func (n *Node) Info() (info comm.Info, err error) {
	err = n.enqueueRpcEvent(func(subs map[cipher.PubKey]struct{}) {
		var a net.Addr
		if a, err = n.pool.ListeningAddress(); err != nil {
			return
		}
		info.Address = a.String()
		info.Feeds = make(map[cipher.PubKey][]string)
		for pk := range subs {
			list := make([]string, 0, 5)
			for _, address := range n.conf.Known[pk] {
				if n.pool.IsConnExist(address) {
					list = append(list, address)
				}
			}
			info.Feeds[pk] = list
		}
	})
	return
}

// Stat is for RPC. It returns database statistic
func (n *Node) Stat() (stat data.Stat, err error) {
	err = n.enqueueRpcEvent(func() {
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
	err = n.enqueueRpcEvent(func() {
		n.close()
	})
	// we will have got ErrClosed from enqueueRpcEvent that
	// is not actual error
	if err == ErrClosed {
		err = nil
	}
	return
}
