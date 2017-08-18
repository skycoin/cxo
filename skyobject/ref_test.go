package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

func TestRef_IsBlank(t *testing.T) {
	// IsBlank() bool

	// TODO (kostyarin): low priority
	t.Skip("not implemented (low priority)")
}

func TestRef_Short(t *testing.T) {
	// Short() string

	// TODO (kostyarin): low priority
	t.Skip("not implemented (low priority)")
}

func TestRef_String(t *testing.T) {
	// String() string

	// TODO (kostyarin): low priority
	t.Skip("not implemented (low priority)")
}

func TestRef_Eq(t *testing.T) {
	// Eq(x *Ref) bool

	// TODO (kostyarin): low priority
	t.Skip("not implemented (low priority)")
}

func TestRef_Schema(t *testing.T) {
	// Schema() Schema

	// detached
	ref := Ref{}
	if ref.Schema() != nil {
		t.Error("unexpected schema")
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

	// nil
	ref = pack.Ref(nil)
	if ref.Schema() != nil {
		t.Error("unexpected schema")
	}

	ref = pack.Ref(User{"Alice", 21, nil})
	if sch := ref.Schema(); sch == nil {
		t.Error("missing schema")
	} else if sch.Name() != "cxo.User" {
		t.Error("wrong schema:", sch.String())
	}

}

func TestRef_Value(t *testing.T) {
	// Value() (obj interface{}, err error)

	// detached
	ref := Ref{}
	if _, err := ref.Value(); err == nil {
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

	// already have
	alice := &User{"Alice", 21, nil}
	ref = pack.Ref(alice)
	if valInterface, err := ref.Value(); err != nil {
		t.Error(err)
	} else if valInterface == nil {
		t.Error("got nil interface{}")
	} else if usr, ok := valInterface.(*User); !ok {
		t.Errorf("unepected type %T", valInterface)
	} else if usr != alice {
		t.Error("wrong pointer")
	}

	// attached empty (nil schema)
	ref = pack.Ref(nil)
	if _, err := ref.Value(); err == nil {
		t.Error("missing error")
	}

	// blank
	ref = pack.Ref(alice)
	ref.rn.value, ref.Hash = nil, cipher.SHA256{} // clear for the test
	if valInterface, err := ref.Value(); err != nil {
		t.Error(err)
	} else if valInterface == nil {
		t.Error("got nil interface{}")
	} else if usr, ok := valInterface.(*User); !ok {
		t.Errorf("unepected type %T", valInterface)
	} else if usr != nil {
		t.Error("got non-nil pointer")
	}

}

func TestRef_SetValue(t *testing.T) {
	// SetValue(obj interface{}) (err error)

	// set nil (detached)
	ref := Ref{Hash: cipher.SHA256{1, 2, 3}}
	if err := ref.SetValue(nil); err != nil {
		t.Error(err)
	}

	alice := &User{"Alice", 21, nil}

	// detached
	if err := ref.SetValue(&alice); err == nil {
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

	// view only
	pack.setFlag(ViewOnly)
	ref = pack.Ref(nil)
	if err := ref.SetValue(alice); err == nil {
		t.Error("missing error")
	}
	pack.unsetFlag(ViewOnly)

	// type is not registered
	type Invader struct {
		Name string
	}
	if err := ref.SetValue(Invader{"Alex"}); err == nil {
		t.Error("missing error")
	}

	// different schema
	ref = pack.Ref(alice)
	if err := ref.SetValue(Group{Name: "VDPG"}); err == nil {
		t.Error("missing error")
	}

	// nil of some type
	ref = pack.Ref((*User)(nil))
	if ref.Schema() == nil {
		t.Error("missing schema")
	} else if valInterface, err := ref.Value(); err != nil {
		t.Error(err)
	} else if valInterface == nil {
		t.Error("got nil-interface{}")
	} else if usr, ok := valInterface.(*User); !ok {
		t.Errorf("unexpected type %T", valInterface)
	} else if usr != nil {
		t.Error("got non-nil")
	}

	// not found in database
	ref = pack.Ref(alice)
	val := pack.unsaved[ref.Hash] // keep for next test
	// clear for the test
	pack.unsaved = make(map[cipher.SHA256][]byte) // clear cache
	ref.rn.value = nil
	if _, err := ref.Value(); err == nil {
		t.Error("missing error")
	}

	// from "database" (from cache)
	pack.unsaved[ref.Hash] = val
	if ref.Schema() == nil {
		t.Error("missing schema")
	} else if valInterface, err := ref.Value(); err != nil {
		t.Error(err)
	} else if valInterface == nil {
		t.Error("got nil-interface{}")
	} else if usr, ok := valInterface.(*User); !ok {
		t.Errorf("unexpected type %T", valInterface)
	} else if usr.Name != alice.Name || usr.Age != alice.Age {
		t.Error("wrong value")
	}

}

func TestRef_Clear(t *testing.T) {
	// Clear() (err error)

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

	ref := pack.Ref(User{"Alice", 21, nil})
	if err := ref.Clear(); err != nil {
		t.Fatal(err)
	}

	if ref.Hash != (cipher.SHA256{}) {
		t.Error("not clear")
	}

	if ref.rn.value != nil {
		t.Error("internal value has not been cleared")
	}

}

func TestRef_Copy(t *testing.T) {
	// Copy() (cp Ref)

	// TODO (kostyarin): low priority
	t.Skip("not implemented (low priority)")
}
