package node

import (
	"errors"
	"net"
	"net/rpc"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
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

func newRPC(s *Node) (r *RPC) {
	r = new(RPC)
	r.ns = s
	r.rs = rpc.NewServer()
	return
}

func (r *RPC) Start(address string) (err error) {
	r.rs.RegisterName("cxo", r)
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

func (r *RPC) Address() (address string) {
	if r.l != nil {
		address = r.l.Addr().String()
	}
	return
}

func (r *RPC) Close() (err error) {
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
// - Tree
// - Terminate

func (r *RPC) Want(feed cipher.PubKey, list *[]cipher.SHA256) (_ error) {
	*list = r.ns.Want(feed)
	return
}

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

func (r *RPC) Subscribe(cf ConnFeed, _ *struct{}) (_ error) {
	if cf.Address == "" {
		r.ns.Subscribe(nil, cf.Feed)
		return
	}
	if c := r.ns.Pool().Connection(cf.Address); c != nil {
		r.ns.Subscribe(c, cf.Feed)
		return
	}
	return errors.New("no such connection: " + cf.Address)
}

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

func (r *RPC) Feeds(_ struct{}, list *[]cipher.PubKey) (_ error) {
	*list = r.ns.Feeds()
	return
}

func (r *RPC) Stat(_ struct{}, stat *data.Stat) (_ error) {
	*stat = r.ns.db.Stat()
	return
}

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

func (r *RPC) Connect(address string, _ *struct{}) (err error) {
	_, err = r.ns.pool.Dial(address)
	return
}

func (r *RPC) Disconnect(address string, _ *struct{}) (err error) {
	if c := r.ns.pool.Connection(address); c != nil {
		err = c.Close()
	}
	return errors.New("no such connection: " + address)
}

func (r *RPC) ListeningAddress(_ struct{}, address *string) (_ error) {
	*address = r.ns.pool.Address()
	return
}

// Tree prints objects tree for given root
func (r *RPC) Tree(feed cipher.PubKey, tree *[]byte) (err error) {
	//
	err = errors.New("not implemented yet")
	return
}

func (r *RPC) Terminate(_ struct{}, _ *struct{}) (err error) {
	if !r.ns.conf.RemoteClose {
		err = errors.New("not allowed")
		return
	}
	r.ns.Close()
	return
}
