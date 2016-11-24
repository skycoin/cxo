package bbs

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/cxo/encoder"
)

type Bbs struct {
	//TODO implement thread lock for Content
	Content encoder.Href `type:"BoardContainer"`
	Store   *encoder.Store
}

func CreateBbs() *Bbs {
	store := encoder.NewStore()
	container := BoardContainer{Boards:encoder.NewHArray()}
	content, _ := store.Save(container)
	return &Bbs{Store:store, Content:content}
}

func (bbs *Bbs) AddBoard(name string) (cipher.SHA256, error) {
	b := Board{Name:name, Threads:encoder.NewHArray()}
	href, _ := bbs.Store.Save(b)

	var container BoardContainer
	bbs.Content.ToObject(bbs.Store, &container)

	container.Boards = container.Boards.Append(href)
	bbs.Content, _ = bbs.Store.Save(container)
	return href.Hash, nil
}

//TODO: Just for test - need refactoring
func (bbs *Bbs) GetBoard(name string) cipher.SHA256 {
	allObjects := bbs.AllBoars()
	container := bbs.continer()
	bbs.Content.ToObject(bbs.Store, &container)
	for i := 0; i < len(allObjects); i++ {
		if (allObjects[i].Name == name) {
			return container.Boards[i].Hash
		}
	}
	return cipher.SHA256{}
}

func (bbs *Bbs) AllBoars() []Board {
	var boards []Board
	boards = bbs.continer().Boards.ToObjects(bbs.Store, Board{}).([]Board)
	return boards
}

func (bbs *Bbs) AddThread(boardKey cipher.SHA256, name string) {
	newThread := Thread{Name:name}
	thread, _ := bbs.Store.Save(newThread)

	var board Board
	bbs.Store.Load(boardKey, &board)
	board.Threads = board.Threads.Append(thread)
	bbs.updateBoard(boardKey, board)
}

func (bbs *Bbs) GetThreads(board Board) []Thread {
	return board.Threads.ToObjects(bbs.Store, Thread{}).([]Thread)
}

func (bbs *Bbs) continer() BoardContainer{
	container := &BoardContainer{}
	bbs.Content.ToObject(bbs.Store, container)
	return *container
}

func (bbs *Bbs) updateBoard(key cipher.SHA256, board Board) {
	newBoard, _ := bbs.Store.Save(board)
	boards := []encoder.Href{newBoard}
	container:= bbs.continer()
	for i := 0; i < len(container.Boards); i++ {
		if container.Boards[i].Hash != key {
			boards = append(boards, container.Boards[i])
		}
	}
	container.Boards = boards
	bbs.Content, _ = bbs.Store.Save(container)
}
