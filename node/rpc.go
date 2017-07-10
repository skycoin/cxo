package node

import (
	"errors"
	"net"
	"net/rpc"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

// A RPC is receiver type for net/rpc.
// It used internally by Server and it's
// exported because net/rpc requires
// exported types. You don't need to
// use the RPC manually
type RPC struct {
	l  net.Listener
	rs *rpc.Server
	ns *Node
}

type rpcServer struct {
	*RPC
}

func newRPC(s *Node) (r *rpcServer) {
	r = new(rpcServer)
	r.RPC = new(RPC)
	r.ns = s
	r.rs = rpc.NewServer()
	return
}

func (r *rpcServer) Start(address string) (err error) {
	r.rs.RegisterName("cxo", r.RPC)
	if r.l, err = net.Listen("tcp", address); err != nil {
		return
	}
	r.ns.await.Add(1)
	go func(l net.Listener) {
		defer r.ns.await.Done()
		r.rs.Accept(l)
	}(r.l)
	return
}

func (r *rpcServer) Address() (address string) {
	if r.l != nil {
		address = r.l.Addr().String()
	}
	return
}

func (r *rpcServer) Close() (err error) {
	if r.l != nil {
		err = r.l.Close()
	}
	return
}

//
// RPC methods
//

// - Want
// - Got
// - Subscribe
// - Unsubscribe
// - Feeds
// - Stat
// - Connections
// - IncomingConnections
// - OutgoingConnections
// - Connect
// - Disconnect
// - Associate
// - ListeningAddress
// - Roots
// - Tree
// - Terminate

// Want on a feed
func (r *RPC) Want(feed cipher.PubKey, list *[]cipher.SHA256) (_ error) {
	*list = r.ns.Want(feed)
	return
}

// Got of a feed
func (r *RPC) Got(feed cipher.PubKey, list *[]cipher.SHA256) (_ error) {
	*list = r.ns.Got(feed)
	return
}

// A ConnFeed represetns connection->feed pair. The struct used
// by RPC methods Subscribe and Unsubscribe
type ConnFeed struct {
	Address string // remote address
	Feed    cipher.PubKey
}

// Subscribe to a connection+feed
func (r *RPC) Subscribe(cf ConnFeed, _ *struct{}) (err error) {
	if cf.Address == "" {
		r.ns.Subscribe(nil, cf.Feed)
		return
	}
	if c := r.ns.Pool().Connection(cf.Address); c != nil {
		err = r.ns.SubscribeResponse(c, cf.Feed)
		return
	}
	return errors.New("no such connection: " + cf.Address)
}

// Unsubscribe from a connection+feed
func (r *RPC) Unsubscribe(cf ConnFeed, _ *struct{}) (_ error) {
	if cf.Address == "" {
		r.ns.Unsubscribe(nil, cf.Feed)
		return
	}
	if c := r.ns.Pool().Connection(cf.Address); c != nil {
		r.ns.Unsubscribe(c, cf.Feed)
		return
	}
	return errors.New("no such connection: " + cf.Address)
}

// Feeds of a node
func (r *RPC) Feeds(_ struct{}, list *[]cipher.PubKey) (_ error) {
	*list = r.ns.Feeds()
	return
}

// Stat of database
func (r *RPC) Stat(_ struct{}, stat *data.Stat) (_ error) {
	*stat = r.ns.db.Stat()
	return
}

// Connections of a node
func (r *RPC) Connections(_ struct{}, list *[]string) (_ error) {
	cs := r.ns.pool.Connections()
	if len(cs) == 0 {
		return
	}
	l := make([]string, 0, len(cs))
	for _, c := range cs {
		l = append(l, c.Address())
	}
	*list = l
	return
}

// IncomingConnections of a node
func (r *RPC) IncomingConnections(_ struct{}, list *[]string) (_ error) {
	var l []string
	for _, c := range r.ns.pool.Connections() {
		if c.IsIncoming() {
			l = append(l, c.Address())
		}
	}
	*list = l
	return
}

// OutgoingConnections of a node
func (r *RPC) OutgoingConnections(_ struct{}, list *[]string) (_ error) {
	var l []string
	for _, c := range r.ns.pool.Connections() {
		if !c.IsIncoming() {
			l = append(l, c.Address())
		}
	}
	*list = l
	return
}

// Connect to a remote peer
func (r *RPC) Connect(address string, _ *struct{}) (err error) {
	_, err = r.ns.pool.Dial(address)
	return
}

// Disconnect from a remote peer
func (r *RPC) Disconnect(address string, _ *struct{}) (err error) {
	if c := r.ns.pool.Connection(address); c != nil {
		err = c.Close()
	}
	return errors.New("no such connection: " + address)
}

// ListeningAddress of Node
func (r *RPC) ListeningAddress(_ struct{}, address *string) (_ error) {
	*address = r.ns.pool.Address()
	return
}

// A RootInfo used by RPC
type RootInfo struct {
	Time   time.Time
	Seq    uint64
	Hash   skyobject.RootReference
	IsFull bool
}

// Roots returns basic information about all root obejcts of a feed.
// It returns (by RPC) list sorted from old roots to new
func (r *RPC) Roots(feed cipher.PubKey, roots *[]RootInfo) (_ error) {
	rs := make([]RootInfo, 0)
	r.ns.so.RangeFeed(feed, func(r *skyobject.Root) (_ error) {
		var ri RootInfo
		ri.Time = time.Unix(0, r.Time())
		ri.Seq = r.Seq()
		ri.Hash = r.Hash()
		ri.IsFull = r.IsFull()
		rs = append(rs, ri)
		return
	})
	*roots = rs
	return
}

// Tree prints objects tree of chosen root object (chosen by hash)
func (r *RPC) Tree(hash cipher.SHA256, tree *string) (_ error) {
	root, ok := r.ns.so.RootByHash(skyobject.RootReference(hash))
	if !ok {
		*tree = "<not found>"
		return
	}
	*tree = root.Inspect()
	return
}

// Terminate remote Node if allowed by it s configurations
func (r *RPC) Terminate(_ struct{}, _ *struct{}) (err error) {
	if !r.ns.conf.RemoteClose {
		err = errors.New("not allowed")
		return
	}
	r.ns.Close()
	return
}
