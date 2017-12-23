package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"

	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/cxo/skyobject/registry"

	"github.com/skycoin/cxo/intro" // types
)

// defaults
const (
	Bind string = "[::1]:8002" // listen on
	RPC  string = "[::1]:7002" // default RPC address

	Discovery string = "[::1]:8008" // discovery server
)

// interest feeds
var (
	// apk is feed the ca interest
	apk, _ = cipher.GenerateDeterministicKeyPair([]byte("A"))
	// the bpk is feed the ca generates, the bsk is secret key
	// that used to sign Root objects of the feed, to proof
	// that the Root objects really belongs to the bpk;
	// short words, bpk is feed, bsk is owner of the feed
	bpk, bsk = cipher.GenerateDeterministicKeyPair([]byte("B"))
)

// wait for SIGINT and return
func waitInterrupt() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}

func main() {

	var c = node.NewConfig()

	c.RPC = RPC // enable RPC

	c.TCP.Listen = Bind // listen
	c.TCP.Discovery = node.Addresses{Discovery}

	c.Public = true // public server

	// use DB in memory for the example
	c.Config.InMemoryDB = true

	c.Config.CacheMaxAmount = 185
	c.Config.CacheMaxVolume = 30 * 1024
	c.Config.CacheMaxItemSize = 512

	// prefix for logs
	c.Logger.Prefix = "[cb] "

	// uncomment to see all debug logs
	//
	// c.Logger.Pins = ^c.Logger.Pins
	// c.Logger.Debug = true

	//
	// callbacks
	//

	// show full root objects
	c.OnRootFilled = showFilledRoot

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

	//
	// add feeds
	//

	if err = n.Share(apk); err != nil {
		log.Fatal(err)
	}

	if err = n.Share(bpk); err != nil {
		log.Fatal(err)
	}

	// the Share method adds feed to underlying Container;
	// it's possible to have a feed, but don't share it

	//
	// generate the B-feed
	//

	// sync
	var (
		wg     sync.WaitGroup        // wait the generate goroutine
		closed = make(chan struct{}) // closed by SIGINT
	)

	wg.Add(1)
	defer wg.Wait()

	go generate(&wg, closed, n)

	// wait for SIGINT
	waitInterrupt()
	close(closed)
}

func generate(wg *sync.WaitGroup, closed <-chan struct{}, n *node.Node) {
	defer wg.Done()

	var (
		usr = intro.User{
			Name: "Eva",
			Age:  19,
		}

		feed = intro.Feed{
			Head: "Eva's feed",
			Info: "Feed of very useful information about Eva's life",
		}
	)

	// Root object
	var r = new(registry.Root)

	r.Pub = bpk                                 // feed of the Root
	r.Nonce = rand.Uint64()                     // head of the feed
	r.Descriptor = []byte("through, version=1") // any data or nothing

	//
	// let's create and publish the first Root
	//

	var c = n.Container()

	// secret key and registry
	var up, err = c.Unpack(bsk, intro.Registry)

	if err != nil {
		log.Fatal(err)
	}

	// the up (*skyobject.Unpack) implements registry.Pack interface
	// and can be used to create new objects

	// Root -> []Dynamic{ User, feed }

	r.Refs = []registry.Dynamic{
		dynamic(up, "intro.User", &usr),
		dynamic(up, "intro.Feed", &feed),
	}

	// let's save the "blank" feed

	if err = c.Save(up, r); err != nil {
		log.Fatal(err)
	}

	// and publish it
	n.Publish(r)

	//
	// now, let's add posts one by one
	//

	var tk = time.NewTicker(5 * time.Second)

	for i := 0; true; i++ {
		select {
		case <-closed:
			return
		case <-tk.C:
		}

		err = feed.Posts.AppendValues(up, intro.Post{
			Head: fmt.Sprintf("Eva's post #%d", i),
			Body: fmt.Sprintf("nothing happens #%d", i),
		})

		if err != nil {
			log.Fatal(err)
		}

		// the feed has been changed
		if err = r.Refs[1].SetValue(up, &feed); err != nil {
			log.Fatal(err)
		}

		if err = c.Save(up, r); err != nil {
			log.Fatal(err)
		}

		n.Publish(r)

	}

}

// create Dynamic reference
func dynamic(
	up *skyobject.Unpack,
	schemaName string,
	obj interface{},
) (
	dr registry.Dynamic,
) {

	// so, it's possible to use Registry.Types() to get schema name
	// but for received registrues this is not an options; and we
	// are using schema name; also, it's possible to use
	// schema reference; but we creating the Dynamic references once
	// and who cares what method is better

	var sch, err = up.Registry().SchemaByName(schemaName)

	if err != nil {
		log.Fatal(err)
	}

	dr.Schema = sch.Reference() // schema reference

	// the SetValue method is usability trick; the method is equal to
	//
	//     var (
	//         val = encoder.Serialize(obj) // cipher/encoder
	//         key = cipher.SumSHA256(val)
	//     )
	//
	//     if err = up.Set(key, val); err != nil {
	//         log.Fatal(err)
	//     }
	//
	//     dr.Hash = key
	//
	// Short words it: (1) serializes given object, (2) calculate SHA256
	// of the serialized value, (3) saves the object (4) set the hash
	// to dr.Hash field
	//
	// Thus, dr.Schema field is not changed after the SetValue and end-user
	// have to care about it.

	if err = dr.SetValue(up, obj); err != nil {
		log.Fatal(err)
	}

	return
}

func showFilledRoot(n *node.Node, r *registry.Root) {

	// just print the Root objects tree

	var pack, err = n.Container().Pack(r, nil)

	if err != nil {
		log.Fatal(err)
	}

	var tree string
	if tree, err = r.Tree(pack); err != nil {
		log.Fatal(err)
	}

	fmt.Println("") // spacing
	fmt.Println(tree)
	fmt.Println("") // spacing

}
