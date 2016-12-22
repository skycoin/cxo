package client

import (
	"github.com/skycoin/cxo/nodeManager"
	"fmt"
	"github.com/skycoin/skycoin/src/cipher"
)

// handle an AnnounceMessage from the node I am subscribed to
func (msg *AnnounceMessage) HandleFromUpstream(cc *nodeManager.UpstreamContext) error {
	//fmt.Println("AnnounceMessage")

	if DB.Has(msg.Hash) {
		return fmt.Errorf("received an announce for data I already have: %v", msg.Hash)
	}

	Sync.OnRequest(cc.Remote, msg.Hash)
	return nil
}

// handle a RequestMessage from a node who is subscribed to me
func (msg *RequestMessage) HandleFromDownstream(cc *nodeManager.DownstreamContext) error {
	// fetch from DB the data by hash
	responseData, ok := DB.Get(msg.Hash)
	if !ok {
		return fmt.Errorf("received request for data I don't have: %v", msg.Hash)
	}

	// fill the response DataMessage
	message := DataMessage{
		Hash: msg.Hash,
		Data:       responseData,
	}

	// send back the response with the requested data
	return cc.Remote.Send(message)
}

// handle a DataMessage from the node I am subscribed to
func (msg *DataMessage) HandleFromUpstream(cc *nodeManager.UpstreamContext) error {
	// TODO: check whether I really requested that data
	// check whether I already have the received data
	//ok := DB.Has(msg.Hash)
	//if ok {
	//	return fmt.Errorf("received data I already have: %v", msg.Hash)
	//}

	hashDoneLocally := cipher.SumSHA256(msg.Data)
	// that the hashes match
	if msg.Hash != hashDoneLocally {
		return fmt.Errorf("received data and it's hash are different: want %v, got %v", msg.Hash, hashDoneLocally)
	}

	// store the new data in the DB
	err := DB.Add(msg.Hash, msg.Data)

	Sync.OnRequest(cc.Remote, msg.Hash)
	return err
}
