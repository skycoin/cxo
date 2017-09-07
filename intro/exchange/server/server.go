package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/node/log"

	cxo "github.com/skycoin/cxo/skyobject"
)

// defaults
const (
	Host        string = "[::1]:8000" // default host address of the server
	RPC         string = "[::1]:7000" // default RPC address
	RemoteClose bool   = false        // don't allow closing by RPC by default

	Discovery string = "[::1]:8008" // discovery server
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

	defer func() {
		if err := recover(); err != nil {
			code = 1
			fmt.Fprintln(os.Stderr, "[PANIC]:", err)
		}
		os.Exit(code) // the Exit "recovers" silently
	}()

	var c = node.NewConfig()

	c.RPCAddress = RPC
	c.Listen = Host
	c.RemoteClose = RemoteClose

	c.PublicServer = true
	c.DiscoveryAddresses = node.Addresses{Discovery}

	c.PingInterval = 0 // suppress pings for this example

	c.DataDir = ""      // don't create ~/.skycoin/cxo
	c.InMemoryDB = true // use DB in memeory

	c.Log.Prefix = "[server] "
	c.Log.Debug = true
	c.Log.Pins = log.All &^ (node.DiscoveryPin | node.HandlePin | node.FillPin |
		node.ConnPin)

	c.Skyobject.Log.Debug = true
	c.Skyobject.Log.Pins = log.All &^ (cxo.VerbosePin | cxo.PackSavePin |
		cxo.FillVerbosePin)
	c.Skyobject.Log.Prefix = "[server cxo] "

	// suppress gnet logger
	c.Config.Logger = log.NewLogger(log.Config{Output: ioutil.Discard})

	c.FromFlags()
	flag.Parse()

	// apk and bk is A-feed and B-feed
	apk, _ := cipher.GenerateDeterministicKeyPair([]byte("A"))
	bpk, _ := cipher.GenerateDeterministicKeyPair([]byte("B"))

	var s *node.Node
	var err error

	// create and launch
	if s, err = node.NewNode(c); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
		return
	}
	defer s.Close() // close

	// add feeds
	for _, pk := range []cipher.PubKey{
		apk,
		bpk,
	} {
		if err = s.AddFeed(pk); err != nil {
			fmt.Println("[ERR] database failure:", err)
			code = 1
			return
		}
	}

	waitInterrupt(s.Closed())
}
