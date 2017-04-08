package node_test

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/skyobject"
)

//
// example objects
//

type Board struct {
	Name    string
	Threads skyobject.References `skyobject:"schema=Thread"`
}

type Thread struct {
	Name  string
	Posts skyobject.References `skyobject:"schema=Post"`
}

type Post struct {
	Header string
	Body   string
}

func Example() {
	db := data.NewDB()
	so := skyobject.NewContainer(db)

	//
	// register objects
	//

	so.Register(
		"Board", Board{},
		"Thread", Thread{},
		"Post", Post{},
	)

	//
	// create node instance
	//

	conf := node.NewConfig()
	conf.Name = "example node"
	conf.Debug = true

	n := node.NewNode(conf, db, so)

	n.Start()
	defer n.Close()

	//
	// example owner
	//

	pub, sec := cipher.GenerateKeyPair()

	// we need to subscribe to the feed to share it
	n.Subscribe(pub)

	// =========================================================================

	//
	// 1) Create, fill down and share a root object
	//

	// =========================================================================

	// execute in main proccessing thread to be
	// sure that accessing skyobject is thread safe
	n.Execute(func() {
		// create root object using public key of the owner
		root := so.NewRoot(pub)

		// inject first board (implicit touch)
		root.Inject(Board{
			Name: "The Board #1",
			Threads: so.SaveArray(
				Thread{
					Name: "The Thread #1.1",
					Posts: so.SaveArray(
						Post{"The Post #1.1.1", "blah"},
						Post{"The Post #1.1.2", "blah"},
						Post{"The Post #1.1.3", "blah"},
					),
				},
				Thread{
					Name: "The Thread #1.2",
					Posts: so.SaveArray(
						Post{"The Post #1.2.1", "blah"},
						Post{"The Post #1.2.2", "blah"},
						Post{"The Post #1.2.3", "blah"},
					),
				},
			),
		})
		so.AddRoot(root, sec) // and sign

		// share the root
		n.Share(pub)
	})

	//
	// some time required for replication
	//

	// =========================================================================

	//
	// 2) Get existing root object, add an object and share
	//

	// =========================================================================

	n.Execute(func() {
		root = so.Root(pub)
		if root == nil { // not found
			// never happens for the example, because the Root exists
			root = so.NewRoot(pub)
		}
		root.Inject(Board{ // implicit touch
			Name: "The Board #2",
			Threads: so.SaveArray(
				Thread{
					Name: "The Thread #2.1",
					Posts: so.SaveArray(
						Post{"The Post #2.1.1", "blah"},
					),
				},
			),
		})
		so.AddRoot(root, sec) // and sign
		n.Share(pub)

	})

	//
	// some time required for replication
	//

	// =========================================================================

	//
	// 3) The same as (2) using InjectHash
	//

	// =========================================================================

	n.Execute(func() {
		root = so.Root(pub)
		if root == nil { // not found
			// never happens for the example, because the Root exists
			root = so.NewRoot(pub)
		}
		// root object works only with Dynamic objects
		board := so.Save(
			so.Dynamic(Board{ // implicit touch
				Name: "The Board #3",
				Threads: so.SaveArray(
					Thread{
						Name: "The Thread #3.1",
						Posts: so.SaveArray(
							Post{"The Post #3.1.1", "blah"},
						),
					},
				),
			}),
		)
		root.InjectHash(board) // inject an object using its hash
		so.AddRoot(root, sec)  // and sign
		n.Share(pub)
	})

	//
	// some time required for replication
	//

	// =========================================================================

	//
	// 4) Drop all references of the root and Inject new one replacing root
	//

	// =========================================================================

	n.Execute(func() {
		// create root object using public key of the owner
		root = so.NewRoot(pub)

		// inject first board (implicit touch)
		root.Inject(Board{
			Name: "The Board #4",
			Threads: so.SaveArray(
				Thread{
					Name: "The Thread #4.1",
					Posts: so.SaveArray(
						Post{"The Post #4.1.1", "blah"},
					),
				},
			),
		})
		// replace previous root object with new one
		so.AddRoot(root, sec) // and sign

		// share the root
		n.Share(pub)
	})

	//
	// some time required for replication
	//

	return
}
