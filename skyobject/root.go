package skyobject

import (
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// A Root represents root object of a feed
type Root struct {
	Refs []Dynamic // main branches

	Reg RegistryRef   // registry
	Pub cipher.PubKey // feed

	Seq  uint64 // seq number
	Time int64  // timestamp (unix nano)

	// sig and hash are anot parts of the Root

	Sig cipher.Sig `enc:"-"` // signature (not part of the Root)

	Hash cipher.SHA256 `enc:"-"` // hash (not part of the Root)
	Prev cipher.SHA256 // hash of previous root

	// machine local fields, not parts of the Root

	// IsFull set to true if DB contains all objects
	// required by this Root
	IsFull bool `enc:"-"`
}

func (r *Root) Encode() []byte {
	return encoder.Serialize(r)
}

// Short retusn string like "[1a2ef33:2]" ({pub_key:seq})
func (r *Root) Short() string {
	return fmt.Sprintf("{%s:%d}",
		r.Pub.Hex()[:7],
		r.Seq)
}

// String implements fmt.Stringer interface
func (r *Root) String() string {
	return fmt.Sprintf("Root{%s:%d:%s}",
		r.Pub.Hex()[:7],
		r.Seq,
		r.Hash.Hex()[:7])
}
