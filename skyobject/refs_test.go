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

	// blank
	refs := Refs{}
	if ln, err := refs.Len(); err != nil {
		t.Error(err)
	} else if ln != 0 {
		t.Error("wrong length")
	}

	// detached
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

	// alice, ammy, eva := &User{"Alice", 21, nil}, &User{"Ammy", 17, nil},
	//	&User{"Eva", 23, nil}
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
