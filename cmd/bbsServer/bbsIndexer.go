package main

import (
	"fmt"

	"github.com/skycoin/cxo/bbs"
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/skycoin/src/cipher"
	// "github.com/skycoin/skycoin/src/mesh/nodeManager"
)

type BBSIndexer struct {
	BBS *bbs.Bbs

	Boards  map[cipher.SHA256]bbs.Board
	Threads map[cipher.SHA256]bbs.Thread
	Posts   map[cipher.SHA256]bbs.Post
}

func MakeBBSIndexer() *BBSIndexer {
	db := data.NewDB()
	security := Signer{}

	bbsIn := bbs.CreateBbs(db, security)
	return &BBSIndexer{
		BBS:     bbsIn,
		Boards:  make(map[cipher.SHA256]bbs.Board),
		Threads: make(map[cipher.SHA256]bbs.Thread),
		Posts:   make(map[cipher.SHA256]bbs.Post),
	}
}

func (bi *BBSIndexer) AddBoard(name string, threads ...bbs.Thread) bbs.Board {
	defer bi.Load()
	return bi.BBS.AddBoard(name, threads...)
}

func (bi *BBSIndexer) AddThread(name string, posts ...bbs.Post) bbs.Thread {
	defer bi.Load()
	return bi.BBS.CreateThread(name, posts...)
}

func (bi *BBSIndexer) CreatePost(header, text string) bbs.Post {
	return bi.BBS.CreatePost(header, text)
}

func (bi *BBSIndexer) Clear() {
	bi.Boards = make(map[cipher.SHA256]bbs.Board)
	bi.Threads = make(map[cipher.SHA256]bbs.Thread)
	bi.Posts = make(map[cipher.SHA256]bbs.Post)
}

func (bi *BBSIndexer) Load() {
	c := bi.BBS.Container

	schemaKey, _ := c.GetSchemaKey("Board")
	keys := c.GetAllBySchema(schemaKey)
	for _, k := range keys {
		var board bbs.Board
		objref := c.GetObjRef(k)
		objref.Deserialize(&board)
		bi.Boards[k] = board
	}

	schemaKey, _ = c.GetSchemaKey("Thread")
	keys = c.GetAllBySchema(schemaKey)
	for _, k := range keys {
		var thread bbs.Thread
		objref := c.GetObjRef(k)
		objref.Deserialize(&thread)
		bi.Threads[k] = thread
	}

	schemaKey, _ = c.GetSchemaKey("Post")
	keys = c.GetAllBySchema(schemaKey)
	for _, k := range keys {
		var post bbs.Post
		objref := c.GetObjRef(k)
		objref.Deserialize(&post)
		bi.Posts[k] = post
	}

	fmt.Printf("Loaded: %d Boards, %d Threads, %d Posts.\n", len(bi.Boards), len(bi.Threads), len(bi.Posts))
}

func (bi *BBSIndexer) GetThreadsFromBoard(boardName string) (threads []bbs.Thread, e error) {
	c := bi.BBS.Container
	// bi.Load()

	// Get Board of name boardName.
	found, key := false, cipher.SHA256{}
	for k, v := range bi.Boards {
		if v.Name == boardName {
			found, key = true, k
			break
		}
	}
	if found == false {
		e = fmt.Errorf("board '%s' not found", boardName)
		return
	}

	// Get Threads from Board.
	boardRef := c.GetObjRef(key)
	threadArrayRef, _ := boardRef.GetFieldAsObj("Threads")
	threadObjArray, _ := threadArrayRef.GetValuesAsObjArray()

	for _, threadObj := range threadObjArray {
		var thread bbs.Thread
		threadObj.Deserialize(&thread)
		threads = append(threads, thread)
	}
	return
}

func (bi *BBSIndexer) GetPostsFromThread(threadName string) (posts []bbs.Post, e error) {
	c := bi.BBS.Container
	// bi.Load()

	// Get Thread of name threadName.
	found, key := false, cipher.SHA256{}
	for k, v := range bi.Threads {
		if v.Name == threadName {
			found, key = true, k
			break
		}
	}
	if found == false {
		e = fmt.Errorf("thread %s not found", threadName)
		return
	}

	// Get Posts from Thread.
	threadRef := c.GetObjRef(key)
	postArrayRef, _ := threadRef.GetFieldAsObj("Posts")
	postObjArray, _ := postArrayRef.GetValuesAsObjArray()

	for _, postObj := range postObjArray {
		var post bbs.Post
		postObj.Deserialize(&post)
		posts = append(posts, post)
	}
	return
}
