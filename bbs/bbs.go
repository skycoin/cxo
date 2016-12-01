package bbs

import (
	"github.com/skycoin/cxo/schema"
	"github.com/skycoin/cxo/data"
)

type Bbs struct {
	//TODO implement thread lock for Content
	//Content schema.Href
	Container *schema.Container
}

func CreateBbs(dataSource data.IDataSource) *Bbs {
	c:= schema.NewContainer(dataSource)
	c.Register(Board{})
	c.Register(Thread{})
	c.Register(Post{})
	return &Bbs{Container:c}
}

func (bbs *Bbs) AddBoard(name string) (schema.HKey, error) {

	b := Board{Name:name, Threads:bbs.Container.CreateArray(Thread{})}
	href, _ := bbs.Container.Save(b)
	boards := bbs.getBoards()
	boards.Items = append(boards.Items, href.Hash)
	bbs.Container.SaveRoot(boards)
	return href.Hash, nil
}

func (bbs *Bbs) CreateBoard(name string, threads ...Thread) Board {
	result := Board{Name:name, Threads:bbs.Container.CreateArray(Thread{})}
	for i := 0; i < len(threads); i++ {
		href, _ := bbs.Container.Save(threads[i])
		result.Threads = result.Threads.Append(href.Hash)
	}
	return result
}

func (bbs *Bbs) AddBoards(boards []Board)  {
	boardToSave := bbs.getBoards()
	for i := 0; i < len(boards); i++ {
		href, _ := bbs.Container.Save(boards[i])
		boardToSave.Items = append(boardToSave.Items, href.Hash)
	}
	bbs.Container.SaveRoot(boardToSave)
}

//TODO: Just for test - need refactoring
func (bbs *Bbs) GetBoard(name string) schema.HKey {
	allObjects := bbs.AllBoars()
	boards := bbs.getBoards()
	for i := 0; i < len(allObjects); i++ {
		if (allObjects[i].Name == name) {
			return boards.Items[i]
		}
	}
	return schema.HKey{}
}

func (bbs *Bbs) AllBoars() []Board {
	var boards []Board
	boards = bbs.getBoards().ToObjects(bbs.Container, Board{}).([]Board)
	return boards
}

func (bbs *Bbs) AddThread(boardKey schema.HKey, name string) {
	newThread := Thread{Name:name}
	href, _ := bbs.Container.Save(newThread)
	var board Board
	bbs.Container.Load(boardKey, &board)
	board.Threads = board.Threads.Append(href.Hash)
	bbs.updateBoard(boardKey, board)
}

func (bbs *Bbs) CreateThread(name string, posts ...Post) Thread {

	result := Thread{Name:name, Posts:bbs.Container.CreateArray(Post{})}
	for i := 0; i < len(posts); i++ {
		h, _ := bbs.Container.Save(posts[i])
		result.Posts = result.Posts.Append(h.Hash)

	}
	return result
}

func (bbs *Bbs) GetThreads(board Board) []Thread {
	return board.Threads.ToObjects(bbs.Container, Thread{}).([]Thread)
}

func (bbs *Bbs) getBoards() schema.HArray {
	bs := bbs.Container.CreateArray(Board{})
	bbs.Container.Load(bbs.Container.Root.Hash, &bs)
	return bs
}

func (bbs *Bbs) updateBoard(key schema.HKey, board Board) {
	container := bbs.getBoards()
	h, _ := bbs.Container.Save(board)
	for i := 0; i < len(container.Items); i++ {
		if container.Items[i] == key {
			container.Items[i] = h.Hash
			break
		}
	}
	bbs.Container.SaveRoot(container)
	//bbs.Content, _ = bbs.Container.Save(container)

	//newBoard, _ := bbs.Container.Save(board)
	////boards := []schema.HKey{newBoard.Hash}
	//boards := bbs.getBoards()
	//boards.Items = append(boards.Items, newBoard.Hash)
	//fmt.Println("boards: ", boards)
	//
	////boardsArray = bbs.Container.CreateArray(Board{}, boards...)
	//bbs.Container.SaveRoot(boards)

	//bbs.Content, _ = bbs.Container.Save(container)
}
