package main

import (
	"fmt"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/cxo/nodeManager"
)

// handle an AnnounceMessage from the node I am subscribed to
func (msg *AnnounceMessage) HandleFromUpstream(cc *nodeManager.UpstreamContext) error {
	fmt.Println("AnnounceMessage")

	if DB.Has(msg.HashOfNewAvailableData) {
		return fmt.Errorf("received an announce for data I already have: %v", msg.HashOfNewAvailableData)
	}

	request := RequestMessage{
		HashOfDataRequested: msg.HashOfNewAvailableData,
	}

	return cc.Remote.Send(request)
}

// handle a RequestMessage from a node who is subscribed to me
func (msg *RequestMessage) HandleFromDownstream(cc *nodeManager.DownstreamContext) error {
	fmt.Println("RequestMessage")

	// fetch from DB the data by hash
	responseData, ok := DB.Get(msg.HashOfDataRequested)
	if !ok {
		return fmt.Errorf("received request for data I don't have: %v", msg.HashOfDataRequested)
	}

	// fill the response DataMessage
	message := DataMessage{
		HashOfData: msg.HashOfDataRequested,
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
	ok := DB.Has(msg.HashOfData)
	if ok {
		return fmt.Errorf("received data I already have: %v", msg.HashOfData)
	}

	// check HASH of the received data

	// hash the received data
	hashDoneLocally := cipher.SumSHA256(msg.Data)
	// that the hashes match
	if msg.HashOfData != hashDoneLocally {
		return fmt.Errorf("received data and it's hash are different: want %v, got %v", msg.HashOfData, hashDoneLocally)
	}

	// print out the received data
	fmt.Println(string(msg.Data))
	time.Sleep(time.Second)

	// store the new data in the DB
	err := DB.Add(msg.HashOfData, msg.Data)

	return err
}
