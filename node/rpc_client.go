package node

import (
	"net/rpc"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/skycoin/src/cipher"
)

type RPCClient struct {
	c *rpc.Client
}

func NewRPCClient(address string) (rc *RPCClient, err error) {
	var c *rpc.Client
	if c, err = rpc.Dial("tcp", address); err != nil {
		return
	}
	rc = new(RPCClient)
	rc.c = c
	return
}

func (r *RPCClient) Want(feed cipher.PubKey) (list []cipher.SHA256, err error) {
	err = r.c.Call("cxo.Want", feed, &list)
	return
}

func (r *RPCClient) Got(feed cipher.PubKey) (list []cipher.SHA256, err error) {
	err = r.c.Call("cxo.Got", feed, &list)
	return
}

func (r *RPCClient) Subscribe(feed cipher.PubKey) (ok bool, err error) {
	err = r.c.Call("cxo.Subscribe", feed, &ok)
	return
}

func (r *RPCClient) Unsubscribe(feed cipher.PubKey) (ok bool, err error) {
	err = r.c.Call("cxo.Unsubscribe", feed, &ok)
	return
}

func (r *RPCClient) Feeds() (list []cipher.PubKey, err error) {
	err = r.c.Call("cxo.Feeds", struct{}{}, &list)
	return
}

func (r *RPCClient) Stat() (stat data.Stat, err error) {
	r.c.Call("cxo.Stat", struct{}{}, &stat)
	return
}

func (r *RPCClient) Connections() (list []string, err error) {
	err = r.c.Call("cxo.Connections", struct{}{}, &list)
	return
}

func (r *RPCClient) IncomingConnections() (list []string, err error) {
	err = r.c.Call("cxo.IncomingConnections", struct{}{}, &list)
	return
}

func (r *RPCClient) OutgoingConnections() (list []string, err error) {
	err = r.c.Call("cxo.OutgoingConnections", struct{}{}, &list)
	return
}

func (r *RPCClient) Connect(address string) (err error) {
	err = r.c.Call("cxo.Connect", address, &struct{}{})
	return
}

func (r *RPCClient) Disconnect(address string) (err error) {
	err = r.c.Call("cxo.Disconnect", address, &struct{}{})
	return
}

func (r *RPCClient) ListeningAddress() (address string, err error) {
	err = r.c.Call("cxo.ListeningAddress", struct{}{}, &address)
	return
}

func (r *RPCClient) Tree(feed cipher.PubKey) (tree []byte, err error) {
	err = r.c.Call("cxo.Tree", feed, &tree)
	return
}

func (r *RPCClient) Terminate() (err error) {
	err = r.c.Call("cxo.Terminate", struct{}{}, &struct{}{})
	return
}

func (r *RPCClient) Close() error {
	return r.c.Close()
}
