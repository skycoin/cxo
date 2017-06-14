package node

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/skyobject"
)

func getConfig() (c NodeConfig) {
	c = NewNodeConfig()
	c.InMemoryDB = true // use datbase in memory for examples
	return
}

func ExampleNewNode() {

	node, err := NewNode(getConfig())
	if err != nil {
		// hanlde error
	}
	defer node.Close()

	if err := node.Start(); err != nil {
		// handle error
		return // don't call Fatal, panic, etc, because of BoltDB
	}

	// stuff here
}

func ExampleNewNodeReg() {

	type User struct {
		Name string
		Age  uint32
	}

	type Group struct {
		Name  string
		Users skyobject.References `skyobject:"schema=example.User"`
	}

	reg := skyobject.NewRegistry()

	reg.Register("example.User", User{})
	reg.Register("example.Group", Group{})

	node, err := NewNodeReg(getConfig(), reg)
	if err != nil {
		// hanlde error
	}
	defer node.Close()

	if err := node.Start(); err != nil {
		// handle error
		return // don't call Fatal, panic, etc, because of BoltDB
	}

	// stuff here
}

func ExampleNode_Pool() {

	conf := getConfig()
	conf.EnableListener = false // disable listening for this example

	node, err := NewNode(conf)
	if err != nil {
		// handle error
	}
	defer node.Close()

	if err := node.Start(); err != nil {
		// handle error
		return
	}

	pool := node.Pool()

	// dialing

	if conn, err := pool.Dial("127.0.0.1:8998"); err != nil {
		// handle error
	} else {
		_ = conn // work with conection
	}

	// call listen manually

	if err = pool.Listen("127.0.0.1:0"); err != nil {
		// handle error
	}

	// disconnect

	if c := pool.Connection("127.0.0.1:8998"); c != nil {
		c.Close()
	}

	// get all conenctions

	for _, c := range pool.Connections() {
		if c.IsIncoming() {
			// stuff here
		} else {
			// outgoing connection
		}
	}

}

var X, Y cipher.PubKey

func ExampleNode_Subscribe() {

	conf := getConfig()
	conf.OnSubscriptionAccepted = func(c *gnet.Conn, feed cipher.PubKey) {
		fmt.Printf("subscribed to %s of %s\n", feed.Hex(), c.Address())
	}
	conf.OnSubscriptionRejected = func(c *gnet.Conn, feed cipher.PubKey) {
		fmt.Printf("remote node %s reject subscription to %s",
			c.Address(),
			feed.Hex())
	}

	node, err := NewNode(conf)
	if err != nil {
		// handle error
	}
	defer node.Close()

	if err := node.Start(); err != nil {
		// handle error
		return
	}

	// make the node subscribed to feed Y

	node.Subscribe(nil, Y)

	// connect to a remote node

	remote, err := node.Pool().Dial("127.0.0.1:8998")
	if err != nil {
		// handle error
		return
	}

	// we know that the remote node shares feed X
	// that we want to subscribe to

	node.Subscribe(remote, X)

	// other stuff

}

func ExampleNode_Unsubscribe() {
	// Unsubscribe(c *gnet.Conn, feed cipher.PubKey)

	conf := getConfig()

	node, err := NewNode(conf)
	if err != nil {
		// handle error
	}
	defer node.Close()

	if err := node.Start(); err != nil {
		// handle error
		return
	}

	// make the node subscribed to feed Y

	node.Subscribe(nil, Y)

	// connect to a remote node and susbcribe to X feed of the node

	remote, err := node.Pool().Dial("127.0.0.1:8998")
	if err != nil {
		// handle error
		return
	}
	node.Subscribe(remote, X)

	// stop receive X from the remote

	node.Unsubscribe(remote, X)

	// stop sharing Y

	node.Unsubscribe(nil, Y)

	// other stuff
}

// TODO low priority

// func ExampleNode_Want() {
// 	// Want(feed cipher.PubKey) (wn []cipher.SHA256)
//
// 	//
// }

// TODO low priority

// func ExampleNode_Got() {
// 	// Want(feed cipher.PubKey) (wn []cipher.SHA256)
//
// 	//
// }

func ExampleNode_Feeds() {

	node, err := NewNode(getConfig())
	if err != nil {
		// handle error
	}
	defer node.Close()

	if err := node.Start(); err != nil {
		// handle error
		return
	}

	// chech out feed we already subscribed to

	if len(node.Feeds()) != 0 {
		// stuff
	}

	node.Subscribe(nil, X)
	node.Subscribe(nil, Y)

	fmt.Println("share:")

	for _, feed := range node.Feeds() {
		fmt.Println(" -", feed.Hex())
	}

	// other stuff

}

func ExampleNode_Quiting() {

	// we  can't call log.Fatal, os.Exit, panic
	// because of datbase. To be safe, we need
	// to close Node

	conf := getConfig()
	conf.RemoteClose = true // allow closing by RPC

	node, err := NewNode(conf)
	if err != nil {
		// handle error
	}
	defer node.Close()

	if err := node.Start(); err != nil {
		// handle error
		return
	}

	go func() {
		defer node.Close()

		// work with node in separate goroutine

		return
	}()

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

	select {
	case <-sig:
		// got SIGINT
	case <-node.Quiting():
		// node closed by a goroutine or using RPC
	}

	// exiting....

}

func ExampleNode_SubscribeResponse() {

	node, err := NewNode(getConfig())
	if err != nil {
		// handle error
	}
	defer node.Close()

	if err := node.Start(); err != nil {
		// handle error
		return
	}

	// connect to remote peer

	peer, err := node.Pool().Dial("127.0.0.1:8998")
	if err != nil {
		// handle error
		return
	}

	// blocking call
	if err := node.SubscribeResponse(peer, X); err != nil {
		// handle error
		return
	}

	// now, we will send/receive updates from/to X feed of the peer

}

func ExampleNode_ListOfFeedsResponse() {

	node, err := NewNode(getConfig())
	if err != nil {
		// handle error
	}
	defer node.Close()

	if err := node.Start(); err != nil {
		// handle error
		return
	}

	// connect to remote public peer

	public, err := node.Pool().Dial("127.0.0.1:8998")
	if err != nil {
		// handle error
		return
	}

	// blocking call
	list, err := node.ListOfFeedsResponse(public)
	if err != nil {
		if err == ErrNonPublicPeer {
			// the peer is not public
			return
		}
		// timeout error
		return
	}

	fmt.Printf("feeds of %s:", public.Address())

	for _, feed := range list {
		fmt.Println(" -", feed.Hex())
	}

}
