package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/skycoin/cxo/node"
)

// defaults
const (
	Host        string = "[::]:8998" // default host address of the server
	RPC         string = "[::]:8997" // default RPC address
	RemoteClose bool   = false       // don't allow closing by RPC by default
)

func waitInterrupt(quit <-chan struct{}) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	select {
	case <-sig:
	case <-quit:
	}
}

func main() {

	var code int

	defer func() { os.Exit(code) }()

	var c = node.NewConfig()

	c.RPCAddress = RPC
	c.Listen = Host
	c.RemoteClose = RemoteClose

	c.FromFlags()
	flag.Parse()

	var s *node.Node
	var err error

	// create and launch
	if s, err = node.NewNode(c); err != nil {
		log.Print(err)
		code = 1
		return
	}
	defer s.Close()

	// TODO: subscribe and connect to KNOWN

	// waiting for SIGINT or termination using RPC

	waitInterrupt(s.Quiting())

}
