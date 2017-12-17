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

// TCPConnect is RPC method
func (r *RPC) TCPConnect(address string, _ *struct{}) (err error) {
	_, err = r.n.TCP().Connect(address)
	return
}

// UDPConnect is RPC method
func (r *RPC) UDPConnect(address string, _ *struct{}) (err error) {
	_, err = r.n.UDP().Connect(address)
}

// TCPDisconnect is RPC method
func (r *RPC) TCPDisconnect(address string, _ *struct{}) (err error) {
	var c *Conn
	if c, err = r.n.TCP().Connect(address); err != nil {
		return
	}
	err = c.Close()
	return
}

// UDPDisconnect is RPC method
func (r *RPC) UDPDisconnect(address string, _ *struct{}) (err error) {
	var c *Conn
	if c, err = r.n.UDP().Connect(address); err != nil {
		return
	}
	err = c.Close()
	return
}

// A ConnFeed represents conenction address and feed
type ConnFeed struct {
	Address string
	Feed    cipher.PubKey
}

// TCPSubscribe is RPC method
func (r *RPC) TCPSubscribe(cf ConnFeed, _ *struct{}) (err error) {
	var c *Conn
	if c, err = r.n.TCP().Connect(cf.Address); err != nil {
		return
	}
	return c.Subscribe(cf.Feed)
}

// UDPSubscribe is RPC method
func (r *RPC) UDPSubscribe(cf ConnFeed, _ *struct{}) (err error) {
	var c *Conn
	if c, err = r.n.UDP().Connect(cf.Address); err != nil {
		return
	}
	return c.Subscribe(cf.Feed)
}

// TCPUnsubscribe is RPC method
func (r *RPC) TCPUnsubscribe(cf ConnFeed, _ *struct{}) (err error) {
	var c *Conn
	if c, err = r.n.TCP().Connect(cf.Address); err != nil {
		return
	}
	return c.Unsubscribe(cf.Feed)
}

// UDPUnsubscribe is RPC method
func (r *RPC) UDPUnsubscribe(cf ConnFeed, _ *struct{}) (err error) {
	var c *Conn
	if c, err = r.n.UDP().Connect(cf.Address); err != nil {
		return
	}
	return c.Unsubscribe(cf.Feed)
}

// TCPRemoteFeeds is RPC method
func (r *RPC) TCPRemoteFeeds(address string, rfs *[]cipher.PubKey) (err error) {
	var c *Conn
	if c, err = r.n.TCP().Connect(address); err != nil {
		return
	}
	var fs []cipher.PubKey
	if fs, err = c.RemoteFeeds(); err != nil {
		return
	}
	*rfs = fs
	return
}

// UDPRemoteFeeds is RPC method
func (r *RPC) UDPRemoteFeeds(address string, rfs *[]cipher.PubKey) (err error) {
	var c *Conn
	if c, err = r.n.UDP().Connect(address); err != nil {
		return
	}
	var fs []cipher.PubKey
	if fs, err = c.RemoteFeeds(); err != nil {
		return
	}
	*rfs = fs
	return
}

// strings with all connections
func (n *Node) connections() (cs []string) {
	n.mx.Lock()
	defer n.mx.Unlock()

	cs = make([]string, 0, len(n.pc)+len(n.ic))

	for c := range n.pc {
		cs = append(cs, c.String()+"(⌛)") // pending
	}

	for _, c := range n.ic {
		cs = append(cs, c.String()+"(✓)") // established
	}

	return
}

// Connections is RPC method
func (r *RPC) Connections(_ struct{}, cs *[]string) (_ error) {
	*cs = r.n.connections()
	return
}

// TCPAddress is RPC method
func (r *RPC) TCPAddress(_ struct{}, address *string) (_ error) {
	*address = r.n.TCP().Address()
	return
}

// UDPAddress is RPC method
func (r *RPC) UDPAddress(_ struct{}, address *string) (_ error) {
	*address = r.n.UDP().Address()
	return
}

// An Info represents brief information about
// the Node and used by RPC methods
type Info struct {
	TCP          string    // listening address or blank string
	UDP          string    // listening address or blank string
	Public       bool      // is public
	TCPDiscovery Addresses // tcp discovery server
	UDPDiscovery Addresses // udp discovery server
}

func (r *RPC) Info(_ struct{}, info *Info) (err error) {

	//

}
