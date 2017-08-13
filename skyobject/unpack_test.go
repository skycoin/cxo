package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

func TestPack_Save(t *testing.T) {

	pk, sk := cipher.GenerateKeyPair()

	t.Run("can't save", func(t *testing.T) {
		c := getCont()
		// no such feed
		pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pack.Save(); err == nil {
			t.Error("missing no-such-feed error")
		}
		if err := c.AddFeed(pk); err != nil {
			t.Fatal(err)
		}
		// empty sec key
		pack, err = c.Unpack(pack.Root(), 0, c.CoreRegistry().Types(),
			cipher.SecKey{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pack.Save(); err == nil {
			t.Error("save with secret key")
		}
		// view only
		pack, err = c.Unpack(pack.Root(), ViewOnly, c.CoreRegistry().Types(),
			sk)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pack.Save(); err == nil {
			t.Error("save with ViewOnly")
		}
	})

	// TODO (kostyarin): Commit error

	t.Run("save", func(t *testing.T) {
		c := getCont()
		if err := c.AddFeed(pk); err != nil {
			t.Fatal(err)
		}
		pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
		if err != nil {
			t.Fatal(err)
		}

		pack.Append(Group{
			Name: "VDPG",
			Leader: pack.Ref(User{
				Name: "Alice",
				Age:  23,
			}),
			Members: pack.Refs(
				User{"Eva", 19, nil},
				User{"Ammy", 17, nil},
				User{"Kate", 21, nil},
			),
			Curator: pack.Dynamic(Developer{
				Name:   "Konstantin",
				GitHub: "logrusorgru",
			}),
		})
		if _, err = pack.Save(); err != nil {
			t.Fatal(err)
		}
		// check out DB
		c.DB().View(func(tx data.Tv) (_ error) {
			// root object
			roots := tx.Feeds().Roots(pk)
			if roots == nil {
				t.Fatal("missing roots")
			}
			rp := roots.Last()
			if rp == nil {
				t.Fatal("misisng root")
			}
			if rp.Seq != pack.Root().Seq {
				t.Fatal("wrong root")
			}
			rt := pack.Root()
			if len(rt.Refs) != 1 {
				t.Fatal("wrong Root.Refs count")
			}
			groupDr := rt.Refs[0]
			// objects
			objs := tx.Objects()
			if groupDr.Object == (cipher.SHA256{}) {
				t.Fatal("wrong Dynaimc")
			}
			if objs.Get(groupDr.Object) == nil {
				t.Fatal("unsaved object")
			}
			// the group
			groupInterface, err := pack.RefByIndex(0)
			if err != nil {
				t.Fatal(err)
			}
			if groupInterface == nil {
				t.Fatal("group is nil")
			}
			group, ok := groupInterface.(*Group)
			if !ok {
				t.Fatal("wrong type")
			}
			if group.Name != "VDPG" {
				t.Fatal("wrong value after saving")
			}
			// leader
			if lead := group.Leader.Hash; lead == (cipher.SHA256{}) {
				t.Fatal("empty ref")
			} else if objs.Get(lead) == nil {
				t.Error("unsaved object")
			}
			leadInterface, err := group.Leader.Value()
			if err != nil {
				t.Fatal(err)
			}
			if leadInterface == nil {
				t.Fatal("nil interface")
			}
			lead, ok := leadInterface.(*User)
			if !ok {
				t.Fatal("wrong type")
			}
			if lead.Name != "Alice" || lead.Age != 23 {
				t.Fatal("wron value after saving", lead)
			}
			// members
			if mms := group.Members.Hash; mms == (cipher.SHA256{}) {
				t.Fatal("empty refs")
			} else if objs.Get(mms) == nil {
				t.Fatal("unsaved refs")
			}
			// explore the refs
			mmsVal := objs.Get(group.Members.Hash)
			var er encodedRefs
			if err := encoder.DeserializeRaw(mmsVal, &er); err != nil {
				t.Fatal(err)
			}
			if len(er.Nested) == 0 {
				t.Fatal("no netsed nodes")
			}
			if er.Depth != 0 {
				t.Error("wrong depth")
			}
			for _, hash := range er.Nested {
				if objs.Get(hash) == nil {
					t.Fatal("missing member")
				}
			}
			// curator
			cur := group.Curator
			if cur.Object == (cipher.SHA256{}) {
				t.Fatal("unsaved dynamic")
			}
			if cur.SchemaRef == (SchemaRef{}) {
				t.Fatal("empty schema ref")
			}
			if objs.Get(cur.Object) == nil {
				t.Fatal("unssaved object")
			}
			curInterface, err := cur.Value()
			if err != nil {
				t.Fatal(err)
			}
			if curInterface == nil {
				t.Fatal("nil interface")
			}
			curg, ok := curInterface.(*Developer)
			if !ok {
				t.Fatal("wrong type")
			}
			if curg.Name != "Konstantin" || curg.GitHub != "logrusorgru" {
				t.Fatal("wrong value")
			}
			return
		})
	})

}

func TestPack_RootRefs(t *testing.T) {
	// RootRefs() (objs []interface{}, err error)

	t.Skip("not implemented") // TODO (kostyarin): implement
}

func TestPack_RefByIndex(t *testing.T) {
	// RefByIndex(i int) (obj interface{}, err error)

	t.Skip("not implemented") // TODO (kostyarin): implement
}

func TestPack_SetRefByIndex(t *testing.T) {
	// SetRefByIndex(i int, obj interface{}) (err error)

	t.Skip("not implemented") // TODO (kostyarin): implement
}

func TestPack_Append(t *testing.T) {
	// Append(objs ...interface{})

	t.Skip("not implemented") // TODO (kostyarin): implement
}

func TestPack_Clear(t *testing.T) {
	// Clear()

	t.Skip("not implemented") // TODO (kostyarin): implement
}

func TestPack_Dynamic(t *testing.T) {
	// Dynamic(obj interface{}) (dr Dynamic)

	t.Skip("not implemented") // TODO (kostyarin): implement
}

func TestPack_Refs(t *testing.T) {
	// Refs(objs ...interface{}) (r Refs)

	t.Skip("not implemented") // TODO (kostyarin): implement
}
