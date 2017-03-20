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
	gnet.RegisterMessage(gnet.MessagePrefixFromString("ROOT"), Root{})
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

// An Announce message is used to notify other nodes about new data we have got
type Announce struct {
	Hash cipher.SHA256
}

func (a *Announce) Handle(ctx *gnet.MessageContext,
	node interface{}) (_ error) {

	node.(*Node).enqueueMsgEvent(a, ctx.Addr)
	return
}

// A Request is used to request data we want
type Request struct {
	Hash cipher.SHA256
}

func (r *Request) Handle(ctx *gnet.MessageContext,
	node interface{}) (_ error) {

	node.(*Node).enqueueMsgEvent(r, ctx.Addr)
	return
}

// A Data is an encoded ([]byte) object we send
type Data struct {
	Data []byte
}

func (d *Data) Handle(ctx *gnet.MessageContext,
	node interface{}) (_ error) {

	node.(*Node).enqueueMsgEvent(d, ctx.Addr)
	return
}

// A Root contains encoded ([]byte) root object
type Root struct {
	Sign cipher.Sig
	Pub  cipher.PubKey
	Root []byte
}

func (r *Root) Handle(ctx *gnet.MessageContext,
	node interface{}) (_ error) {

	node.(*Node).enqueueMsgEvent(r, ctx.Addr)
	return
}
