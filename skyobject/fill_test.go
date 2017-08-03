package skyobject

import (
	"sync"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	//"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node/log"
)

func Test_filling(t *testing.T) {

	type User struct {
		Name string
		Age  uint32
		/*		Ary   [3]uint64
				Slice []string*/
	}

	type Group struct {
		Name     string
		Leader   Ref  `skyobject:"schema=User"`
		Memebers Refs `skyobject:"schema=User"`
		Curator  Dynamic
		// List     []Dynamic
		// Ary      [5]Dynamic
	}

	type Man struct {
		Name   string
		Height uint32
	}

	// create c1

	var getCont = func(name string) *Container {
		conf := NewConfig()
		conf.Log.Prefix = "[" + name + "] "
		conf.Registry = NewRegistry(func(r *Reg) {
			r.Register("User", User{})
			r.Register("Group", Group{})
			r.Register("Man", Man{})
		})
		if testing.Verbose() {
			conf.Log.Debug = true
			conf.Log.Pins = log.All
		}
		return NewContainer(data.NewMemoryDB(), conf)
	}

	c1 := getCont("emitter")
	defer c1.Close()

	c2 := getCont("collector")
	defer c2.Close()

	reg1 := c1.CoreRegistry()
	reg2 := c2.CoreRegistry()

	if reg1.Reference() != reg2.Reference() {
		t.Fatal("fatality")
	}

	for name, sch := range reg1.reg {
		if reg2.reg[name].Reference() != sch.Reference() {
			t.Error("brutality")
		}
	}

	pk, sk := cipher.GenerateKeyPair()

	if err := c1.AddFeed(pk); err != nil {
		t.Fatal(err)
	}

	if err := c2.AddFeed(pk); err != nil {
		t.Fatal(err)
	}

	// test fill and knows about

	// empty references and slices/arrays of Dynamic

	pack, err := c1.NewRoot(pk, sk, HashTableIndex, reg1.Types())
	if err != nil {
		t.Fatal(err)
	}

	group := new(Group)
	group.Name = "the Group"

	pack.Append(group)

	if group.Leader.walkNode == nil {
		t.Error("Leader not initialized")
	}
	if group.Memebers.wn == nil {
		t.Error("Members not initialized")
	}
	if group.Curator.walkNode == nil {
		t.Error("Curator not initialized")
	}

	rp, err := pack.Save()
	if err != nil {
		t.Fatal(err)
	}

	r2, err := c2.AddRoot(pk, &rp)
	if err != nil {
		t.Fatal(err)
	}

	wantq := make(chan WCXO, 1)
	fullq := make(chan *Root)
	dropq := make(chan DropRootError)
	wg := new(sync.WaitGroup)

	filler := c2.NewFiller(r2, wantq, fullq, dropq, wg)

Loop1:
	for {
		select {
		case wcxo := <-wantq:
			val := c1.Get(wcxo.Hash)
			c2.Set(wcxo.Hash, val)
			select {
			case wcxo.GotQ <- val:
				t.Log("sent", wcxo.Hash.Hex()[:7])
			case de := <-dropq:
				t.Fatal(de)
			}
		case r := <-fullq:
			t.Log("filled", r.Short())
			break Loop1
		case de := <-dropq:
			t.Log(c1.Inspect(pack.Root()))
			t.Fatal(de)
		}
	}

	filler.Close()

	var drainChans = func(wantq chan WCXO, fullq chan *Root,
		dropq chan DropRootError) {
		for {
			select {
			case <-wantq:
			case <-fullq:
			case <-dropq:
			default:
				return
			}
		}
	}

	drainChans(wantq, fullq, dropq)

	// fill referenes

	if err := group.Leader.SetValue(&User{"Alex", 32}); err != nil {
		t.Fatal(err)
	}
	if err := group.Curator.SetValue(&Man{"Jho", 44}); err != nil {
		t.Fatal(err)
	}
	err = group.Memebers.Append(
		&User{"A1", 1},
		&User{"A2", 2},
		&User{"A3", 3},
		&User{"A4", 4},
	)
	if err != nil {
		t.Fatal(err)
	}

	pack.Append(group)
	rp, err = pack.Save()

	if err != nil {
		t.Fatal(err)
	}

	r2, err = c2.AddRoot(pk, &rp)
	if err != nil {
		t.Fatal(err)
	}

	// fill fill fill

	filler = c2.NewFiller(r2, wantq, fullq, dropq, wg)

Loop2:
	for {
		select {
		case wcxo := <-wantq:
			val := c1.Get(wcxo.Hash)
			c2.Set(wcxo.Hash, val)
			select {
			case wcxo.GotQ <- val:
				t.Log("sent", wcxo.Hash.Hex()[:7])
			case de := <-dropq:
				t.Fatal(de)
			}
		case r := <-fullq:
			t.Log("filled", r.Short())
			break Loop2
		case de := <-dropq:
			t.Log(c1.Inspect(pack.Root()))
			t.Log(group.Memebers.DebugString())
			t.Fatal(de)
		}
	}

	filler.Close()

	//

}
