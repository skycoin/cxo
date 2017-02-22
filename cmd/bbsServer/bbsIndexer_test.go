package main

import (
	// "fmt"
	// "github.com/skycoin/cxo/bbs"
	"testing"
)

func TestBBS(t *testing.T) {
	indexer := MakeBBSIndexerSimple()
	indexer.AddBoard("Test1")
	indexer.AddBoard("Test2")
	indexer.AddBoard("Test3")
	indexer.AddBoard("Test4")
}
