package skyobject

import (
	"bytes"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

func TestRoot_Touch(t *testing.T) {
	//
}

func TestRoot_Inject(t *testing.T) {
	//
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
	r := c.NewRoot(pubKey())
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
	r := c.NewRoot(pubKey())
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
	r1 := c1.NewRoot(pub)
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

func TestRoot_Got(t *testing.T) {
	t.Run("all", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		pk, sk := cipher.GenerateKeyPair()
		root := c.NewRoot(pk)
		c.Register("User", User{})   // s: 1
		c.Register("Group", Group{}) // s: +1 -> 2
		c.Register("List", List{})   // s: +1 -> 3
		c.Register("Man", Man{})     // s: +1 -> 4
		root.Inject(Group{           // o: +1 -> 5
			Name: "a group",
			Leader: c.Save(User{ // o: +1 -> 6
				"Billy Kid", 16, 90,
			}),
			Members: c.SaveArray(
				User{"Bob Marley", 21, 0},   // o: +1 -> 7
				User{"Alice Cooper", 19, 0}, // o: +1 -> 8
				User{"Eva Brown", 30, 0},    // o: +1 -> 9
			),
			Curator: c.Dynamic(Man{ // o: +2 -> 10
				Name:    "Ned Kelly",
				Age:     28,
				Seecret: []byte("secret key"),
				Owner:   Group{},
				Friends: List{},
			}),
		})
		c.AddRoot(root, sk)
		set, err := root.Got()
		if err != nil {
			t.Error("unexpected error:", err)
			return
		}
		if l := len(set); l != 10 {
			t.Error("unexpects got objects: ", l)
		}
	})
	t.Run("no", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		pk, sk := cipher.GenerateKeyPair()
		root := c.NewRoot(pk)
		c.Register("User", User{})   // s: +1
		c.Register("Group", Group{}) // s: +1 -> 2
		c.Register("List", List{})   // s: +1 -> 3
		c.Register("Man", Man{})     // s: +1 -> 4
		leader := User{
			"Billy Kid", 16, 90,
		}
		members := []interface{}{
			User{"Bob Marley", 21, 0},
			User{"Alice Cooper", 19, 0},
			User{"Eva Brown", 30, 0},
		}
		root.Inject(Group{ // o: +1 -> 5
			Name:    "a group",
			Leader:  getHash(leader),
			Members: getHashes(members...),
			Curator: c.Dynamic(Man{ // o: +1 -> 6
				Name:    "Ned Kelly",
				Age:     28,
				Seecret: []byte("secret key"),
				Owner:   Group{},
				Friends: List{},
			}),
		})
		c.AddRoot(root, sk)
		set, _ := root.Got()
		if l := len(set); l != 6 {
			t.Error("unexpects count of got objects: ", l)
		}
		c.Save(leader)
		set, _ = root.Got()
		if l := len(set); l != 7 {
			t.Error("unexpects count of got objects: ", l)
		}
		c.SaveArray(members...)
		set, _ = root.Got()
		if l := len(set); l != 10 {
			t.Error("unexpects count of got objects: ", l)
		}
	})
}
