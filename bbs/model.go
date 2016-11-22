package bbs

import (
	"github.com/skycoin/cxo/schema"
)

type Board struct {
	Name    string
	Threads schema.HArray `type:"Thread"`
}

type Thread struct {
	Name  string
	Posts schema.HArray `type:"Post"`
}

type Post struct {
	Poster []byte
	Text   string
}
