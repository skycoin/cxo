package bbs

import (
	"github.com/skycoin/cxo/schema"
	"fmt"
)

type Bbs struct {
	Boards schema.HrefStatic `boards:"[]Boards"`
	Store  *schema.Store
}

func CreateBbs() *Bbs {
	return &Bbs{Store:schema.NewStore(), Boards:schema.HrefStatic{}}
}

func (bbs *Bbs) AddBoard(name string) (Board, error) {
	boards := []Board{}
	bbs.Store.Load(bbs.Boards.Hash, &boards)

	b := Board{Name:name, Threads:schema.HrefStatic{}}
	boards = append(boards, b)
	bbs.Store.Save(b)

	bbs.updateBoard(boards)
	return b, nil
}

func (bbs *Bbs) AllBoars() []Board {
	var res []Board
	bbs.Store.Load(bbs.Boards.Hash, &res)
	return res
}

func (bbs *Bbs) AddThread(board Board, name string) {
	var threads []Thread = []Thread{}
	e := bbs.Store.Load(board.Threads.Hash, &threads)
	fmt.Println("Current threads", threads)
	if (e != nil) {
		fmt.Println(e)
	}
	newThread := Thread{Name:name}

	threadKey, _ := bbs.Store.Save(newThread)
	fmt.Println("threadKey", threadKey)

	threads = append(threads, newThread)

	bbs.updateThreads(board, threads)
}

func (bbs *Bbs) GetThreads(board Board) []Thread {
	var threads []Thread = []Thread{}

	e := bbs.Store.Load(board.Threads.Hash, &threads)
	if (e != nil) {
		fmt.Println(e)
	}

	return threads
}

func (bbs *Bbs) updateBoard(boards []Board) {
	bbs.Boards, _ = bbs.Store.BuildHref(boards)
}

func (bbs *Bbs) updateThreads(board Board, threads []Thread) {
	boards := []Board{}
	bbs.Store.Load(bbs.Boards.Hash, &boards)

	for i := 0; i < len(boards); i++ {
		//TODO: Refactor. Names could be equal. Need better way to compare objects(May be calculate and compare hash)
		if (boards[i].Name == board.Name) {
			boards[i].Threads, _ = bbs.Store.BuildHref(threads)
			bbs.updateBoard(boards)
			break
		}
	}
}
