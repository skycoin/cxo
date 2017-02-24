package bbs

import (
	// "fmt"
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/nodeManager"
	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/skycoin/src/cipher"
)

type Bbs struct {
	//TODO implement thread lock for Content
	Container skyobject.ISkyObjects
	Board     cipher.SHA256
	sign      cipher.Sig
	security  nodeManager.INodeSecurity
}

func CreateBbs(dataSource data.IDataSource, security nodeManager.INodeSecurity) *Bbs {
	c := skyobject.SkyObjects(dataSource)
	c.RegisterSchema(Board{})
	c.RegisterSchema(Thread{})
	c.RegisterSchema(Post{})
	return &Bbs{Container: c, security: security}
}

func (bbs *Bbs) AddBoard(name string, threads ...Thread) Board {

	sl := skyobject.NewArray(threads)
	bbs.Container.Save(&sl)

	board := Board{Name: name, Threads: sl}
	bl := skyobject.NewObject(board)
	ref := bbs.Container.Save(&bl)

	sign := bbs.security.Sign(ref.Ref)
	bbs.Board = bbs.Container.Publish(ref, &sign)

	return board
}

func (bbs *Bbs) CreateThread(name string, posts ...Post) Thread {
	sl := skyobject.NewArray(posts)
	bbs.Container.Save(&sl)
	return Thread{Name: name, Posts: sl}

}

func (bbs *Bbs) CreatePost(header string, text string) Post {
	//poster := skyobject.NewObject(Poster{})
	//bbs.Container.Save(&poster)

	return Post{Header: header, Text: text}
}
