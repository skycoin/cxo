package skyobject

import (
	"github.com/skucoin/skycoin/src/cipher/encoder"
	"github.com/skycoin/skycoin/src/cipher"
)

type Head struct {
	Refs interface{} // Refs

	Pub cipher.PubKey // feed
	Sec cipher.SecKey // sign

	Reg *Registry // regsitry of the Head

	pu    PackUnpacker             // pack/unpack
	cache map[cipher.SHA256][]byte // cache objects
}

func NewHead(pu PackUnpacker) *Head {
	if pu == nil {
		panic("missing PackUnpacker")
	}

	h = new(Head)

	return
}

// Publish changes. It returns nil if
// nothing has been changed
func (h *Head) Publish() (commit *Root) {
	//
	return
}

// Root returns Root even if nothing
// has been changed
func (h *Head) Root() (r *Root) {
	//
	return
}

// Add a value to Root references (to
// list of references cosest to root)
func (h *Head) Add(val interface{}) (err error) {
	//
	return
}
