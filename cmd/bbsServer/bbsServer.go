package main

import (
	"errors"
	"fmt"
	// "github.com/skycoin/cxo/data"
	// "github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/skycoin/src/mesh/messages"
	// "io"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

const DEFAULT_PORT = "1235"

type RPCReceiver struct {
	I *BBSIndexer
}

func MakeRPCReceiver() (receiver *RPCReceiver) {
	// db := data.NewDB()
	receiver = &RPCReceiver{
		I: MakeBBSIndexerSimple(),
	}
	return
}

func (r *RPCReceiver) Greet(args []string, result *[]byte) error {
	replyMsg := "Hello "
	if len(args) == 0 {
		fmt.Println("Greeting anonymous.")
		replyMsg += "anonymous"
	} else {
		fmt.Println("Greeting ", args[0])
		replyMsg += args[0]

	}
	*result = messages.Serialize((uint16)(0), &replyMsg)
	return nil
}

func (r *RPCReceiver) GenerateRandomData(_ []string, result *[]byte) error {
	post1 := r.I.CreatePost("post1", "bla 111")
	post2 := r.I.CreatePost("post2", "bla 222")
	post3 := r.I.CreatePost("post3", "bla 333")
	thread1 := r.I.AddThread("thread1", post1, post2)
	thread2 := r.I.AddThread("thread2", post3)
	thread3 := r.I.AddThread("thread3")
	r.I.AddBoard("board1", thread1)
	r.I.AddBoard("board2", thread2, thread3)

	replyMsg := "Generated 3 posts, 3 threads and 2 boards."
	*result = messages.Serialize((uint16)(0), &replyMsg)
	return nil
}

func (r *RPCReceiver) ListBoards(_ []string, result *[]byte) error {
	r.I.Load()
	fmt.Printf("Listing %d boards.\n", len(r.I.Boards))
	*result = messages.Serialize((uint16)(0), r.I.Boards)
	return nil
}

func (r *RPCReceiver) ListThreadsForBoard(args []string, result *[]byte) error {
	if len(args) < 1 {
		return errors.New("no Board specified")
	}
	// TODO: Implement.
	return nil
}

func (r *RPCReceiver) ListPostsForThread(args []string, result *[]byte) error {
	if len(args) < 1 {
		return errors.New("no Thread specified")
	}
	// TODO: Implement.
	return nil
}

func main() {
	port := os.Getenv("BBS_SERVER_PORT")
	if port == "" {
		log.Println("No BBS_SERVER_PORT environmental variable found, assigning default port value:", DEFAULT_PORT)
		port = DEFAULT_PORT
	}

	receiver := MakeRPCReceiver()
	e := rpc.Register(receiver)
	if e != nil {
		panic(e)
	}
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":"+port)
	if e != nil {
		panic(e)
	}
	e = http.Serve(l, nil)
	if e != nil {
		panic(e)
	}
}
