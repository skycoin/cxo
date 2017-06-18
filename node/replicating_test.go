package node

import (
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/skyobject"
)

type User struct {
	Name string
	Age  uint32
}

type Group struct {
	Name   string
	Leader skyobject.Reference  `skyobject:"schema=test.User"`
	Users  skyobject.References `skyobject:"schema=test.User"`
}

func testRegistry() (reg *skyobject.Registry) {

	// tpyes

	reg = skyobject.NewRegistry()

	reg.Register("test.User", User{})
	reg.Register("test.Group", Group{})

	return

}

// from one node to another
func Test_replicating(t *testing.T) {

	// feed and owner

	pk, sk := cipher.GenerateKeyPair()

	// a config

	aconf := newConfig(false)

	accept := make(chan struct{})

	aconf.OnSubscriptionAccepted = func(*gnet.Conn, cipher.PubKey) {
		accept <- struct{}{}
	}

	// b config

	bconf := newConfig(true)

	received := make(chan struct{})
	filled := make(chan struct{})

	bconf.OnRootReceived = func(root *Root) {
		if root.Pub() != pk {
			t.Error("wrong feed")
		}
		hash := root.Hash()
		t.Log("received root object:", shortHex(hash.String()))
		received <- struct{}{}
	}
	bconf.OnRootFilled = func(root *Root) {
		if root.Pub() != pk {
			t.Error("wrong feed")
		}
		filled <- struct{}{}
	}

	a, b, ac, _, err := newConnectedNodes(aconf, bconf)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	defer b.Close()

	b.Subscribe(nil, pk)
	a.Subscribe(ac, pk)

	select {
	case <-accept:
	case <-time.After(TM):
		t.Error("slow")
		return
	}

	c := a.Container()

	// I don't want to reimplement newConenctedNodes, thus
	// I will use NewRootReg instead of NewRoot

	reg := testRegistry()

	// core registry will be never removed by GC
	// but this non-core registry can be removed by GC
	// and we need to lock GC and unlock it later

	var root *Root

	c.LockGC()
	{

		c.AddRegistry(reg)

		root, err = c.NewRootReg(pk, sk, reg.Reference())
		if err != nil {
			t.Error(err)
			c.UnlockGC()
			return
		}

		root.Append(root.MustDynamic("test.User", User{
			Name: "Alice",
			Age:  19,
		}))

		// after this Inject call root object saved and
		// referes to the registry, now we can safely unlock GC

	}
	c.UnlockGC()

	select {
	case <-received:
		t.Log("received")
	case <-time.After(TM * 10):
		t.Error("slow")
		return
	}

	select {
	case <-filled:
		t.Log("filled")
	case <-time.After(TM * 10): // increase timeout
		t.Error("slow")
		return
	}

}

// the same as replicating, but 1 -> 2 -> 3
func Test_passThrough(t *testing.T) {

	// feed and owner

	pk, sk := cipher.GenerateKeyPair()

	// a config

	aconf := newConfig(false)
	aconf.GCInterval = 0 // disable GC

	accept := make(chan struct{}) // used by a and c

	aconf.OnSubscriptionAccepted = func(*gnet.Conn, cipher.PubKey) {
		accept <- struct{}{}
	}

	// b conf (no changes)

	// c config

	cconf := newConfig(true)

	received := make(chan struct{})
	filled := make(chan struct{})

	cconf.OnSubscriptionAccepted = aconf.OnSubscriptionAccepted
	cconf.OnRootReceived = func(root *Root) {
		if root.Pub() != pk {
			t.Error("wrong feed")
		}
		hash := root.Hash()
		t.Log("received root object:", shortHex(hash.String()))
		received <- struct{}{}
	}
	cconf.OnRootFilled = func(root *Root) {
		if root.Pub() != pk {
			t.Error("wrong feed")
		}
		filled <- struct{}{}
	}

	//
	// registry
	//

	reg := testRegistry()

	//
	// a
	//

	a, err := NewNodeReg(aconf, reg)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	//
	// b
	//

	b, err := NewNode(newConfig(true)) // listen
	if err != nil {
		t.Error(err)
		return
	}
	defer b.Close()

	//
	// c
	//

	c, err := NewNode(cconf)
	if err != nil {
		t.Error(err)
		return
	}
	defer c.Close()

	//
	// dial
	//

	ac, err := a.Pool().Dial(b.Pool().Address())
	if err != nil {
		t.Error(err)
		return
	}

	cc, err := c.Pool().Dial(b.Pool().Address())
	if err != nil {
		t.Error(err)
		return
	}

	//
	// subscribe
	//

	b.Subscribe(nil, pk)
	a.Subscribe(ac, pk)
	c.Subscribe(cc, pk)

	for i := 0; i < 2; i++ {
		if !receiveChanTimeout(accept, TM) {
			t.Error("slow")
			return
		}
	}

	//
	// generate
	//

	root, err := a.Container().NewRoot(pk, sk)
	if err != nil {
		t.Error(err)
		return
	}

	root.Append(root.MustDynamic("test.User", User{
		Name: "Alice",
		Age:  19,
	}))

	//
	// receive / fill
	//

	select {
	case <-received:
		t.Log("received")
	case <-time.After(TM * 10):
		t.Error("slow")
		return
	}

	select {
	case <-filled:
		t.Log("filled")
	case <-time.After(TM * 10):
		t.Error("slow")
		return
	}

}
