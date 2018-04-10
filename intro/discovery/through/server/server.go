package main

// the server doesn't import the through package and knows
// nothing about its types and registries; but the server
// can collect and share any objects (any feeds)

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"
)

// defaults
const (
	Bind string = "[::1]:8000" // default host address of the node
	RPC  string = "[::1]:7000" // default RPC address

	Discovery string = "[::1]:8008" // discovery server
)

// feeds we are going to share
var (
	apk, _ = cipher.GenerateDeterministicKeyPair([]byte("A")) // A-feed (ca)
	bpk, _ = cipher.GenerateDeterministicKeyPair([]byte("B")) // B-feed (cb)
)

// wait for SIGINT and return
func waitInterrupt() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}

func main() {

	var c = node.NewConfig()

	c.RPC = RPC                                 // enable RPC
	c.TCP.Listen = Bind                         // listen
	c.TCP.Discovery = node.Addresses{Discovery} // conenct o discovery server

	c.Public = true // share list of feeds

	// use DB in memeory for the example
	c.Config.InMemoryDB = true

	// logs
	c.Logger.Prefix = "[server] "

	// uncomment to see all debug logs
	//
	// c.Logger.Pins = ^c.Logger.Pins
	// c.Logger.Debug = true

	// obtain configurations from commandline flags
	c.FromFlags()
	flag.Parse()

	var (
		n   *node.Node
		err error
	)

	// creating node, we creates container instance
	if n, err = node.NewNode(c); err != nil {
		log.Fatal(err)
	}
	defer n.Close() // close

	// share the feeds
	//
	// we have to add the feed to the server; after the
	// appending, the server sends list of its feeds to
	// the discovery server and the discovery server
	// connects other nodes to the server (other nodes,
	// that share feeds apk and bpk)

	if err = n.Share(apk); err != nil {
		log.Fatal(err)
	}

	if err = n.Share(bpk); err != nil {
		log.Fatal(err)
	}

	// daemon mode: on

	waitInterrupt()

}
