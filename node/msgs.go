package node

import (
	"github.com/skycoin/skycoin/src/cipher"
)

type Sync struct {
	Feeds []cipher.PubKey // feeds to subscribe to
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
