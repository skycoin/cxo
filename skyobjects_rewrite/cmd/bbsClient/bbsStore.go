package main

import (
	"fmt"

	"github.com/skycoin/cxo/bbs"
	"github.com/skycoin/skycoin/src/cipher"
)

var tree = Tree{
	Branches:          make(map[cipher.SHA256]Branch),
	DownloadedBoards:  make(map[cipher.SHA256]bbs.SBoard),
	DownloadedThreads: make(map[cipher.SHA256]bbs.Thread),
}

// Branch is a branch of tree.
type Branch struct {
	Ready  bool
	Leaves []cipher.SHA256
}

func newBranch() Branch {
	return Branch{Ready: false}
}

// Tree is a tree.
type Tree struct {
	Branches          map[cipher.SHA256]Branch
	DownloadedBoards  map[cipher.SHA256]bbs.SBoard
	DownloadedThreads map[cipher.SHA256]bbs.Thread
}

// AddBranch adds an unready branch to tree.
func (t *Tree) AddBranch(key cipher.SHA256) error {
	if _, has := t.Branches[key]; has {
		return fmt.Errorf("branch '%s' exists", key.Hex())
	}
	t.Branches[key] = newBranch()
	return nil
}

// AddBranches adds many branches to tree, removing those unused.
func (t *Tree) AddBranches(keys []cipher.SHA256) {
	for _, key := range keys {
		t.AddBranch(key)
	}
	// Remove unused.
	for oldKey := range t.Branches {
		keep := false
		for _, newKey := range keys {
			if newKey == oldKey {
				keep = true
			}
		}
		if keep == false {
			delete(t.Branches, oldKey)
		}
	}
}

// GetBoard returns the board of specified key.
func (t *Tree) GetBoard(key cipher.SHA256) bbs.SBoard {
	return t.DownloadedBoards[key]
}

// GetUndownloadedBoardKeys gets all the keys of boards that need to be downloaded.
func (t *Tree) GetUndownloadedBoardKeys() (keys []cipher.SHA256) {
	for key := range t.Branches {
		if t.IsDownloadedBoard(key) == false {
			keys = append(keys, key)
		}
	}
	return
}

// IsDownloadedBoard determines if key is of downloaded board.
func (t *Tree) IsDownloadedBoard(boardKey cipher.SHA256) bool {
	for key, board := range t.DownloadedBoards {
		if key == boardKey && board.Name != "" {
			return true
		}
	}
	return false
}

// AddDownloadedBoards adds downloaded boards to tree.
func (t *Tree) AddDownloadedBoards(keys []cipher.SHA256, boards []bbs.SBoard) error {
	if len(keys) != len(boards) {
		return fmt.Errorf("uneven lengths")
	}
	for i, key := range keys {
		t.DownloadedBoards[key] = boards[i]
	}
	return nil
}
