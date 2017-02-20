package main

import (
	// "errors"
	"fmt"
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
}

func (receiver *RPCReceiver) Greet(args []string, results *[]byte) error {
	replyMsg := "Hello "
	if len(args) == 0 {
		fmt.Println("Greeting anonymous.")
		replyMsg += "anonymous"
	} else {
		fmt.Println("Greeting ", args[0])
		replyMsg += args[0]

	}
	*results = messages.Serialize((uint16)(0), &replyMsg)
	return nil
}

func main() {
	port := os.Getenv("BBS_SERVER_PORT")
	if port == "" {
		log.Println("No BBS_SERVER_PORT environmental variable found, assigning default port value:", DEFAULT_PORT)
		port = DEFAULT_PORT
	}

	receiver := new(RPCReceiver)
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
