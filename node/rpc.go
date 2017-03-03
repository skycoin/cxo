package node

import (
	"net"
	"net/rpc"

	"golang.org/x/net/netutil"

	"github.com/skycoin/cxo/data"
)

//
// "rpc.Connect",    "127.0.0.1:9090", *struct{}
// "rpc.Disconnect", "127.0.0.1:9090", *struct{}
// "rpc.List",       struct{}{},       *struct{List []string, Err error}
// "rpc.Info",       struct{}{}.       *struct{
//                                         Address string,
//                                         Stat    struct{
//                                             Total  int
//                                             Memory int
//                                         },
//                                         Err error,
//                                      }
//

// A RPC is used for Node internals
type RPC struct {
	log    Logger
	events chan func(*Node)

	quit chan struct{}
	done chan struct{}
}

// NewRPC is used for Node internals
func NewRPC(chanSize int, log Logger) *RPC {
	return &RPC{
		log:    log,
		events: make(chan func(*Node), chanSize),
		quit:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

func (r *RPC) start(address string, max int) (err error) {
	r.log.Debug("[DBG] start RPC")
	var l net.Listener
	if l, err = net.Listen("tcp", address); err != nil {
		return
	}
	r.log.Print("[INF] RPC is listening on ", l.Addr())
	if max > 0 {
		l = netutil.LimitListener(l, max)
	}
	rpc.RegisterName("rpc", r)
	go rpc.Accept(l)
	go r.handleQuit(r.quit, r.done, l)
	return
}

func (r *RPC) handleQuit(quit, done chan struct{}, l net.Listener) {
	<-quit
	l.Close()
	// drain evenst
	for {
		select {
		case evt := <-r.events:
			// we need to call events to release pending requests
			evt(nil)
		default:
			close(done)
			return
		}
	}
}

func (r *RPC) close() {
	r.log.Debug("[DBG] closing RPC...")
	close(r.quit)
	<-r.done
	r.log.Debug("[DBG] RPC was closed")
}

// Connect is remote procedure
func (r *RPC) Connect(address string, _ *struct{}) error {
	err := make(chan error)
	r.events <- func(node *Node) {
		if node == nil { // for drain
			err <- ErrClosed
			return
		}
		_, e := node.pool.Connect(address)
		err <- e
	}
	return <-err
}

// Disconnect is remote procedure
func (r *RPC) Disconnect(address string, _ *struct{}) error {
	err := make(chan error)
	r.events <- func(node *Node) {
		if node == nil { // for drain
			err <- ErrClosed
			return
		}
		if gc, ok := node.pool.Addresses[address]; ok {
			node.pool.Disconnect(gc, ErrManualDisconnect)
			err <- nil
			return
		}
		err <- ErrNotFound
	}
	return <-err
}

// A ListReply is reply to RPC-client when he calls List
type ListReply struct {
	List []string
	Err  error // always nil and used for internals
}

// List in remote procedure
func (r *RPC) List(_ struct{}, reply *ListReply) error {
	result := make(chan ListReply)
	r.events <- func(node *Node) {
		var r ListReply
		if node == nil { // for drain
			r.Err = ErrClosed
		} else {
			r.List = make([]string, 0, len(node.pool.Addresses))
			for a := range node.pool.Addresses {
				r.List = append(r.List, a)
			}
		}
		result <- r
	}
	*reply = <-result
	if reply.Err != nil {
		return reply.Err
	}
	return nil
}

// InfoReplry is reply to RPC-client when he calls Info
type InfoReply struct {
	Address string
	Stat    data.Stat
	Err     error
}

// Info is a remote procedure
func (r *RPC) Info(_ struct{}, reply *InfoReply) error {
	result := make(chan InfoReply)
	r.events <- func(node *Node) {
		var r InfoReply
		if node == nil { // for drain
			r.Err = ErrClosed
		} else {
			r.Stat = node.db.Stat()
			if a, e := node.pool.ListeningAddress(); e == nil {
				r.Address = a.String()
			}
		}
		result <- r
	}
	*reply = <-result
	if reply.Err != nil {
		return reply.Err
	}
	return nil
}
