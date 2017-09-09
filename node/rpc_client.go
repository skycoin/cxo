package node

import (
	"net/rpc"

	"github.com/skycoin/skycoin/src/cipher"
)

// A RPCClient represents RPC client to
// control and explore a Node
type RPCClient struct {
	c *rpc.Client
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

// AddFeed to the Node. If given fed already exists, then this
// method does nothing
func (r *RPCClient) AddFeed(pk cipher.PubKey) error {
	return r.c.Call("cxo.AddFeed", pk, &struct{}{})
}

// DelFeed from the Node. The method never
// returns 'not found' error
func (r *RPCClient) DelFeed(pk cipher.PubKey) error {
	return r.c.Call("cxo.DelFeed", pk, &struct{}{})
}

// Subscribe to feed. If address is empty string, then it make
// the node susbcribed to the feed. Otherwise, it subscribes
// to feed of a peer
func (r *RPCClient) Subscribe(address string, feed cipher.PubKey) (err error) {
	var cf ConnFeed
	cf.Address = address
	cf.Feed = feed
	err = r.c.Call("cxo.Subscribe", cf, &struct{}{})
	return
}

// Unsubscribe from a feed. If address is emty string then it
// unsubscribes from feed entirely. Otherwise, from a feed of
// a peer
func (r *RPCClient) Unsubscribe(address string,
	feed cipher.PubKey) (err error) {

	var cf ConnFeed
	cf.Address = address
	cf.Feed = feed
	err = r.c.Call("cxo.Unsubscribe", cf, &struct{}{})
	return
}

// Feeds returns list of feeds the node subscribed to
func (r *RPCClient) Feeds() (list []cipher.PubKey, err error) {
	err = r.c.Call("cxo.Feeds", struct{}{}, &list)
	return
}

// Connections return list of all connections
func (r *RPCClient) Connections() (list []ConnectionInfo, err error) {
	err = r.c.Call("cxo.Connections", struct{}{}, &list)
	return
}

// IncomingConnections returns list of all incoming connections
func (r *RPCClient) IncomingConnections() (list []ConnectionInfo, err error) {
	err = r.c.Call("cxo.IncomingConnections", struct{}{}, &list)
	return
}

// OutgoingConnections returns list of all outgoing connections
func (r *RPCClient) OutgoingConnections() (list []ConnectionInfo, err error) {
	err = r.c.Call("cxo.OutgoingConnections", struct{}{}, &list)
	return
}

// Connect to a peer
func (r *RPCClient) Connect(address string) (err error) {
	err = r.c.Call("cxo.Connect", address, &struct{}{})
	return
}

// Disconnect from a peer
func (r *RPCClient) Disconnect(address string) (err error) {
	err = r.c.Call("cxo.Disconnect", address, &struct{}{})
	return
}

// Connection returns list of feeds of given connection
func (r *RPCClient) Connection(address string) (feeds []cipher.PubKey,
	err error) {

	err = r.c.Call("cxo.Connection", address, &feeds)
	return
}

// Feed returns all connections of given feed
func (r *RPCClient) Feed(pk cipher.PubKey) (addresses []string, err error) {
	err = r.c.Call("cxo.Feed", pk, &addresses)
	return
}

// ListeningAddress of the Node
func (r *RPCClient) ListeningAddress() (address string, err error) {
	err = r.c.Call("cxo.ListeningAddress", struct{}{}, &address)
	return
}

// Info returns brief information about a Node
func (r *RPCClient) Info() (info *Info, err error) {
	info = new(Info)
	err = r.c.Call("cxo.Info", struct{}{}, info)
	return
}

// Roots returns brief information about all root obejts of a feed
func (r *RPCClient) Roots(feed cipher.PubKey) (ris []RootInfo, err error) {
	err = r.c.Call("cxo.Roots", feed, &ris)
	return
}

// Tree returns strigified objects tree of a root object. The
// method useful for inspecting
func (r *RPCClient) Tree(pk cipher.PubKey, seq uint64,
	lastFull bool) (tree string, err error) {

	err = r.c.Call("cxo.Tree", SelectRoot{pk, seq, lastFull}, &tree)
	return
}

// Stat returns statistic
func (r *RPCClient) Stat() (stat Stat, err error) {
	r.c.Call("cxo.Stat", struct{}{}, &stat)
	return
}

// Terminate the node if allowed
func (r *RPCClient) Terminate() (err error) {
	err = r.c.Call("cxo.Terminate", struct{}{}, &struct{}{})
	return
}

// Close the RPCClient
func (r *RPCClient) Close() error {
	return r.c.Close()
}
