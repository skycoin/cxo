package skyobject

import (
	"bytes"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

func TestRoot_Touch(t *testing.T) {
	// TODO: mid. priority
}

func TestRoot_Inject(t *testing.T) {
	// TODO: mid. priority
}

func Test_signature(t *testing.T) {
	pub, sec := cipher.GenerateKeyPair()
	b := []byte("hello")
	hash := cipher.SumSHA256(b)
	sig := cipher.SignHash(hash, sec)
	if err := cipher.VerifySignature(pub, sig, hash); err != nil {
		t.Error(err)
	}
}

func Test_encodeEqual(t *testing.T) {
	c := getCont()
	pk, sk := cipher.GenerateKeyPair()
	r := c.NewRoot(pk, sk)
	p := r.Encode()
	if bytes.Compare(p, r.Encode()) != 0 {
		t.Error("encode produce different result")
	}
	c.Register("User", User{})
	p = r.Encode()
	if bytes.Compare(p, r.Encode()) != 0 {
		t.Error("encode produce different result")
	}
	c.Register("Group", Group{})
	p = r.Encode()
	if bytes.Compare(p, r.Encode()) != 0 {
		t.Error("encode produce different result")
	}
	c.Register("List", List{})
	p = r.Encode()
	if bytes.Compare(p, r.Encode()) != 0 {
		t.Error("encode produce different result")
	}
	c.Register("Man", Man{})
	p = r.Encode()
	if bytes.Compare(p, r.Encode()) != 0 {
		t.Error("encode produce different result")
	}
}

func Test_encodeDecode(t *testing.T) {
	c := getCont()
	pk, sk := cipher.GenerateKeyPair()
	r := c.NewRoot(pk, sk)
	c.Register("User", User{})
	p := r.Encode()
	re, err := decodeRoot(p)
	if err != nil {
		t.Error(err)
		return
	}
	if re.Time != r.Time {
		t.Error("wrong time")
	}
	if re.Seq != r.Seq {
		t.Error("wrong seq")
	}
	if len(re.Refs) != len(r.Refs) {
		t.Error("wrong Refs length")
	}
	for i, ref := range re.Refs {
		if r.Refs[i] != ref {
			t.Error("wrong reference", i)
		}
	}
	if len(r.reg.reg) != len(re.Reg) {
		t.Error("wrong Reg length")
	}
	for _, ent := range re.Reg {
		if r.reg.reg[ent.K] != ent.V {
			t.Error("wrong entity ", ent.K)
		}
	}
}

func TestRoot_Encode(t *testing.T) {
	pub, sec := cipher.GenerateKeyPair()
	// encode
	c1 := getCont()
	r1 := c1.NewRoot(pub, sec)
	c1.Register("User", User{})
	c1.Register("Group", Group{})
	r1.Sign(sec)
	sig := r1.Sig
	p := r1.Encode()
	if r1.Pub != pub {
		t.Error("pub key was changed during encoding")
	}
	if r1.Sig != sig {
		t.Error("signature was changed during encoding")
	}
	// decode
	c2 := getCont()
	if ok, err := c2.AddEncodedRoot(p, r1.Pub, r1.Sig); err != nil {
		t.Error(err)
	} else if !ok {
		t.Error("can't set encoded root")
	} else if len(c2.reg.reg) != len(c1.reg.reg) {
		t.Error("wrong registry")
	}
}

func TestRoot_Values(t *testing.T) {
	//
}

func TestRoot_GotFunc(t *testing.T) {
	t.Run("all", func(t *testing.T) {
		// # registered schemas
		//
		// f170128 <- User
		// f885957 <- Group
		// 715fd47 <- List
		// c4b24cf <- Man
		//
		// # got
		//
		//  1 f885957 <- Group (s) Group (Injected)
		//  2 f91b1eb          (o)
		//  3 f170128 <- User (s) Leader
		//  4 586bafc         (o)
		//  5 f170128 <- User (s) Members
		//  6 06e6d9e         (o)
		//  7 60a5738         (o)
		//  8 295e985         (o)
		//  9 c4b24cf <- Man (s) Curator
		// 10 7ef26f6        (o)
		// 11 f885957 <- Group Curator/Owner
		// 12 f170128 <- User  Curator/Owner/Leader
		// 13 f170128 <- User  Curator/Owner/Members
		// 14 715fd47 <- List  Curator/Friends
		// 15 f170128 <- User  Curator/Friends/Members
		// 16 f885957 <- Group Curator/Friends/MemberOf
		// total:  16
		c := NewContainer(data.NewDB())
		pk, sk := cipher.GenerateKeyPair()
		root := c.NewRoot(pk, sk)
		c.Register("User", User{})
		c.Register("Group", Group{})
		c.Register("List", List{})
		c.Register("Man", Man{})
		t.Log("registered schemas")
		for n, r := range c.reg.reg {
			t.Logf(" - %q %s", n, shortHex(r.String()))
		}
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
		err := root.GotFunc(func(hash Reference) (_ error) {
			i++
			t.Logf("  %d hash of GotFunc is %s",
				i,
				shortHex(hash.String()))
			return
		})
		if err != nil {
			t.Error("unexpected error:", err)
			return
		}
		if i != 16 {
			t.Error("unexpects got objects: ", i)
		}
	})
	t.Run("no", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		pk, sk := cipher.GenerateKeyPair()
		root := c.NewRoot(pk, sk)
		c.Register("User", User{})
		c.Register("Group", Group{})
		c.Register("List", List{})
		c.Register("Man", Man{})
		t.Log("registered schemas")
		for n, r := range c.reg.reg {
			t.Logf(" - %q %s", n, shortHex(r.String()))
		}
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
		var i int = 0
		gotFunc := func(hash Reference) (_ error) {
			i++
			t.Logf("  %d hash of GotFunc is %s",
				i,
				shortHex(hash.String()))
			return
		}
		if err := root.GotFunc(gotFunc); err != nil {
			t.Error("unexpected error:", err)
		}
		if i != 3 {
			t.Error("unexpected count of got objects: ", i)
		}
		c.Save(leader)
		i = 0
		if err := root.GotFunc(gotFunc); err != nil {
			t.Error("unexpected error:", err)
		}
		if i != 13 {
			t.Error("unexpected count of got objects: ", i)
		}
		c.SaveArray(members...)
		i = 0
		if err := root.GotFunc(gotFunc); err != nil {
			t.Error("unexpected error:", err)
		}
		if i != 16 {
			t.Error("unexpected count of got objects: ", i)
		}
	})
}

func shortHex(a string) string {
	return string([]byte(a)[:7])
}

func TestRoot_GotFunc_order(t *testing.T) {
	// TODO: low. proirity
}
