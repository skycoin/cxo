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
	RemoteClose bool   = false
)

func waitInterrupt(quit <-chan struct{}) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	select {
	case <-sig:
		os.Exit(0)
	case <-quit:
	}
}

func main() {

	var c node.ServerConfig = node.NewServerConfig()

	c.RPCAddress = RPC
	c.Listen = Host
	c.RemoteClose = RemoteClose

	c.FromFlags()
	flag.Parse()

	var s *node.Server
	var err error

	if s, err = node.NewServer(c); err != nil {
		log.Print(err)
		return
	}

	if err = s.Start(); err != nil {
		log.Print(err)
		return
	}
	defer s.Close()

	// TODO: subscribe and connect to KNOWN

	// waiting for SIGINT or termination using RPC

	waitInterrupt(s.Quiting())

}
