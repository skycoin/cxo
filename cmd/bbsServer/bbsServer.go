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
		I: MakeBBSIndexer(),
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
	r.I.AddBoard("board1", thread1, thread2)
	r.I.AddBoard("board2", thread2)

	replyMsg := fmt.Sprintf(
		"We now have %d boards, %d threads and %d posts.",
		len(r.I.Boards), len(r.I.Threads), len(r.I.Posts),
	)

	*result = messages.Serialize((uint16)(0), &replyMsg)
	return nil
}

func (r *RPCReceiver) ListBoards(_ []string, result *[]byte) error {
	r.I.Load()
	fmt.Printf("Listing %d boards.\n", len(r.I.Boards))
	*result = messages.Serialize((uint16)(0), r.I.Boards)
	return nil
}

func (r *RPCReceiver) ListThreads(args []string, result *[]byte) error {
	r.I.Load()

	switch len(args) {
	case 0:
		fmt.Printf("Listing all threads: %d.\n", len(r.I.Threads))
		*result = messages.Serialize((uint16)(0), r.I.Threads)

	case 1:
		key := args[0]
		threads, e := r.I.GetThreadsFromBoard(key)
		if e != nil {
			fmt.Println(e)
			return e
		}

		fmt.Printf("Listing %d threads for board '%s'.\n", len(threads), key)
		*result = messages.Serialize((uint16)(0), threads)

	default:
		return errors.New("invalid number of arguments")
	}

	return nil
}

func (r *RPCReceiver) ListPosts(args []string, result *[]byte) error {
	r.I.Load()

	switch len(args) {
	case 0:
		fmt.Printf("Listing all posts: %d.\n", len(r.I.Posts))
		*result = messages.Serialize((uint16)(0), r.I.Posts)

	case 1:
		key := args[0]
		posts, e := r.I.GetPostsFromThread(key)
		if e != nil {
			fmt.Println(e)
			return e
		}

		fmt.Printf("Listing %d posts for thread '%s'.\n", len(posts), key)
		*result = messages.Serialize((uint16)(0), posts)

	default:
		return errors.New("invalid number of arguments")
	}

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
