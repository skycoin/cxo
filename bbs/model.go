package bbs

import (
	"github.com/skycoin/cxo/schema"
)

type Board struct {
	Name    string
	Threads schema.HArray
}

type Thread struct {
	Name  string
	Posts schema.HArray
}

type Post struct {
	Poster schema.Href
	Text   string
}
