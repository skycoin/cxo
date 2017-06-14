package node

import (
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/skyobject"
)

func Test_replicating(t *testing.T) {

	// tpyes

	type User struct {
		Name string
		Age  uint32
	}

	type Group struct {
		Name   string
		Leader skyobject.Reference  `skyobject:"schema=test.User"`
		Users  skyobject.References `skyobject:"schema=test.User"`
	}

	// feed and owner

	pk, sk := cipher.GenerateKeyPair()

	// a config

	aconf := newNodeConfig(false)

	accept := make(chan struct{})

	aconf.OnSubscriptionAccepted = func(*gnet.Conn, cipher.PubKey) {
		accept <- struct{}{}
	}

	// b config

	bconf := newNodeConfig(true)

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

	reg := skyobject.NewRegistry()

	reg.Register("test.User", User{})
	reg.Register("test.Group", Group{})

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

		root.Inject("test.User", User{
			Name: "Alice",
			Age:  19,
		})

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
		//
	case <-time.After(TM * 10): // increase timeout
		t.Error("slow")
		return
	}

}
