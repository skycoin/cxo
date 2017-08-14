package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
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

	// detached blank
	refs := Refs{}
	if ln, err := refs.Len(); err != nil {
		t.Error(err)
	} else if ln != 0 {
		t.Error("wrong length:", ln)
	}

	// detached non-blank
	refs.Hash = cipher.SHA256{1, 2, 3}
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
	pack.flags = pack.flags | HashTableIndex
	if _, err := refs.Len(); err == nil {
		t.Error("missing error")
	} else if refs.index == nil {
		t.Error("index has not been created")
	}
	pack.flags = 0
	refs.index = nil

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

	// reduce the degree nand repeat to force the Refs has branches
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
	refs.branches, refs.leafs, refs.index = nil, nil, nil

	if ln, err := refs.Len(); err != nil {
		t.Error(err)
	} else if ln != 3 {
		t.Error("wrong length:", ln)
	}

}

func TestRefs_RefByIndex(t *testing.T) {
	// RefByIndex(i int) (ref *Ref, err error)

	//
}

func TestRefs_RefByHash(t *testing.T) {
	// RefByHash(hash cipher.SHA256) (needle *Ref, err error)

	//
}

func TestRefs_Ascend(t *testing.T) {
	// Ascend(irf IterateRefsFunc) (err error)

	//
}

func TestRefs_Descend(t *testing.T) {
	// Descend(irf IterateRefsFunc) (err error)

	//
}

func TestRefs_Append(t *testing.T) {
	// Append(objs ...interface{}) (err error)

	//
}

func TestRefs_Clear(t *testing.T) {
	// Clear()

	//
}
