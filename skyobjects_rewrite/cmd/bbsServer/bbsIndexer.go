package main

import (
	"fmt"

	"github.com/skycoin/cxo/bbs"
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/skycoin/src/cipher"
)

// BBSIndexer indexes the Bbs.
type BBSIndexer struct {
	*bbs.Bbs
}

// MakeBBSIndexer makes a BBSIndexer.
func MakeBBSIndexer() *BBSIndexer {
	return &BBSIndexer{
		bbs.CreateBbs(
			data.NewDB(),
			Signer{},
		),
	}
}

// Update updates the root and returns count and timestamp of indexed items.
func (bi *BBSIndexer) Update() (count, timestamp int64) {
	timestamp = bi.Container.GetRootTimestamp()
	fmt.Printf("Update: Root at %d.\n", timestamp)
	return
}

// GetBoardIndexes retrieves all board indexes.
func (bi *BBSIndexer) GetBoardIndexes() (keys []cipher.SHA256) {
	for _, ref := range bi.Container.GetRootValues() {
		keys = append(keys, ref.GetKey())
	}
	return
}

// GetThreadIndexesFromBoard retrieves all thread indexes from specified board.
func (bi *BBSIndexer) GetThreadIndexesFromBoard(boardKey cipher.SHA256) (threadKeys []cipher.SHA256) {
	boardRef := bi.Container.Get(boardKey)
	if boardRef == nil || boardRef.GetSchema(bi.Container).Name != "Board" {
		return
	}
	for _, threadRef := range boardRef.GetChildren(bi.Container) {
		threadKeys = append(threadKeys, threadRef.GetKey())
	}
	return
}

// GetBoards gets boards from list of keys.
func (bi *BBSIndexer) GetBoards(keys []cipher.SHA256) (boards []bbs.SBoard) {
	for _, key := range keys {
		var board bbs.Board
		if ref := bi.Container.Get(key); ref != nil {
			ref.Deserialize(&board)
		}
		boards = append(boards, board.ToS())
	}
	return
}

// GetThreads gets threads from a list of keys.
func (bi *BBSIndexer) GetThreads(keys []cipher.SHA256) (threads []bbs.Thread) {
	for _, key := range keys {
		var thread bbs.Thread
		if ref := bi.Container.Get(key); ref != nil {
			ref.Deserialize(&thread)
		}
		threads = append(threads, thread)
	}
	return
}
