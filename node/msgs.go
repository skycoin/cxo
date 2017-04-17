package node

import (
	"github.com/skycoin/skycoin/src/cipher"
)

//
// server -> server messages
//

// server sends the message after
// connecting to another server to sync
type ServerConnect struct {
	Feeds []cipher.PubKey
}

// sync feeds list with another server
// (reply for ServerMsg and update the
// list if need any time)
type ServerSync struct {
	Feeds []cipher.PubKey
}

// say to another servers about quiting
type ServerQuit struct{}

// client -> server messages

type ClientSync struct {
	Feeds []cipher.PubKey
}

//
// data transport
//

// update root object
type Root struct {
	Feed cipher.PubKey // feed
	Sig  cipher.SecKey // signature
	Root []byte        // encoded root object
}

// request missing data
type Request struct {
	Hash cipher.SHA256 // checksum of the data
}

// send requested data
type Data struct {
	Data []byte // encoded data
}
