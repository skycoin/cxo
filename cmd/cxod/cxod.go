package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/skycoin/src/mesh/app"
	"github.com/skycoin/skycoin/src/mesh/messages"
	"github.com/skycoin/skycoin/src/mesh/nodemanager"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/skyobject"
)

type Config struct {
	Host string
}

func (c *Config) fromFlags() {
	flag.StringVar(&c.Host,
		"h",
		"[::]",
		"server host")
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

	var (
		db *data.DB             = data.NewDB()
		so *skyobject.Container = skyobject.NewContainer(db)
		nd *node.Node           = node.NewNode(db, so)
		c  Config
	)

	c.fromFlags()
	flag.Parse()

	meshnet := nodemanager.NewNetwork()
	defer nodemanager.Shutdown()

	address := meshnet.AddNewNode(c.Host)

	fmt.Println("host:           ", c.Host)
	fmt.Println("server address: ", address.Hex())

	_, err := app.NewServer(meshnet, address, Handler(db, so, nd))
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("started")

	// rpc

	// waiting for SIGINT or termination using RPC

	waitInterrupt(n.Quiting())

}

func Handler(db *data.DB, so *skyobject.Container,
	nd *node.Node) func([]byte) []byte {

	want := make(skyobject.Set)

	return func(in []byte) (out []byte) {
		msg, err := node.Decode(in)
		if err != nil {
			fmt.Println("error decoding message:", err)
			return
		}
		switch m := msg.(type) {
		case *node.SyncMsg:
			// add feeds of remote node to internal list
		case *node.RootMsg:
			for _, f := range nd.Feeds() {
				if f == m.Feed {
					ok, err := so.AddEncodedRoot(m.Root, m.Feed, m.Sig)
					if err != nil {
						fmt.Println("error adding root object: ", err)
						// TODO: close connection
						return
					}
					if !ok {
						return // older then existsing one
					}
					// todo: send updates to related connections except this one
					return
				}
			}
		case *node.RequestMsg:
			if data, ok := db.Get(m.Hash); ok {
				return node.Encode(&node.DataMsg{data})
			}
		case *node.DataMsg:
			hash := cipher.SumSHA256(m.Data)
			if _, ok := want[skyobject.Reference(hash)]; ok {
				db.Set(hash, m.Data)
				delete(want, hash)
			}
		default:
			fmt.Printf("unexpected message type: %T\n", msg)
		}
	}
}
