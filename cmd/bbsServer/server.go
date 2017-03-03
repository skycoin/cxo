package main

import (
	"github.com/skycoin/cxo/gui"
)

// BbsServer handles subscription to CXO Daemon and Connection with RPC Client.
type BbsServer struct {
	Router *gui.Router
}

// NewBbsServer creates a new BbsServer.
func NewBbsServer() *BbsServer {
	server := BbsServer{
		Router: gui.NewRouter(),
	}
	server.Router.Serve("http://127.0.0.1:6482")
	return &server
}

// Run runs the server.
func (s *BbsServer) Run() {
	for {

	}
}

func main() {
	server := NewBbsServer()
	server.Run()
}
