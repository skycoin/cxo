package node

import (
	"github.com/iketheadore/skycoin/src/daemon/gnet"
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject"
)

func init() {
	gnet.RegisterMessage(gnet.MessagePrefixFromString("PING"), Ping{})
	gnet.RegisterMessage(gnet.MessagePrefixFromString("PONG"), Pong{})
	gnet.RegisterMessage(gnet.MessagePrefixFromString("ANNC"), Announce{})
	gnet.RegisterMessage(gnet.MessagePrefixFromString("REQT"), Request{})
	gnet.RegisterMessage(gnet.MessagePrefixFromString("DATA"), Data{})
	gnet.VerifyMessages()
}

// A Ping is used to keep conections alive
type Ping struct{}

// Handle implements *gnet.Message interface and sends Pong back
func (p *Ping) Handle(ctx *gnet.MessageContext, node interface{}) (_ error) {
	var n *Node = node.(*Node)
	n.pool.SendMessage(ctx.Addr, &Pong{})
	return
}

// A Pong is just reply for Ping
type Pong struct{}

func (*Pong) Handle(_ *gnet.MessageContext, _ interface{}) (_ error) {
	return
}

// Announce is used to notify remote node about data the node has got
type Announce struct {
	Hash cipher.SHA256
}

func (a *Announce) Handle(ctx *gnet.MessageContext,
	node interface{}) (terminate error) {

	var n *Node = node.(*Node)
	if n.hot == nil { // we don't have a root yet
		return
	}
	if _, ok := n.hot[a.Hash]; !ok { // don't want this object
		return
	}
	n.pool.SendMessage(ctx.Addr, &Request{a.Hash})
	return
}

// Request is used to request data
type Request struct {
	Hash cipher.SHA256
}

func (r *Request) Handle(ctx *gnet.MessageContext,
	node interface{}) (terminate error) {

	var n *Node = node.(*Node)
	if data, ok := n.db.Get(r.Hash); ok {
		n.pool.SendMessage(ctx.Addr, &Data{data})
	}
	return
}

// Data is a pice of data
type Data struct {
	Data []byte
}

func (d *Data) Handle(ctx *gnet.MessageContext,
	node interface{}) (terminate error) {

	var n *Node = node.(*Node)
	if n.hot == nil {
		return // i don't want any data (i don't have root object yet)
	}
	hash := cipher.SumSHA256(d.Data)
	if _, ok := n.hot[hash]; !ok {
		return // i don't want the data
	}
	n.db.Set(hash, d.Data)
	delete(n.hot, hash)
	// Broadcast
	for _, gc := range n.pool.GetConnections() {
		if gc.Id != ctx.ConnID {
			n.pool.SendMessage(gc.Addr(), &Announce{hash})
		}
	}
	return
}

// Root contains root object
type Root struct {
	Root []byte
}

func (r *Root) Handle(ctx *gnet.MessageContext,
	node interface{}) (terminate error) {

	var n *Node = node.(*Node)
	var root *skyobject.Root
	if root, terminate = skyobject.DecodeRoot(r.Root); terminate != nil {
		return // terminate connection that sends malformed messages
	}
	if !n.so.SetRoot(root) {
		// older or the same
		return
	}
	// refresh list of wanted objects
	n.hot, _ = n.so.Want()
	// Broadcast the root
	for _, gc := range n.pool.GetConnections() {
		if gc.Id != ctx.ConnID {
			n.pool.SendMessage(gc.Addr(), r)
		}
	}
	return
}
