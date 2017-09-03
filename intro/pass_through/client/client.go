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
	sky "github.com/skycoin/cxo/skyobject"

	passThrough "github.com/skycoin/cxo/intro/pass_through"
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
		r.Register("pt.Vote", passThrough.Vote{})
		r.Register("pt.Content", passThrough.Content{})
	})

	var c = node.NewConfig()

	c.RPCAddress = RPC
	c.Listen = Host
	c.RemoteClose = RemoteClose

	c.PingInterval = 0 // suppress ping logs

	c.DataDir = ""      // don't create ~/.skycoin/cxo
	c.InMemoryDB = true // use DB in memory

	// suppress gnet logs
	c.Config.Logger = log.NewLogger(log.Config{Output: ioutil.Discard})

	//c.Log.Debug = true
	//c.Log.Pins = log.All // all
	c.Log.Prefix = "[client] "

	c.Skyobject.Registry = reg // <-- registry

	//c.Skyobject.Log.Debug = true
	//c.Skyobject.Log.Pins = log.All // all
	c.Skyobject.Log.Prefix = "[client cxo]"

	c.FromFlags()
	flag.Parse()

	// show full root objects
	c.OnRootFilled = func(c *node.Conn, r *sky.Root) {
		fmt.Println("\n\n\n") // space
		fmt.Println("----")
		fmt.Println(c.Node().Container().Inspect(r))
		fmt.Println("----")
	}

	pk, _ := cipher.GenerateDeterministicKeyPair([]byte("x"))

	c.OnCreateConnection = func(c *node.Conn) {
		fmt.Println("OnCreateConnection {", c.Address())
		if err := c.Subscribe(pk); err != nil {
			fmt.Println("[ERR] subscribing:", err)
			c.Close()
		}
		fmt.Println("OnCreateConnection }", c.Address())
	}

	c.OnCloseConnection = func(c *node.Conn) {
		fmt.Println("OnCloseConnection", c.Address())
	}

	c.OnRootReceived = func(c *node.Conn, r *sky.Root) {
		fmt.Println("OnRootReceived", c.Address(), r.Short())

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

	if err = s.AddFeed(pk); err != nil {
		s.Println("[ERR] database failure:", err)
		return
	}
	// conenct to pipe

	if err := s.Connect(PipeAddress); err != nil {
		s.Println("[ERR] can't conenct to server:", err)
		return
	}

	waitInterrupt(s.Quiting())
}
