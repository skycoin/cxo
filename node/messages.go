package node

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"
)

func init() {
	// service
	gnet.RegisterMessage(gnet.MessagePrefixFromString("PING"), Ping{})
	gnet.RegisterMessage(gnet.MessagePrefixFromString("PONG"), Pong{})

	// data
	gnet.RegisterMessage(gnet.MessagePrefixFromString("ANNC"), Announce{})
	gnet.RegisterMessage(gnet.MessagePrefixFromString("RQST"), Request{})
	gnet.RegisterMessage(gnet.MessagePrefixFromString("DATA"), Data{})

	// root
	gnet.RegisterMessage(gnet.MessagePrefixFromString("ANRT"), AnnounceRoot{})
	gnet.RegisterMessage(gnet.MessagePrefixFromString("RQRT"), RequestRoot{})
	gnet.RegisterMessage(gnet.MessagePrefixFromString("DTRT"), DataRoot{})

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

//
// data
//

type Announce struct {
	Hashes []cipher.SHA256
}

func (a *Announce) Handle(c *gnet.MessageContext, n interface{}) (_ error) {
	n.(*Node).enqueueMsgEvent(a, c.Addr)
	return
}

type Request struct {
	Hashes []cipher.SHA256
}

func (r *Request) Handle(c *gnet.MessageContext, n interface{}) (_ error) {
	n.(*Node).enqueueMsgEvent(r, c.Addr)
	return
}

type Data struct {
	Data [][]byte
}

func (d *Data) Handle(c *gnet.MessageContext, n interface{}) (_ error) {
	n.(*Node).enqueueMsgEvent(d, c.Addr)
	return
}

//
// Root
//

type RootHead struct {
	Pub  cipher.PubKey
	Time int64
}

type AnnounceRoot struct {
	Roots []RootHead
}

func (a *AnnounceRoot) Handle(c *gnet.MessageContext, n interface{}) (_ error) {
	n.(*Node).enqueueMsgEvent(a, c.Addr)
	return
}

type RequestRoot struct {
	Roots []RootHead
}

func (r *RequestRoot) Handle(c *gnet.MessageContext, n interface{}) (_ error) {
	n.(*Node).enqueueMsgEvent(r, c.Addr)
	return
}

type RootBody struct {
	Pub  cipher.PubKey
	Sig  cipher.Sig
	Root []byte
}

type DataRoot struct {
	Roots []RootBody
}

func (d *DataRoot) Handle(c *gnet.MessageContext, n interface{}) (_ error) {
	n.(*Node).enqueueMsgEvent(d, c.Addr)
	return
}
