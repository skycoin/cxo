package main

import (
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
