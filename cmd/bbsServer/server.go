package main

import (
	"fmt"
	"strings"

	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// Server represents a RPC server for bbs.
type Server struct {
	ix *Indexer
}

// NewServer creates a new server.
func NewServer() *Server {
	return &Server{
		ix: NewIndexer(),
	}
}

// Greet greets the user with given name.
func (s *Server) Greet(args []string, response *[]byte) error {
	if len(args) == 0 {
		*response = encoder.Serialize("Hello Anonymous!")
		return nil
	}
	name := strings.TrimSpace(args[0])
	if name == "" {
		name = "Anonymous"
	}
	*response = encoder.Serialize("Hello " + name + "!")
	return nil
}

// AddBoard adds a board to root.
// arg[0] : Name of board to be created and added.
// arg[1] : Description of board to be created and added.
func (s *Server) AddBoard(args []string, response *[]byte) error {
	if len(args) != 2 {
		return fmt.Errorf("invalid arguments")
	}
	bKey := s.ix.CreateBoard(args[0], args[1])
	return s.ix.addBoardOfKeyToRoot(bKey)
}

// RemoveBoard removes a board from root.
// args[0] : Name of board to remove.
func (s *Server) RemoveBoard(args []string, response *[]byte) error {
	if len(args) != 1 {
		return fmt.Errorf("invalid arguments")
	}
	bKey, has := s.ix.boards[args[0]]
	if !has {
		return fmt.Errorf("board does not exist locally")
	}
	return s.ix.removeBoardOfKeyFromRoot(bKey)
}

// ListBoards retrieves a list of boards.
func (s *Server) ListBoards(args []string, response *[]byte) error {
	if len(args) != 0 {
		return fmt.Errorf("invalid arguments")
	}
	boards, e := s.ix.GetBoards()
	if e != nil {
		return e
	}
	*response = encoder.Serialize(boards)
	return nil
}

// AddThread adds a thread to specified board.
// args[0] : Name of thread to add.
// args[1] : Name of board to add thread to.
func (s *Server) AddThread(args []string, response *[]byte) error {
	// TODO: Complete.
	return nil
}
