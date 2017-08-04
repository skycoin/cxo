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

// tpyes

type User struct {
	Name string
	Age  uint32
}

type Group struct {
	Name   string
	Leader skyobject.Ref  `skyobject:"schema=cxo.User"`
	Users  skyobject.Refs `skyobject:"schema=cxo.User"`
}

func Example_sourceDestination() {

	// feed and owner

	feed, owner := cipher.GenerateKeyPair()

	// launch

	src, err := SourceNodeSrcDst(feed, owner)
	if err != nil {
		log.Print(err)
		return
	}
	defer src.Close()

	dst, err := DestinationNodeSrcDst(feed, src.Pool().Address())
	if err != nil {
		log.Print(err)
		return
	}
	defer dst.Close()

	select {
	case <-src.Quiting():
	case <-dst.Quiting():
	}
}

func SourceNodeSrcDst(feed cipher.PubKey, owner cipher.SecKey) (src *node.Node,
	err error) {

	// registry
	reg := skyobject.NewRegistry(func(r *skyobject.Reg) {
		r.Register("cxo.User", User{})
		r.Register("cxo.Group", Group{})
	})
	// config
	conf := node.NewConfig()
	conf.Listen = "127.0.0.1:0" // arbitrary assignment
	conf.InMemoryDB = true      // use in-memory database
	conf.EnableRPC = false      // disable RPC
	conf.Log.Prefix = "[SRC] "
	conf.Skyobject = skyobject.NewConfig()
	conf.Skyobject.Registry = reg // <--- CoreRegistry
	// conf.Log.Debug = true
	// node
	if src, err = node.NewNode(conf); err != nil {
		return
	}
	// feed
	src.Subscribe(nil, feed)
	// don't block
	go generateSrcDst(src, feed, owner)
	return
}

func generateSrcDst(src *node.Node, feed cipher.PubKey, owner cipher.SecKey) {
	defer src.Close()
	// for this example (never need in real case)
	time.Sleep(1 * time.Second)
	// container
	cnt := src.Container()
	// generate groups
	pack, err := cnt.NewRoot(feed, owner, 0, cnt.CoreRegistry().Types())
	if err != nil {
		log.Print(err)
		return
	}
	for i := 0; i < 2; i++ {
		generateGroup(i, src, pack)
		// for this example
		time.Sleep(1 * time.Second)
	}
}

func generateGroup(i int, src *node.Node, pack *skyobject.Pack) {
	pack.Append(&Group{
		Name: fmt.Sprint("Group #", i),
		Leader: pack.Ref(&User{
			Name: "Elisabet Bathory",
			Age:  30,
		}),
		Users: pack.Refs(
			&User{fmt.Sprintf("Alice #%d", i), 19 + uint32(i)},
			&User{fmt.Sprintf("Eva #%d", i), 21 + uint32(i)},
		),
	})
	if _, err := pack.Save(); err != nil {
		log.Println(err)
		return
	}
	src.Publish(pack.Root()) // share
}

