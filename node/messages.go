package node

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"

	"github.com/skycoin/cxo/skyobject"
)

func init() {
	gnet.RegisterMessage(gnet.MessagePrefixFromString("PING"), Ping{})
	gnet.RegisterMessage(gnet.MessagePrefixFromString("PONG"), Pong{})
	gnet.RegisterMessage(gnet.MessagePrefixFromString("ANNC"), Announce{})
	gnet.RegisterMessage(gnet.MessagePrefixFromString("RQST"), Request{})
	gnet.RegisterMessage(gnet.MessagePrefixFromString("DATA"), Data{})
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

// Announce new and existing data
type Announce struct {
	Hash cipher.SHA256
}

func (a *Announce) Handle(ctx *gnet.MessageContext, n interface{}) (_ error) {
	n.(*Node).enqueueMsgEvent(a, ctx.Addr)
	return
}

func (n *Node) handleAnnounce(a *Announce, address string, want skyobject.Set) {
	if _, ok := want[skyobject.Reference(a.Hash)]; ok { // want the object
		n.pool.SendMessage(address, &Request{a.Hash})
	}
}

// Requset data
type Request struct {
	Hash cipher.SHA256
}

func (r *Request) Handle(ctx *gnet.MessageContext, n interface{}) (_ error) {
	n.(*Node).enqueueMsgEvent(r, ctx.Addr)
	return
}

func (n *Node) handleRequest(r *Request, address string) {
	if data, ok := n.db.Get(r.Hash); ok {
		n.pool.SendMessage(address, &Data{data})
	}
}

// receive data
type Data struct {
	Data []byte
}

func (d *Data) Handle(ctx *gnet.MessageContext, n interface{}) (_ error) {
	n.(*Node).enqueueMsgEvent(d, ctx.Addr)
	return
}

func (n *Node) handleData(d *Data, address string, want skyobject.Set) {
	hash := cipher.SumSHA256(d.Data)
	if _, ok := want[skyobject.Reference(hash)]; !ok { // want the object
		return
	}
	n.db.Set(hash, d.Data)
	delete(want, skyobject.Reference(hash))
	n.wanted(want)
	// TODO: gnet needs exclusive broadcast
	n.pool.BroadcastMessage(&Announce{hash}) // notify about new data
}

// all current wanted objects
func (n *Node) wanted(want skyobject.Set) {
	for pub := range n.subs {
		n.wantOfRoot(n.so.Root(pub), want)
	}
}

func (n *Node) wantOfRoot(root *skyobject.Root, want skyobject.Set) {
	if root == nil {
		return
	}
	set, err := root.Want()
	if err != nil {
		n.Debug("[ERR] ", err)
		return
	}
	for ref := range set {
		want[ref] = struct{}{}
	}
}

// AnnounceRoot used to notify about new and wanted root objects
type AnnounceRoot struct {
	Pub  cipher.PubKey
	Time int64
}

func (a *AnnounceRoot) Handle(ctx *gnet.MessageContext,
	n interface{}) (_ error) {

	n.(*Node).enqueueMsgEvent(a, ctx.Addr)
	return
}

func (n *Node) handleAnnounceRoot(ar *AnnounceRoot, address string) {
	if _, ok := n.subs[ar.Pub]; !ok {
		return
	}
	root := n.so.Root(ar.Pub)
	if root != nil {
		if root.Time > ar.Time {
			return
		}
	}
	n.pool.SendMessage(address, &RequestRoot{ar.Pub})
}

type RequestRoot struct {
	Pub cipher.PubKey
}

func (r *RequestRoot) Handle(ctx *gnet.MessageContext,
	n interface{}) (_ error) {

	n.(*Node).enqueueMsgEvent(r, ctx.Addr)
	return
}

func (n *Node) handleRequestRoot(rr *RequestRoot, address string) {
	root := n.so.Root(rr.Pub)
	if root == nil {
		return
	}
	n.pool.SendMessage(address, &DataRoot{
		Pub:  root.Pub,
		Sig:  root.Sig,
		Root: root.Encode(),
	})
	got, _ := root.Got()
	for k := range got {
		n.pool.SendMessage(address, &Announce{cipher.SHA256(k)})
	}
}

// DataRoot objects
type DataRoot struct {
	Pub  cipher.PubKey
	Sig  cipher.Sig
	Root []byte
}

func (d *DataRoot) Handle(ctx *gnet.MessageContext,
	n interface{}) (_ error) {

	n.(*Node).enqueueMsgEvent(d, ctx.Addr)
	return
}

func (n *Node) handleDataRoot(dr *DataRoot, address string,
	want skyobject.Set) {

	if _, ok := n.subs[dr.Pub]; !ok {
		return
	}
	ok, err := n.so.AddEncodedRoot(dr.Root, dr.Pub, dr.Sig)
	if err != nil {
		n.Print("[ERR] ", err)
		return
	}
	if !ok {
		n.Debug("[DBG] older root received")
		return
	}
	// n.wantOfRoot(n.so.Root(dr.Pub), want)
	root := n.so.Root(dr.Pub)
	if root == nil {
		return
	}
	set, err := root.Want()
	if err != nil {
		n.Debug("[ERR] ", err)
		return
	}
	for ref := range set {
		want[ref] = struct{}{}                                    // merge
		n.pool.SendMessage(address, &Request{cipher.SHA256(ref)}) // request
	}
}
