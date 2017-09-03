package node

import (
	"errors"
	"net"
	"net/rpc"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

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

// AddFeed to the Node
func (r *RPC) AddFeed(pk cipher.PubKey, _ *struct{}) error {
	return r.ns.AddFeed(pk)
}

// DelFeed of the node
func (r *RPC) DelFeed(pk cipher.PubKey, _ *struct{}) error {
	return r.ns.DelFeed(pk)
}

// internal/RPC
type ConnFeed struct {
	Address string // remote address
	Feed    cipher.PubKey
}

// Subscribe to a connection + feed
func (r *RPC) Subscribe(cf ConnFeed, _ *struct{}) (err error) {
	if cf.Address == "" {
		return errors.New("missing address")
	}
	if gc := r.ns.Pool().Connection(cf.Address); gc != nil {
		if c, ok := gc.Value().(*Conn); ok {
			return c.Subscribe(cf.Feed)
		}
	}
	return errors.New("no such connection: " + cf.Address)
}

// Unsubscribe from a connection + feed
func (r *RPC) Unsubscribe(cf ConnFeed, _ *struct{}) (_ error) {
	if cf.Address == "" {
		return errors.New("missing address")
	}
	if gc := r.ns.Pool().Connection(cf.Address); gc != nil {
		if c, ok := gc.Value().(*Conn); ok {
			c.Unsubscribe(cf.Feed)
			return
		}
	}
	return errors.New("no such connection: " + cf.Address)
}

// Feeds of a node
func (r *RPC) Feeds(_ struct{}, list *[]cipher.PubKey) (_ error) {
	*list = r.ns.Feeds()
	return
}

/*
type Stat struct {
	Data data.Stat      // data.DB
	CXO  skyobject.Stat // skyobject.Container
	// TODO: node stat
}

// Stat of database
func (r *RPC) Stat(_ struct{}, stat *Stat) (_ error) {
	*stat = r.ns.Stat()
	return
}
*/

// Connections of a node
func (r *RPC) Connections(_ struct{}, list *[]string) (_ error) {
	cs := r.ns.Connections()
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
	for _, c := range r.ns.Connections() {
		if c.gc.IsIncoming() {
			l = append(l, c.Address())
		}
	}
	*list = l
	return
}

// OutgoingConnections of a node
func (r *RPC) OutgoingConnections(_ struct{}, list *[]string) (_ error) {
	var l []string
	for _, c := range r.ns.Connections() {
		if !c.gc.IsIncoming() {
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
	Time time.Time
	Seq  uint64
	Hash cipher.SHA256
	Prev cipher.SHA256

	CreateTime time.Time
	AccessTime time.Time
	RefsCount  uint32
}

func decodeRoot(val []byte) (r *skyobject.Root, err error) {
	r = new(skyobject.Root)
	if err = encoder.DeserializeRaw(val, r); err != nil {
		r = nil
	}
	return
}

// Roots returns basic information about all root obejcts of a feed.
// It returns (by RPC) list sorted from old roots to new
func (r *RPC) Roots(feed cipher.PubKey, roots *[]RootInfo) (err error) {
	ris := make([]RootInfo, 0)
	err = r.ns.DB().IdxDB().Tx(func(feeds data.Feeds) (err error) {
		var rs data.Roots
		if rs, err = feeds.Roots(feed); err != nil {
			return
		}
		return rs.Ascend(func(ir *data.Root) (err error) {
			var ri RootInfo
			ri.Hash = ir.Hash
			ri.Seq = ir.Seq

			ri.CreateTime = time.Unix(0, ir.CreateTime)
			ri.AccessTime = time.Unix(0, ir.AccessTime)

			var val []byte
			var rc uint32
			if val, rc, err = r.ns.DB().CXDS().Get(ir.Hash); err != nil {
				return
			}

			ri.RefsCount = rc

			var r *skyobject.Root
			if r, err = decodeRoot(val); err != nil {
				return
			}

			ri.Time = time.Unix(0, r.Time)
			ri.Prev = r.Prev

			ris = append(ris, ri)
			return
		})
	})
	if err != nil {
		return
	}
	*roots = ris
	return
}

type SelectRoot struct {
	Pub      cipher.PubKey
	Seq      uint64
	LastFull bool // ignore the seq and print last full of the feed
}

// Tree prints objects tree of chosen root object (chosen by pk+seq)
func (r *RPC) Tree(sel SelectRoot, tree *string) (err error) {
	var root *skyobject.Root
	if sel.LastFull == true {
		root, err = r.ns.so.LastRoot(sel.Pub)
	} else {
		root, err = r.ns.so.Root(sel.Pub, sel.Seq)
	}
	if err != nil {
		return
	}
	if root == nil {
		*tree = "<not found>"
	}
	*tree = r.ns.so.Inspect(root)
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
