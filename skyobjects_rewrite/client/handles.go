package client

import (
	"fmt"

	"github.com/skycoin/cxo/node"
	"github.com/skycoin/skycoin/src/cipher"
)

// introductinc Replier interface for Sync.OnRequest
type Replier interface {
	Reply(msg interface{}) (err error) // reply for message
}

// nodeManager -> node refactoring
//
// HandleFromUpstream   -> HandleIncoming(IncomingContext) (terminate error)
// HandleFromDownstream -> HandleOutgoing(OutgoingContext) (terminate error)

// handle an AnnounceMessage from the node I am subscribed to
func (msg *AnnounceMessage) HandleIncoming(
	ic node.IncomingContext) (terminate error) {

	if DB.Has(msg.Hash) {
		logger.Info("received an announce for data I already have: %v",
			msg.Hash.Hex())
		return // nil (keep connection alive)
	}
	Sync.OnRequest(ic, msg.Hash)
	return nil
}

// handle a RequestMessage from a node who is subscribed to me
func (msg *RequestMessage) HandleOutgoing(
	oc node.OutgoingContext) (terminate error) {

	// fetch from DB the data by hash
	responseData, ok := DB.Get(msg.Hash)
	if !ok {
		logger.Error("received request for data I don't have: %v",
			msg.Hash.Hex())
		return nil // keep connection alive
	}
	// send back the response with the requested data
	err := oc.Reply(DataMessage{
		Hash: msg.Hash,
		Data: responseData,
	})
	if err != nil {
		logger.Error("error sending reply: %v", err)
	}
	return // keep alive
}

// handle a DataMessage from the node I am subscribed to
func (msg *DataMessage) HandleIncoming(
	ic node.IncomingContext) (terminate error) {

	// TODO: check whether I really requested that data
	// check whether I already have the received data
	//ok := DB.Has(msg.Hash)
	//if ok {
	//	return fmt.Errorf("received data I already have: %v", msg.Hash)
	//}

	hashDoneLocally := cipher.SumSHA256(msg.Data)
	// that the hashes match
	if msg.Hash != hashDoneLocally {
		terminate = fmt.Errorf(
			"received data and it's hash are different: want %v, got %v",
			msg.Hash,
			hashDoneLocally)
		return // terminate connection
	}

	// store the new data in the DB
	if err := DB.Add(msg.Hash, msg.Data); err != nil {
		logger.Error("error adding value to DB: %v, %s", err, msg.Hash.Hex())
	}
	Sync.OnRequest(ic, msg.Hash)
	return
}
