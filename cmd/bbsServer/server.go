package main

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
