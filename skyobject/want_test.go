package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
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
		if ref == sk {
			name, ok = n, true
			break
		}
	}
	return
}

func TestRoot_WantFunc(t *testing.T) {
	t.Run("all", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		pk, sk := cipher.GenerateKeyPair()
		root := c.NewRoot(pk, sk)
		c.Register("User", User{})
		c.Register("Group", Group{})
		c.Register("List", List{})
		c.Register("Man", Man{})
		root.Inject(Group{
			Name: "a group",
			Leader: c.Save(User{
				"Billy Kid", 16, 90,
			}),
			Members: c.SaveArray(
				User{"Bob Marley", 21, 0},
				User{"Alice Cooper", 19, 0},
				User{"Eva Brown", 30, 0},
			),
			Curator: c.Dynamic(Man{
				Name:    "Ned Kelly",
				Age:     28,
				Seecret: []byte("secret key"),
				Owner:   Group{},
				Friends: List{},
			}),
		}, sk)
		var i int
		err := root.WantFunc(func(hash Reference) error {
			i++
			return nil
		})
		if err != nil {
			t.Error("unexpected error:", err)
			return
		}
		if i != 0 {
			t.Error("unexpects wanted objects: ", i)
		}
	})
	t.Run("no", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		pk, sk := cipher.GenerateKeyPair()
		root := c.NewRoot(pk, sk)
		c.Register("User", User{})   // 1
		c.Register("Group", Group{}) // +1 -> 2
		c.Register("List", List{})   // +1 -> 3
		c.Register("Man", Man{})     // +1 -> 4
		leader := User{
			"Billy Kid", 16, 90,
		}
		members := []interface{}{
			User{"Bob Marley", 21, 0},
			User{"Alice Cooper", 19, 0},
			User{"Eva Brown", 30, 0},
		}
		root.Inject(Group{
			Name:    "a group",
			Leader:  getHash(leader),
			Members: getHashes(members...),
			Curator: c.Dynamic(Man{
				Name:    "Ned Kelly",
				Age:     28,
				Seecret: []byte("secret key"),
				Owner:   Group{},
				Friends: List{},
			}),
		}, sk)
		var i int
		wantFunc := func(hash Reference) (_ error) {
			i++
			return
		}
		err := root.WantFunc(wantFunc)
		if err != nil {
			t.Error("unexpected error:", err)
			return
		}
		if i != 4 {
			t.Error("unexpects count of wanted objects: ", i)
		}
		c.Save(leader)
		err = root.WantFunc(wantFunc)
		if err != nil {
			t.Error("unexpected error:", err)
			return
		}
		if i != 3 {
			t.Error("unexpects count of wanted objects: ", i)
		}
		c.SaveArray(members...)
		err = root.WantFunc(wantFunc)
		if err != nil {
			t.Error("unexpected error:", err)
			return
		}
		if i != 0 {
			t.Error("unexpects count of wanted objects: ", i)
		}
	})
}
