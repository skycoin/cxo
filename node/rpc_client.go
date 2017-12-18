package node

import (
	"net/rpc"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject/registry"
)

// A RPCClient represents client for
// RPC methods of the Node
type RPCClient struct {
	c *rpc.Client
}

// Close client
func (r *RPCClient) Close() (err error) {
	return r.c.Close()
}

// Node related methods
func (r *RPCClient) Node() (n *RPCClientNode) {
	return &RPCClientNode{r}
}

// TCP related methods
func (r *RPCClient) TCP() (t *RPCClientTCP) {
	return &RPCClientTCP{r}
}

// UDP related methods
func (r *RPCClient) UDP() (t *RPCClientUDP) {
	return &RPCClientUDP{r}
}

// Root objects related methods
func (r *RPCClient) Root() (t *RPCClientRoot) {
	return &RPCClientRoot{r}
}

// NewRPCClient creates RPC client connected to RPC server with
// given address
func NewRPCClient(address string) (rc *RPCClient, err error) {
	var c *rpc.Client
	if c, err = rpc.Dial("tcp", address); err != nil {
		return
	}
	rc = new(RPCClient)
	rc.c = c
	return
}

// An RPCCLientNode implements RPC
// methods related to the Node
type RPCClientNode struct {
	r *RPCClient
}

// Share given feed
func (r *RPCClientNode) Share(pk cipher.PubKey) (err error) {
	return r.r.c.Call("node.Share", pk, &struct{}{})
}

func (r *RPCClientNode) DontShare(pk cipher.PubKey) (err error) {
	return r.r.c.Call("node.DontShare", pk, &struct{}{})
}

func (r *RPCClientNode) Feeds() (fs []cipher.PubKey, err error) {
	err = r.r.c.Call("node.Feeds", struct{}{}, &fs)
	return
}

func (r *RPCClientNode) IsSharing(pk cipher.PubKey) (yep bool, err error) {
	err = r.r.c.Call("node.IsSharing", struct{}{}, &yep)
	return
}

func (r *RPCClientNode) Connections() (cs []string, err error) {
	err = r.r.c.Call("node.Connections", struct{}{}, &cs)
	return
}

func (r *RPCClientNode) Config() (config *Config, err error) {
	var c Config
	if err = r.r.c.Call("node.Config", struct{}{}, &c); err != nil {
		return
	}
	return &c, nil
}

func (r *RPCClientNode) Stat() (stat *Stat, err error) {
	var s Stat
	if err = r.r.c.Call("node.Stat", struct{}{}, &s); err != nil {
		return
	}
	return &s, nil
}

// A RPCClientTCP implements RPC
// methods related to TCP transport
type RPCClientTCP struct {
	r *RPCClient
}

func (r *RPCClientTCP) Connect(address string) (err error) {
	return r.r.c.Call("tcp.Connect", address, &struct{}{})
}

func (r *RPCClientTCP) Disconnect(address string) (err error) {
	return r.r.c.Call("tcp.Disconnect", address, &struct{}{})
}

func (r *RPCClientTCP) Subscribe(address string, pk cipher.PubKey) (err error) {
	return r.r.c.Call("tcp.Subscribe", ConnFeed{address, pk}, &struct{}{})
}

func (r *RPCClientTCP) Unsubscribe(
	address string,
	pk cipher.PubKey,
) (
	err error,
) {
	return r.r.c.Call("tcp.Unsubscribe", ConnFeed{address, pk}, &struct{}{})
}

func (r *RPCClientTCP) RemoteFeeds(
	address string, //      :
) (
	rfs []cipher.PubKey, // :
	err error, //           :
) {
	err = r.r.c.Call("tcp.RemoteFeeds", address, &rfs)
	return
}

func (r *RPCClientTCP) Address() (address string, err error) {
	err = r.r.c.Call("tcp.Address", struct{}{}, &address)
	return
}

// A RPCClientUDP implements RPC
// methods related to UDP transport
type RPCClientUDP struct {
	r *RPCClient
}

func (r *RPCClientUDP) Connect(address string) (err error) {
	return r.r.c.Call("udp.Connect", address, &struct{}{})
}

func (r *RPCClientUDP) Disconnect(address string) (err error) {
	return r.r.c.Call("udp.Disconnect", address, &struct{}{})
}

func (r *RPCClientUDP) Subscribe(address string, pk cipher.PubKey) (err error) {
	return r.r.c.Call("udp.Subscribe", ConnFeed{address, pk}, &struct{}{})
}

func (r *RPCClientUDP) Unsubscribe(
	address string,
	pk cipher.PubKey,
) (
	err error,
) {
	return r.r.c.Call("udp.Unsubscribe", ConnFeed{address, pk}, &struct{}{})
}

func (r *RPCClientUDP) RemoteFeeds(
	address string, //      :
) (
	rfs []cipher.PubKey, // :
	err error, //           :
) {
	err = r.r.c.Call("udp.RemoteFeeds", address, &rfs)
	return
}

func (r *RPCClientUDP) Address() (address string, err error) {
	err = r.r.c.Call("udp.Address", struct{}{}, &address)
	return
}

// A RPCClientRoot implements RPC
// methods related to Root objects
type RPCClientRoot struct {
	r *RPCClient
}

func (r *RPCClientRoot) Show(
	feed cipher.PubKey,
	nonce uint64,
	seq uint64,
) (
	z *registry.Root,
	err error,
) {

	var x registry.Root
	err = r.r.c.Call("root.Show", RootSelector{feed, nonce, seq}, &x)
	if err != nil {
		return
	}
	return &x, nil
}

func (r *RPCClientRoot) Tree(
	feed cipher.PubKey,
	nonce uint64,
	seq uint64,
) (
	tree string,
	err error,
) {
	err = r.r.c.Call("root.Tree", RootSelector{feed, nonce, seq}, &tree)
	return
}

func (r *RPCClientRoot) Last(
	feed cipher.PubKey,
) (
	z *registry.Root,
	err error,
) {

	var x registry.Root
	if err = r.r.c.Call("root.Last", feed, &x); err != nil {
		return
	}
	return &x, nil
}
