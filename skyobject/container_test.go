package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

func TestNewContainer(t *testing.T) {
	t.Run("with registry", func(t *testing.T) {
		r := NewRegistry()
		c := NewContainer(r)
		if c.db == nil {
			t.Error("misisng database")
		}
		if c.coreRegistry != r {
			t.Error("wrong core registry")
		}
		if len(c.registries) == 0 {
			t.Error("registry missing in map")
		} else if c.registries[r.Reference()] != r {
			t.Error("wrong registry in map")
		} else if r.done != true {
			t.Error("Done wasn't called")
		}
	})
	t.Run("wihtout registry", func(t *testing.T) {
		c := NewContainer(nil)
		if c.db == nil {
			t.Error("misisng database")
		}
		if c.coreRegistry != nil {
			t.Error("wrong core registry")
		}
		if len(c.registries) != 0 {
			t.Error("unexpected registries in map")
		}
	})
}

func TestNewContainerDB(t *testing.T) {
	//
}

func TestContainer_CoreRegistry(t *testing.T) {
	//
}

func TestContainer_Registry(t *testing.T) {
	//
}

func TestContainer_DB(t *testing.T) {
	//
}

func TestContainer_Get(t *testing.T) {
	//
}

func TestContainer_Save(t *testing.T) {
	//
}

func TestContainer_SaveArray(t *testing.T) {
	//
}

func TestContainer_Dynamic(t *testing.T) {
	//
}

func TestContainer_NewRoot(t *testing.T) {
	t.Run("nil reg", func(t *testing.T) {
		c := NewContainer(nil)
		pk, sk := cipher.GenerateKeyPair()
		defer shouldPanic(t)
		c.NewRoot(pk, sk)
	})
	t.Run("empty pk", func(t *testing.T) {
		c := NewContainer(NewRegistry())
		_, sk := cipher.GenerateKeyPair()
		defer shouldPanic(t)
		c.NewRoot(cipher.PubKey{}, sk)
	})
	t.Run("empty sk", func(t *testing.T) {
		c := NewContainer(NewRegistry())
		pk, _ := cipher.GenerateKeyPair()
		defer shouldPanic(t)
		c.NewRoot(pk, cipher.SecKey{})
	})
	t.Run("different sk", func(t *testing.T) {
		c := NewContainer(NewRegistry())
		pk, sk := cipher.GenerateKeyPair()
		r := c.NewRoot(pk, sk)
		r.Touch() // save
		_, sn := cipher.GenerateKeyPair()
		defer shouldPanic(t)
		c.NewRoot(pk, sn)
	})
}

func TestContainer_AddEncodedRoot(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()

	reg := NewRegistry()
	reg.Regsiter("cxo.User", User{})
	reg.Regsiter("cxo.Group", Group{})
	reg.Done()

	c1 := NewContainer(reg)
	r1 := c1.NewRoot(pk, sk)

	r1.Inject(User{Name: "motherfucker"})

	c2 := NewContainer(nil)

	r2, err := c2.AddEncodedRoot(r1.Encode()) // []byte, sig
	if err != nil {
		t.Fatal(err)
	}

	if r2.Refs()[0] != r1.Refs()[0] {
		t.Error("missmatch")
	}

	if r2.RegistryReference() != r1.RegistryReference() {
		t.Error("missmatch")
	}

}

func TestContainer_LastRoot(t *testing.T) {
	// high
}

func TestContainer_LastFullRoot(t *testing.T) {
	// high
}

func TestContainer_RootBySeq(t *testing.T) {
	// mid.
}

func TestContainer_GC(t *testing.T) {
	// mid.
}

func TestContainer_RootsGC(t *testing.T) {
	// mid.
}

func TestContainer_RegsitryGC(t *testing.T) {
	// mid.
}

func TestContainer_ObjectsGC(t *testing.T) {
	// mid.
}
