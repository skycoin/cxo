package client_test

import (
	"fmt"

	"github.com/skycoin/cxo/node/rpc/client"
)

const (
	ADDRESS = ""
	HOST    = ""
)

func Example() {
	node, err := client.Dial("tcp", ADDRESS)
	if err != nil {
		// handle error
		return
	}
	// close the client (not the remote node we are conencted to)
	defer node.Close()
	connections, err := node.List()
	if err != nil {
		// handle error
		return
	}
	for _, address := range connections {
		fmt.Println("connected to ", address)
	}
	if err = node.Connect(HOST); err != nil {
		fmt.Println("connection failed:", err)
	}
	if err = node.Terminate(); err != nil {
		fmt.Println("can't terminate node:", err)
	}
}
