package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"

	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// DefaultPort is the default port for RPC Server.
const DefaultPort = "1235"

// RPCReceiver represents a RPC Reciever.
type RPCReceiver struct {
	I *BBSIndexer
}

// MakeRPCReceiver makes a RPCReceiver
func MakeRPCReceiver() *RPCReceiver {
	return &RPCReceiver{MakeBBSIndexer()}
}

// Greet sends a greeting message.
func (r *RPCReceiver) Greet(args []string, result *[]byte) error {
	replyMsg := "Hello "
	if len(args) == 0 {
		fmt.Println("Greeting anonymous.")
		replyMsg += "anonymous"
	} else {
		fmt.Println("Greeting ", args[0])
		replyMsg += args[0]

	}
	*result = encoder.Serialize(&replyMsg)
	return nil
}

func main() {
	port := os.Getenv("BBS_SERVER_PORT")
	if port == "" {
		log.Println("No BBS_SERVER_PORT environmental variable found, assigning default port value:", DefaultPort)
		port = DefaultPort
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
