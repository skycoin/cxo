package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"

	"github.com/skycoin/cxo/skyobject/registry"
)

// defaults
const (
	Seed string = "[::1]:8001" // address of the seed
	RPC  string = "[::1]:7002" // default RPC address
)

// feeds of the seed
var (
	apk, _ = cipher.GenerateDeterministicKeyPair([]byte("A"))
	bpk, _ = cipher.GenerateDeterministicKeyPair([]byte("B"))
)

func main() {

	var c = node.NewConfig()

	c.RPC = RPC       // enable RPC
	c.TCP.Listen = "" // don't listen

	// not public

	// use DB in memory for the example
	c.Config.InMemoryDB = true

	// change cache parameters for example
	c.Config.CacheMaxAmount = 185
	c.Config.CacheMaxVolume = 30 * 1024
	c.Config.CacheMaxItemSize = 512

	// prefix for logs
	c.Logger.Prefix = "[peer] "

	// uncomment to see all debug logs
	//
	// c.Logger.Pins = ^c.Logger.Pins
	// c.Logger.Debug = true

	// obtain configs from flags
	c.FromFlags()
	flag.Parse()

	// create node

	var (
		n   *node.Node
		err error
	)

	// create and launch
	if n, err = node.NewNode(c); err != nil {
		log.Fatal(err)
		return
	}
	defer n.Close() // close

	// connect to the seed

	var conn *node.Conn
	if conn, err = n.TCP().Connect(Seed); err != nil {
		log.Fatal(err)
	}

	//
	// interactive shell
	//

	fmt.Println("interactive commands:")
	fmt.Println("  A       preview A feed")
	fmt.Println("  B       preview B feed")
	fmt.Println("  quit    leave app")

	for {

		fmt.Print("> ")

		var cmd string
		if _, err = fmt.Scanln(&cmd); err != nil {
			log.Fatal(err)
		}

		cmd = strings.TrimSpace(cmd)

		switch cmd {
		case "A":
			preview(conn, apk)
		case "B":
			preview(conn, bpk)
		case "quit", "exit":
			return
		}

	}

}

func preview(conn *node.Conn, pk cipher.PubKey) {

	var err = conn.Preview(pk,
		func(pack registry.Pack, r *registry.Root) (subscribe bool) {

			var tree, err = r.Tree(pack)

			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(tree)

			return // never subscribe
		})

	if err != nil {
		log.Fatal(err)
	}

}