func DestinationNodeSrcDst(feed cipher.PubKey, address string) (dst *node.Node,
	err error) {

	// you can create the same regsitry here if you want to
	// extract received obejcts to golang values

	// config
	conf := node.NewConfig()
	conf.EnableListener = false // disable listener for this example
	conf.InMemoryDB = true      // use database in memory
	conf.EnableRPC = false
	conf.Log.Prefix = "[DST] "
	// conf.Log.Debug = true

	// while a root object and all related objects received
	conf.OnRootFilled = func(n *node.Node, _ *gnet.Conn, root *skyobject.Root) {
		// don't block messages handling;
		// it's not nessesary for this example, but
		// if you want to perform a long running
		// task in this callback, then you need
		// to keep in mind that this callback
		// blocks goroutine that handles incoming
		// messages from this connection
		go printTreeSrcDst(n.Container(), root)
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

func printTreeSrcDst(cnt *skyobject.Container, root *skyobject.Root) {
	fmt.Println("----")
	defer fmt.Println("----")

	fmt.Println(cnt.Inspect(root))
}

/*

Output:
--------------------------------------------------------------------------------
[SRC] 00:46:08 node.go:284: listen on 127.0.0.1:43579
----
(root) 4b3995a446da275db1f2e20d15454e3636eb936a6dcbca78de985cad70e4cb1b
└── *(dynamic) {a1cab7ccf5bed7c4fcf21fb6bdcfb304f042264a3e45946c812ce84d587ca02b, 137f4a9c68b1c5c21458b962187d66e9948fe8a658d87fbdec2c692aff0c0c89}
    └── struct cxo.Group
        ├── (field) Name
        │   └── Group #0
        ├── (field) Leader
        │   └── *(static) f0e08559a53eff8d33061396d0515416beccc234dc6496275b7a1712e7a5d865
        │       └── struct cxo.User
        │           ├── (field) Name
        │           │   └── Elisabet Bathory
        │           └── (field) Age
        │               └── 30
        └── (field) Users
            └── []cxo.User
                ├── *(static) 39c2abd5512a56d8d3ff64430a4268356b964e6377a82ab3e8e0e2b7c8d9b926
                │   └── struct cxo.User
                │       ├── (field) Name
                │       │   └── Alice #0
                │       └── (field) Age
                │           └── 19
                └── *(static) 2effedfc0530f310366603a79669802c8af518e8f29e2768949aeb1039365462
                    └── struct cxo.User
                        ├── (field) Name
                        │   └── Eva #0
                        └── (field) Age
                            └── 21

----
----
(root) 7ac6fee78fd7202f870af2bcc3ba4e09fe00068ed1c3b527e1440aff331de553
├── *(dynamic) {a1cab7ccf5bed7c4fcf21fb6bdcfb304f042264a3e45946c812ce84d587ca02b, 137f4a9c68b1c5c21458b962187d66e9948fe8a658d87fbdec2c692aff0c0c89}
│   └── struct cxo.Group
│       ├── (field) Name
│       │   └── Group #0
│       ├── (field) Leader
│       │   └── *(static) f0e08559a53eff8d33061396d0515416beccc234dc6496275b7a1712e7a5d865
│       │       └── struct cxo.User
│       │           ├── (field) Name
│       │           │   └── Elisabet Bathory
│       │           └── (field) Age
│       │               └── 30
│       └── (field) Users
│           └── []cxo.User
│               ├── *(static) 39c2abd5512a56d8d3ff64430a4268356b964e6377a82ab3e8e0e2b7c8d9b926
│               │   └── struct cxo.User
│               │       ├── (field) Name
│               │       │   └── Alice #0
│               │       └── (field) Age
│               │           └── 19
│               └── *(static) 2effedfc0530f310366603a79669802c8af518e8f29e2768949aeb1039365462
│                   └── struct cxo.User
│                       ├── (field) Name
│                       │   └── Eva #0
│                       └── (field) Age
│                           └── 21
└── *(dynamic) {a1cab7ccf5bed7c4fcf21fb6bdcfb304f042264a3e45946c812ce84d587ca02b, 2e9ce252ebc836d41abfdbb3555d5d791c9e5bd702e865434222a5b2d05ca279}
    └── struct cxo.Group
        ├── (field) Name
        │   └── Group #1
        ├── (field) Leader
        │   └── *(static) f0e08559a53eff8d33061396d0515416beccc234dc6496275b7a1712e7a5d865
        │       └── struct cxo.User
        │           ├── (field) Name
        │           │   └── Elisabet Bathory
        │           └── (field) Age
        │               └── 30
        └── (field) Users
            └── []cxo.User
                ├── *(static) 58a436f55033d7c8f6b29e81955f63e387d2076fdf5752da2f6576e5edb10ee0
                │   └── struct cxo.User
                │       ├── (field) Name
                │       │   └── Alice #1
                │       └── (field) Age
                │           └── 20
                └── *(static) 5b6cd2e396ab946c85d6d10c63785572ec9094afea6669134ebe494ccc33bbf4
                    └── struct cxo.User
                        ├── (field) Name
                        │   └── Eva #1
                        └── (field) Age
                            └── 22

----
[DST] 00:46:11 conn.go:474: [ERR] 127.0.0.1:43579 reading error: EOF
--------------------------------------------------------------------------------

*/
