package bbs

import "github.com/skycoin/cxo/encoder"

type BoardContainer struct {
	Boards encoder.HArray `type:"Board"`
}

type Board struct {
	Name    string
	Threads encoder.HArray `type:"Thread"`
}

type Thread struct {
	Name  string
	Posts encoder.HArray `type:"Post"`
}

type Post struct {
	Poster []byte
	Text   string
}
