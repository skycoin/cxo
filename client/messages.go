package client

import "github.com/skycoin/skycoin/src/cipher"

// upstream -> downstream (HandleFromUpstream)
// AnnounceMessage is a message that announces to subscribers
// that it has new data
type AnnounceMessage struct {
	HashOfNewAvailableData cipher.SHA256
}

// downstream -> upstream (HandleFromDownstream)
// RequestMessage is a request from a subscriber for data
type RequestMessage struct {
	HashOfDataRequested cipher.SHA256
}

// upstream -> downstream (HandleFromUpstream)
// DataMessage is the response to a RequestMessage, with the data requested
type DataMessage struct {
	HashOfData cipher.SHA256
	Data       []byte
}
