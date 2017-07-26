package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
)

// A Root represents root object of a feed
type Root struct {
	Refs []Dynamic // main branches

	Reg RegistryReference // registry
	Pub cipher.PubKey     // feed

	Seq  uint64 // seq number
	Time int64  // timestamp (unix nano)

	Sig cipher.Sig // signature

	Hash cipher.SHA256 // hash
	Prev cipher.SHA256 // hash of previous root
}

// PH retusn string like "[1a2ef33:98a7ec5]"
// ([pub_key:root_hash]). First 7 characters
func (r *Root) PH() string {
	return "[" + r.Pub.Hex()[:7] + ":" + r.Hash.Hex()[:7] + "]"
}
