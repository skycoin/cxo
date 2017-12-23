package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"
)

func waitInterrupt() {
	var sig = make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}

func main() {

	var c = node.NewConfig()
	c.OnSubscribeRemote = acceptAllSubscriptions

	c.FromFlags()
	flag.Parse()

	var (
		n   *node.Node
		err error
	)

	// create and launch
	if n, err = node.NewNode(c); err != nil {
		log.Fatal(err)
	}
	defer n.Close()

	// waiting for SIGINT
	waitInterrupt()
}

// accept all incoming subscriptions
func acceptAllSubscriptions(c *node.Conn, pk cipher.PubKey) (_ error) {
	if err := c.Node().Share(pk); err != nil {
		log.Fatal("DB failure:", err) // DB failure
	}
	return
}
