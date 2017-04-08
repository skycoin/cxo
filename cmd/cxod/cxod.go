package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/rpc/server"
	"github.com/skycoin/cxo/skyobject"
)

const (
	// default listening address
	ADDRESS = "127.0.0.1"
	PORT    = 9987
)

// Fill down known hosts
var knownHosts = map[cipher.SHA256][]string{}

func getConfigs() (nc node.Config, rc server.Config) {
	var lc log.Config
	// get defaults
	nc = node.NewConfig()
	rc = server.NewConfig()
	//
	flag.StringVar(&nc.Listen,
		"address",
		ADDRESS,
		"Address to listen on. Set to empty string for arbitrary assignment")

	nc.Config.FromFlags() // gnet.Config

	lc.FromFlags() // logger configs

	flag.BoolVar(&nc.RemoteClose,
		"remote-close",
		nc.RemoteClose,
		"allow close the node using RPC client")

	flag.BoolVar(&rc.Enable,
		"rpc",
		rc.Enable,
		"use rpc")
	flag.IntVar(&nc.RPCEvents,
		"rpc-events",
		nc.RPCEvents,
		"rpc events chan size")
	flag.StringVar(&rc.Address,
		"rpc-address",
		rc.Address,
		"address for rpc")
	flag.IntVar(&rc.Max,
		"rpc-max",
		rc.Max,
		"maximum rpc-connections")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <flags> [known hosts]\n", os.Args[0])
		flag.PrintDefaults()
	}

	var help bool
	flag.BoolVar(&help, "h", false, "show help")

	flag.Parse()

	if help {
		fmt.Printf("Usage: %s <flags> [subscribe to]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}

	for _, sb := range flag.Args() { // subscribe
		pk, err := cipher.PubKeyFromHex(sb)
		if err != nil {
			fmt.Fprintf(os.Stderr, "malformed public key to subscribe to:\n"+
				"  %q\n"+
				"  %v\n", sb, err)
			os.Exit(1)
		}
		nc.Subscribe = append(nc.Subscribe, pk)
	}

	nc.Logger = log.NewLogger("["+lc.Prefix+"] ", lc.Debug)
	nc.Config.Logger = log.NewLogger("[p:"+lc.Prefix+"] ", lc.Debug)

	return
}

func waitInterrupt(quit <-chan struct{}) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	select {
	case <-sig:
	case <-quit:
	}
}

func main() {

	// exit code
	var code int
	defer func() { os.Exit(code) }()

	var err error

	var (
		db  *data.DB
		so  *skyobject.Container
		n   *node.Node
		rpc *server.Server

		nc node.Config
		rc server.Config
	)

	db = data.NewDB()

	so = skyobject.NewContainer(db)

	//
	// Get configurations from flags
	//

	nc, rc = getConfigs() // node config, rpc config

	//
	// Node
	//

	n = node.NewNode(nc, db, so)

	// can panic
	n.Start()
	defer n.Close()

	//
	// RPC
	//

	if rc.Enable {
		rpc = server.NewServer(rc, n) // , so)
		if err = rpc.Start(); err != nil {
			fmt.Fprintln(os.Stderr, "error starting RPC:", err)
			code = 1
			return
		}
		defer rpc.Close()
	}

	// waiting for SIGING or termination using RPC

	waitInterrupt(n.Quiting())

}
