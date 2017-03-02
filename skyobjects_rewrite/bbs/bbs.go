package bbs

import (
	// "fmt"
	"fmt"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobjects"
	"github.com/skycoin/skycoin/src/cipher"
)

// Signer is a replacemant for obsolete cxo/nodeManager.INodeSecurity { Sign(blah) }
// Because of Golang naming convention, now it called Signer and placed here.
type Signer interface {
	Sign(hash cipher.SHA256) cipher.Sig
}

// Bbs represents Bulletin Board System.
type Bbs struct {
	//TODO implement thread lock for Content
	Container *skyobjects.Container
	sign      cipher.Sig
	security  Signer
}

// CreateBbs creates a Bbs object.
func CreateBbs(dataSource data.IDataSource, security Signer) *Bbs {
	c := skyobjects.NewContainer(dataSource)
	c.Schema.Register(Board{}, Thread{}, Post{})
	return &Bbs{Container: c, security: security}
}

// AddBoard adds a board to the Bbs if it does not exist.
func (bbs *Bbs) AddBoard(name, description string) (cipher.SHA256, error) {
	// Check if board with same name already exists.
	if ref, _ := bbs.FindBoard(name); ref != nil {
		return cipher.SHA256{}, fmt.Errorf("board of name '%s' already exists", name)
	}
	// Add new board.
	boardRef := skyobjects.NewObjectReference(Board{Name: name, Description: description})
	return bbs.Container.StoreToRoot(boardRef).GetKey(), nil
}

// RemoveBoard removes a board from Bbs.
func (bbs *Bbs) RemoveBoard(name string) error {
	// Find the board.
	if ref, _ := bbs.FindBoard(name); ref != nil {
		return bbs.Container.RemoveFromRoot(ref.GetKey())
	}
	return fmt.Errorf("board of name '%s' doesn't exist or is removed", name)
}

// AddThreadToBoard adds a thread to specified board.
func (bbs *Bbs) AddThreadToBoard(threadName, boardName string) error {
	var threads []Thread

	// Find the board with name 'boardName'.
	boardRef, board := bbs.FindBoard(boardName)

	if boardRef == nil {
		return fmt.Errorf("board of name `%s` doesn't exist", boardName)
	}
	// See if thread already exists whilst looping through.
	for _, threadRef := range boardRef.GetChildren(bbs.Container) {
		var thread Thread
		threadRef.Deserialize(&thread)
		if thread.Name == threadName {
			return fmt.Errorf(
				"thread of name '%s' already exists in board of name %s",
				threadName, boardName,
			)
		}
		threads = append(threads, thread)
	}

	// Create new thread and append to thread array.
	threads = append(threads, Thread{Name: threadName})
	threadsRef := skyobjects.NewArrayReference(threads)
	bbs.Container.Store(threadsRef)

	// Add threads to new board.
	board.Threads = *threadsRef
	newBoardRef := skyobjects.NewObjectReference(*board)
	bbs.Container.StoreToRoot(newBoardRef)

	// Remove old board from root.
	bbs.Container.RemoveFromRoot(boardRef.GetKey())

	return nil
}

// FindBoard finds board of specified name.
func (bbs *Bbs) FindBoard(name string) (skyobjects.IReference, *Board) {
	for _, ref := range bbs.Container.GetRootValues() {
		var board Board
		if ref.Deserialize(&board); board.Name == name {
			return ref, &board
		}
	}
	return nil, nil
}

// RemoveThreadFromBoard removes a thread from specified board.
func (bbs *Bbs) RemoveThreadFromBoard(threadName, boardName string) error {
	var threads []Thread

	// Find board.
	boardRef, board := bbs.FindBoard(boardName)
	if boardRef == nil {
		return fmt.Errorf("board of name '%s' doesn't exist", boardName)
	}
	// Find thread.
	threadFound := false
	for _, threadRef := range boardRef.GetChildren(bbs.Container) {
		var thread Thread
		if threadRef.Deserialize(&thread); thread.Name == threadName {
			threadFound = true
			continue
		}
		threads = append(threads, thread)
	}
	if threadFound == false {
		return fmt.Errorf(
			"thread of name '%s' already doesn't exist in board of name '%s'",
			threadName, boardName,
		)
	}

	// Recreate new thread array.
	newThreadsRef := skyobjects.NewArrayReference(threads)
	bbs.Container.Store(newThreadsRef)

	// Add threads to new board.
	board.Threads = *newThreadsRef
	newBoardRef := skyobjects.NewObjectReference(board)
	bbs.Container.StoreToRoot(newBoardRef)

	// Remove old board from root.
	bbs.Container.RemoveFromRoot(boardRef.GetKey())

	return nil
}

// CreateThread creates a thread.
func (bbs *Bbs) CreateThread(name string) Thread { return Thread{name} }

// AddPost adds a new post to Bbs.
func (bbs *Bbs) AddPost(header string, text string, thread Thread) Post {

	tl := skyobjects.NewObjectReference(thread)
	bbs.Container.Store(tl)

	post := Post{Header: header, Text: text, Thread: *tl}
	pl := skyobjects.NewObjectReference(post)
	bbs.Container.Store(pl)

	return post
}

func (bbs *Bbs) addBoard(name, description string, threads ...Thread) cipher.SHA256 {
	sl := skyobjects.NewArrayReference(threads)
	bbs.Container.Store(sl)

	board := Board{Name: name, Description: description, Threads: *sl}
	bl := skyobjects.NewObjectReference(board)
	bbs.Container.StoreToRoot(bl)

	return bl.GetKey()
}
