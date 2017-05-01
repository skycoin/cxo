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
	ns *Server
}

func newRPC(s *Server) (r *RPC) {
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
// - AddFeed
// - DelFeed
// - Feeds
// - Stat
// - Connections
// - IncomingConnections
// - OutgoingConnections
// - Connect
// - Disconnect
// - ListeningAddress
// - Tree
// - Terminate

func (r *RPC) Want(feed cipher.PubKey, list *[]cipher.SHA256) (_ error) {
	*list = r.ns.Want(feed)
	return
}

func (r *RPC) Got(feed cipher.PubKey, list *[]cipher.SHA256) (err error) {
	*list, err = r.ns.Got(feed)
	return
}

func (r *RPC) AddFeed(feed cipher.PubKey, ok *bool) (err error) {
	*ok = r.ns.AddFeed(feed)
	return
}

func (r *RPC) DelFeed(feed cipher.PubKey, ok *bool) (err error) {
	*ok = r.ns.DelFeed(feed)
	return
}

func (r *RPC) Feeds(_ struct{}, list *[]cipher.PubKey) (_ error) {
	*list = r.ns.Feeds()
	return
}

func (r *RPC) Stat(_ struct{}, stat *data.Stat) (_ error) {
	*stat = r.ns.Stat()
	return
}

func (r *RPC) Connections(_ struct{}, list *[]string) (_ error) {
	*list = r.ns.Connections()
	return
}

func (r *RPC) IncomingConnections(_ struct{}, list *[]string) (_ error) {
	var l []string
	for _, address := range r.ns.pool.Connections() {
		c := r.ns.pool.Connection(address)
		if c.IsIncoming() {
			l = append(l, address)
		}
	}
	*list = l
	return
}

func (r *RPC) OutgoingConnections(_ struct{}, list *[]string) (_ error) {
	var l []string
	for _, address := range r.ns.pool.Connections() {
		c := r.ns.pool.Connection(address)
		if !c.IsIncoming() {
			l = append(l, address)
		}
	}
	*list = l
	return
}

func (r *RPC) Connect(address string, _ *struct{}) (err error) {
	err = r.ns.Connect(address)
	return
}

func (r *RPC) Disconnect(address string, _ *struct{}) (err error) {
	err = r.ns.Disconnect(address)
	return
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
