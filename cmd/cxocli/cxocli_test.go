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
	Users skyobject.Refs `skyobject:"schema=User"`
}

var testOut = new(bytes.Buffer)

func init() {
	out = testOut // owerwrite
	if !testing.Verbose() {
		log.SetOutput(ioutil.Discard)
	}
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
		n.AddFeed(pk)

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

		n.AddFeed(pk)

		cnt := n.Container()
		pack, err := cnt.NewRoot(pk, sk, 0, cnt.CoreRegistry().Types())
		if err != nil {
			t.Error(err)
			return
		}

		// 1
		if err = pack.Save(); err != nil {
			t.Error(err)
			return
		}
		n.Publish(pack.Root())
		h1 := pack.Root().Hash
		t1 := time.Unix(0, pack.Root().Time)
		// 2
		if err = pack.Save(); err != nil {
			t.Error(err)
			return
		}
		n.Publish(pack.Root())
		h2 := pack.Root().Hash
		t2 := time.Unix(0, pack.Root().Time)

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
			h1.Hex(),
			t1.String(),
			h2.Hex(),
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

	n.AddFeed(pk)

	cnt := n.Container()

	pack, err := cnt.NewRoot(pk, sk, 0, cnt.CoreRegistry().Types())
	if err != nil {
		t.Error(err)
		return
	}

	pack.Append(&Group{
		Name: "Just an average Group",
		Users: pack.Refs(
			&User{
				Name: "Bob Simple",
				Age:  40},
			&User{
				Name: "Jim Cobley",
				Age:  80}),
	})
	if err = pack.Save(); err != nil {
		t.Error(err)
		return
	}

	err = tree(cl, []string{
		"roots",
		pack.Root().Pub.Hex(),
		fmt.Sprint(pack.Root().Seq),
	})
	if err != nil {
		t.Error(err)
		return
	}

	const want = `(root) %s (reg: ecf785f)
└── *(dynamic) {7409ab3:baef146}
    └── Group
        ├── Name: "Just an average Group"
        └── Users: *(refs) d31349b 2
            ├── *(ref) ff1eaf4
            │   └── User
            │       ├── Name: "Bob Simple"
            │       └── Age: 40
            └── *(ref) 915fbaa
                └── User
                    ├── Name: "Jim Cobley"
                    └── Age: 80

`

	// for windows
	out := strings.Replace(testOut.String(), "\r\n", "\n", -1)

	if out != fmt.Sprintf(want, pack.Root().Short()) {
		t.Error("wrong output")
		t.Logf("%q", out)
		t.Logf("%q", fmt.Sprintf(want, pack.Root().Short()))
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

func newNodeConfig() (conf node.Config) {
	conf = node.NewConfig()
	conf.InMemoryDB = true
	conf.Listen = "127.0.0.1:0"
	conf.EnableListener = false
	conf.RPCAddress = "127.0.0.1:0"
	conf.Log.Debug = testing.Verbose()
	conf.RemoteClose = false
	return
}

func launchNode(conf node.Config) (n *node.Node, err error) {
	if conf.Skyobject == nil {
		conf.Skyobject = skyobject.NewConfig()
	}
	conf.Skyobject.Registry = skyobject.NewRegistry(func(r *skyobject.Reg) {
		r.Register("User", User{})
		r.Register("Group", Group{})
	})
	if !testing.Verbose() {
		conf.Log.Output = ioutil.Discard
		conf.Skyobject.Log.Output = ioutil.Discard
	}
	if n, err = node.NewNode(conf); err != nil {
		return
	}
	return
}
