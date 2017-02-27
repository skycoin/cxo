package client

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"
)

func init() {
	// check implementation of required interfaces
	var (
		_ node.OutgoingHandler = &Announce{}
		_ node.IncomingHandler = &Request{}
		_ node.OutgoingHandler = &Data{}
	)
}

// A Replier is context of a message
type Replier interface {
	Reply(msg interface{}) (err error) // reply for message
}

// A feed sends Announce message with hash of new object it has
type Announce struct {
	Hash cipher.SHA256
}

// handle on side of subscriber
func (ann *Announce) HandleOutgoing(ctx node.MsgContext,
	client interface{}) (terminate error) {

	var c *Client = client.(*Client)

	if c.db.Has(ann.Hash) {
		logger.Info("got announce for data I already have: %v",
			ann.Hash.Hex())
		return // nil (keep connection alive)
	}
	c.requestMissing(ctx, ann.Hash)
	return // nil (keep connection alive)
}

// A subscriber sends Request if it has got Announce with hash of
// object it doesn't have
type Request struct {
	Hash cipher.SHA256
}

// handled from feed
func (req *Request) HandleIncoming(ctx node.MsgContext,
	client interface{}) (terminate error) {

	var (
		c *Client = client.(*Client)

		data []byte
		ok   bool
	)

	if data, ok = c.db.Get(req.Hash); !ok {
		logger.Error("got request for data I don't have: %v",
			req.Hash.Hex())
		return // keep connection alive
	}
	// send back the response with the requested data
	err := ctx.Reply(Data{
		Hash: req.Hash,
		Data: data,
	})
	if err != nil {
		logger.Error("error sending reply: %v", err)
	}
	return // keep alive
}

// Data is response fro request. It sent by feed to subscriber
type Data struct {
	Hash cipher.SHA256
	Data []byte
}

// sent by feed, handled by subscriber
func (dat *Data) HandleOutgoing(ctx node.MsgContext,
	client interface{}) (terminate error) {

	var c *Client = client.(*Client)

	// if we got dat.Hash == cipher.SHA256{} (empty hash),
	// then adding value to database causes panic
	if dat.Hash == (cipher.SHA256{}) {
		logger.Errorf("got data with empty hash from %s %s",
			ctx.Remote().Address(),
			ctx.Remote().PubKey().Hex())
		terminate = node.ErrMalformedMessage
		return // terminate connection
	}

	// Do we really need it?
	//
	// hashDoneLocally := cipher.SumSHA256(msg.Data)
	// // that the hashes match
	// if msg.Hash != hashDoneLocally {
	// 	terminate = fmt.Errorf(
	// 		"received data and it's hash are different: want %v, got %v",
	// 		msg.Hash,
	// 		hashDoneLocally)
	// 	return // terminate connection
	// }

	// store the new data in the DB
	if err := c.db.Add(dat.Hash, dat.Data); err != nil {
		// error can be "already exists" only, ignore it
		logger.Error("error adding value to DB: %v, %s", err, dat.Hash.Hex())
		return // keep connection
	}
	c.gotNewData(ctx, dat.Hash)
	return
}
