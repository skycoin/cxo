package node

import (
	"net"
	"net/rpc"

	"github.com/skycoin/skycoin/src/cipher"
)

// wrap the RPC
type rpcServer struct {
	*RPC
}

// create RPC server
func (n *Node) createRPC() (r *rpcServer) {
	r = new(rpcServer)
	r.RPC = new(RPC)
	r.n = n
	r.r = rpc.NewServer()
	return
}

func (r *rpcServer) Listen(address string) (err error) {
	r.r.RegisterName("cxo", r.RPC)
	if r.l, err = net.Listen("tcp", address); err != nil {
		return
	}
	r.n.await.Add(1)
	go r.run()
}

func (r *rpcServer) run() {
	defer r.n.await.Done()
	r.r.Accept(r.l)
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

// A RPC represnet RPC server of the Node.
// The RPC is exported because the net/rpc
// package requires it. E.g. the RPC is
// registered by the rpc package. The RPC
// can't be used outside the Node.
//
// Short words, the RPC is internal
type RPC struct {
	n *Node        // back reference
	l net.Listener // underlying listener
	r *rpc.Server  //
}

// Share is RPC method
func (r *RPC) Share(pk cipher.PubKey, _ *struct{}) (err error) {
	return r.n.Share(pk)
}

// DontShare is RPC method
func (r *RPC) DontShare(pk cipher.PubKey, _ *struct{}) (err error) {
	return r.n.DontShare(pk)
}

// Feeds is RPC method
func (r *RPC) Feeds(_ struct{}, fs *[]cipher.PubKey) (_ error) {
	*fs = r.n.Feeds()
	return
}

// IsSharing is RPC method
func (r *RPC) IsSharing(pk cipher.PubKey, yep *bool) (_ error) {
	*yep = r.n.IsSharing(pk)
	return
}
