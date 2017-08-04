package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"
	sky "github.com/skycoin/cxo/skyobject"

	"github.com/skycoin/cxo/intro"
)

// defaults
const (
	Host        string = "[::]:8000" // default host address of the server
	RPC         string = "[::]:7000" // default RPC address
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

	defer func() {
		if err := recover(); err != nil {
			code = 1
			fmt.Fprintln(os.Stderr, "[PANIC]:", err)
		}
		os.Exit(code) // the Exit "recovers" silently
	}()

	reg := sky.NewRegistry(func(r *sky.Reg) {
		r.Register("intro.Vote", intro.Vote{})
		r.Register("intro.Content", intro.Content{})
	})

	var c = node.NewConfig()

	c.RPCAddress = RPC
	c.Listen = Host
	c.RemoteClose = RemoteClose

	c.DBPath = "./server.db"

	c.Log.Prefix = "[server] "
	c.Log.Debug = true
	c.Log.Pins = log.All // all

	c.Skyobject.Registry = reg

	c.Skyobject.Log.Debug = true
	c.Skyobject.Log.Pins = sky.CleanUpPin | sky.PackSavePin | sky.VerbosePin // all
	c.Skyobject.Log.Prefix = "[server cxo] "

	c.FromFlags()
	flag.Parse()

	// subscribe all incoming connections
	pk, sk := cipher.GenerateDeterministicKeyPair([]byte("x"))

	c.OnCreateConnection = func(s *node.Node, c *gnet.Conn) {
		if c.IsIncoming() {
			go s.Subscribe(c, pk) // don't block
		}
	}

	var s *node.Node
	var err error

	// create and launch
	if s, err = node.NewNode(c); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
		return
	}
	defer s.Close() // close

	s.Subscribe(nil, pk)

	var wg sync.WaitGroup
	stop := make(chan struct{})

	wg.Add(1)
	go fictiveVotes(s, &wg, pk, sk, stop)

	defer wg.Wait()   // wait
	defer close(stop) // stop fictiveVotes call

	waitInterrupt(s.Quiting())
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

	content := new(intro.Content)

	// create Root (and initialize the "content" var)
	pack.Append(content)
	if _, err := pack.Save(); err != nil {
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
		case <-time.After(5 * time.Second):
		}

		// new random votes

		content.Post.Append(&intro.Vote{i%3 != 0, uint32(i)})
		content.Thread.Append(&intro.Vote{i%2 == 0, uint32(i)})

		// replace Content with new one
		if err := pack.SetRefByIndex(0, content); err != nil {
			c.Print("[FATAL] ", err)
			return
		}
		if _, err := pack.Save(); err != nil {
			c.Print("[FATAL] ", err)
			return
		}
		s.Publish(pack.Root()) // publish the Root
	}

}
