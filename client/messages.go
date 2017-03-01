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

	logger.Debugf("got announce %s", ann.Hash.Hex())

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

	logger.Debugf("got request %s", req.Hash.Hex())

	var (
		c *Client = client.(*Client)

		data []byte
		ok   bool

		err error
	)

	// if request is empty then we've got request for root object
	if req.Hash == (cipher.SHA256{}) {
		req.Hash = c.root
	}

	if data, ok = c.db.Get(req.Hash); !ok {
		logger.Error("got request for data I don't have: %v",
			req.Hash.Hex())
		return // keep connection alive
	}
	// send back the response with the requested data...
	// ...or may be it's root?
	if req.Hash == c.root {
		logger.Debugf("send root %s", req.Hash.Hex())
		err = ctx.Reply(Root{
			Hash: req.Hash,
			Data: data,
		})
	} else {
		logger.Debugf("send data %s", req.Hash.Hex())
		err = ctx.Reply(Data{
			Hash: req.Hash,
			Data: data,
		})
	}
	if err != nil {
		logger.Error("error sending reply: %v", err)
	}
	return // keep alive
}

// Data is response for request. It sent by feed to subscriber
type Data struct {
	Hash cipher.SHA256
	Data []byte
}

// sent by feed, handled by subscriber
func (dat *Data) HandleOutgoing(ctx node.MsgContext,
	client interface{}) (terminate error) {

	logger.Debugf("got data %s", dat.Hash.Hex())

	var c *Client = client.(*Client)

	if terminate = isHashEmpty(ctx, dat.Hash); terminate != nil {
		return
	}

	if terminate = isHashValid(ctx, dat.Hash, dat.Data); terminate != nil {
		return
	}

	// store the new data in the DB
	if err := c.db.Add(dat.Hash, dat.Data); err != nil {
		// error can be "already exists" only, ignore it
		logger.Errorf("error adding value to DB: %v, %s", err, dat.Hash.Hex())
		return // keep connection
	}
	c.gotNewData(ctx, dat.Hash)
	return
}

// A Root is the same as Data but its content is a root object
type Root struct {
	Hash cipher.SHA256
	Data []byte
}

// sent by feed, handled by subscriber
func (rt *Root) HandleOutgoing(ctx node.MsgContext,
	client interface{}) (terminate error) {

	logger.Debugf("got root %s", rt.Hash.Hex())

	var c *Client = client.(*Client)

	if terminate = isHashEmpty(ctx, rt.Hash); terminate != nil {
		return
	}

	if terminate = isHashValid(ctx, rt.Hash, rt.Data); terminate != nil {
		return
	}

	// store the new data in the DB
	if err := c.db.Add(rt.Hash, rt.Data); err != nil {
		// error can be "already exists" only, ignore it
		logger.Errorf("error adding value to DB: %v, %s", err, rt.Hash.Hex())
		return // keep connection
	}
	// it is root
	c.setRoot(rt.Hash)
	c.gotNewData(ctx, rt.Hash)
	return
}

// if we got dat.Hash == cipher.SHA256{} (empty hash),
// then adding value to database causes panic
func isHashEmpty(ctx node.MsgContext, hash cipher.SHA256) (err error) {
	if hash == (cipher.SHA256{}) {
		logger.Errorf("got data with empty hash from %s %s",
			ctx.Remote().Address(),
			ctx.Remote().PubKey().Hex())
		err = node.ErrMalformedMessage
	}
	return
}

func isHashValid(ctx node.MsgContext, hash cipher.SHA256,
	data []byte) (err error) {

	if local := cipher.SumSHA256(data); local != hash {
		logger.Errorf(
			"missmatch received hash and hash of received data from %s %s",
			ctx.Remote().Address(),
			ctx.Remote().PubKey().Hex())
		err = node.ErrMalformedMessage
	}
	return
}
