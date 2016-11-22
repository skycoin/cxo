package bbs

import (
	"github.com/skycoin/cxo/schema"
)

type Board struct {
	Name    string
	Threads schema.HrefStatic `href:"[]Thread"`
}

type Thread struct {
	Name  string
	Board schema.HrefStatic `href:"[]Post"`
}

type Post struct {
	Poster []byte
	Text   string
}



