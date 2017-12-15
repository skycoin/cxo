package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/node/log"

	cxo "github.com/skycoin/cxo/skyobject"

	"github.com/skycoin/cxo/intro/exchange"
)

// defaults
const (
	Host        string = "127.0.0.1:8001" // default host address of the node
	RPC         string = "127.0.0.1:7001" // default RPC address
	RemoteClose bool   = false            // don't allow closing by RPC

	Discovery string = "127.0.0.1:8008" // discovery server
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

	// register types to use
	reg := cxo.NewRegistry(func(r *cxo.Reg) {
		r.Register("exchg.Vote", exchange.Vote{})
		r.Register("exchg.Content", exchange.Content{})
	})

	var c = node.NewConfig()

	c.RPCAddress = RPC
	c.Listen = Host
	c.RemoteClose = RemoteClose

	c.DiscoveryAddresses = node.Addresses{Discovery}

	c.PingInterval = 0 // suppress ping logs

	c.DataDir = ""      // don't create ~/.skycoin/cxo
	c.InMemoryDB = true // use DB in memeory

	// node logger
	c.Log.Prefix = "[ca] "
	c.Log.Pins = log.All //
	c.Log.Debug = true   //

	// suppress gnet logger
	c.Config.Logger = log.NewLogger(log.Config{Output: ioutil.Discard})

	c.Skyobject.Registry = reg // <-- registry

	// skyobject logger
	c.Skyobject.Log.Prefix = "[ca cxo]"

	// show full root objects
	c.OnRootFilled = func(c *node.Conn, r *cxo.Root) {
		fmt.Print("\n\n") // space
		fmt.Println("----")
		fmt.Println(r.Pub.Hex())
		fmt.Println(c.Node().Container().Inspect(r))
		fmt.Println("----")
	}

	// obtain configs from flags
	c.FromFlags()
	flag.Parse()

	// apk and bk is A-feed and B-feed, the ask is
	// secret key required for creating
	apk, ask := cipher.GenerateDeterministicKeyPair([]byte("A"))
	bpk, _ := cipher.GenerateDeterministicKeyPair([]byte("B"))

	var s *node.Node
	var err error

	// create and launch
	if s, err = node.NewNode(c); err != nil {
		fmt.Fprintln(os.Stderr, "[FATAL]:", err)
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

	var wg sync.WaitGroup
	stop := make(chan struct{})

	wg.Add(1)
	go fictiveVotes(s, &wg, apk, ask, stop)

	defer wg.Wait()   // wait
	defer close(stop) // stop fictiveVotes call

	waitInterrupt(s.Closed())
}

func fictiveVotes(s *node.Node, wg *sync.WaitGroup, pk cipher.PubKey,
	sk cipher.SecKey, stop chan struct{}) {

	defer wg.Done()

	s.Debug(log.All, "fictiveVotes")

	c := s.Container()

	pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
	if err != nil {
		c.Print("[FATAL] ", err)
		return
	}
	defer pack.Close()

	content := new(exchange.Content)

	// create Root (and initialize the "content" var)
	pack.Append(content)
	if err := pack.Save(); err != nil {
		c.Print("[FATAL] ", err)
		return
	}
	s.Publish(pack.Root()) // publish the Root

	// update the Root time to time

	for i := 0; true; i++ {
		select {
		case <-stop:
			c.Print("[STOP]")
			return
		case <-s.Quiting():
			c.Print("[STOP]")
			return
		case <-time.After(5 * time.Second):
		}

		// new random votes

		content.Post.Append(&exchange.Vote{
			Up:    i%3 != 0,
			Index: uint32(i),
		})
		content.Thread.Append(&exchange.Vote{
			Up:    i%2 == 0,
			Index: uint32(i),
		})

		// replace Content with new one
		if err := pack.SetRefByIndex(0, content); err != nil {
			c.Print("[FATAL] ", err)
			return
		}
		if err := pack.Save(); err != nil {
			c.Print("[FATAL] ", err)
			return
		}
		s.Publish(pack.Root()) // publish the Root
	}

}
