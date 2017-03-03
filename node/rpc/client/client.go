// Package client is RPC-client for node/. For example
// see cmd/cli
//
package client

import (
	"io"
	"net/rpc"

	"github.com/skycoin/cxo/data"
)

//     "rpc.Connect",     address string,    _       *struct{}
//     "rpc.Disconnect",  address string,    _       *struct{}
//     "rpc.List"         _       struct{},  list    *[]string
//     "rpc.Info",        _       struct{},  address *string
//     "rpc.Stat",        _       struct{},  stat    *data.Stat
//     "rpc.Terminate",   _       struct{},  _       *struct{}

// A Client represents RPC client
type Client struct {
	r *rpc.Client
}

// Dial connects to RPC-server and return Client or an error
func Dial(net string, address string) (c *Client, err error) {
	c = new(Client)
	c.r, err = rpc.Dial(net, address)
	return
}

// Close is used shutdonw the Client
func (c *Client) Close() (err error) {
	err = c.r.Close()
	return
}

//
// RPC methods
//

// Connect to given address
func (c *Client) Connect(address string) (err error) {
	err = c.r.Call("rpc.Connect", address, nil)
	return
}

// Disconnect from given address
func (c *Client) Disconnect(address string) (err error) {
	err = c.r.Call("rpc.Disconnect", address, nil)
	return
}

func (c *Client) List() (list []string, err error) {
	err = c.r.Call("rpc.List", struct{}{}, &list)
	return
}

// Info returns listening address
func (c *Client) Info() (address string, err error) {
	err = c.r.Call("rpc.Info", struct{}{}, &address)
	return
}

// Stat returns database Statistic
func (c *Client) Stat() (stat data.Stat, err error) {
	err = c.r.Call("rpc.Stat", struct{}{}, &stat)
	return
}

// Terminate is used to close node remotely
func (c *Client) Terminate() (err error) {
	err = c.r.Call("rpc.Terminate", struct{}{}, nil)
	if err == io.ErrUnexpectedEOF {
		err = nil
	}
	return
}
