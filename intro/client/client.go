package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/node/gnet"
	sky "github.com/skycoin/cxo/skyobject"

	"github.com/skycoin/cxo/intro"
)

// defaults
const (
	Host        string = "[::]:8002" // default host address of the server
	RPC         string = "[::]:7002" // default RPC address
	RemoteClose bool   = false       // don't allow closing by RPC by default

	PipeAddress string = "[::]:8001"
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

	c.DBPath = "./client.db"

	c.Log.Debug = true
	c.Log.Pins = ^c.Log.Pins // all
	c.Log.Prefix = "[client] "

	c.Skyobject.Registry = reg // <-- registry

	c.Skyobject.Log.Debug = true
	c.Skyobject.Log.Pins = ^c.Skyobject.Log.Pins // all
	c.Skyobject.Log.Prefix = "[client cxo]"

	c.FromFlags()
	flag.Parse()

	// show full root objects
	c.OnRootFilled = func(s *node.Node, c *gnet.Conn, r *sky.Root) {
		fmt.Println("\n\n\n") // space
		fmt.Println("----")
		s.Container().Inspect(r)
		fmt.Println("----")
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

	pk, _ := cipher.GenerateDeterministicKeyPair([]byte("x"))
	s.Subscribe(nil, pk)
	// conenct to pipe

	conn, err := s.Pool().Dial(PipeAddress)
	if err != nil {
		s.Println("[ERR] can't conenct to server:", err)
		return
	}
	s.Subscribe(conn, pk) // subscribe to conn

	waitInterrupt(s.Quiting())

}
