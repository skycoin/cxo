package skyobject

import (
	"reflect"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func TestPack_newOf(t *testing.T) {

	c := getCont()
	pk, sk := cipher.GenerateKeyPair()

	pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
	if err != nil {
		t.Fatal(err)
	}

	for _, sn := range []struct {
		schemaName string
		typ        reflect.Type
	}{
		{"cxo.User", reflect.TypeOf(User{})},
		{"cxo.Group", reflect.TypeOf(Group{})},
		{"cxo.Developer", reflect.TypeOf(Developer{})},
	} {

		val, err := pack.newOf(sn.schemaName)

		if err != nil {
			t.Error(err)
			continue
		}

		if val.Kind() != reflect.Ptr {
			t.Error("not a pointer")
			continue
		}

		if val.IsNil() {
			t.Error("is nil")
			continue
		}

		elem := val.Elem()
		if elem.Type() != sn.typ {
			t.Error("wrong type")
		}

	}

	if _, err := pack.newOf("unknown"); err == nil {
		t.Error("misisng error")
	}

}

func TestPack_unpackToGo(t *testing.T) {

	c := getCont()
	pk, sk := cipher.GenerateKeyPair()

	pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
	if err != nil {
		t.Fatal(err)
	}

	for _, sn := range []struct {
		schemaName string
		value      interface{}
	}{
		{"cxo.User", User{"Alice", 21, nil}},
		{"cxo.Group", Group{Name: "VDPG"}},
		{"cxo.Developer", Developer{Name: "I'm, in person"}},
	} {

		obj, err := pack.unpackToGo(sn.schemaName, encoder.Serialize(sn.value))
		if err != nil {
			t.Error(err)
			continue
		}

		switch sn.schemaName {
		case "cxo.User":
			src := sn.value.(User)
			if usr, ok := obj.(*User); !ok {
				t.Errorf("wrong type %T", obj)
			} else if usr.Name != src.Name || usr.Age != src.Age {
				t.Error("wrong decode value")
			}
		case "cxo.Group":
			src := sn.value.(Group)
			if grp, ok := obj.(*Group); !ok {
				t.Errorf("wrong type %T", obj)
			} else if grp.Name != src.Name {
				t.Error("wrong decode value")
			} else if grp.Leader.isInitialized() == false {
				t.Error("Ref's not initialized")
			} else if grp.Members.isInitialized() == false {
				t.Error("Refs were not initialized")
			} else if grp.Curator.isInitialized() == false {
				t.Error("Dynamic's not initialized")
			}
		case "cxo.Developer":
			src := sn.value.(Developer)
			if dev, ok := obj.(*Developer); !ok {
				t.Errorf("wrong type %T", obj)
			} else if dev.Name != src.Name {
				t.Error("wrong decode value")
			}
		}

	}

}

func TestPack_setupToGo(t *testing.T) {
	c := getCont()

	t.Run("load all", func(t *testing.T) {
		pk, sk := cipher.GenerateKeyPair()

		pack, err := c.NewRoot(pk, sk, EntireTree, c.CoreRegistry().Types())
		if err != nil {
			t.Fatal(err)
		}

		usr := User{"Alice", 21, nil}
		usrHash, _ := pack.save(usr)

		g1, g2, g3 := User{"Eva", 17, nil}, User{"Ammy", 16, nil},
			User{"Bob", 25, nil}
		g1Hash, _ := pack.save(g1)
		g2Hash, _ := pack.save(g2)
		g3Hash, _ := pack.save(g3)

		var er encodedRefs
		er.Degree = uint32(c.conf.MerkleDegree)
		er.Length = 3
		er.Depth = 0
		er.Nested = []cipher.SHA256{g1Hash, g2Hash, g3Hash}

		refsHash, _ := pack.save(er)

		cur := Developer{"Nii", "niihub.tld/nii"}
		curHash, _ := pack.save(cur)

		curSchema, err := pack.Registry().SchemaByName("cxo.Developer")

		grp := &Group{
			Name:    "vdpg",
			Leader:  Ref{Hash: usrHash},
			Members: Refs{Hash: refsHash},
			Curator: Dynamic{SchemaRef: curSchema.Reference(), Object: curHash},
		}

		val := reflect.ValueOf(grp)

		if err := pack.setupToGo(val); err != nil {
			t.Fatal(err)
		}

		if grp.Leader.isInitialized() == false {
			t.Error("Ref not initialized")
		}
		if grp.Members.isInitialized() == false {
			t.Error("Refs not initialized")
		}
		if grp.Curator.isInitialized() == false {
			t.Error("Dynamic not initialized")
		}

		// Leader
		usrI, err := grp.Leader.Value()
		if err != nil {
			t.Fatal(err)
		}
		usrl := usrI.(*User)
		if usrl.Age != usr.Age || usrl.Name != usr.Name {
			t.Error("wrong Leader")
		}

		// Members
		mem := grp.Members

		if mem.rn.index != nil {
			t.Error("unexpected index")
		}

		if mem.upper != nil {
			t.Error("unexpected upper")
		}

		if mem.degree != c.conf.MerkleDegree {
			t.Error("wrong degree", mem.degree, c.conf.MerkleDegree)
		}

		if mem.depth != 0 {
			t.Error("wrong length", mem.depth)
		}

		if mem.length != 3 {
			t.Error("wrong length", mem.length)
		}

		for i, g := range []User{g1, g2, g3} {
			gRef, err := grp.Members.RefByIndex(i)
			if err != nil {
				t.Fatal(err)
			}
			gI, err := gRef.Value()
			if err != nil {
				t.Fatal(err)
			}
			gl := gI.(*User)
			if gl.Name != g.Name || gl.Age != g.Age {
				t.Error("wrong Memeber", i)
			}
		}

		// Curator
		curI, err := grp.Curator.Value()
		if err != nil {
			t.Fatal(err)
		}
		if curI == nil {
			t.Fatal("nil Curator")
		}
		curl := curI.(*Developer)
		if curl.GitHub != cur.GitHub || curl.Name != cur.Name {
			t.Error("wrong Curator")
		}

	})

	t.Run("load no", func(t *testing.T) {
		pk, sk := cipher.GenerateKeyPair()

		pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
		if err != nil {
			t.Fatal(err)
		}

		usr := User{"Alice", 21, nil}
		usrHash, _ := pack.save(usr)

		g1, g2, g3 := User{"Eva", 17, nil}, User{"Ammy", 16, nil},
			User{"Bob", 25, nil}
		g1Hash, _ := pack.save(g1)
		g2Hash, _ := pack.save(g2)
		g3Hash, _ := pack.save(g3)

		var er encodedRefs
		er.Degree = 8
		er.Length = 3
		er.Depth = 0
		er.Nested = []cipher.SHA256{g1Hash, g2Hash, g3Hash}

		refsHash, _ := pack.save(er)

		cur := Developer{"Nii", "niihub.tld/nii"}
		curHash, _ := pack.save(cur)

		curSchema, err := pack.Registry().SchemaByName("cxo.Developer")

		grp := &Group{
			Name:    "vdpg",
			Leader:  Ref{Hash: usrHash},
			Members: Refs{Hash: refsHash},
			Curator: Dynamic{SchemaRef: curSchema.Reference(), Object: curHash},
		}

		val := reflect.ValueOf(grp)

		if err := pack.setupToGo(val); err != nil {
			t.Fatal(err)
		}

		if grp.Leader.isInitialized() == false {
			t.Error("Ref not initialized")
		}

		if grp.Members.isInitialized() == false {
			t.Error("Refs not initialized")
		}

		if grp.Curator.isInitialized() == false {
			t.Error("Dynamic not initialized")
		}

		//

	})

}
