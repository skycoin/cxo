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
	// TODO:
	// if len(r.reg.reg) != len(re.Reg) {
	// 	t.Error("wrong Reg length")
	// }
	// for _, ent := range re.Reg {
	// 	if r.reg.reg[ent.K] != ent.V {
	// 		t.Error("wrong entity ", ent.K)
	// 	}
	// }
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
	}
	// else if len(c2.reg.reg) != len(c1.reg.reg) {
	// 	t.Error("wrong registry")
	// }
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
	// # registered schemas
	//
	// f170128 <- User
	// f885957 <- Group
	// 715fd47 <- List
	// c4b24cf <- Man
	//
	// # got
	//
	//  1 f885957 <- Group (s) Group (Injected) (have)
	//  2 f91b1eb          (o)                  (have)
	//  3 f170128 <- User (s) Leader            (have)
	//  4 586bafc         (o)
	//  5 f170128 <- User (s) Members           (have)
	//  6 06e6d9e         (o)
	//  7 60a5738         (o)
	//  8 295e985         (o)
	//  9 c4b24cf <- Man (s) Curator            (have)
	// 10 7ef26f6        (o)
	// 11 f885957 <- Group (s) Curator/Owner            (have)
	// 12 f170128 <- User  (s) Curator/Owner/Leader     (have)
	// 13 f170128 <- User  (s) Curator/Owner/Members    (have)
	// 14 715fd47 <- List  (s) Curator/Friends          (have)
	// 15 f170128 <- User  (s) Curator/Friends/Members  (have)
	// 16 f885957 <- Group (s) Curator/Friends/MemberOf (have)
	// total:  9
	t.Run("all", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		pk, sk := cipher.GenerateKeyPair()
		root := c.NewRoot(pk, sk)
		c.Register("User", User{})
		c.Register("Group", Group{})
		c.Register("List", List{})
		c.Register("Man", Man{})
		leader := User{
			"Billy Kid", 16, 90,
		}
		members := []interface{}{
			User{"Bob Marley", 21, 0},
			User{"Alice Cooper", 19, 0},
			User{"Eva Brown", 30, 0},
		}
		curator := Man{
			Name:    "Ned Kelly",
			Age:     28,
			Seecret: []byte("secret key"),
			Owner:   Group{},
			Friends: List{},
		}
		group := Group{
			Name:   "a group",
			Leader: c.Save(leader),
			Members: References{
				c.Save(members[0]),
				c.Save(members[1]),
				c.Save(members[2]),
			},
			Curator: Dynamic{
				c.SchemaReference(Man{}),
				c.Save(curator),
			},
		}
		root.Inject(group, sk)
		order := References{
			//  1 f885957 <- Group (s) Group (Injected) (have)
			c.SchemaReference(Group{}),
			//  2 f91b1eb          (o)                  (have)
			getHash(group),
			//  3 f170128 <- User (s) Leader            (have)
			c.SchemaReference(User{}),
			//  4 586bafc         (o)
			getHash(leader),
			//  5 f170128 <- User (s) Members           (have)
			c.SchemaReference(User{}),
			//  6 06e6d9e         (o)
			getHash(members[0]),
			//  7 60a5738         (o)
			getHash(members[1]),
			//  8 295e985         (o)
			getHash(members[2]),
			//  9 c4b24cf <- Man (s) Curator            (have)
			c.SchemaReference(Man{}),
			// 10 7ef26f6        (o)
			getHash(curator),
			// 11 f885957 <- Group (s) Curator/Owner            (have)
			c.SchemaReference(Group{}),
			// 12 f170128 <- User  (s) Curator/Owner/Leader     (have)
			c.SchemaReference(User{}),
			// 13 f170128 <- User  (s) Curator/Owner/Members    (have)
			c.SchemaReference(User{}),
			// 14 715fd47 <- List  (s) Curator/Friends          (have)
			c.SchemaReference(List{}),
			// 15 f170128 <- User  (s) Curator/Friends/Members  (have)
			c.SchemaReference(User{}),
			// 16 f885957 <- Group (s) Curator/Friends/MemberOf (have)
			c.SchemaReference(Group{}),
		}
		var i int
		gotFunc := func(hash Reference) (_ error) {
			if hash != order[i] {
				t.Error("fatality:",
					i+1,
					shortHex(hash.String()),
					shortHex(order[i].String()))
			}
			i++
			return
		}
		if err := root.GotFunc(gotFunc); err != nil {
			t.Fatal("unexpected error:", err)
		}
		if i != 16 {
			t.Error("unexpected amount of objects: ", i)
		}
	})
	t.Run("no", func(t *testing.T) {
		t.Skip("NOT IMPLEMENTED")
		c := NewContainer(data.NewDB())
		pk, sk := cipher.GenerateKeyPair()
		root := c.NewRoot(pk, sk)
		c.Register("User", User{})
		c.Register("Group", Group{})
		c.Register("List", List{})
		c.Register("Man", Man{})
		leader := User{
			"Billy Kid", 16, 90,
		}
		members := []interface{}{
			User{"Bob Marley", 21, 0},
			User{"Alice Cooper", 19, 0},
			User{"Eva Brown", 30, 0},
		}
		curator := Man{
			Name:    "Ned Kelly",
			Age:     28,
			Seecret: []byte("secret key"),
			Owner:   Group{},
			Friends: List{},
		}
		group := Group{
			Name:    "a group",
			Leader:  getHash(leader),
			Members: getHashes(members...),
			Curator: Dynamic{
				c.SchemaReference(Man{}),
				getHash(curator),
			},
		}
		root.Inject(group, sk)
		encoded := root.Encode()
		// use anotuer container toreceive the data
		receiver := getCont() //
		ok, err := receiver.AddEncodedRoot(encoded, pk, root.Sig)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("can't add encoded root")
		}
		// =====================================================================
		// stage 1 (want schema of Group)
		// =====================================================================
		//  1 f885957 <- Group (s) Group (Injected)
		//    - no schema, no traversing
		// total: 1
		var i int = 0
		order := []Reference{
			//  1 f885957 <- Group (s) Group (Injected)
			c.SchemaReference(Group{}),
		}
		wantFunc := func(hash Reference) (_ error) {
			if hash != order[i] {
				t.Error("fatality:",
					i+1,
					shortHex(hash.String()),
					shortHex(order[i].String()))
			}
			i++
			return
		}
		receiverRoot := receiver.Root(pk)
		if err := receiverRoot.WantFunc(wantFunc); err != nil {
			t.Fatal("unexpected error:", err)
		}
		if i != 1 {
			t.Error("unexpects count of wanted objects: ", i)
		}
		// =====================================================================
		// stage 2 (want the root Group)
		// =====================================================================
		//  1 f885957 <- Group (s) Group (Injected)               (already have)
		//  2 f91b1eb          (o)
		//    - no object, no traversing
		// total: 1
		data, _ := c.get(c.reg.reg["Group"]) // receive schema of the Group
		receiver.db.AddAutoKey(data)         // ---------------------------
		i = 0
		order = []Reference{
			//  2 f91b1eb          (o)
			getHash(group),
		}
		if err := receiverRoot.WantFunc(wantFunc); err != nil {
			t.Fatal("unexpected error:", err)
		}
		if i != 1 {
			t.Error("unexpects count of wanted objects: ", i)
		}
		// =====================================================================
		// stage 3 (what schema of User)
		// =====================================================================
		//  1 f885957 <- Group (s) Group (Injected)               (already have)
		//  2 f91b1eb          (o)                                (already have)
		//  3 f170128 <- User (s) Leader
		//    - no schema, no traversing more
		// total: 1
		receiver.Save(group) // receive the root Group
		// -------------------------------------------
		i = 0
		order = []Reference{
			//  3 f170128 <- User (s) Leader
			c.SchemaReference(User{}),
		}
		if err := receiverRoot.WantFunc(wantFunc); err != nil {
			t.Fatal("unexpected error:", err)
		}
		if i != 1 {
			t.Error("unexpects count of wanted objects: ", i)
		}
		// =====================================================================
		// stage 4 (want Group/User objects + schema of Man)
		// =====================================================================
		//  1 f885957 <- Group (s) Group (Injected)               (already have)
		//  2 f91b1eb          (o)                                (already have)
		//  3 f170128 <- User (s) Leader                          (already have)
		//  4 586bafc         (o)
		//  5 f170128 <- User (s) Members                         (already have)
		//  6 06e6d9e         (o)
		//  7 60a5738         (o)
		//  8 295e985         (o)
		//  9 c4b24cf <- Man (s) Curator
		//    - no schema, no traversing more
		// total: 5
		data, _ = c.get(c.reg.reg["User"]) // receive schema of the User
		receiver.db.AddAutoKey(data)       // ---------------------------
		i = 0
		order = []Reference{
			//  4 586bafc         (o)
			getHash(leader),
			//  6 06e6d9e         (o)
			getHash(members[0]),
			//  7 60a5738         (o)
			getHash(members[1]),
			//  8 295e985         (o)
			getHash(members[2]),
			//  9 c4b24cf <- Man (s) Curator
			c.SchemaReference(Man{}),
		}
		if err := receiverRoot.WantFunc(wantFunc); err != nil {
			t.Fatal("unexpected error:", err)
		}
		if i != 5 {
			t.Error("unexpects count of wanted objects: ", i)
		}
		// =====================================================================
		// stage 5 (want Man object)
		// =====================================================================
		//  1 f885957 <- Group (s) Group (Injected)               (already have)
		//  2 f91b1eb          (o)                                (already have)
		//  3 f170128 <- User (s) Leader                          (already have)
		//  4 586bafc         (o)                                 (already have)
		//  5 f170128 <- User (s) Members                         (already have)
		//  6 06e6d9e         (o)                                 (already have)
		//  7 60a5738         (o)                                 (already have)
		//  8 295e985         (o)                                 (already have)
		//  9 c4b24cf <- Man (s) Curator                          (already have)
		// 10 7ef26f6        (o)
		//    - no object, no traversing more
		// total: 1
		data, _ = c.get(c.reg.reg["Man"]) // receive schema of the Man,
		receiver.db.AddAutoKey(data)      // Leader, and all Members
		receiver.Save(leader)             //
		receiver.Save(members[0])         //
		receiver.Save(members[1])         //
		receiver.Save(members[2])         // ------------------------------
		i = 0
		order = []Reference{
			// 10 7ef26f6        (o)
			getHash(curator),
		}
		if err := receiverRoot.WantFunc(wantFunc); err != nil {
			t.Fatal("unexpected error:", err)
		}
		if i != 1 {
			t.Error("unexpects count of wanted objects: ", i)
		}
		// =====================================================================
		// stage 6 (want schema of List)
		// =====================================================================
		//  1 f885957 <- Group (s) Group (Injected)               (already have)
		//  2 f91b1eb          (o)                                (already have)
		//  3 f170128 <- User (s) Leader                          (already have)
		//  4 586bafc         (o)                                 (already have)
		//  5 f170128 <- User (s) Members                         (already have)
		//  6 06e6d9e         (o)                                 (already have)
		//  7 60a5738         (o)                                 (already have)
		//  8 295e985         (o)                                 (already have)
		//  9 c4b24cf <- Man (s) Curator                          (already have)
		// 10 7ef26f6        (o)                                  (already have)
		// 11 f885957 <- Group (s) Curator/Owner                  (already have)
		// 12 f170128 <- User  (s) Curator/Owner/Leader           (already have)
		// 13 f170128 <- User  (s) Curator/Owner/Members          (already have)
		// 14 715fd47 <- List  (s) Curator/Friends
		// 15 f170128 <- User  (s) Curator/Friends/Members        (already have)
		// 16 f885957 <- Group (s) Curator/Friends/MemberOf       (already have)
		// total: 1
		receiver.Save(curator) // receive the Man
		// --------------------------------------
		i = 0
		order = []Reference{
			// 14 715fd47 <- List  (s) Curator/Friends
			c.SchemaReference(List{}),
		}
		if err := receiverRoot.WantFunc(wantFunc); err != nil {
			t.Fatal("unexpected error:", err)
		}
		if i != 1 {
			t.Error("unexpects count of wanted objects: ", i)
		}
		// =====================================================================
		// stage 7 (has got everything need)
		// =====================================================================
		//  1 f885957 <- Group (s) Group (Injected)               (already have)
		//  2 f91b1eb          (o)                                (already have)
		//  3 f170128 <- User (s) Leader                          (already have)
		//  4 586bafc         (o)                                 (already have)
		//  5 f170128 <- User (s) Members                         (already have)
		//  6 06e6d9e         (o)                                 (already have)
		//  7 60a5738         (o)                                 (already have)
		//  8 295e985         (o)                                 (already have)
		//  9 c4b24cf <- Man (s) Curator                          (already have)
		// 10 7ef26f6        (o)                                  (already have)
		// 11 f885957 <- Group (s) Curator/Owner                  (already have)
		// 12 f170128 <- User  (s) Curator/Owner/Leader           (already have)
		// 13 f170128 <- User  (s) Curator/Owner/Members          (already have)
		// 14 715fd47 <- List  (s) Curator/Friends                (already have)
		// 15 f170128 <- User  (s) Curator/Friends/Members        (already have)
		// 16 f885957 <- Group (s) Curator/Friends/MemberOf       (already have)
		// total: 1
		data, _ = c.get(c.reg.reg["List"]) // receive schema of the List
		receiver.db.AddAutoKey(data)       // --------------------------
		i = 0
		order = []Reference{}
		if err := receiverRoot.WantFunc(wantFunc); err != nil {
			t.Fatal("unexpected error:", err)
		}
		if i != 0 {
			t.Error("unexpects count of wanted objects: ", i)
		}
	})
}
