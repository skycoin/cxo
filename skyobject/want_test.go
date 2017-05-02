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

func TestRoot_Want_and_Got(t *testing.T) {
	//
}
