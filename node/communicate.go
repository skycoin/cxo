package node

import (
	"errors"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"
)

func init() {
	// gnet.Message interface
	var (
		_ gnet.Message = &Announce{}
		_ gnet.Message = &Request{}
		_ gnet.Message = &Data{}
	)

	// Hm, must be non-pointer
	gnet.RegisterMessage(gnet.MessagePrefixFromString("ANN"), Announce{})
	gnet.RegisterMessage(gnet.MessagePrefixFromString("REQ"), Request{})
	gnet.RegisterMessage(gnet.MessagePrefixFromString("DAT"), Data{})

	gnet.VerifyMessages()
}

var (
	ErrUnexpectedMessage gnet.DisconnectReason = errors.New(
		"unexpected message")
)

// An Announce message sent to subscribers
// when new object given
type Announce struct {
	Hash cipher.SHA256
}

func (a *Announce) Handle(ctx *gnet.MessageContext, ni interface{}) error {
	var node Node = getNodeFromState(ni)
	// must be at inflow side
	if !node.isInflow(ctx) {
		node.Debug("[ERR] got announce message at feed side")
		return ErrUnexpectedMessage
	}
	// do we have the object?
	if node.DB().Has(a.Hash) {
		return nil
	}
	// we don't have the object; let's request it
	ctx.Conn.ConnectionPool.SendMessage(ctx.Conn, &Request{
		a.Hash,
	})
	return nil
}

// A Request sent to feed when hash of received Annonce
// doesn't exists in DB
type Request struct {
	Hash cipher.SHA256
}

func (r *Request) Handle(ctx *gnet.MessageContext, ni interface{}) error {
	var (
		node Node = getNodeFromState(ni)
		data []byte
		ok   bool
	)
	// must be at feed side
	if !node.isFeed(ctx) {
		node.Debug("[ERR] got request message at inflow side")
		return ErrUnexpectedMessage // terminate connection
	}
	// obtain requested object from DB
	if data, ok = node.DB().Get(r.Hash); !ok {
		node.Debug("[WRN] request object I don't have")
		return nil // keep connection
	}
	// send requested object back
	ctx.Conn.ConnectionPool.SendMessage(ctx.Conn, &Data{
		Data: data,
	})
	return nil
}

// A Data is a new object
type Data struct {
	Data []byte
}

func (d *Data) Handle(ctx *gnet.MessageContext, ni interface{}) error {
	var (
		node Node = getNodeFromState(ni)

		data []byte
		err  error

		hash cipher.SHA256

		val interface{}

		ref Referencer
		ok  bool
	)
	// must be at inflow side
	if !node.isInflow(ctx) {
		node.Debug("[ERR] got data message at feed side")
		return ErrUnexpectedMessage
	}
	// check message length
	if len(d.Data) < 4 {
		node.Print("[ERR] recived data is too short")
		return ErrMalformedMessage // terminate connection
	}
	// see hash
	hash = cipher.SumSHA256(d.Data)
	// do we already have it?
	if node.DB().Has(hash) {
		node.Debug("receive data I already have")
		return nil // keep connection
	}
	// decode data
	if val, err = node.Encoder().Decode(d.Data); err != nil {
		node.Print("[ERR] error decoding incomind data message")
		return ErrMalformedMessage // terminate connection
	}
	// ok, data was decoded successfully, it's safe to store it now
	node.DB().Set(hash, data)
	// and safe to send announce to my subscribers
	node.Broadcast(hash)
	// check references
	if ref, ok = val.(Referencer); ok {
		// range over references of the object
		for _, h := range ref.References() {
			// if we don't have ...
			if !node.DB().Has(h) {
				// ... send requst back to get it
				ctx.Conn.ConnectionPool.SendMessage(ctx.Conn, &Request{
					Hash: h,
				})
			}
		}
	}
	return nil
}

// A Referencer implements a message that references to
// another messages
type Referencer interface {
	// References returns complite set of all references
	References() []cipher.SHA256
}
