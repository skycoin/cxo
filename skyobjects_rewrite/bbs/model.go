package bbs

import (
	"github.com/skycoin/cxo/skyobjects"
	"github.com/skycoin/skycoin/src/cipher"
)

//type BoardType skyobject.HashLink
//

// Board represents a board which contains threads.
type Board struct {
	Name        string
	Description string
	Threads     skyobjects.ArrayReference
	PublicKey   cipher.PubKey
}

// ToS converts Board to simple
func (b *Board) ToS() SBoard {
	return SBoard{
		Name:        b.Name,
		Description: b.Description,
	}
}

// Thread contains posts.
type Thread struct {
	Name string
}

// Post represents a post.
type Post struct {
	Header string
	Text   string
	Thread skyobjects.ObjectReference
}

// SBoard is simple representation of Board.
type SBoard struct {
	Name        string
	Description string
}
