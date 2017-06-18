package node_test

import (
	"fmt"
	"log"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/skyobject"
)

//
// Linked list: use skyobject.Reference instead of pointer. So,
// we can't use double-linked-list becasue of back reference
//

// real linked list
type any struct {
	Data string
	Next *any
}

var linkedList *any // create the linked-list

// CX linked list
type Any struct {
	Data string
	Next skyobject.Reference `skyobject:"schema=Any"`
}

func init() {
	log.SetFlags(log.Lshortfile | log.Ltime)
}

func fillLinkedList() {
	linkedList = &any{
		"one",
		&any{
			"two",
			&any{
				"three",
				nil,
			},
		},
	}
}

func Example_linkedList() {

	// fill linked list
	fillLinkedList()

	// feed and owner

	feed, owner := cipher.GenerateKeyPair()

	// launch src->pipe->dst

	pipe, err := PipeNodeLinkedList(feed)
	if err != nil {
		log.Print(err)
		return
	}
	defer pipe.Close()

	src, err := SourceNodeLinkedList(feed, owner, pipe.Pool().Address())
	if err != nil {
		log.Print(err)
		return
	}
	defer src.Close()

	dst, err := DestinationNodeLinkedList(feed, pipe.Pool().Address())
	if err != nil {
		log.Print(err)
		return
	}
	defer dst.Close()

	select {
	case <-src.Quiting():
	case <-pipe.Quiting():
	case <-dst.Quiting():
	}

}

func PipeNodeLinkedList(feed cipher.PubKey) (pipe *node.Node, err error) {
	conf := node.NewConfig()
	conf.Listen = "127.0.0.1:0" // arbitrary assignment (for example)
	conf.InMemoryDB = true      // use in-memory database (for example)
	conf.EnableRPC = false      // disable RPC (for example)
	conf.Log.Prefix = "[PIPE] "
	// conf.Log.Debug = true

	// node
	if pipe, err = node.NewNode(conf); err != nil {
		return
	}

	// feed
	pipe.Subscribe(nil, feed)

	return
}

func SourceNodeLinkedList(feed cipher.PubKey, owner cipher.SecKey,
	address string) (src *node.Node, err error) {

	// create registry and register all types we are going to use
	reg := skyobject.NewRegistry()
	reg.Register("Any", Any{})

	// config

	conf := node.NewConfig()
	conf.EnableListener = false // don't listen (for example)
	conf.InMemoryDB = true      // use in-memory database (for example)
	conf.EnableRPC = false      // disable RPC (for example)
	conf.Log.Prefix = "[SRC] "
	// conf.Log.Debug = true

	// node
	if src, err = node.NewNodeReg(conf, reg); err != nil {
		return
	}

	// dial to pipe
	var pc *gnet.Conn
	pc, err = src.Pool().Dial(address)
	if err != nil {
		src.Close()
		log.Print(err)
		return
	}

	// subscribe to pipe+feed and make the pipe
	// subscribed to the feed of the src
	if err = src.SubscribeResponse(pc, feed); err != nil {
		src.Close()
		log.Print(err)
		return
	}

	// don't block

	go generateLinkedList(src, feed, owner)

	return

}

func generateLinkedList(src *node.Node, feed cipher.PubKey,
	owner cipher.SecKey) {

	defer src.Close()

	// for this example (never need in real case)
	time.Sleep(1 * time.Second)

	// container will push all updates to subscribed peers (to the Pipe)
	cnt := src.Container()

	// root object that refers to our registry and
	// we will attach the linked list to the root
	root, err := cnt.NewRoot(feed, owner)
	if err != nil {
		log.Print(err)
		return
	}

	shareLinkedList(cnt, root)

	// for this example (waiting for replication)
	time.Sleep(1 * time.Second)
}

func shareLinkedList(cnt *node.Container, root *node.Root) {

	// actually, GC disabled by default and we don't need the LockGC / UnlockGC
	cnt.LockGC()
	defer cnt.UnlockGC()

	// find a tail of the list
	var chain []*any

	for elem := linkedList; elem != nil; elem = elem.Next {
		chain = append(chain, elem)
	}

	// save elements of the list from tail
	var a *any
	var prev skyobject.Reference
	for i := len(chain) - 1; i >= 0; i-- {
		a = chain[i]
		prev = root.Save(Any{
			Data: a.Data,
			Next: prev,
		})
	}

	// now the prev contains first element of the list;
	// let's attach it to root

	// we can attach to Root Dynamic reference only

	sch, err := root.SchemaReferenceByName("Any")
	if err != nil {
		log.Print(err)
		return
	}

	first := skyobject.Dynamic{
		Schema: sch,  // schema reference
		Object: prev, // object reference
	}

	if _, err := root.Append(first); err != nil {
		log.Print(err)
	}

	// this way, the root points to registry and first element of te list,
	// first element knows what shcema of next element (skyobject struct tag)
	// and the first element points to next, and so on

}

func DestinationNodeLinkedList(feed cipher.PubKey,
	address string) (dst *node.Node, err error) {

	// config

	conf := node.NewConfig()
	conf.EnableListener = false // disable listener for this example
	conf.InMemoryDB = true      // use database in memory
	conf.EnableRPC = false
	conf.Log.Prefix = "[DST] "
	// conf.Log.Debug = true

	// while a root object and all related objects received
	conf.OnRootFilled = func(root *node.Root) {
		// don't block messages handling;
		// it's not nessesary for this example, but
		// if you want to perform a long running
		// task in this callback, then you need
		// to keep in mind that this callback
		// blocks goroutine that handles incoming
		// messages from this connection
		go printTreeLinkedList(root)
	}

	// node

	dst, err = node.NewNode(conf)
	if err != nil {
		return
	}

	var src *gnet.Conn

	if src, err = dst.Pool().Dial(address); err != nil {
		log.Print(err)
		dst.Close()
		return
	}

	if err = dst.SubscribeResponse(src, feed); err != nil {
		log.Print(err)
		dst.Close()
		return
	}

	return

}

func printTreeLinkedList(root *node.Root) {

	fmt.Println("----")
	defer fmt.Println("----")

	fmt.Println(root.Inspect())
}

/*

Output:
--------------------------------------------------------------------------------
[PIPE] 00:43:37 node.go:284: listen on 127.0.0.1:37205
----
(root) 41d87baf46f73ba78c5a857f6a2696e82eb8602a8e6a715091af67913f8cd1d3
└── *(dynamic) {e936b0757713505f4a29fa6e33d5d7c474eef2c2f334b81619c64dd9b96af614, 4c87cc858135b5735934137b43b05921a36e85eb1ec1832d4785d9cd9bb0e461}
    └── struct Any
        ├── (field) Data
        │   └── one
        └── (field) Next
            └── *(static) 2c9f6080e1112d909ca3e4c5e08e22baf91933b41d22eec58d68e9ee77dc973a
                └── struct Any
                    ├── (field) Data
                    │   └── two
                    └── (field) Next
                        └── *(static) 57f57aac0390f14afc4287ca8f10abb6296b542d50fd8698012cd839a5f76a87
                            └── struct Any
                                ├── (field) Data
                                │   └── three
                                └── (field) Next
                                    └── *(static) 0000000000000000000000000000000000000000000000000000000000000000
                                        └── nil

----
[PIPE] 00:43:39 conn.go:474: [ERR] 127.0.0.1:48326 reading error: EOF
[PIPE] 00:43:39 conn.go:474: [ERR] 127.0.0.1:48328 reading error: EOF
--------------------------------------------------------------------------------

*/
