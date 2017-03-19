package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
	//"github.com/skycoin/cxo/data"
)

func getHash(i interface{}) Reference {
	return Reference(cipher.SumSHA256(encoder.Serialize(i)))
}

func getHashes(ary ...interface{}) (rs References) {
	rs = make(References, len(ary))
	for j, x := range ary {
		rs[j] = getHash(x)
	}
	return
}

func pubKey() (pk cipher.PubKey) {
	pk, _ = cipher.GenerateKeyPair()
	return
}

func findSchemaName(sr *Registry, ref Reference) (name string, ok bool) {
	for n, sk := range sr.reg {
		if cipher.SHA256(ref) == sk {
			name, ok = n, true
			break
		}
	}
	return
}

func TestSet_Add(t *testing.T) {
	// doesn't need
}

func TestContainer_Want(t *testing.T) {
	t.Run("all", func(t *testing.T) {
		// c := NewContainer(data.NewDB())
		// root := c.NewRoot(pubKey())
		// root.RegisterSchema("User", User{})
		// root.Set(Group{
		// 	Name: "a group",
		// 	Leader: root.Save(User{
		// 		"Billy Kid", 16, 90,
		// 	}),
		// 	Members: root.SaveArray(
		// 		User{"Bob Marley", 21, 0},
		// 		User{"Alice Cooper", 19, 0},
		// 		User{"Eva Brown", 30, 0},
		// 	),
		// 	Curator: root.Dynamic(Man{
		// 		Name:    "Ned Kelly",
		// 		Age:     28,
		// 		Seecret: []byte("secret key"),
		// 		Owner:   Group{},
		// 		Friends: List{},
		// 	}),
		// })
		// c.AddRoot(root)
		// set, err := root.Want()
		// if err != nil {
		// 	t.Error("unexpected error:", err)
		// 	return
		// }
		// if l := len(set); l != 0 {
		// 	t.Error("unexpects wanted objects: ", l)
		// 	for k := range set {
		// 		t.Error("missing: ", k.String())
		// 	}
		// }
	})
	t.Run("no", func(t *testing.T) {
		// c := NewContainer(data.NewDB())
		// root := c.NewRoot(pubKey())
		// root.RegisterSchema("User", User{})
		// leader := User{
		// 	"Billy Kid", 16, 90,
		// }
		// members := []interface{}{
		// 	User{"Bob Marley", 21, 0},
		// 	User{"Alice Cooper", 19, 0},
		// 	User{"Eva Brown", 30, 0},
		// }
		// root.Set(Group{
		// 	Name:    "a group",
		// 	Leader:  getHash(leader),
		// 	Members: getHashes(members...),
		// 	Curator: root.Dynamic(Man{
		// 		Name:    "Ned Kelly",
		// 		Age:     28,
		// 		Seecret: []byte("secret key"),
		// 		Owner:   Group{},
		// 		Friends: List{},
		// 	}),
		// })
		// c.AddRoot(root)
		// set, err := root.Want()
		// if err != nil {
		// 	t.Error("unexpected error:", err)
		// 	return
		// }
		// if l := len(set); l != 5 {
		// 	t.Error("unexpects count of wanted objects: ", l)
		// }
		// root.Save(leader)
		// set, err = root.Want()
		// if err != nil {
		// 	t.Error("unexpected error:", err)
		// 	return
		// }
		// if l := len(set); l != 4 {
		// 	t.Error("unexpects count of wanted objects: ", l)
		// }
		// root.SaveArray(members...)
		// set, err = root.Want()
		// if err != nil {
		// 	t.Error("unexpected error:", err)
		// 	return
		// }
		// if l := len(set); l != 0 {
		// 	t.Error("unexpects count of wanted objects: ", l)
		// }
	})
}

func TestContainer_wantKeys(t *testing.T) {
	//
}

func TestContainer_wantSchemaObjKey(t *testing.T) {
	//
}

func TestContainer_wantSchemaObjData(t *testing.T) {
	//
}

func TestContainer_wantField(t *testing.T) {
	//
}
