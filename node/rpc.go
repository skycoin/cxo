package node

import (
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/rpc/comm"
	"github.com/skycoin/cxo/skyobject"
)

// An rpcEvent represent RPC event
type rpcEvent func()

// enqueue rpc event
func (n *Node) enqueueRpcEvent(evt rpcEvent) (err error) {
	var done = make(chan struct{})
	// enquue
	select {
	case n.rpce <- func() {
		defer close(done)
		evt()
	}:
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

// Subscribe to a feed by public key
func (n *Node) Subscribe(pub cipher.PubKey) {
	n.enqueueRpcEvent(func() {
		n.subs[pub] = struct{}{}
		for _, address := range n.conf.Known[pub] {
			n.pool.Connect(address)
		}
		n.Share(pub) // trigger update of wanted objects etc
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
		n.pool.Disconnect(address)
	}
	return
}

func (n *Node) Inject(hash cipher.SHA256,
	pub cipher.PubKey, sec cipher.SecKey) (err error) {

	if sec == (cipher.SecKey{}) {
		err = ErrEmptySecret
		return
	}
	if pub == (cipher.PubKey{}) {
		err = ErrNotFound
		return
	}
	n.enqueueRpcEvent(func() {
		// skip root if we don't share it
		if _, ok := n.subs[pub]; !ok {
			err = ErrNotFound
			return
		}
		root := n.so.Root(pub) // get root by public key
		if root == nil {
			err = ErrNotFound
			return
		}
		// inject the hash
		if err = root.InjectHash(skyobject.Reference(hash)); err != nil {
			return
		}
		root.Touch()            // update timestamp and seq
		n.so.AddRoot(root, sec) // replace with previous and sign
		n.Share(pub)            // send the new root to subscribers
	})
	return
}

// List should be called from RPC server. The List returns all
// connections
func (n *Node) List() []string {
	return n.pool.Connections()
}

//
// TODO: Info
//

// Info is for RPC. It returns all useful inforamtions about the node
// except statistic. I.e. it returns listening address and feed the node
// subscribed to
func (n *Node) Info() (info comm.Info, err error) {
	err = n.enqueueRpcEvent(func() {
		info.Address = n.pool.Address()
		if len(n.subs) == 0 {
			return
		}
		info.Feeds = make([]cipher.PubKey, 0, len(n.subs))
		for pk := range n.subs {
			info.Feeds = append(info.Feeds, pk)
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

// Terminate is the same as Close but designed for RPC.
// If a node created with Config, RemoteClose field of which
// set to true then it's possible to terminate the node
// using RPC. Otherwise, the method returns ErrNotAllowed
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

// Execute some task in main thread to be sure
// that accessing skyobject.Container is
// thread safe
func (n *Node) Execute(task func()) (err error) {
	err = n.enqueueRpcEvent(task)
	return
}

// Want returns missing objects of given root
func (n *Node) Want(pub cipher.PubKey) (w []cipher.SHA256, err error) {
	n.enqueueRpcEvent(func() {
		if _, ok := n.subs[pub]; !ok {
			err = fmt.Errorf("not subscribed to %s", pub.Hex())
			return
		}
		root := n.so.Root(pub)
		if root == nil {
			return // not an error
		}
		var set skyobject.Set
		if set, err = root.Want(); err != nil {
			return
		}
		if len(set) == 0 {
			return // no objects
		}
		w = make([]cipher.SHA256, 0, len(set))
		for k := range set {
			w = append(w, cipher.SHA256(k))
		}
	})
	return
}

// Got returns full list of objects that the root object has got.
// The method returns reference key and size of the object in bytes
func (n *Node) Got(pub cipher.PubKey) (g map[cipher.SHA256]int, err error) {
	n.enqueueRpcEvent(func() {
		if _, ok := n.subs[pub]; !ok {
			err = fmt.Errorf("not subscribed to %s", pub.Hex())
			return
		}
		root := n.so.Root(pub)
		if root == nil {
			return // not an error
		}
		var set skyobject.Set
		if set, err = root.Got(); err != nil {
			return
		}
		if len(set) == 0 {
			return // no objects
		}
		g = make(map[cipher.SHA256]int, len(set))
		for k := range set {
			data, _ := n.db.Get(cipher.SHA256(k))
			g[cipher.SHA256(k)] = len(data)
		}
	})
	return
}
