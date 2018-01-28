package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/skyobject/registry"

	"github.com/skycoin/cxo/intro" // types
)

// defaults
const (
	Src string = "[::1]:8001" // listening address fo the src
	RPC string = "[::1]:7002" // default RPC address
)

// interest feeds
var (
	// apk is feed the src generates, and the dst receive
	apk, _ = cipher.GenerateDeterministicKeyPair([]byte("A"))
)

// wait for SIGINT and return
func waitInterrupt() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}

func main() {

	var (
		c = node.NewConfig()
		r = new(receiver) // show received objects
	)

	c.RPC = RPC // enable RPC

	c.TCP.Listen = "" // don't listen

	// no discovery
	// not a publuc

	// use DB in memory for the example
	c.Config.InMemoryDB = true

	// change cache parameters for example
	c.Config.CacheMaxAmount = 185
	c.Config.CacheMaxVolume = 30 * 1024
	c.Config.CacheMaxItemSize = 512

	// prefix for logs
	c.Logger.Prefix = "[dst] "

	// uncomment to see all debug logs
	//
	// c.Logger.Pins = ^c.Logger.Pins
	// c.Logger.Debug = true

	//
	// callbacks
	//

	// show full root objects
	c.OnRootFilled = r.showFilledRoot

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

	// connect to the src node

	var conn *node.Conn
	if conn, err = n.TCP().Connect(Src); err != nil {
		log.Fatal(err)
	}

	// subscribe to apk feed of the src

	if err = conn.Subscribe(apk); err != nil {
		log.Fatal(err)
	}

	// As you can see, we are using conn.Subscribe. E.g. a connection
	// means nothing without subscription. The Subscribe methods calls
	// n.Share inside.

	// wait for SIGINT

	waitInterrupt()
}

type receiver struct {
	usr  intro.User
	feed intro.Feed

	lastPost int
}

func (r *receiver) showFilledRoot(n *node.Node, root *registry.Root) {

	// just print the Root objects tree

	var pack, err = n.Container().Pack(root, nil)

	if err != nil {
		log.Fatal(err)
	}

	// the r is Root that can contain:
	//
	//     r.Refs[0] -> intro.User
	//     r.Refs[1] -> intro.Feed
	//

	// (1) we print the User only if it is not the same as the receiver has
	// (2) we print posts of the Feed starting after the lastPost of the
	//     recevier
	// (3) we print info about the Feed if it chagned

	fmt.Println("")       // spacing
	defer fmt.Println("") // spacing

	fmt.Println("Root", root.Short())

	if len(root.Refs) == 0 {
		fmt.Println("  blank")
		return
	}

	// user
	if len(root.Refs) > 0 {

		var usr intro.User
		if err = root.Refs[0].Value(pack, &usr); err != nil {
			log.Fatal(err)
		}

		if usr != r.usr {
			// changed

			r.usr = usr // keep

			fmt.Println("  User")
			fmt.Println("    Name:", usr.Name)
			fmt.Println("    Age: ", usr.Age)

		}

	}

	// feed
	if len(root.Refs) > 1 {

		var feed intro.Feed
		if err = root.Refs[1].Value(pack, &feed); err != nil {
			log.Fatal(err)
		}

		fmt.Println("  Feed")

		if feed.Head != r.feed.Head {
			r.feed.Head = feed.Head
			fmt.Println("    Head:", feed.Head)
		}

		if feed.Info != r.feed.Info {
			r.feed.Info = feed.Info
			fmt.Println("    Info:", feed.Info)
		}

		var ln int
		if ln, err = feed.Posts.Len(pack); err != nil {
			log.Fatal(err)
		}

		if ln <= r.lastPost {
			fmt.Println("    no new posts")
			return
		}

		// print new posts
		for i := r.lastPost; i < ln; i++ {

			var post intro.Post
			if _, err = feed.Posts.ValueByIndex(pack, i, &post); err != nil {
				log.Fatal(err)
			}

			fmt.Println("    ---")
			fmt.Println("   ", post.Head)
			fmt.Println("   ", post.Body)

		}

		r.lastPost = ln

	}

	// So, but the `src` node delete a post, and add new, we skip it.
	// But it's just example

}
