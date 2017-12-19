package node

import (
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/cxo/skyobject/registry"
)

func getTestConfigNotListen(prefix string) (c *Config) {
	c = getTestConfig(prefix)
	c.TCP.Listen = ""
	c.UDP.Listen = ""
	return
}

func getTestNodeNotListen(prefix string) (n *Node) {
	var err error
	if n, err = NewNode(getTestConfigNotListen(prefix)); err != nil {
		panic(err)
	}
	return
}

func TestNode_ID(t *testing.T) {

	var n = getTestNodeNotListen("test")
	defer n.Close()

	if n.ID() == (cipher.PubKey{}) {
		t.Error("blank node ID")
	}

}

func TestNode_Config(t *testing.T) {
	// (conf *Config)

	var (
		conf   = getTestConfigNotListen("test")
		n, err = NewNode(conf)
	)

	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()

	if n.Config() != conf {
		t.Error("wrong config")
	}

	var sc = skyobject.NewConfig()
	sc.InMemoryDB = true

	var c *skyobject.Container
	if c, err = skyobject.NewContainer(sc); err != nil {
		t.Fatal(err)
	}

	if n, err = NewNodeContainer(conf, c); err != nil {
		t.Fatal(err)
	}
	defer n.Close()

	if nc := n.Config(); nc != conf {
		t.Error("wrong node config")
	} else if nc == nil {
		t.Error("nil config")
	} else if nc.Config != sc {
		t.Error("wrong Container config")
	}

}

func TestNode_Container(t *testing.T) {
	// (c *skyobject.Container)

	var config = getTestConfigNotListen("test")

	var n, err = NewNode(config)

	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()

	if n.Container() == nil {
		t.Error("missing container")
	}

	var sc = skyobject.NewConfig()
	sc.InMemoryDB = true

	var c *skyobject.Container
	if c, err = skyobject.NewContainer(sc); err != nil {
		t.Fatal(err)
	}

	if n, err = NewNodeContainer(config, c); err != nil {
		t.Fatal(err)
	}
	defer n.Close()

	if n.Container() != c {
		t.Error("wrong container")
	}

}

func TestNode_Publish(t *testing.T) {
	// (r *registry.Root)

	// broadcast to all subscribers

	// create listener (server, generator of root objects)

	var (
		ln = getTestNode("server")

		gr1 = make(chan *registry.Root, 1) // sc1
		gr2 = make(chan *registry.Root, 1) // sc2

		gn1 = make(chan *registry.Root, 1) // should not receive (uc1)

		sc1 = getTestConfigNotListen("sc1") // subscriber
		sc2 = getTestConfigNotListen("sc2") // subscriber

		uc1 = getTestConfigNotListen("uc1") // not subscribed
	)

	defer ln.Close()

	// handle received roots

	sc1.OnRootReceived = func(_ *Conn, r *registry.Root) (_ error) {
		gr1 <- r
		return
	}
	sc2.OnRootReceived = func(_ *Conn, r *registry.Root) (_ error) {
		gr2 <- r
		return
	}
	uc1.OnRootReceived = func(_ *Conn, r *registry.Root) (_ error) {
		gn1 <- r
		return
	}

	// create clients (subscribed, subscribed, and not subscribed)

	var (
		sn1, sn2, un1 *Node
		err           error
	)

	if sn1, err = NewNode(sc1); err != nil {
		t.Fatal(err)
	}
	defer sn1.Close()

	if sn2, err = NewNode(sc2); err != nil {
		t.Fatal(err)
	}
	defer sn2.Close()

	if un1, err = NewNode(uc1); err != nil {
		t.Fatal(err)
	}
	defer un1.Close()

	// feed and owner

	var pk, sk = cipher.GenerateKeyPair()

	if err = ln.Share(pk); err != nil {
		t.Fatal(err)
	}

	// conenct to the listener (and subscribe / not susbcribe)

	if c, err := sn1.TCP().Connect(ln.TCP().Address()); err != nil {
		t.Fatal(err)
	} else if err = c.Subscribe(pk); err != nil {
		t.Fatal(err)
	}

	if c, err := sn2.TCP().Connect(ln.TCP().Address()); err != nil {
		t.Fatal(err)
	} else if err = c.Subscribe(pk); err != nil {
		t.Fatal(err)
	}

	if _, err = un1.TCP().Connect(ln.TCP().Address()); err != nil {
		t.Fatal(err)
	}

	// generate Root

	var up *skyobject.Unpack
	if up, err = ln.Container().Unpack(sk, getTestRegistry()); err != nil {
		t.Fatal(err)
	}

	var r = new(registry.Root)

	r.Pub = pk
	r.Nonce = 9012

	// save blank Root (it's enough for this test)

	if err = ln.Container().Save(up, r); err != nil {
		t.Error(err)
	}

	// publish

	ln.Publish(r)

	// check out channels

	select {
	case <-gr1:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("not received")
	}

	select {
	case <-gr2:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("not received")
	}

	select {
	case <-gn1:
		t.Fatal("received")
	case <-time.After(100 * time.Millisecond):
	}

}

