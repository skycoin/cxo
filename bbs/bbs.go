package bbs

import (
	"github.com/skycoin/cxo/schema"
)

type Bbs struct {
	//TODO implement thread lock for Content
	Content schema.Href
	Store   *schema.Store
}

func CreateBbs() *Bbs {
	store := schema.NewStore()
	container := BoardContainer{Boards:store.CreateArray(Board{})}
	content, _ := store.Save(container)
	return &Bbs{Store:store, Content:content}
}

func (bbs *Bbs) AddBoard(name string) (schema.HKey, error) {

	b := Board{Name:name, Threads:bbs.Store.CreateArray(Thread{})}
	href, _ := bbs.Store.Save(b)

	var container BoardContainer
	bbs.Content.ToObject(bbs.Store, &container)

	container.Boards = container.Boards.Append(href.Hash)
	bbs.Content, _ = bbs.Store.Save(container)
	return href.Hash, nil
}

func (bbs *Bbs) CreateBoard(name string, threads ...Thread) Board {
	result := Board{Name:name, Threads:bbs.Store.CreateArray(Thread{})}
	for i := 0; i < len(threads); i++ {
		h, _ := bbs.Store.Save(threads[i])
		result.Threads = result.Threads.Append(h.Hash)
	}
	return result
}

func (bbs *Bbs) AddBoards(boards []Board)  {
	var container BoardContainer
	bbs.Content.ToObject(bbs.Store, &container)
	for i := 0; i < len(boards); i++ {
		h, _ := bbs.Store.Save(boards[i])
		container.Boards = container.Boards.Append(h.Hash)
	}
	bbs.Content, _ = bbs.Store.Save(container)
}

//TODO: Just for test - need refactoring
func (bbs *Bbs) GetBoard(name string) schema.HKey {
	allObjects := bbs.AllBoars()
	container := bbs.container()
	bbs.Content.ToObject(bbs.Store, &container)
	for i := 0; i < len(allObjects); i++ {
		if (allObjects[i].Name == name) {
			return container.Boards.Items[i]
		}
	}
	return schema.HKey{}
}

func (bbs *Bbs) AllBoars() []Board {
	var boards []Board
	boards = bbs.container().Boards.ToObjects(bbs.Store, Board{}).([]Board)
	return boards
}

func (bbs *Bbs) AddThread(boardKey schema.HKey, name string) {
	newThread := Thread{Name:name}
	thread, _ := bbs.Store.Save(newThread)

	var board Board
	bbs.Store.Load(boardKey, &board)
	board.Threads = board.Threads.Append(thread.Hash)
	bbs.updateBoard(boardKey, board)
}

func (bbs *Bbs) CreateThread(name string, posts ...Post) Thread {

	result := Thread{Name:name, Posts:bbs.Store.CreateArray(Post{})}
	for i := 0; i < len(posts); i++ {
		h, _ := bbs.Store.Save(posts[i])
		result.Posts = result.Posts.Append(h.Hash)

	}
	return result
}

func (bbs *Bbs) GetThreads(board Board) []Thread {
	return board.Threads.ToObjects(bbs.Store, Thread{}).([]Thread)
}

func (bbs *Bbs) container() BoardContainer {
	container := &BoardContainer{Boards:bbs.Store.CreateArray(Board{})}
	bbs.Content.ToObject(bbs.Store, container)
	return *container
}

func (bbs *Bbs) updateBoard(key schema.HKey, board Board) {
	newBoard, _ := bbs.Store.Save(board)
	boards := []schema.HKey{newBoard.Hash}
	container := bbs.container()
	for i := 0; i < len(container.Boards.Items); i++ {
		if container.Boards.Items[i] != key {
			boards= append(boards, container.Boards.Items[i])
		}
	}
	container.Boards = bbs.Store.CreateArray(Board{}, boards...)
	bbs.Content, _ = bbs.Store.Save(container)
}
