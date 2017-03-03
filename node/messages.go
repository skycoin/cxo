package node

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"
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

	n := node.(*Node)
	if n.db.Has(a.Hash) {
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

	n := node.(*Node)
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

	// TODO: drop data I don't asking for

	n := node.(*Node)
	hash := cipher.SumSHA256(d.Data)
	n.db.Set(hash, d.Data)
	// Broadcast
	for _, gc := range n.pool.Pool {
		if gc != ctx.Conn {
			n.pool.SendMessage(gc, &Announce{hash})
		}
	}
	return
}
