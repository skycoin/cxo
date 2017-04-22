package bbs

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

// A Board
type Board struct {
	Header  string
	Threads skyobject.References
}

// A Thread
type Thread struct {
	Header string
	Posts  skyobject.References
}

// A Post
type Post struct {
	Header string
	Body   string
}

// A BBS instance that used by an owner of a feed
type BBS struct {
	owner cipher.SecKey
	feed  cipher.PubKey
	so    *skyobject.Container
}

// NewBBS creates BBS of given owner to wrok with given feed
func NewBBS(owner cipher.SecKey, feed cipher.PubKey,
	so *skyobject.Container) (b *BBS) {

	b = new(BBS)

	if so == nil {
		panic("nil container")
	}
	if owner == (cipher.SecKey{}) {
		panic("empty owner")
	}
	if feed == (cipher.PubKey{}) {
		panic("empty feed")
	}

	b.owner = owner
	b.feed = feed
	b.so = so

	return
}

func (b *BBS) name() {

}
