package bbs

import "github.com/skycoin/skycoin/src/cipher"

// Board type represents a board.
type Board struct {
	Name    string
	Desc    string
	Threads []cipher.SHA256 `skyobjects:"href"`
}

// Thread type represents a thread.
type Thread struct {
	Name string
}

// Post type represents a post.
type Post struct {
	Header string
	Body   string
	Thread cipher.SHA256 `skyobjects:"href"`
}
