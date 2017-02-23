package main

import (
	"fmt"
	"github.com/skycoin/cxo/bbs"
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/encoder"
	"github.com/skycoin/cxo/nodeManager"
	// "github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/skycoin/src/cipher"
	// "strings"
)

type BBSIndexer struct {
	BBS *bbs.Bbs

	Boards  map[cipher.SHA256]bbs.Board
	Threads map[cipher.SHA256]bbs.Thread
	Posts   map[cipher.SHA256]bbs.Post
}

func MakeBBSIndexer() *BBSIndexer {
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
		_, data := c.GetObject(k)
		encoder.DeserializeRaw(data, &board)
		bi.Boards[k] = board
	}

	schemaKey, _ = c.GetSchemaKey("Thread")
	keys = c.GetAllBySchema(schemaKey)
	for _, k := range keys {
		var thread bbs.Thread
		_, data := c.GetObject(k)
		encoder.DeserializeRaw(data, &thread)
		bi.Threads[k] = thread
	}

	schemaKey, _ = c.GetSchemaKey("Post")
	keys = c.GetAllBySchema(schemaKey)
	for _, k := range keys {
		var post bbs.Post
		_, data := c.GetObject(k)
		encoder.DeserializeRaw(data, &post)
		bi.Posts[k] = post
	}

	fmt.Printf("Loaded: %d Boards, %d Threads, %d Posts.\n", len(bi.Boards), len(bi.Threads), len(bi.Posts))
}

func (bi *BBSIndexer) GetThreadsFromBoard(boardName string) (threads []bbs.Thread, e error) {
	c := bi.BBS.Container
	// bi.Load()

	// Get Board of name boardName.
	var key cipher.SHA256

	for k, v := range bi.Boards {
		if v.Name == boardName {
			key = k
			break
		}
	}

	// Get Threads from Board.
	typ, data := c.GetObject(key)
	threadArrayKey := c.GetField(typ, data, "Threads")
	threadMap := c.GetMap(threadArrayKey, "Thread")

	for _, threadData := range threadMap {
		var thread bbs.Thread
		encoder.DeserializeRaw(threadData, &thread)
		threads = append(threads, thread)
	}

	return
}

func (bi *BBSIndexer) GetPostsFromThread(threadName string) (posts []bbs.Post, e error) {
	c := bi.BBS.Container
	// bi.Load()

	// Get Thread of name threadName.
	var key cipher.SHA256

	for k, v := range bi.Threads {
		if v.Name == threadName {
			key = k
			break
		}
	}

	// Get Posts from Thread.
	typ, data := c.GetObject(key)
	postArrayKey := c.GetField(typ, data, "Posts")
	postMap := c.GetMap(postArrayKey, "Post")

	for _, postData := range postMap {
		var post bbs.Post
		encoder.DeserializeRaw(postData, &post)
		posts = append(posts, post)
	}

	return
}

// func (bi *BBSIndexer) loadBoards() {
// 	c := bi.BBS.Container
// 	schemaKey, _ := c.GetSchemaKey("Board")
// 	keys := c.GetAllBySchema(schemaKey)
// 	for _, k := range keys {
// 		ref := skyobject.Href{Ref: k}
// 		bi.Boards = append(bi.Boards, objectLink{ID: k.Hex(), Name: ref.String(c)})
// 	}

// 	fmt.Println("\n[BOARDS]\n")
// 	for _, k := range keys {
// 		var boardExtract bbs.Board // <----------------------------------------- (BOARD)
// 		boardType, boardData := c.GetObject(k)
// 		encoder.DeserializeRaw(boardData, &boardExtract)
// 		bi.BoardMap[boardExtract.Name] = k

// 		var threadsExtract []bbs.Thread // <------------------------------------ (CHILDREN THREADS)
// 		threadsKey := c.GetField(boardType, boardData, "Threads")
// 		threadDataArray := c.GetArray(threadsKey, "Thread")

// 		for _, threadData := range threadDataArray {
// 			var threadExtract bbs.Thread // <----------------------------------- (CHILD THREAD)
// 			encoder.DeserializeRaw(threadData, &threadExtract)
// 			threadsExtract = append(threadsExtract, threadExtract)
// 		}

// 		fmt.Println(boardExtract.Name)
// 		for _, v := range threadsExtract {
// 			fmt.Println("", "-", v.Name)
// 		}
// 	}
// }

// func (bi *BBSIndexer) loadThreads() {
// 	c := bi.BBS.Container
// 	schemaKey, _ := c.GetSchemaKey("Thread")
// 	keys := c.GetAllBySchema(schemaKey)
// 	for _, k := range keys {
// 		ref := skyobject.Href{Ref: k}
// 		bi.Threads = append(bi.Threads, objectLink{ID: k.Hex(), Name: ref.String(c)})
// 	}

// 	fmt.Println("\n[THREADS]\n")
// 	for _, k := range keys {
// 		var threadExtract bbs.Thread // <--------------------------------------- (THREAD)
// 		threadType, threadData := c.GetObject(k)
// 		encoder.DeserializeRaw(threadData, &threadExtract)
// 		bi.ThreadMap[threadExtract.Name] = k

// 		var postsExtract []bbs.Post // <---------------------------------------- (CHILDREN POSTS)
// 		postsKey := c.GetField(threadType, threadData, "Posts")
// 		postDataArray := c.GetArray(postsKey, "Post")

// 		for _, postData := range postDataArray {
// 			var postExtract bbs.Post // <--------------------------------------- (CHILD POST)
// 			encoder.DeserializeRaw(postData, &postExtract)
// 			postsExtract = append(postsExtract, postExtract)
// 		}

// 		fmt.Println(threadExtract.Name)
// 		for _, v := range postsExtract {
// 			fmt.Println("", "-", v.Header)
// 		}
// 	}
// }

// func (bi *BBSIndexer) loadPosts() {
// 	c := bi.BBS.Container
// 	schemaKey, _ := c.GetSchemaKey("Post")
// 	keys := c.GetAllBySchema(schemaKey)
// 	for _, k := range keys {
// 		ref := skyobject.Href{Ref: k}
// 		bi.Posts = append(bi.Posts, objectLink{ID: k.Hex(), Name: ref.String(c)})
// 	}

// 	fmt.Println("\n[POSTS]\n")
// 	for _, k := range keys {
// 		var postExtract bbs.Post
// 		_, postData := c.GetObject(k)
// 		encoder.DeserializeRaw(postData, &postExtract)
// 		bi.PostMap[postExtract.Header] = postExtract.Text

// 		fmt.Println(postExtract.Header, "-", postExtract.Text)
// 	}
// 	fmt.Print('\n')
// }
