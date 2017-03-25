// Package server is used to control a cxo/node using RPC.
// The list of remote procedures:
//
//     "rpc.Connect",     address      string,  _       *struct{}
//     "rpc.Disconnect",  address      string,  _       *struct{}
//     "rpc.Inject",      params  comm.Inject,  _       *struct{}
//     "rpc.List"         _          struct{},  list    *[]string
//     "rpc.Info",        _          struct{},  info    *comm.Info
//     "rpc.Stat",        _          struct{},  stat    *data.Stat
//     "rpc.Terminate",   _          struct{},  _       *struct{}
//
package server

import (
	"net"
	"net/rpc"

	"golang.org/x/net/netutil"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/rpc/comm"
)

// A Config contains RPC-server configurations
type Config struct {
	Enable  bool
	Address string // listening address
	Max     int    // maximum connections
}

// NewConig retusn Cofig filled down with default values
func NewConfig() (rc Config) {
	rc.Enable = true
	rc.Max = 10
	return
}

// A Server represents RPC server for node.Node
type Server struct {
	n *node.Node // reference to node, wich also used as logger

	c Config // configurations

	quit chan struct{}
	done chan struct{}
}

// NewServer creates new RPC server for given node with given configs
func NewServer(conf Config, n *node.Node) *Server {
	return &Server{
		n:    n,
		c:    conf,
		quit: make(chan struct{}),
		done: make(chan struct{}),
	}
}

// Start is used to launch RPC server
func (s *Server) Start() (err error) {
	s.n.Debug("[DBG] [RPC] staring RPC server")
	var l net.Listener
	if l, err = net.Listen("tcp", s.c.Address); err != nil {
		return
	}
	s.n.Print("[INF] [RPC] RPC server is listening on ", l.Addr())
	if s.c.Max > 0 {
		l = netutil.LimitListener(l, s.c.Max)
	}
	rpc.RegisterName("rpc", s)
	go s.accept(l, s.done)
	go s.handleQuit(s.quit, l)
	return
}

func (s *Server) accept(l net.Listener, done chan struct{}) {
	defer close(done)
	rpc.Accept(l)
}

func (s *Server) handleQuit(quit chan struct{}, l net.Listener) {
	<-quit
	l.Close()
}

// Close is used to shutdown the Server
func (s *Server) Close() {
	s.n.Debug("[DBG] [RPC] closing RPC...")
	close(s.quit)
	<-s.done
	s.n.Debug("[DBG] [RPC] RPC was closed")
}

//
// RPC Procedures
//

func (s *Server) Connect(address string, _ *struct{}) (err error) {
	err = s.n.Connect(address)
	return
}

func (s *Server) Subscribe(pub cipher.PubKey, _ *struct{}) (_ error) {
	s.n.Subscribe(pub)
	return
}

func (s *Server) Disconnect(address string, _ *struct{}) (err error) {
	err = s.n.Disconnect(address)
	return
}

func (s *Server) Inject(args comm.Inject, _ *struct{}) (err error) {
	err = s.n.Inject(args.Hash, args.Pub, args.Sec)
	return
}

func (s *Server) List(_ struct{}, list *[]string) (err error) {
	var l []string
	if l, err = s.n.List(); err != nil {
		return
	}
	*list = l
	return
}

func (s *Server) Info(_ struct{}, info *comm.Info) (err error) {
	var inf comm.Info
	if inf, err = s.n.Info(); err != nil {
		return
	}
	*info = inf
	return
}

func (s *Server) Stat(_ struct{}, stat *data.Stat) (err error) {
	var st data.Stat
	if st, err = s.n.Stat(); err != nil {
		return
	}
	*stat = st
	return
}

func (s *Server) Terminate(_ struct{}, _ *struct{}) (err error) {
	err = s.n.Terminate()
	return
}
