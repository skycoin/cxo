package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/skyobject"
)

const (
	TM time.Duration = 100 * time.Millisecond
)

type User struct {
	Name string
	Age  uint32
}

type Group struct {
	Name  string
	Users skyobject.References `skyobject:"schema=User"`
}

var testOut *bytes.Buffer = new(bytes.Buffer)

func init() {
	out = testOut // owerwrite
	if !testing.Verbose() {
		log.SetOutput(ioutil.Discard)
	}
}

func Test_want(t *testing.T) {
	// want(rpc, ss)

	// TODO: low priority
}

func Test_got(t *testing.T) {
	// got(rpc, ss)

	// TODO: low priority
}

func Test_subscribe(t *testing.T) {
	// subscribe(rpc, ss)

	//
}

func Test_subscribe_to(t *testing.T) {
	// subscribe_to(rpc, ss)

	//
}

func Test_unsubscribe(t *testing.T) {
	// unsubscribe(rpc, ss)

	//
}

func Test_unsubscribe_from(t *testing.T) {
	// unsubscribe_from(rpc, ss)

	//
}

func Test_feeds(t *testing.T) {
	// feeds(rpc)

	//
}

func Test_stat(t *testing.T) {
	// stat(rpc)

	//
}

func Test_connections(t *testing.T) {
	// connections(rpc)

	//
}

func Test_incoming_connections(t *testing.T) {
	// incoming_connections(rpc)

	//
}

func Test_outgoing_connections(t *testing.T) {
	// outgoing_connections(rpc)

	//
}

func Test_connect(t *testing.T) {
	// connect(rpc, ss)

	//
}

func Test_disconnect(t *testing.T) {
	// disconnect(rpc, ss)

	//
}

func Test_listening_address(t *testing.T) {
	// listening_address(rpc)

	//
}

func Test_roots(t *testing.T) {
	// roots(rpc, ss)

	t.Run("empty feed", func(t *testing.T) {
		defer testOut.Reset()

		n, err := launchNode(newNodeConfig())
		if err != nil {
			t.Fatal(err)
		}
		defer n.Close()

		cl, err := node.NewRPCClient(n.RPCAddress())
		if err != nil {
			t.Error(err)
			return
		}

		pk, _ := cipher.GenerateKeyPair()

		err = roots(cl, []string{
			"roots",
			pk.Hex(),
		})
		if err != nil {
			t.Error(err)
			return
		}

		if testOut.String() != "  empty feed\n" {
			t.Errorf("wrong output %q", testOut.String())
		}
	})

	t.Run("two", func(t *testing.T) {
		defer testOut.Reset()

		n, err := launchNode(newNodeConfig())
		if err != nil {
			t.Fatal(err)
		}
		defer n.Close()

		cl, err := node.NewRPCClient(n.RPCAddress())
		if err != nil {
			t.Error(err)
			return
		}

		pk, sk := cipher.GenerateKeyPair()

		n.Subscribe(nil, pk)

		root, err := n.Container().NewRoot(pk, sk)
		if err != nil {
			t.Error(err)
			return
		}

		// 1
		if _, err = root.Touch(); err != nil {
			t.Error(err)
			return
		}
		h1 := root.Hash()
		t1 := time.Unix(0, root.Time())
		// 2
		if _, err = root.Touch(); err != nil {
			t.Error(err)
			return
		}
		h2 := root.Hash()
		t2 := time.Unix(0, root.Time())

		err = roots(cl, []string{
			"roots",
			pk.Hex(),
		})
		if err != nil {
			t.Error(err)
			return
		}

		const format = `  - %s
      time: %s
      seq: 0
      fill: true
  - %s
      time: %s
      seq: 1
      fill: true
`

		// for windows
		out := strings.Replace(testOut.String(), "\r\n", "\n", -1)
		want := fmt.Sprintf(format,
			h1.String(),
			t1.String(),
			h2.String(),
			t2.String())

		if out != want {
			t.Error("wrong output")
		}

	})
}

