package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"
	// sky "github.com/skycoin/cxo/skyobject"
)

// defaults
const (
	Host        string = "[::]:8001" // default host address of the server
	RPC         string = "[::]:7001" // default RPC address
	RemoteClose bool   = false       // don't allow closing by RPC by default

	ServerAddress string = "[::]:8000"
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

	c.DBPath = "./pipe.db"

	// node logger
	c.Log.Prefix = "[pipe] "
	c.Log.Debug = true
	c.Log.Pins = log.All // all

	// no registry

	// skyobject logger
	c.Skyobject.Log.Prefix = "[pipe cxo]"
	c.Skyobject.Log.Debug = true
	c.Skyobject.Log.Pins = log.All // all

	c.FromFlags()
	flag.Parse()

	// subscribe all incoming connections
	pk, _ := cipher.GenerateDeterministicKeyPair([]byte("x"))

	c.OnCreateConnection = func(s *node.Node, c *gnet.Conn) {
		if c.IsIncoming() {
			go s.Subscribe(c, pk) // don't block
		}
	}

	var s *node.Node
	var err error

	// create and launch
	if s, err = node.NewNode(c); err != nil {
		fmt.Fprintln(os.Stderr, "[FATAL]:", err)
		code = 1
		return
	}
	defer s.Close() // close

	s.Subscribe(nil, pk)

	// conenct to server

	conn, err := s.Pool().Dial(ServerAddress)
	if err != nil {
		s.Println("[ERR] can't conenct to server:", err)
		return
	}
	s.Subscribe(conn, pk) // subscribe to conn

	waitInterrupt(s.Quiting())

}
