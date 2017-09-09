package skyobject

import (
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

func testGetByHash(t *testing.T, db data.CXDS, hash cipher.SHA256,
	obj interface{}) {

	val, _, _ := db.Get(hash)
	if val == nil {
		t.Fatal("nil val")
	}
	if err := encoder.DeserializeRaw(val, obj); err != nil {
		t.Fatal(err)
	}
}

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
	gr := new(Group)
	testGetByHash(t, ec.db.CXDS(), dr.Object, gr)
	if gr.Name != "the Group" {
		t.Fatal("wrong name")
	}

	// leader
	leader := new(User)
	testGetByHash(t, ec.db.CXDS(), gr.Leader.Hash, leader)
	if leader.Name != "Alice" {
		t.Fatal("wrong name")
	}

	// curator
	curator := new(Developer)
	testGetByHash(t, ec.db.CXDS(), gr.Curator.Object, curator)
	if curator.Name != "kostyarin" {
		t.Fatal("wrong name")
	}

	// refs
	ers := new(encodedRefs)
	testGetByHash(t, ec.db.CXDS(), gr.Members.Hash, ers)

	if ers.Degree != uint32(ec.conf.MerkleDegree) {
		t.Fatal("wrong degree")
	}
	if ers.Length != 18 {
		t.Fatal("wrong length")
	}
	if ers.Depth != 2 {
		t.Fatal("wrong depth", ers.Depth)
	}
	if len(ers.Nested) != 1 {
		t.Fatal("wrong nested length", len(ers.Nested))
	}

	// nested node (split to 2)
	//

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
		FullQ: make(chan FullRoot, 128),
		DropQ: make(chan DropRootError, 128),
	}

	fl := rc.NewFiller(rr, bus)

	tc := time.After(time.Second)
	var fr FullRoot

Loop:
	for {
		select {
		case wcxo := <-bus.WantQ:
			t.Log("want", wcxo.Key.Hex()[:7])
			val, _, err := ec.db.CXDS().Get(wcxo.Key)
			if err != nil {
				t.Fatal(err)
			}
			if val == nil {
				t.Fatal("val is nil")
			}
			// save receiver side
			if _, err := rc.db.CXDS().Set(wcxo.Key, val); err != nil {
				t.Fatal(err)
			}
			wcxo.GotQ <- val
		case dre := <-bus.DropQ:
			t.Fatal(dre.Err)
		case fr = <-bus.FullQ:
			break Loop
		case <-tc:
			t.Fatal("too long")
		}
	}

	t.Log(rc.Inspect(fr.Root))

	_ = fl

}
