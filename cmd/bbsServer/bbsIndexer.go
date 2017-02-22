package main

import (
	"fmt"
	"github.com/skycoin/cxo/bbs"
	"github.com/skycoin/cxo/data"
	// "github.com/skycoin/cxo/encoder"
	"github.com/skycoin/cxo/nodeManager"
	"github.com/skycoin/cxo/skyobject"
	// "github.com/skycoin/skycoin/src/cipher"
)

type objectLink struct {
	ID   string
	Name string
}

type BBSIndexer struct {
	BBS     *bbs.Bbs
	Boards  []objectLink
	Threads []objectLink
	Posts   []objectLink
}

func MakeBBSIndexer(bbsIn *bbs.Bbs) *BBSIndexer {
	return &BBSIndexer{
		BBS: bbsIn,
	}
}

func MakeBBSIndexerSimple() *BBSIndexer {
	db := data.NewDB()

	config := nodeManager.NewManagerConfig()
	manager, e := nodeManager.NewManager(config)
	if e != nil {
		fmt.Println(e)
		return nil
	}
	node := manager.NewNode()

	bbsIn := bbs.CreateBbs(db, node)
	return &BBSIndexer{
		BBS: bbsIn,
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
	bi.Boards = []objectLink{}
	bi.Threads = []objectLink{}
	bi.Posts = []objectLink{}
}

func (bi *BBSIndexer) Load() {
	bi.Clear()
	bi.loadBoards()
	bi.loadThreads()
	bi.loadPosts()
	fmt.Printf("Generated: %d Boards, %d Threads, %d Posts.\n", len(bi.Boards), len(bi.Threads), len(bi.Posts))
}

func (bi *BBSIndexer) loadBoards() {
	c := bi.BBS.Container
	schemaKey, _ := c.GetSchemaKey("Board")
	keys := c.GetAllBySchema(schemaKey)
	for _, k := range keys {
		ref := skyobject.Href{Ref: k}
		bi.Boards = append(bi.Boards, objectLink{ID: k.Hex(), Name: ref.String(c)})
	}
}

func (bi *BBSIndexer) loadThreads() {
	c := bi.BBS.Container
	schemaKey, _ := c.GetSchemaKey("Thread")
	keys := c.GetAllBySchema(schemaKey)
	for _, k := range keys {
		ref := skyobject.Href{Ref: k}
		bi.Threads = append(bi.Threads, objectLink{ID: k.Hex(), Name: ref.String(c)})
	}
}

func (bi *BBSIndexer) loadPosts() {
	c := bi.BBS.Container
	schemaKey, _ := c.GetSchemaKey("Post")
	keys := c.GetAllBySchema(schemaKey)
	for _, k := range keys {
		ref := skyobject.Href{Ref: k}
		bi.Posts = append(bi.Posts, objectLink{ID: k.Hex(), Name: ref.String(c)})
	}
}
