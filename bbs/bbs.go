package bbs

import (
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/skycoin/src/cipher"
)

type Bbs struct {
	//TODO implement thread lock for Content
	Container skyobject.ISkyObjects
	board    skyobject.Href
	sign      cipher.Sig
}

func CreateBbs(dataSource data.IDataSource) *Bbs {
	c := skyobject.SkyObjects(dataSource)
	//c.RegisterSchema(BoardType{})
	c.RegisterSchema(Board{})
	c.RegisterSchema(Thread{})
	c.RegisterSchema(Post{})
	return &Bbs{Container:c}
}
func (bbs *Bbs) AddBoard(name string, threads ...Thread) Board {
	sl := skyobject.NewSlice(Thread{}, threads)
	bbs.Container.Save(&sl)
	board := Board{Name:name, Threads:sl}
	bl := skyobject.NewLink(board)
	bbs.Container.Save(&bl)
	bl = skyobject.NewLink(board)
	bbs.board = bbs.Container.Save(&bl)
	return board
}

func (bbs *Bbs) CreateThread(name string, posts ...Post) Thread {
	sl := skyobject.NewSlice(Post{}, posts)
	bbs.Container.Save(&sl)
	return Thread{Name:name, Posts:sl}

}

func (bbs *Bbs) CreatePost(header string, text string) Post {
	poster := skyobject.NewLink(Poster{})
	bbs.Container.Save(&poster)

	return Post{Header:header, Text:text, Poster:poster}
}
