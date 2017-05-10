package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
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
	t.Run("without db", func(t *testing.T) {
		defer shouldPanic(t)
		NewContainerDB(nil, nil)
	})
	t.Run("with db", func(t *testing.T) {
		db := data.NewDB()
		c := NewContainerDB(db, nil)
		if c.db != db {
			t.Error("wrong database")
		}
	})
}

func TestContainer_AddRegistry(t *testing.T) {
	type User struct {
		Name string
		Age  uint32
	}
	c := NewContainer(nil)
	reg := NewRegistry()
	reg.Regsiter("test.User", User{})
	c.AddRegistry(reg)
	if len(c.registries) != 1 {
		t.Fatal("misisng reg")
	}
	if c.registries[reg.Reference()] != reg {
		t.Fatal("unexpected")
	}
	if reg.done != true {
		t.Error("Done haven't been called")
	}
	// try to replace with equal
	rep := NewRegistry()
	rep.Regsiter("test.User", User{})
	rep.Done()
	if rep.Reference() != reg.Reference() {
		t.Fatal("missmatch references")
	}
	c.AddRegistry(rep)
	if len(c.registries) != 1 {
		t.Error("add equal registry")
	}
	if c.registries[reg.Reference()] != reg {
		t.Error("replaced")
	}
}

func TestContainer_CoreRegistry(t *testing.T) {
	t.Run("with", func(t *testing.T) {
		reg := NewRegistry()
		c := NewContainer(reg)
		if c.CoreRegistry() != reg {
			t.Error("misisng core regstry")
		}
	})
	t.Run("without", func(t *testing.T) {
		c := NewContainer(nil)
		if c.CoreRegistry() != nil {
			t.Error("unexpected core reg")
		}
	})
}

func TestContainer_Registry(t *testing.T) {
	type User struct {
		Name string
	}
	reg := NewRegistry()
	reg.Regsiter("test.User", User{})
	reg.Done()
	rr := reg.Reference()
	c := NewContainer(nil)
	r, err := c.Registry(rr)
	if err == nil {
		t.Error("missing error")
	}
	if r != nil {
		t.Error("unexpected regstry")
	}
	c.AddRegistry(reg)
	if r, err := c.Registry(reg.Reference()); err != nil {
		t.Error("unexpected error: ", err)
	} else if r != reg {
		t.Error("wrong registry instance")
	}
}

func TestContainer_WantRegistry(t *testing.T) {
	type User struct {
		Name string
	}
	reg := NewRegistry()
	reg.Regsiter("test.User", User{})
	reg.Done()
	rr := reg.Reference()

	c := NewContainer(reg)

	pk, sk := cipher.GenerateKeyPair()
	root := c.NewRoot(pk, sk)

	sig, p := root.Touch()

	x := NewContainer(nil)
	if _, err := x.AddEncodedRoot(p, sig); err != nil {
		t.Fatal(err)
	}

	if !x.WantRegistry(rr) {
		t.Error("registry should be wanted")
	}

	x.AddRegistry(reg)

	if x.WantRegistry(rr) {
		t.Error("registry should not be wanted")
	}
}

func TestContainer_Registries(t *testing.T) {
	type User struct {
		Name string
	}
	r1 := NewRegistry()
	r1.Regsiter("test.User", User{})
	r1.Done()

	type Developer struct {
		Name string
	}
	r2 := NewRegistry()
	r2.Regsiter("test.Developer", Developer{})
	r2.Done()

	c := NewContainer(r1)

	cr := c.Registries()
	if len(cr) != 1 {
		t.Error("wrong number of registries of container")
	} else if cr[0] != r1.Reference() {
		t.Error("wrong reference")
	}

	c.AddRegistry(r2)

	cr = c.Registries()
	if len(cr) != 2 {
		t.Error("wrong number of registries of contaiener")
	} else {
		pass := (cr[0] == r1.Reference() && cr[1] == r2.Reference()) ||
			(cr[1] == r1.Reference() && cr[0] == r2.Reference())
		if !pass {
			t.Error("wrong references")
		}
	}
}

func TestContainer_DB(t *testing.T) {
	if NewContainer(nil).DB() == nil {
		t.Error("misisng database")
	}
	db := data.NewDB()
	c := NewContainerDB(db, nil)
	if c.DB() != db {
		t.Error("wrong or missing db")
	}
}

func TestContainer_Get(t *testing.T) {
	//
}

func TestContainer_Set(t *testing.T) {
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

func TestContainer_Feeds(t *testing.T) {
	//
}

func TestContainer_WantFeed(t *testing.T) {
	//
}

func TestContainer_GotFeed(t *testing.T) {
	//
}

func TestContainer_DelFeed(t *testing.T) {
	//
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
