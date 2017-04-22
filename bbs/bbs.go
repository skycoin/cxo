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
	so    *skyobject.Container
}
