package bbs

import (
	"fmt"
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/skycoin/src/cipher"
)

// developer note:
// this is replacemant for obsolete cxo/nodeManager.INodeSecurity { Sign(blah) }
// Because of Golang naming convention, now it called Signer and placed here
type Signer interface {
	Sign(hash cipher.SHA256) cipher.Sig //
}

type Bbs struct {
	//TODO implement thread lock for Content
	Container skyobject.ISkyObjects
	Board     cipher.SHA256
	sign      cipher.Sig
	security  Signer
}

func CreateBbs(dataSource data.IDataSource, security Signer) *Bbs {

	c := skyobject.SkyObjects(dataSource)
	c.RegisterSchema(Board{})
	c.RegisterSchema(Thread{})
	c.RegisterSchema(Post{})
	return &Bbs{Container: c, security: security}
}

func (bbs *Bbs) AddBoard(name string, threads ...Thread) Board {
	sl := skyobject.NewArray(threads)
	fmt.Println("Create threads")
	bbs.Container.Save(&sl)
	board := Board{Name: name, Threads: sl}
	bl := skyobject.NewObject(board)
	ref := bbs.Container.Save(&bl)
	sign := bbs.security.Sign(ref.Ref)
	fmt.Println("Sign", sign)
	bbs.Board = bbs.Container.Publish(ref, &sign)
	fmt.Println(bbs.Board)
	return board
}

func (bbs *Bbs) CreateThread(name string, posts ...Post) Thread {
	sl := skyobject.NewArray(posts)
	bbs.Container.Save(&sl)
	return Thread{
		Name:  name,
		Posts: sl,
	}
}

func (bbs *Bbs) CreatePost(header string, text string) Post {
	//poster := skyobject.NewObject(Poster{})
	//bbs.Container.Save(&poster)

	return Post{Header: header, Text: text}
}
