package main

import (
	"fmt"

	"github.com/skycoin/cxo/bbs"
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobjects"
	"github.com/skycoin/skycoin/src/cipher"
)

// Indexer represents a skyobjects indexer.
type Indexer struct {
	c           *skyobjects.Container
	boards      map[string]cipher.SHA256
	threads     map[string]cipher.SHA256
	posts       map[string]cipher.SHA256
	_boardType  cipher.SHA256
	_threadType cipher.SHA256
	_postType   cipher.SHA256
}

// NewIndexer creates a new Indexer.
func NewIndexer() *Indexer {
	container := skyobjects.NewContainer(data.NewDB())
	return &Indexer{
		c:           container,
		boards:      make(map[string]cipher.SHA256),
		threads:     make(map[string]cipher.SHA256),
		posts:       make(map[string]cipher.SHA256),
		_boardType:  container.SaveSchema(bbs.Board{}),
		_threadType: container.SaveSchema(bbs.Thread{}),
		_postType:   container.SaveSchema(bbs.Post{}),
	}
}

// CreateRoot creates a new root.
func (ix *Indexer) CreateRoot(boardKeys ...cipher.SHA256) (key cipher.SHA256) {
	root := skyobjects.NewRoot(0)
	root.AddChildren(boardKeys...)
	return ix.c.SaveRoot(root)
}

// CreateBoard creates a new board.
func (ix *Indexer) CreateBoard(name, desc string, threadKeys ...cipher.SHA256) (key cipher.SHA256) {
	board := bbs.Board{Name: name, Desc: desc, Threads: threadKeys}
	key = ix.c.SaveObject(ix._boardType, board)
	ix.boards[name] = key
	return
}

// CreateThread creates a new thread.
func (ix *Indexer) CreateThread(name string) (key cipher.SHA256) {
	thread := bbs.Thread{Name: name}
	key = ix.c.SaveObject(ix._threadType, thread)
	ix.threads[name] = key
	return
}

// CreatePost creates a new post.
func (ix *Indexer) CreatePost(header, body string, threadKey cipher.SHA256) (key cipher.SHA256) {
	post := bbs.Post{Header: header, Body: body, Thread: threadKey}
	key = ix.c.SaveObject(ix._postType, post)
	ix.posts[header] = key
	return
}

// GetBoards gets all the boards connected to the most recent root.
func (ix *Indexer) GetBoards() (boards []bbs.Board, err error) {
	for _, k := range ix.c.GetCurrentRootChildren() {
		var b bbs.Board
		ref, e := ix.c.Get(k)
		if e != nil {
			switch e.(type) {
			case *skyobjects.ErrorSkyObjectNotFound:
				fmt.Println("Skyobject not found locally. Feature to request it is not implemented.")
				continue
			default:
				err = e
				return
			}
		}
		ref.Deserialize(&b)
		boards = append(boards, b)
	}
	return
}

// GetThreadsOfBoard gets all the threads connected to the specified board.
func (ix *Indexer) GetThreadsOfBoard(bName string) (threads []bbs.Thread, err error) {
	bRef, e := ix.c.Get(ix.threads[bName])
	if e != nil {
		switch e.(type) {
		case *skyobjects.ErrorSkyObjectNotFound:
			fmt.Println("Skyobject not found locally. Feature to request it is not implemented.")
			err = e
			return
		default:
			err = e
			return
		}
	}
	for k, has := range ix.c.GetChildren(bRef) {
		if !has {
			fmt.Println("Skyobject not found locally. Feature to request it is not implemented.")
			continue
		}
		ref, e := ix.c.Get(k)
		if e != nil {
			err = e
			return
		}
		var t bbs.Thread
		ref.Deserialize(&t)
		threads = append(threads, t)
	}
	return
}

// GetPostsOfThread gets all the posts that reference the specified thread.
func (ix *Indexer) GetPostsOfThread(tName string) (posts []bbs.Post, err error) {
	tKey, has := ix.threads[tName]
	if !has {
		fmt.Println("Skyobject not found locally. Feature to request it is not implemented.")
		err = skyobjects.ErrorSkyObjectNotFound{}
		return
	}
	for _, k := range ix.c.GetAllOfSchema(ix._postType) {
		var p bbs.Post
		ref, e := ix.c.Get(k)
		if e != nil {
			err = e
			return
		}
		ref.Deserialize(&p)
		if p.Thread == tKey {
			posts = append(posts, p)
		}
	}
	return
}

func (ix *Indexer) addBoardOfKeyToRoot(bKey cipher.SHA256) error {
	bKeys := ix.c.GetCurrentRootChildren()
	if SHA256SliceHas(bKeys, bKey) {
		return fmt.Errorf("board is already a child of root")
	}
	bKeys = append(bKeys, bKey)
	ix.CreateRoot(bKeys...)
	return nil
}

func (ix *Indexer) removeBoardOfKeyFromRoot(bKey cipher.SHA256) error {
	bKeys := ix.c.GetCurrentRootChildren()
	for i, bk := range bKeys {
		if bk == bKey {
			bKeys[i], bKeys[0] = bKeys[0], bKeys[i]
			bKeys = bKeys[1:]
			ix.CreateRoot(bKeys...)
			return nil
		}
	}
	return fmt.Errorf("board is not a child of root")
}
