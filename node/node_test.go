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
	if address, err := n1.Info(); err != nil {
		t.Error(err)
		return
	} else {
		if err := n2.Connect(address); err != nil {
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

type SmallGroup struct {
	Header  string
	Leader  cipher.SHA256   `skyobject:"href,schema=User"`
	Members []cipher.SHA256 `skyobject:"href,schema=User"`
}

//                                                                            //
////////////////////////////////////////////////////////////////////////////////

func filledDownNode() *Node {
	db := data.NewDB()
	so := skyobject.NewContainer(db)
	conf := newConfig()
	conf.Name = "source"
	n := NewNode(conf, db, so)
	root := so.NewRoot()
	root.Register("User", User{})
	root.Set(SmallGroup{
		Header: "Widdecombe Fair",
		Leader: so.Save(User{"Old Uncle Tom Cobley", 75}),
		Members: so.SaveArray(
			User{"Bill Brewer", 50},
			User{"Jan Stewer", 51},
			User{"Peter Gurney", 52},
			User{"Peter Davy", 53},
			User{"Dan'l Whiddon", 54},
			User{"Harry Hawke", 55},
		),
	})
	so.SetRoot(root)
	return n
}

func newNode(name string) *Node {
	db := data.NewDB()
	so := skyobject.NewContainer(db)
	conf := newConfig()
	conf.Name = name
	return NewNode(conf, db, so)
}

func TestNode_replication(t *testing.T) {
	// create filled down node
	source := filledDownNode()
	source.Start()
	defer source.Close()
	// other nodes
	n1, n2, n3, n4 := newNode("n1"), newNode("n2"), newNode("n3"), newNode("n4")
	// start n1-n4
	for _, nd := range []*Node{n1, n2, n3, n4} {
		nd.Start()
		defer nd.Close()
	}
	// connect n1-n4
	for i, nd := range []*Node{n1, n2, n3, n4} {
		address, err := nd.Info()
		if err != nil {
			t.Error(err)
			return
		}
		for j, ns := range []*Node{n1, n2, n3, n4} {
			if i == j {
				continue
			}
			if err = ns.Connect(address); err != nil {
				t.Error("connecting error: ", err)
			}
		}
	}
	// connect n1 to source
	address, err := source.Info()
	if err != nil {
		t.Error(err)
		return
	}
	if err = n1.Connect(address); err != nil {
		t.Error(err)
		return
	}
	// wait
	time.Sleep(500 * time.Millisecond)
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
