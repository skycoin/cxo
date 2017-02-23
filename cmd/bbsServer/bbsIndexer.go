package main

import (
	"fmt"
	"github.com/skycoin/cxo/bbs"
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/encoder"
	"github.com/skycoin/cxo/nodeManager"
	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/skycoin/src/cipher"
	"strings"
)

type objectLink struct {
	ID   string
	Name string
}

type href struct {
	Type cipher.SHA256
	Data []byte
}

type BBSIndexer struct {
	BBS *bbs.Bbs

	Boards  []objectLink
	Threads []objectLink
	Posts   []objectLink

	// [Key] : Item Name, [Data] : Object Key
	BoardMap  map[string]cipher.SHA256
	ThreadMap map[string]cipher.SHA256
	PostMap   map[string]cipher.SHA256
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
	fmt.Printf("Loaded: %d Boards, %d Threads, %d Posts.\n", len(bi.Boards), len(bi.Threads), len(bi.Posts))
}

func (bi *BBSIndexer) GetThreadsFromBoard(key string) (threads []objectLink, e error) {
	c := bi.BBS.Container
	bi.Load()

	boardKey, e := cipher.SHA256FromHex(key)
	if e != nil {
		return
	}

	ref := skyobject.Href{Ref: boardKey}
	threadKeyMap := ref.References(c)

	for k, _ := range threadKeyMap {
		threadRef := skyobject.Href{Ref: k}
		if key = k.Hex(); bi.isThreadKey(key) {
			threads = append(threads, objectLink{ID: key, Name: threadRef.String(c)})
		}
	}

	return
}

func (bi *BBSIndexer) GetPostsFromThread(key string) (posts []objectLink, e error) {
	c := bi.BBS.Container
	bi.Load()

	threadKey, e := cipher.SHA256FromHex(key)
	if e != nil {
		return
	}

	ref := skyobject.Href{Ref: threadKey}
	postKeyMap := ref.References(c)

	for k, _ := range postKeyMap {
		postRef := skyobject.Href{Ref: k}
		if key = k.Hex(); bi.isPostKey(key) {
			posts = append(posts, objectLink{ID: key, Name: postRef.String(c)})
		}
	}

	return
}

func (bi *BBSIndexer) isBoardKey(key string) bool {
	key = strings.TrimSpace(key)
	for _, v := range bi.Boards {
		if key == v.ID {
			return true
		}
	}
	return false
}

func (bi *BBSIndexer) isThreadKey(key string) bool {
	key = strings.TrimSpace(key)
	for _, v := range bi.Threads {
		if key == v.ID {
			return true
		}
	}
	return false
}

func (bi *BBSIndexer) isPostKey(key string) bool {
	key = strings.TrimSpace(key)
	for _, v := range bi.Posts {
		if key == v.ID {
			return true
		}
	}
	return false
}

type testHref struct {
	Type cipher.SHA256
	Data []byte
}

func (bi *BBSIndexer) loadBoards() {
	c := bi.BBS.Container
	schemaKey, _ := c.GetSchemaKey("Board")
	keys := c.GetAllBySchema(schemaKey)
	for _, k := range keys {
		ref := skyobject.Href{Ref: k}
		bi.Boards = append(bi.Boards, objectLink{ID: k.Hex(), Name: ref.String(c)})
	}

	fmt.Println("\n[BOARDS]\n")
	for _, k := range keys {
		var boardExtract bbs.Board // <----------------------------------------- (BOARD)
		boardType, boardData := c.GetObject(k)
		encoder.DeserializeRaw(boardData, &boardExtract)

		var threadsExtract []bbs.Thread // <------------------------------------ (CHILDREN THREADS)
		threadsKey := c.GetField(boardType, boardData, "Threads")
		threadDataArray := c.GetArray(threadsKey, "Thread")

		for _, threadData := range threadDataArray {
			var threadExtract bbs.Thread // <----------------------------------- (CHILD THREAD)
			encoder.DeserializeRaw(threadData, &threadExtract)
			threadsExtract = append(threadsExtract, threadExtract)
		}

		fmt.Println(boardExtract.Name)
		for _, v := range threadsExtract {
			fmt.Println("", "-", v.Name)
		}
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

	fmt.Println("\n[THREADS]\n")
	for _, k := range keys {
		var threadExtract bbs.Thread // <--------------------------------------- (THREAD)
		threadType, threadData := c.GetObject(k)
		encoder.DeserializeRaw(threadData, &threadExtract)

		var postsExtract []bbs.Post // <---------------------------------------- (CHILDREN POSTS)
		postsKey := c.GetField(threadType, threadData, "Posts")
		postDataArray := c.GetArray(postsKey, "Post")

		for _, postData := range postDataArray {
			var postExtract bbs.Post // <--------------------------------------- (CHILD POST)
			encoder.DeserializeRaw(postData, &postExtract)
			postsExtract = append(postsExtract, postExtract)
		}

		fmt.Println(threadExtract.Name)
		for _, v := range postsExtract {
			fmt.Println("", "-", v.Header)
		}
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

	fmt.Println("\n[POSTS]\n")
	for _, k := range keys {
		var postExtract bbs.Post
		_, postData := c.GetObject(k)
		encoder.DeserializeRaw(postData, &postExtract)

		fmt.Println(postExtract.Header, "-", postExtract.Text)
	}
	fmt.Print('\n')
}
