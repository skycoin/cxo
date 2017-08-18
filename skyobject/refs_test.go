package skyobject

import (
	"fmt"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func TestRefs_IsBlank(t *testing.T) {
	// IsBlank() bool

	// TODO (kostyarin): low priority
	t.Skip("not implemented (low priority)")
}

func TestRefs_Eq(t *testing.T) {
	// Eq(x *Refs) bool

	// TODO (kostyarin): low priority
	t.Skip("not implemented (low priority)")
}

func TestRefs_Short(t *testing.T) {
	// Short() string

	// TODO (kostyarin): low priority
	t.Skip("not implemented (low priority)")
}

func TestRefs_String(t *testing.T) {
	// String() string

	// TODO (kostyarin): low priority
	t.Skip("not implemented (low priority)")
}

func TestRefs_Len(t *testing.T) {
	// Len() (ln int, err error)

	// detached
	var refs Refs
	if _, err := refs.Len(); err == nil {
		t.Error("missing error")
	}

	c := getCont()
	defer c.db.Close()
	defer c.Close()

	pk, sk := cipher.GenerateKeyPair()
	if err := c.AddFeed(pk); err != nil {
		t.Fatal(err)
	}

	pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
	if err != nil {
		t.Fatal(err)
	}

	// blank
	refs = pack.Refs()
	if ln, err := refs.Len(); err != nil {
		t.Error(err)
	} else if ln != 0 {
		t.Error("wrong length")
	}

	// not found in DB
	refs = pack.Refs()
	refs.Hash = cipher.SHA256{1, 2, 3} // fictive hash

	if _, err := refs.Len(); err == nil {
		t.Error("missing error")
	}

	// not found in DB (use HashTableIndex flag)
	pack.SetFlag(HashTableIndex)
	if _, err := refs.Len(); err == nil {
		t.Error("missing error")
	} else if refs.rn.index == nil {
		t.Error("index has not been created")
	}
	pack.UnsetFlag(HashTableIndex)
	refs.rn.index = nil

	if refs.degree != c.conf.MerkleDegree {
		t.Error("unexpected degree of new created Refs", refs.degree)
	}

	// the Refs is root that contains leafs (degree = 8)
	if c.conf.MerkleDegree < 3 {
		t.Fatal("set proper degree manually or chagne default configs")
	}

	alice, ammy, eva := &User{"Alice", 21, nil}, &User{"Ammy", 17, nil},
		&User{"Eva", 23, nil}

	refs = pack.Refs(alice, ammy, eva)

	if ln, err := refs.Len(); err != nil {
		t.Error(err)
	} else if ln != 3 {
		t.Error("wrong length:", ln)
	}

	// reduce the degree and repeat to force the Refs has branches
	// this way it calls: cahngeDepth and then Append

	c.conf.MerkleDegree = 2

	refs = pack.Refs(alice, ammy, eva)

	if ln, err := refs.Len(); err != nil {
		t.Error(err)
	} else if ln != 3 {
		t.Error("wrong length:", ln)
	}

	// load from "database"
	refs.length, refs.depth = 0, 0
	refs.branches, refs.leafs, refs.rn.index = nil, nil, nil

	if ln, err := refs.Len(); err != nil {
		t.Error(err)
	} else if ln != 3 {
		t.Error("wrong length:", ln)
	}

}

func TestRefs_RefByIndex(t *testing.T) {
	// RefByIndex(i int) (ref *Ref, err error)

	c := getCont()
	defer c.db.Close()
	defer c.Close()

	pk, sk := cipher.GenerateKeyPair()
	if err := c.AddFeed(pk); err != nil {
		t.Fatal(err)
	}

	pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
	if err != nil {
		t.Fatal(err)
	}

	alice, ammy, eva := &User{"Alice", 21, nil}, &User{"Ammy", 17, nil},
		&User{"Eva", 23, nil}

	c.conf.MerkleDegree = 16 // for sure

	refs := pack.Refs(alice, ammy, eva)

	// invalid index (negative)
	if _, err := refs.RefByIndex(-3); err == nil {
		t.Error("missing error")
	}

	// invalid index (out of range)
	if _, err := refs.RefByIndex(10); err == nil {
		t.Error("missing error")
	}

	// the users are direct childrens of the refs
	for i, u := range []*User{
		alice,
		ammy,
		eva,
	} {
		if ref, err := refs.RefByIndex(i); err != nil {
			t.Error(err)
		} else if valInterface, err := ref.Value(); err != nil {
			t.Error(err)
		} else if valInterface == nil {
			t.Error("got nil-interface{}")
		} else if usr, ok := valInterface.(*User); !ok {
			t.Errorf("unexpected type %T", valInterface)
		} else if usr != u {
			t.Error("wrong pointer")
		}
	}

	// force refs to have branches (not direct leafs)
	c.conf.MerkleDegree = 2
	refs = pack.Refs(alice, ammy, eva) // recreate

	// the users are childrens of branches of the refs
	for i, u := range []*User{
		alice,
		ammy,
		eva,
	} {
		if ref, err := refs.RefByIndex(i); err != nil {
			t.Error(err)
		} else if valInterface, err := ref.Value(); err != nil {
			t.Error(err)
		} else if valInterface == nil {
			t.Error("got nil-interface{}")
		} else if usr, ok := valInterface.(*User); !ok {
			t.Errorf("unexpected type %T", valInterface)
		} else if usr != u {
			t.Error("wrong pointer")
		}
	}

}

func TestRefs_RefByHash(t *testing.T) {
	// RefByHash(hash cipher.SHA256) (needle *Ref, err error)

	c := getCont()
	defer c.db.Close()
	defer c.Close()

	pk, sk := cipher.GenerateKeyPair()
	if err := c.AddFeed(pk); err != nil {
		t.Fatal(err)
	}

	pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
	if err != nil {
		t.Fatal(err)
	}

	alice, ammy, eva := &User{"Alice", 21, nil}, &User{"Ammy", 17, nil},
		&User{"Eva", 23, nil}

	hashes := []cipher.SHA256{}

	for _, u := range []*User{
		alice,
		ammy,
		eva,
	} {
		key, _ := pack.dsave(u)
		hashes = append(hashes, key)
	}

	// using HashTableIndex
	pack.SetFlag(HashTableIndex)
	refs := pack.Refs(alice, ammy, eva)

	for i, u := range []*User{
		alice,
		ammy,
		eva,
	} {
		if ref, err := refs.RefByHash(hashes[i]); err != nil {
			t.Error(err)
		} else if valInterface, err := ref.Value(); err != nil {
			t.Error(err)
		} else if valInterface == nil {
			t.Error("got nil-interface{}")
		} else if usr, ok := valInterface.(*User); !ok {
			t.Errorf("unexpected type %T", valInterface)
		} else if usr != u {
			t.Error("wrong pointer")
		}
	}

	// not found (HashTableIndex)
	if ref, err := refs.RefByHash(cipher.SHA256{1, 2, 3}); err != nil {
		t.Error(err)
	} else if ref != nil {
		t.Error("found")
	}

	// itterate ascending
	pack.UnsetFlag(HashTableIndex)
	refs = pack.Refs()
	if err := refs.Append(alice, ammy, eva); err != nil {
		t.Fatal(err)
	}

	for i, u := range []*User{
		alice,
		ammy,
		eva,
	} {
		if ref, err := refs.RefByHash(hashes[i]); err != nil {
			t.Error(err)
		} else if valInterface, err := ref.Value(); err != nil {
			t.Error(err)
		} else if valInterface == nil {
			t.Error("got nil-interface{}")
		} else if usr, ok := valInterface.(*User); !ok {
			t.Errorf("unexpected type %T", valInterface)
		} else if usr != u {
			t.Error("wrong pointer")
		}
	}

	// not found (itterate ascending)
	if ref, err := refs.RefByHash(cipher.SHA256{1, 2, 3}); err != nil {
		t.Error(err)
	} else if ref != nil {
		t.Error("found")
	}

}

func TestRefs_RefByHashWithIndex(t *testing.T) {
	// RefByHashWithIndex(cipher.SHA256) (int, *RefsElem, error)

	c := getCont()
	defer c.db.Close()
	defer c.Close()

	pk, sk := cipher.GenerateKeyPair()
	if err := c.AddFeed(pk); err != nil {
		t.Fatal(err)
	}

	pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
	if err != nil {
		t.Fatal(err)
	}

	var users []interface{}
	for i := 0; i < 100; i++ {
		users = append(users, &User{
			Name: fmt.Sprintf("User %d", i),
			Age:  (uint32(i) % 30) | 15,
		})
	}

	var getHash = func(i interface{}) cipher.SHA256 {
		return cipher.SumSHA256(encoder.Serialize(i))
	}

	refs := pack.Refs(users...)

	for i, u := range users {
		hash := getHash(u)
		k, needle, err := refs.RefByHashWithIndex(hash)
		if err != nil {
			t.Error(err)
			continue
		}
		if needle == nil {
			t.Error("not found")
			continue
		}
		if valInterface, err := needle.Value(); err != nil {
			t.Error(err)
		} else if usr, ok := valInterface.(*User); !ok {
			t.Errorf("wrong type %T", valInterface)
		} else if usr != u.(*User) {
			t.Error("wrong pointer")
		}
		if k != i {
			t.Error("wrong index")
		}
	}

	// delete odd
	var shift int
	var i int
	for _, u := range users {
		if i%2 == 0 {
			continue
		}
		el, err := refs.RefByIndex(i - shift)
		if err != nil {
			t.Error(err)
			continue
		}
		if err := el.Delete(); err != nil {
			t.Error(err)
		}
		shift++
		users[i] = u
	}
	users = users[:i] // delete

	// and try again
	for i, u := range users {
		hash := getHash(u)
		k, needle, err := refs.RefByHashWithIndex(hash)
		if err != nil {
			t.Error(err)
			continue
		}
		if needle == nil {
			t.Error("not found")
			continue
		}
		if valInterface, err := needle.Value(); err != nil {
			t.Error(err)
		} else if usr, ok := valInterface.(*User); !ok {
			t.Errorf("wrong type %T", valInterface)
		} else if usr != u.(*User) {
			t.Error("wrong pointer")
		}
		if k != i {
			t.Error("wrong index")
		}
	}

}

func TestRefs_Ascend(t *testing.T) {
	// Ascend(irf IterateRefsFunc) (err error)

	c := getCont()
	defer c.db.Close()
	defer c.Close()

	pk, sk := cipher.GenerateKeyPair()
	if err := c.AddFeed(pk); err != nil {
		t.Fatal(err)
	}

	pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
	if err != nil {
		t.Fatal(err)
	}

	alice, ammy, eva := &User{"Alice", 21, nil}, &User{"Ammy", 17, nil},
		&User{"Eva", 23, nil}

	// direct leafs
	c.conf.MerkleDegree = 16 // for sure
	refs := pack.Refs(alice, ammy, eva)
	t.Log(refs.DebugString(false))

	users := []*User{alice, ammy, eva}

	err = refs.Ascend(func(i int, el *RefsElem) (_ error) {
		if el.Hash == (cipher.SHA256{}) {
			t.Error("empty Hash", i)
		}
		if i < 0 || i >= len(users) {
			t.Error("index out of range:", i)
			return ErrStopIteration
		}
		u := users[i]

		if valInterface, err := el.Value(); err != nil {
			t.Error(err)
		} else if valInterface == nil {
			t.Error("got nil-interface{}")
		} else if usr, ok := valInterface.(*User); !ok {
			t.Errorf("unexpected type %T", valInterface)
		} else if usr != u {
			t.Error("wrong pointer", usr)
		}

		if el.upper != &refs {
			t.Error("upper of RefsElem not points to Refs")
		}

		return
	})
	if err != nil {
		t.Error(err)
	}

	// remove ammy and try again
	if el, err := refs.RefByIndex(1); err != nil {
		t.Fatal(err)
	} else if el.rn == nil {
		t.Fatal("not initialized")
	} else if up := el.upper; up == nil {
		t.Fatal("missing upper of RefsElem in Refs")
	} else if up != &refs {
		t.Logf("up:   %#v", up)
		t.Logf("refs: %#v", refs)
		t.Fatalf("wrong upper Refs: %s:%p - %s:%p", up.Short(), up,
			refs.Short(), &refs)
	} else if err = el.SetValue(nil); err != nil {
		t.Fatal(err)
	}

	if ln, err := refs.Len(); err != nil {
		t.Fatal(err)
	} else if ln != 2 {
		t.Fatal("wrong length:", ln)
	}
	t.Log(refs.DebugString(false))

	users = []*User{alice, eva}

	err = refs.Ascend(func(i int, el *RefsElem) (_ error) {
		if el.Hash == (cipher.SHA256{}) {
			t.Error("empty Hash", i)
		}
		if i < 0 || i >= len(users) {
			t.Error("index out of range:", i)
			return ErrStopIteration
		}
		u := users[i]
		if valInterface, err := el.Value(); err != nil {
			t.Error(err)
		} else if valInterface == nil {
			t.Error("got nil-interface{}")
		} else if usr, ok := valInterface.(*User); !ok {
			t.Errorf("unexpected type %T", valInterface)
		} else if usr != u {
			t.Error("wrong pointer", usr)
		}

		return
	})
	if err != nil {
		t.Fatal(err)
	}

	// delete eva inside the Ascend and try again
	err = refs.Ascend(func(i int, el *RefsElem) (_ error) {
		if i == 1 {
			return el.Delete()
		}
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	if ln, err := refs.Len(); err != nil {
		t.Fatal(err)
	} else if ln != 1 {
		t.Fatal("wrong length", ln)
	}
	t.Log(refs.DebugString(false))

	err = refs.Ascend(func(i int, el *RefsElem) (_ error) {
		if el.Hash == (cipher.SHA256{}) {
			t.Error("empty Hash", i)
		}
		if i != 0 {
			t.Error("index out of range", i)
			return ErrStopIteration
		}
		if valInterface, err := el.Value(); err != nil {
			t.Error(err)
		} else if valInterface == nil {
			t.Error("got nil-interface{}")
		} else if usr, ok := valInterface.(*User); !ok {
			t.Errorf("unexpected type %T", valInterface)
		} else if usr != alice {
			t.Error("wrong pointer", usr)
		}

		return
	})
	if err != nil {
		t.Fatal(err)
	}

	// rebuild tree after Ascend
	c.conf.MerkleDegree = 2
	refs = pack.Refs(alice, ammy, eva)

	err = refs.Ascend(func(i int, el *RefsElem) (_ error) {
		if i == 1 || i == 2 {
			el.Delete() // delte ammy and eva
		}
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(refs.DebugString(false))

	if refs.depth != 0 ||
		refs.length != 1 ||
		len(refs.branches) != 0 ||
		len(refs.leafs) != 1 {
		t.Errorf("wrong tree %#v", refs)
	}

	// delete all inside Ascend
	users = nil
	for i := 0; i < 100; i++ {
		users = append(users, &User{
			Name: fmt.Sprintf("User %d", i),
			Age:  (uint32(i) % 30) | 15,
		})
	}
	// build the tree one-by-one
	refs = pack.Refs()
	for i, u := range users {
		if err := refs.Append(u); err != nil {
			t.Fatal(err)
		}
		if ln, err := refs.Len(); err != nil {
			t.Fatal(err)
		} else if ln != i+1 {
			t.Fatal("wrong length", i+1, ln)
		}
	}

	// chech how the ErrStopIteration works
	var count int
	err = refs.Ascend(func(int, *RefsElem) (_ error) {
		count++
		return ErrStopIteration
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Error("ErrStopIteration doesn't stop the iteration")
	}

	// delete all inside ascend
	err = refs.Ascend(func(i int, el *RefsElem) (_ error) {
		if i < 0 || i > len(users) {
			t.Error("index out or range")
			return
		}
		if valInterface, err := el.Value(); err != nil {
			t.Error(err)
		} else if usr, ok := valInterface.(*User); !ok {
			t.Errorf("wrong type %T", valInterface)
		} else if usr != users[i] {
			t.Error("wrong pointer")
		}
		return el.Delete()
	})
	if err != nil {
		t.Fatal(err)
	}

	if ln, err := refs.Len(); err != nil {
		t.Error(err)
	} else if ln != 0 {
		t.Error("wrong length")
	}

}

func TestRefs_Descend(t *testing.T) {
	// Descend(irf IterateRefsFunc) (err error)

	c := getCont()
	defer c.db.Close()
	defer c.Close()

	pk, sk := cipher.GenerateKeyPair()
	if err := c.AddFeed(pk); err != nil {
		t.Fatal(err)
	}

	pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
	if err != nil {
		t.Fatal(err)
	}

	alice, ammy, eva := &User{"Alice", 21, nil}, &User{"Ammy", 17, nil},
		&User{"Eva", 23, nil}

	// direct leafs
	c.conf.MerkleDegree = 16 // for sure
	refs := pack.Refs(alice, ammy, eva)
	t.Log(refs.DebugString(false))

	users := []*User{alice, ammy, eva}

	err = refs.Descend(func(i int, el *RefsElem) (_ error) {
		if el.Hash == (cipher.SHA256{}) {
			t.Error("empty Hash", i)
		}
		if i < 0 || i >= len(users) {
			t.Error("index out of range:", i)
			return ErrStopIteration
		}
		u := users[i]

		if valInterface, err := el.Value(); err != nil {
			t.Error(err)
		} else if valInterface == nil {
			t.Error("got nil-interface{}")
		} else if usr, ok := valInterface.(*User); !ok {
			t.Errorf("unexpected type %T", valInterface)
		} else if usr != u {
			t.Error("wrong pointer", usr)
		}

		if el.upper != &refs {
			t.Error("upper of RefsElem not points to Refs")
		}

		return
	})
	if err != nil {
		t.Error(err)
	}

	// remove ammy and try again
	if el, err := refs.RefByIndex(1); err != nil {
		t.Fatal(err)
	} else if el.rn == nil {
		t.Fatal("not initialized")
	} else if up := el.upper; up == nil {
		t.Fatal("missing upper of RefsElem in Refs")
	} else if up != &refs {
		t.Logf("up:   %#v", up)
		t.Logf("refs: %#v", refs)
		t.Fatalf("wrong upper Refs: %s:%p - %s:%p", up.Short(), up,
			refs.Short(), &refs)
	} else if err = el.SetValue(nil); err != nil {
		t.Fatal(err)
	}

	if ln, err := refs.Len(); err != nil {
		t.Fatal(err)
	} else if ln != 2 {
		t.Fatal("wrong length:", ln)
	}
	t.Log(refs.DebugString(false))

	users = []*User{alice, eva}

	err = refs.Descend(func(i int, el *RefsElem) (_ error) {
		if el.Hash == (cipher.SHA256{}) {
			t.Error("empty Hash", i)
		}
		if i < 0 || i >= len(users) {
			t.Error("index out of range:", i)
			return ErrStopIteration
		}
		u := users[i]
		if valInterface, err := el.Value(); err != nil {
			t.Error(err)
		} else if valInterface == nil {
			t.Error("got nil-interface{}")
		} else if usr, ok := valInterface.(*User); !ok {
			t.Errorf("unexpected type %T", valInterface)
		} else if usr != u {
			t.Error("wrong pointer", usr)
		}

		return
	})
	if err != nil {
		t.Fatal(err)
	}

	// delete eva inside the Descend and try again
	err = refs.Descend(func(i int, el *RefsElem) (_ error) {
		if i == 1 {
			return el.Delete()
		}
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	if ln, err := refs.Len(); err != nil {
		t.Fatal(err)
	} else if ln != 1 {
		t.Fatal("wrong length", ln)
	}
	t.Log(refs.DebugString(false))

	err = refs.Descend(func(i int, el *RefsElem) (_ error) {
		if el.Hash == (cipher.SHA256{}) {
			t.Error("empty Hash", i)
		}
		if i != 0 {
			t.Error("index out of range", i)
			return ErrStopIteration
		}
		if valInterface, err := el.Value(); err != nil {
			t.Error(err)
		} else if valInterface == nil {
			t.Error("got nil-interface{}")
		} else if usr, ok := valInterface.(*User); !ok {
			t.Errorf("unexpected type %T", valInterface)
		} else if usr != alice {
			t.Error("wrong pointer", usr)
		}

		return
	})
	if err != nil {
		t.Fatal(err)
	}

	// rebuild tree after Descend
	c.conf.MerkleDegree = 2
	refs = pack.Refs(alice, ammy, eva)

	err = refs.Descend(func(i int, el *RefsElem) (_ error) {
		if i == 1 || i == 2 {
			el.Delete() // delte ammy and eva
		}
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(refs.DebugString(false))

	if refs.depth != 0 ||
		refs.length != 1 ||
		len(refs.branches) != 0 ||
		len(refs.leafs) != 1 {
		t.Errorf("wrong tree %#v", refs)
	}

	// delete all inside Descend
	users = nil
	for i := 0; i < 100; i++ {
		users = append(users, &User{
			Name: fmt.Sprintf("User %d", i),
			Age:  (uint32(i) % 30) | 15,
		})
	}
	// build the tree one-by-one
	refs = pack.Refs()
	for i, u := range users {
		if err := refs.Append(u); err != nil {
			t.Fatal(err)
		}
		if ln, err := refs.Len(); err != nil {
			t.Fatal(err)
		} else if ln != i+1 {
			t.Fatal("wrong length", i+1, ln)
		}
	}

	// chech how the ErrStopIteration works
	var count int
	err = refs.Descend(func(int, *RefsElem) (_ error) {
		count++
		return ErrStopIteration
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Error("ErrStopIteration doesn't stop the iteration")
	}

	// delete all inside ascend
	err = refs.Descend(func(i int, el *RefsElem) (_ error) {
		if i < 0 || i > len(users) {
			t.Error("index out or range")
			return
		}
		if valInterface, err := el.Value(); err != nil {
			t.Error(err)
		} else if usr, ok := valInterface.(*User); !ok {
			t.Errorf("wrong type %T", valInterface)
		} else if usr != users[i] {
			t.Error("wrong pointer")
		}
		return el.Delete()
	})
	if err != nil {
		t.Fatal(err)
	}

	if ln, err := refs.Len(); err != nil {
		t.Error(err)
	} else if ln != 0 {
		t.Error("wrong length")
	}
}

func TestRefs_Append(t *testing.T) {
	// Append(objs ...interface{}) (err error)

	//
}

func TestRefs_Clear(t *testing.T) {
	// Clear()

	//
}
