package client

import (
	"github.com/skycoin/cxo/nodeManager"
	"fmt"
	"github.com/skycoin/skycoin/src/cipher"
)

// handle an AnnounceMessage from the node I am subscribed to
func (msg *AnnounceMessage) HandleFromUpstream(cc *nodeManager.UpstreamContext) error {
	fmt.Println("AnnounceMessage")

	if DB.Has(msg.Hash) {
		return fmt.Errorf("received an announce for data I already have: %v", msg.Hash)
	}


	fmt.Println("Got hash", msg.Hash)
	fmt.Println("Got hash hex", msg.Hash.Hex())


	//request := RequestMessage{
	//	Hash: msg.Hash,
	//}
	//
	requestFunc := func (hash cipher.SHA256) error{
		return cc.Remote.Send(RequestMessage{Hash:hash})
	}
	Syncronizer.Sync(msg.Hash, requestFunc)
	return nil
}

func (msg *RequestMessage) HandleFromUpstream(cc *nodeManager.UpstreamContext) error {
	fmt.Println("Request Upstream")
	//
	//if DB.Has(msg.Hash) {
	//	return fmt.Errorf("received an announce for data I already have: %v", msg.Hash)
	//}
	//
	//
	//fmt.Println("Got hash", msg.Hash)
	//fmt.Println("Got hash hex", msg.Hash.Hex())
	//
	//request := RequestMessage{
	//	Hash: msg.Hash,
	//}



	return nil
}

// handle a RequestMessage from a node who is subscribed to me
func (msg *RequestMessage) HandleFromDownstream(cc *nodeManager.DownstreamContext) error {
	fmt.Println("RequestMessage")

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
	fmt.Println("DataMessage")

	// TODO: check whether I really requested that data
	// check whether I already have the received data
	ok := DB.Has(msg.Hash)
	if ok {
		return fmt.Errorf("received data I already have: %v", msg.Hash)
	}

	// check HASH of the received data

	// hash the received data
	hashDoneLocally := cipher.SumSHA256(msg.Data)
	// that the hashes match
	if msg.Hash != hashDoneLocally {
		return fmt.Errorf("received data and it's hash are different: want %v, got %v", msg.Hash, hashDoneLocally)
	}

	// print out the received data
	fmt.Println(msg.Data)

	// store the new data in the DB
	err := DB.Add(msg.Hash, msg.Data)
	//Sync.Accept(msg.Hash, msg.Data)

	requestFunc := func (hash cipher.SHA256) error{
		return cc.Remote.Send(RequestMessage{Hash:hash})
	}
	Syncronizer.Sync(msg.Hash, requestFunc)
	return err
}
