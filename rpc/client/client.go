// Package client is RPC-client for CXO daemon. For example
// see cmd/cli
package client

import (
	"io"
	"net/rpc"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/rpc/comm"
)

//     "rpc.Connect",     address      string,  _       *struct{}
//     "rpc.Disconnect",  address      string,  _       *struct{}
//     "rpc.Inject",      params  comm.Inject,  _       *struct{}
//     "rpc.List"         _          struct{},  list    *[]string
//     "rpc.Want"         pub   cipher.PubKey,  w       []cipher.SHA256
//     "rpc.Got"          pub   cipher.PubKey,  g       map[cipher.SHA256]int
//     "rpc.Info",        _          struct{},  info    *comm.Info
//     "rpc.Stat",        _          struct{},  stat    *data.Stat
//     "rpc.Terminate",   _          struct{},  _       *struct{}

// A Client represents RPC client
type Client struct {
	r *rpc.Client
}

// Dial connects to RPC-server and returns Client or an error
func Dial(net string, address string) (c *Client, err error) {
	c = new(Client)
	c.r, err = rpc.Dial(net, address)
	return
}

// Close is used to shutdonw the Client
func (c *Client) Close() (err error) {
	err = c.r.Close()
	return
}

//
// RPC methods
//

// Connect is used to connect remote node to given address
func (c *Client) Connect(address string) (err error) {
	err = c.r.Call("rpc.Connect", address, nil)
	return
}

// Disconnect is used to disconnect remote node from given address
func (c *Client) Disconnect(address string) (err error) {
	err = c.r.Call("rpc.Disconnect", address, nil)
	return
}

// Subscribe subscribes node to given feed (by public key)
func (c *Client) Subscribe(pub cipher.PubKey) (err error) {
	err = c.r.Call("rpc.Subscribe", pub, nil)
	return
}

// Inject injects hash to root object of the feed (by public key)
func (c *Client) Inject(args comm.Inject) (err error) {
	err = c.r.Call("rpc.Inject", args, nil)
	return
}

// List returns all connections of remote node
func (c *Client) List() (list []string, err error) {
	err = c.r.Call("rpc.List", struct{}{}, &list)
	return
}

// Info returns listening address of remote node
func (c *Client) Info() (info comm.Info, err error) {
	err = c.r.Call("rpc.Info", struct{}{}, &info)
	return
}

// Stat returns database statistic of remote node
func (c *Client) Stat() (stat data.Stat, err error) {
	err = c.r.Call("rpc.Stat", struct{}{}, &stat)
	return
}

// Terminate is used to close remote node if possible.
// If the node was launched with -remote-close=t flag,
// we can to shutdown it remotely
func (c *Client) Terminate() (err error) {
	err = c.r.Call("rpc.Terminate", struct{}{}, nil)
	if err == io.ErrUnexpectedEOF {
		err = nil
	}
	return
}

// Want returns list of missing object for given root
func (c *Client) Want(pub cipher.PubKey) (w []cipher.SHA256, err error) {
	err = c.r.Call("rpc.Want", pub, &w)
	return
}

// Got returns list of objects of given root that the node
// has got. The map containe object key mapped to size of
// object in bytes
func (c *Client) Got(pub cipher.PubKey) (g map[cipher.SHA256]int, err error) {
	err = c.r.Call("rpc.Got", pub, &g)
	return
}