func TestNode_ConnectionsOfFeed(t *testing.T) {
	// (feed cipher.PubKey) (cs []*Conn)

	//

}

func TestNode_Connections(t *testing.T) {
	// (cs []*Conn)

	//

}

func TestNode_TCP(t *testing.T) {
	// (tcp *TCP)

	//

}

func TestNode_UDP(t *testing.T) {
	// (udp *UDP)

	// TOOD (kostyarin): does udp works?
}

func TestNode_Share(t *testing.T) {
	// (feed cipher.PubKey) (err error)

	var (
		n = getTestNodeNotListen("test")

		err error

		pk1, _ = cipher.GenerateKeyPair()
		pk2, _ = cipher.GenerateKeyPair()
		pk3, _ = cipher.GenerateKeyPair()
	)

	defer n.Close()

	if err = n.Container().AddFeed(pk1); err != nil {
		t.Fatal(err)
	}

	if len(n.Feeds()) != 0 {
		t.Fatal("share something, but should not")
	}

	if n.IsSharing(pk1) == true {
		t.Error("share, but should not")
	}

	if err = n.DontShare(pk1); err != nil {
		t.Fatal(err)
	}

	if n.IsSharing(pk1) == true {
		t.Error("share, but should not")
	}

	// add

	if err = n.Share(pk1); err != nil {
		t.Fatal(err)
	}

	if n.IsSharing(pk1) == false {
		t.Error("doesn't share, but should")
	}

	var feeds = n.Feeds()

	if len(feeds) != 1 {
		t.Fatal("wrong feeds length:", len(feeds))
	}

	if feeds[0] != pk1 {
		t.Fatal("wrong feed")
	}

	// del

	if err = n.DontShare(pk1); err != nil {
		t.Fatal(err)
	}

	if n.IsSharing(pk1) == true {
		t.Error("share, but should not")
	}

	if len(n.Feeds()) != 0 {
		t.Fatal("share something, but should not")
	}

	// add, add, add

	var fs = []cipher.PubKey{pk1, pk2, pk3}

	for _, pk := range fs {
		if err = n.Share(pk); err != nil {
			t.Fatal(err)
		}
	}

	for _, pk := range fs {
		if n.IsSharing(pk) == false {
			t.Error("doesn't share, but should")
		}
	}

	if feeds = n.Feeds(); len(feeds) != 3 {
		t.Fatal("wrong length:", len(feeds))
	}

	var fm = map[cipher.PubKey]struct{}{
		pk1: {},
		pk2: {},
		pk3: {},
	}

	for _, pk := range feeds {
		if _, ok := fm[pk]; ok == false {
			t.Fatal("missing feed")
		}
		delete(fm, pk)
	}

}

func TestNode_DontShare(t *testing.T) {
	// (feed cipher.PubKey) (err error)

	// moved to Share

}

func TestNode_Feeds(t *testing.T) {
	// (feeds []cipher.PubKey)

	// moved to Share

}

func TestNode_IsSharing(t *testing.T) {
	// (feed cipher.PubKey) (ok bool)

	// moved to Share

}

func TestNode_Stat(t *testing.T) {
	// (s *Stat)

	// TOOD (kostyarin): the lowest priority

}

func TestNode_Close(t *testing.T) {
	// (err error)

	// TOOD (kostyarin): the lowest priority

}