func Test_tree(t *testing.T) {
	// tree(rpc, ss)

	defer testOut.Reset()

	n, err := launchNode(newNodeConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()

	cl, err := node.NewRPCClient(n.RPCAddress())
	if err != nil {
		t.Error(err)
		return
	}

	pk, sk := cipher.GenerateKeyPair()

	n.Subscribe(nil, pk)

	root, err := n.Container().NewRoot(pk, sk)
	if err != nil {
		t.Error(err)
		return
	}

	root.Append(root.MustDynamic("Group", Group{
		Name: "Just an average Group",
		Users: root.SaveArray(
			User{
				Name: "Bob Simple",
				Age:  40,
			},
			User{
				Name: "Jim Cobley",
				Age:  80,
			},
		),
	}))

	hash := root.Hash()

	err = tree(cl, []string{
		"roots",
		hash.String(),
	})
	if err != nil {
		t.Error(err)
		return
	}

	const want = `(root) %s
└── struct Group
    ├── (field) Name
    │   └── Just an average Group
    └── (field) Users
        └── []User
            ├── *(static) ff1eaf416e8281b398693ce5934af57482493622e4e2978c751e8b12542f04cb
            │   └── struct User
            │       ├── (field) Name
            │       │   └── Bob Simple
            │       └── (field) Age
            │           └── 40
            └── *(static) 915fbaac74e9a55c9ed7050a829d117d8734d0444e3543b8d67557ed9ae813bc
                └── struct User
                    ├── (field) Name
                    │   └── Jim Cobley
                    └── (field) Age
                        └── 80

`

	// for windows
	out := strings.Replace(testOut.String(), "\r\n", "\n", -1)

	if out != fmt.Sprintf(want, hash.String()) {
		t.Error("wrong output")
		t.Log(out)
		t.Log(fmt.Sprintf(want, hash.String()))
	}
}

func Test_term(t *testing.T) {
	// term(rpc)

	t.Run("allowed", func(t *testing.T) {
		defer testOut.Reset()

		conf := newNodeConfig()
		conf.RemoteClose = true

		n, err := launchNode(conf)
		if err != nil {
			t.Fatal(err)
		}
		defer n.Close()

		cl, err := node.NewRPCClient(n.RPCAddress())
		if err != nil {
			t.Error(err)
			return
		}

		if err := term(cl); err != nil {
			t.Error(err)
			return
		}

		select {
		case <-n.Quiting():
			if testOut.String() != "  terminated\n" {
				t.Errorf("wrong output %q", testOut.String())
			}
		case <-time.After(TM):
			t.Error("slow or not closed by RPC")
		}
	})

	t.Run("not allowed", func(t *testing.T) {
		defer testOut.Reset()

		n, err := launchNode(newNodeConfig())
		if err != nil {
			t.Fatal(err)
		}
		defer n.Close()

		cl, err := node.NewRPCClient(n.RPCAddress())
		if err != nil {
			t.Error(err)
			return
		}

		if err := term(cl); err == nil {
			t.Error("misisng error")
		} else if err.Error() != "not allowed" {
			t.Error("wrong errro")
		} else if testOut.Len() != 0 {
			t.Error("something on stdout")
		}
	})

}

func newNodeConfig() (conf node.NodeConfig) {
	conf = node.NewNodeConfig()
	conf.InMemoryDB = true
	conf.Listen = "127.0.0.1:0"
	conf.EnableListener = false
	conf.RPCAddress = "127.0.0.1:0"
	conf.Log.Debug = testing.Verbose()
	conf.RemoteClose = false
	conf.GCInterval = 0
	return
}

func launchNode(conf node.NodeConfig) (n *node.Node, err error) {
	reg := skyobject.NewRegistry()
	reg.Register("User", User{})
	reg.Register("Group", Group{})
	if n, err = node.NewNodeReg(conf, reg); err != nil {
		return
	}
	if !testing.Verbose() {
		n.Logger.SetOutput(ioutil.Discard)
	}
	if err = n.Start(); err != nil {
		n.Close()
		return
	}
	return
}
