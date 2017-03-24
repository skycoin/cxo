package node

import (
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

func newConfig() Config {
	c := NewConfig()
	c.Debug = testing.Verbose()
	return c
}

func TestNode_connecting(t *testing.T) {
	conf := newConfig()
	db := data.NewDB()
	so := skyobject.NewContainer(db)
	n1, n2 := NewNode(conf, db, so), NewNode(conf, db, so)
	n1.Start()
	defer n1.Close()
	n2.Start()
	defer n2.Close()
	if info, err := n1.Info(); err != nil {
		t.Error(err)
		return
	} else {
		if err := n2.Connect(info.Address); err != nil {
			t.Error(err)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
//                     R E Q U I R E D    T Y P E S                           //

type User struct {
	Name string
	Age  int64
}

type Man struct {
	Name   string
	Soname string
}

type SmallGroup struct {
	Header  string
	Leader  skyobject.Reference  `skyobject:"schema=User"`
	Members skyobject.References `skyobject:"schema=User"`
	FallGuy skyobject.Dynamic
}

//                                                                            //
////////////////////////////////////////////////////////////////////////////////

func filledDownNode(pub cipher.PubKey, sec cipher.SecKey) *Node {
	db := data.NewDB()
	so := skyobject.NewContainer(db)
	conf := newConfig()
	conf.Name = "source"
	n := NewNode(conf, db, so)
	root := so.NewRoot(pub)
	root.Register("User", User{})
	root.Inject(SmallGroup{
		Header: "Widdecombe Fair",
		Leader: root.Save(User{"Old Uncle Tom Cobley", 75}),
		Members: root.SaveArray(
			User{"Bill Brewer", 50},
			User{"Jan Stewer", 51},
			User{"Peter Gurney", 52},
			User{"Peter Davy", 53},
			User{"Dan'l Whiddon", 54},
			User{"Harry Hawke", 55},
		),
		FallGuy: root.Dynamic(Man{
			Name:   "Bob",
			Soname: "Cobley",
		}),
	})
	root.Touch() // update timestamp
	so.AddRoot(root, sec)
	return n
}

func newNode(name string) *Node {
	db := data.NewDB()
	so := skyobject.NewContainer(db)
	conf := newConfig()
	conf.Name = name
	return NewNode(conf, db, so)
}

func TestNode_sourceDrain(t *testing.T) {
	// the feed an downer
	pub, sec := cipher.GenerateKeyPair()
	// create filled down node
	source := filledDownNode(pub, sec)
	source.Start()
	defer source.Close()
	// we need to subscribe to the node to make it share the feed
	source.Subscribe(pub) // subscribe to and share the feed
	// drain node
	drain := newNode("drain")
	drain.Start()
	defer drain.Close()
	// subscribe to the feed
	drain.Subscribe(pub)
	// connect to source
	info, err := source.Info()
	if err != nil {
		t.Error(err)
		return
	}
	// we don't have known hosts: use Connect method
	if err = drain.Connect(info.Address); err != nil {
		t.Error(err)
		return
	}
	// wait
	time.Sleep(500 * time.Millisecond)
	// inspect
	ss := source.db.Stat()
	t.Log("source: ", ss)
	xs := drain.db.Stat()
	t.Logf("drain: %v", xs)
	if xs.Total != ss.Total {
		t.Errorf("wrong object count: want %d, got %d", ss.Total, xs.Total)
	}
	if xs.Memory != ss.Memory {
		t.Errorf("wrong amount of memory: want %s, got %s",
			data.HumanMemory(ss.Memory),
			data.HumanMemory(xs.Total))
	}
}

func TestNode_replication(t *testing.T) {
	// the feed an downer
	pub, sec := cipher.GenerateKeyPair()
	// create filled down node
	source := filledDownNode(pub, sec)
	source.Start()
	defer source.Close()
	//
	source.Subscribe(pub) // subscribe to and share the feed
	// other nodes
	n1, n2, n3, n4 := newNode("n1"), newNode("n2"), newNode("n3"), newNode("n4")
	// start n1-n4
	for _, nd := range []*Node{n1, n2, n3, n4} {
		nd.Start()
		defer nd.Close()
	}
	// connect n1-n4
	for i, nd := range []*Node{n1, n2, n3, n4} {
		info, err := nd.Info()
		if err != nil {
			t.Error(err)
			return
		}
		for j, ns := range []*Node{n1, n2, n3, n4} {
			if i == j {
				continue
			}
			if err = ns.Connect(info.Address); err != nil {
				t.Error("connecting error: ", err)
			}
		}
		nd.Subscribe(pub) // subscribe to the feed
	}
	// connect n1 to source
	info, err := source.Info()
	if err != nil {
		t.Error(err)
		return
	}
	if err = n1.Connect(info.Address); err != nil {
		t.Error(err)
		return
	}
	// wait
	time.Sleep(5000 * time.Millisecond)
	// inspect
	ss := source.db.Stat()
	t.Log("source: ", ss)
	for i, nd := range []*Node{n1, n2, n3, n4} {
		xs := nd.db.Stat()
		t.Logf("n%d: %v", i, xs)
		if xs.Total != ss.Total {
			t.Errorf("wrong object count: want %d, got %d", ss.Total, xs.Total)
		}
		if xs.Memory != ss.Memory {
			t.Errorf("wrong amount of memory: want %s, got %s",
				data.HumanMemory(ss.Memory),
				data.HumanMemory(xs.Total))
		}
	}

}
