package node

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node/gnet"
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
func (p *Ping) Handle(ctx *gnet.MessageContext, _ interface{}) (_ error) {
	ctx.Conn.ConnectionPool.SendMessage(ctx.Conn, &Pong{})
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
	if len(n.hot) == 0 { // we don't have a root yet or it's full
		return
	}
	if _, ok := n.hot[a.Hash]; !ok { // don't want this object
		return
	}
	ctx.Conn.ConnectionPool.SendMessage(ctx.Conn, &Request{a.Hash})
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
		ctx.Conn.ConnectionPool.SendMessage(ctx.Conn, &Data{data})
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
	if len(n.hot) == 0 {
		// i don't have root object yet or it's full (i don't want any data)
		return
	}
	hash := cipher.SumSHA256(d.Data)
	if _, ok := n.hot[hash]; !ok {
		return // i don't want the data
	}
	n.db.Set(hash, d.Data)
	delete(n.hot, hash)
	// Broadcast
	for _, gc := range n.pool.Pool {
		if gc.Id != ctx.Conn.Id {
			n.pool.SendMessage(gc, &Announce{hash})
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

	var (
		n  *Node = node.(*Node)
		ok bool
	)
	if ok, terminate = n.so.SetEncodedRoot(r.Root); terminate != nil {
		return // terminate connection that sends malformed messages
	}
	if !ok { // older or the same
		return
	}
	// refresh list of wanted objects
	n.hot, _ = n.so.Want()
	// Broadcast the root
	for id, gc := range n.pool.Pool {
		if id != ctx.Conn.Id {
			n.pool.SendMessage(gc, r)
		}
	}
	return
}
