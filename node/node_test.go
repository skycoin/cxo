package node

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/skyobject"
)

func newConfig() (c Config) {
	c = NewConfig()
	if testing.Verbose() {
		c.Logger = log.NewLogger("[node] ", true)
		c.Config.Logger = log.NewLogger("[pool] ", true)
	} else {
		c.Logger = log.NewLogger("", false)
		c.Logger.SetOutput(ioutil.Discard)
	}
	return
}

func newConfigName(name string) (c Config) {
	c = NewConfig()
	if testing.Verbose() {
		c.Logger = log.NewLogger("["+name+"] ", true)
		c.Config.Logger = log.NewLogger("[p:"+name+"] ", true)
	} else {
		c.Logger = log.NewLogger("", false)
		c.Logger.SetOutput(ioutil.Discard)
	}
	return
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

func filledNode(pub cipher.PubKey, sec cipher.SecKey) *Node {
	db := data.NewDB()
	so := skyobject.NewContainer(db)
	conf := newConfigName("source")
	n := NewNode(conf, db, so)
	root := so.NewRoot(pub)
	so.Register(
		"User", User{},
		"Man", Man{},
		"SmallGroup", SmallGroup{},
	)
	root.Inject(SmallGroup{
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
		FallGuy: so.Dynamic(Man{
			Name:   "Bob",
			Soname: "Cobley",
		}),
	})
	so.AddRoot(root, sec)
	return n
}

func newNode(name string) *Node {
	db := data.NewDB()
	so := skyobject.NewContainer(db)
	conf := newConfigName(name)
	return NewNode(conf, db, so)
}

// source -> drain
func TestNode_sourceDrain(t *testing.T) {
	// the feed an downer
	pub, sec := cipher.GenerateKeyPair()
	// create filled down node
	source := filledNode(pub, sec)
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
	// # connect to source
	// we don't have known hosts: use Connect method
	if err := drain.Connect(source.pool.Address()); err != nil {
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

// source -> pipe -> drain
// 1) connect pipe to source
// 2) connect drain to pipe
func TestNode_sourcePipeDrain(t *testing.T) {
	// the feed an downer
	pub, sec := cipher.GenerateKeyPair()
	// # source
	// create filled down node
	source := filledNode(pub, sec)
	source.Start()
	defer source.Close()
	// we need to subscribe to the node to make it share the feed
	source.Subscribe(pub) // subscribe to and share the feed
	// # pipe
	pipe := newNode("pipe")
	pipe.Start()
	defer pipe.Close()
	pipe.Subscribe(pub)
	// # drain
	drain := newNode("drain")
	drain.Start()
	defer drain.Close()
	drain.Subscribe(pub)
	// # connect pipe to source
	t.Log("source address: ", source.pool.Address())
	if err := pipe.Connect(source.pool.Address()); err != nil {
		t.Error(err)
		return
	}
	// # connect drain to pipe
	t.Log("pipe address: ", pipe.pool.Address())
	if err := drain.Connect(pipe.pool.Address()); err != nil {
		t.Error(err)
		return
	}
	// wait
	time.Sleep(500 * time.Millisecond)
	// inspect
	ss := source.db.Stat()
	t.Log("source: ", ss)
	names := []string{"pipe", "drain"}
	for i, x := range []*Node{pipe, drain} {
		xs := x.db.Stat()
		t.Logf("%s: %v", names[i], xs)
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

// source -> pipe -> drain
// 1) connect drain to pipe
// 2) connect pipe to source
func TestNode_sourcePipeDrain2(t *testing.T) {
	// the feed an downer
	pub, sec := cipher.GenerateKeyPair()
	// # source
	// create filled down node
	source := filledNode(pub, sec)
	source.Start()
	defer source.Close()
	// we need to subscribe to the node to make it share the feed
	source.Subscribe(pub) // subscribe to and share the feed
	// # pipe
	pipe := newNode("pipe")
	pipe.Start()
	defer pipe.Close()
	pipe.Subscribe(pub)
	// # drain
	drain := newNode("drain")
	drain.Start()
	defer drain.Close()
	drain.Subscribe(pub)
	// # connect drain to pipe
	if err := drain.Connect(pipe.pool.Address()); err != nil {
		t.Error(err)
		return
	}
	// # connect pipe to source
	if err := pipe.Connect(source.pool.Address()); err != nil {
		t.Error(err)
		return
	}
	// wait
	time.Sleep(500 * time.Millisecond)
	// inspect
	ss := source.db.Stat()
	t.Log("source: ", ss)
	names := []string{"pipe", "drain"}
	for i, x := range []*Node{pipe, drain} {
		xs := x.db.Stat()
		t.Logf("%s: %v", names[i], xs)
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

// source -> pipe -> drain
// 1) connect pipe to source
// 2) connect pipe to drain
func TestNode_sourcePipeDrain3(t *testing.T) {
	// the feed an downer
	pub, sec := cipher.GenerateKeyPair()
	// # source
	// create filled down node
	source := filledNode(pub, sec)
	source.Start()
	defer source.Close()
	// we need to subscribe to the node to make it share the feed
	source.Subscribe(pub) // subscribe to and share the feed
	// # pipe
	pipe := newNode("pipe")
	pipe.Start()
	defer pipe.Close()
	pipe.Subscribe(pub)
	// # drain
	drain := newNode("drain")
	drain.Start()
	defer drain.Close()
	drain.Subscribe(pub)
	// # connect pipe to source
	if err := pipe.Connect(source.pool.Address()); err != nil {
		t.Error(err)
		return
	}
	// # connect pipe to drain
	if err := pipe.Connect(drain.pool.Address()); err != nil {
		t.Error(err)
		return
	}
	// wait
	time.Sleep(500 * time.Millisecond)
	// inspect
	ss := source.db.Stat()
	t.Log("source: ", ss)
	names := []string{"pipe", "drain"}
	for i, x := range []*Node{pipe, drain} {
		xs := x.db.Stat()
		t.Logf("%s: %v", names[i], xs)
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

// source -> pipe -> drain
// 1) connect pipe to drain
// 2) connect pipe to source
func TestNode_sourcePipeDrain4(t *testing.T) {
	// the feed an downer
	pub, sec := cipher.GenerateKeyPair()
	// # source
	// create filled down node
	source := filledNode(pub, sec)
	source.Start()
	defer source.Close()
	// we need to subscribe to the node to make it share the feed
	source.Subscribe(pub) // subscribe to and share the feed
	// # pipe
	pipe := newNode("pipe")
	pipe.Start()
	defer pipe.Close()
	pipe.Subscribe(pub)
	// # drain
	drain := newNode("drain")
	drain.Start()
	defer drain.Close()
	drain.Subscribe(pub)
	// # connect pipe to drain
	if err := pipe.Connect(drain.pool.Address()); err != nil {
		t.Error(err)
		return
	}
	// # connect pipe to source
	if err := pipe.Connect(source.pool.Address()); err != nil {
		t.Error(err)
		return
	}
	// wait
	time.Sleep(500 * time.Millisecond)
	// inspect
	ss := source.db.Stat()
	t.Log("source: ", ss)
	names := []string{"pipe", "drain"}
	for i, x := range []*Node{pipe, drain} {
		xs := x.db.Stat()
		t.Logf("%s: %v", names[i], xs)
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

// source -> n1 <-> n2 <-> n3 <-> n4
func TestNode_replication(t *testing.T) {
	// the feed an downer
	pub, sec := cipher.GenerateKeyPair()
	// create filled down node
	source := filledNode(pub, sec)
	source.Start()
	defer source.Close()
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
		address := nd.pool.Address()
		// subscribe to the feed first
		nd.Subscribe(pub)
		for j, ns := range []*Node{n1, n2, n3, n4} {
			if i == j {
				continue
			}
			if err := ns.Connect(address); err != nil {
				t.Error("connecting error: ", err)
			}
		}
	}
	// connect n1 to source
	if err := n1.Connect(source.pool.Address()); err != nil {
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

//
// closer to cmd/test
//

func TestNode_sourcePipeDrainTest(t *testing.T) {
	// required types
	type Board struct {
		Head    string
		Threads skyobject.References `skyobject:"schema=Thread"`
		Owner   skyobject.Dynamic
	}
	type Thread struct {
		Head  string
		Posts skyobject.References `skyobject:"schema=Post"`
	}
	type Post struct {
		Head string
		Body string
	}
	type User struct {
		Name   string
		Age    int32
		Hidden string `enc:"-"`
	}
	type Man struct {
		Name string
		Age  int32
	}
	// nodes
	var source, pipe, drain *Node
	// required keys
	pub, sec := cipher.GenerateKeyPair()
	// create source
	{
		db := data.NewDB()
		so := skyobject.NewContainer(db)
		conf := newConfigName("source")
		source = NewNode(conf, db, so)
		source.Start()
		defer source.Close()
		source.Subscribe(pub)
		so.Register(
			"Board", Board{},
			"Thread", Thread{},
			"Post", Post{},
			"User", User{},
		)
		root := so.NewRoot(pub)
		root.Inject(Board{
			"Board #1",
			so.SaveArray(
				Thread{"Thread #1.1", so.SaveArray(
					Post{"Post #1.1.1", "Body #1.1.1"},
					Post{"Post #1.1.2", "Body #1.1.2"})},
				Thread{"Thread #1.2", so.SaveArray(
					Post{"Post #1.2.1", "Body #1.2.1"},
					Post{"Post #1.2.2", "Body #1.2.2"})},
				Thread{"Thread #1.3", so.SaveArray(
					Post{"Post #1.3.1", "Body #1.3.1"},
					Post{"Post #1.3.2", "Body #1.3.2"})},
				Thread{"Thread #1.4", so.SaveArray(
					Post{"Post #1.4.1", "Body #1.4.1"},
					Post{"Post #1.4.2", "Body #1.4.2"})}),
			so.Dynamic(User{"Billy Kid", 16, "secret"}),
		})
		so.AddRoot(root, sec)
		source.Share(pub)
	}
	// create drain
	{
		drain = newNode("drain")
		drain.Start()
		defer drain.Close()
		drain.Subscribe(pub)
	}
	// create pipe
	{
		pipe = newNode("pipe")
		pipe.Start()
		defer pipe.Close()
		pipe.Subscribe(pub)
	}
	// connect pipe to drain
	{
		info, err := drain.Info()
		if err != nil {
			t.Error(err)
			return
		}
		if err = pipe.Connect(info.Address); err != nil {
			t.Error(err)
			return
		}
	}
	// connect pipe to source
	{
		info, err := source.Info()
		if err != nil {
			t.Error(err)
			return
		}
		if err = pipe.Connect(info.Address); err != nil {
			t.Error(err)
			return
		}
	}
	// wait
	time.Sleep(500 * time.Millisecond)
	// inspect
	ss := source.db.Stat()
	t.Log("source: ", ss)
	names := []string{"pipe", "drain"}
	for i, x := range []*Node{pipe, drain} {
		xs := x.db.Stat()
		t.Logf("%s: %v", names[i], xs)
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
