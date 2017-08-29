package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func Test_filling(t *testing.T) {

	pk, sk := cipher.GenerateKeyPair()

	// emitter
	ec := getCont()
	defer ec.Close()

	if err := ec.AddFeed(pk); err != nil {
		t.Fatal(err)
	}

	pack, err := ec.NewRoot(pk, sk, 0, ec.CoreRegistry().Types())
	if err != nil {
		t.Fatal(err)
	}

	plead := pack.Ref(User{"Alice", 21, nil})
	pmemb := pack.Refs(
		User{"U#01", 19, nil},
		User{"U#02", 20, nil},
		User{"U#03", 21, nil},
		User{"U#04", 22, nil},
		User{"U#05", 23, nil},
		User{"U#06", 24, nil},
		User{"U#07", 25, nil},
		User{"U#08", 26, nil},
		User{"U#09", 27, nil},
		User{"U#10", 28, nil},
		User{"U#11", 29, nil},
		User{"U#12", 30, nil},
		User{"U#13", 31, nil},
		User{"U#14", 31, nil},
		User{"U#15", 33, nil},
		User{"U#16", 34, nil},
		User{"U#16", 35, nil},
		User{"U#17", 36, nil},
	)

	t.Log(pmemb.DebugString(false))

	pcur := pack.Dynamic(Developer{"kostyarin", "logrusorgru"})

	pack.Append(Group{
		Name:    "the Group",
		Leader:  plead,
		Members: pmemb,
		Curator: pcur,
	})

	if err := pack.Save(); err != nil {
		t.Fatal(err)
	}

	er := pack.Root()

	//
	// explore the Root first
	//

	// group
	dr := er.Refs[0]
	val, _, _ := ec.db.CXDS().Get(dr.Object)
	if val == nil {
		t.Fatal("nil val")
	}
	gr := new(Group)
	if err := encoder.DeserializeRaw(val, gr); err != nil {
		t.Fatal(err)
	}
	if gr.Name != "the Group" {
		t.Fatal("wrong name")
	}

	// leader
	val, _, _ = ec.db.CXDS().Get(gr.Leader.Hash)
	if val == nil {
		t.Fatal("nil val")
	}
	leader := new(User)
	if err := encoder.DeserializeRaw(val, leader); err != nil {
		t.Fatal(err)
	}
	if leader.Name != "Alice" {
		t.Fatal("wrong name")
	}

	// curator
	val, _, _ = ec.db.CXDS().Get(gr.Curator.Object)
	if val == nil {
		t.Fatal("nil val")
	}
	curator := new(Developer)
	if err := encoder.DeserializeRaw(val, curator); err != nil {
		t.Fatal(err)
	}
	if curator.Name != "kostyarin" {
		t.Fatal("wrong name")
	}

	// refs
	val, _, _ = ec.db.CXDS().Get(gr.Members.Hash)
	if val == nil {
		t.Fatal("nil val", gr.Members.Hash.Hex())
	}
	ers := new(encodedRefs)
	if err := encoder.DeserializeRaw(val, ers); err != nil {
		t.Fatal(err)
	}
	if ers.Degree != uint32(ec.conf.MerkleDegree) {
		t.Fatal("wrong degree")
	}
	if ers.Length != 18 {
		t.Fatal("wrong length")
	}
	if ers.Depth != 2 {
		t.Fatal("wrong depth", ers.Depth)
	}
	if len(ers.Nested) != 2 {
		t.Log(ec.Inspect(er))
		t.Fatal("wrong nested length", len(ers.Nested))
	}

	//
	//
	//

	// receiver
	rc := getCont()
	defer rc.Close()

	if err := rc.AddFeed(pk); err != nil {
		t.Fatal(err)
	}

	rr, err := rc.AddEncodedRoot(er.Sig, er.Encode())
	if err != nil {
		t.Fatal(err)
	}

	bus := FillingBus{
		WantQ: make(chan WCXO, 128),
		FullQ: make(chan *Root, 128),
		DropQ: make(chan DropRootError, 128),
	}

	fl := rc.NewFiller(rr, bus)

	for {
		select {
		case wcxo := <-bus.WantQ:
			t.Log("want", wcxo.Key)
			val, _, err := ec.db.CXDS().Get(wcxo.Key)
			if err != nil {
				t.Fatal(err)
			}
			if val == nil {
				t.Fatal("val is nil")
			}
			wcxo.GotQ <- val
		case dre := <-bus.DropQ:
			t.Fatal(dre.Err)
		case fr := <-bus.FullQ:
			t.Fatal("FULL", fr.Short())
		}
	}

	_ = fl

}
