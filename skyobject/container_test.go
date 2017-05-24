package skyobject

import (
	"testing"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/skycoin/src/cipher"
)

func TestNewContainer(t *testing.T) {
	t.Run("missing DB", func(t *testing.T) {
		defer shouldPanic(t)
		NewContainer(nil, nil)
	})
	t.Run("registry", func(t *testing.T) {
		reg := NewRegistry()
		db := data.NewMemoryDB()
		c := NewContainer(db, reg)
		if c.DB() == nil {
			t.Error("missing database")
		} else if c.DB() != db {
			t.Error("wrong database")
		} else if c.coreRegistry != reg {
			t.Error("wrong core registry")
		} else {
			if reg.done != true {
				t.Error("missing (*Registry).Done in NewContainer")
			}
			if _, ok := c.DB().Get(cipher.SHA256(reg.Reference())); !ok {
				t.Error("registry wasn't saved")
			}
		}
		if _, err := c.Registry(reg.Reference()); err != nil {
			t.Error("can't give core registry by reference")
		}
	})
}

func TestContainer_AddRegistry(t *testing.T) {
	c := NewContainer(data.NewMemoryDB(), nil)
	reg := NewRegistry()
	c.AddRegistry(reg)
	if reg.done != true {
		t.Error("missing (*Registry).Done in AddRegistry")
	} else if _, err := c.Registry(reg.Reference()); err != nil {
		t.Error("can't give registyr by reference")
	} else if _, ok := c.DB().Get(cipher.SHA256(reg.Reference())); !ok {
		t.Error("registry wasn't saved")
	}
}

func TestContainer_CoreRegistry(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		c := NewContainer(data.NewMemoryDB(), nil)
		if c.CoreRegistry() != nil {
			t.Error("unexpected core registry")
		}
	})
	t.Run("got", func(t *testing.T) {
		reg := NewRegistry()
		c := NewContainer(data.NewMemoryDB(), reg)
		if c.CoreRegistry() != reg {
			t.Error("missing or wrong registry")
		}
	})
}

func TestContainer_Registry(t *testing.T) {
	// core
	cr := NewRegistry()
	cr.Register("cxo.User", User{})
	// add
	ar := NewRegistry()
	ar.Register("cxo.Developer", Developer{})
	//
	c := NewContainer(data.NewMemoryDB(), cr)
	c.AddRegistry(ar)
	//
	if _, err := c.Registry(RegistryReference{}); err == nil {
		t.Error("missing error")
	}
	if _, err := c.Registry(cr.Reference()); err != nil {
		t.Error("missing core registry")
	}
	if _, err := c.Registry(ar.Reference()); err != nil {
		t.Error("missing added registry")
	}
}

func TestContainer_WantRegistry(t *testing.T) {
	t.Run("dont want", func(t *testing.T) {
		c := NewContainer(data.NewMemoryDB(), nil)
		reg := NewRegistry()
		reg.Done()
		if c.WantRegistry(reg.Reference()) {
			t.Error("unexpected want")
		}
	})
	t.Run("want", func(t *testing.T) {
		reg := NewRegistry()
		reg.Register("cxo.User", User{})
		c1 := NewContainer(data.NewMemoryDB(), reg)
		pk, sk := cipher.GenerateKeyPair()
		r, err := c1.NewRoot(pk, sk)
		if err != nil {
			t.Fatal(err)
		}
		_, rp, err := r.Inject("cxo.User", User{"Alice", 20, nil})
		if err != nil {
			t.Fatal(err)
		}
		if c1.WantRegistry(reg.Reference()) {
			t.Error("missing want")
		}
		c2 := NewContainer(data.NewMemoryDB(), nil)
		if _, r := c2.AddRootPack(&rp); r != nil {
			t.Fatal(err)
		}
		if !c2.WantRegistry(reg.Reference()) {
			t.Error("missing want")
		}
	})
}

func TestContainer_Registries(t *testing.T) {
	t.Run("core", func(t *testing.T) {
		reg := NewRegistry()
		c := NewContainer(data.NewMemoryDB(), reg)
		regs := c.Registries()
		if len(regs) != 1 {
			t.Error("wrong registries")
		} else if regs[0] != reg.Reference() {
			t.Error("wrong reference")
		}
	})
	t.Run("no core", func(t *testing.T) {
		c := NewContainer(data.NewMemoryDB(), nil)
		reg := NewRegistry()
		c.AddRegistry(reg)
		regs := c.Registries()
		if len(regs) != 1 {
			t.Error("wrong registries")
		} else if regs[0] != reg.Reference() {
			t.Error("wrong reference")
		}
	})
}

func TestContainer_DB(t *testing.T) {
	db := data.NewMemoryDB()
	c := NewContainer(db, nil)
	if c.DB() != db {
		t.Error("wrong db")
	}
}

func TestContainer_Get(t *testing.T) {
	//
}

func TestContainer_Set(t *testing.T) {
	//
}

func TestContainer_NewRoot(t *testing.T) {
	//
}

func TestContainer_NewRootReg(t *testing.T) {
	//
}

func TestContainer_AddRootPack(t *testing.T) {
	//
}

func TestContainer_LastRoot(t *testing.T) {
	//
}

func TestContainer_LastRootSk(t *testing.T) {
	//
}

func TestContainer_LastFullRoot(t *testing.T) {
	c := getCont()
	pk, sk := cipher.GenerateKeyPair()
	r, err := c.NewRoot(pk, sk)
	if err != nil {
		t.Error(err)
		return
	}
	r.Inject("cxo.User", User{"Alice", 15, nil})
	r.Inject("cxo.User", User{"Eva", 16, nil})
	r.Inject("cxo.User", User{"Ammy", 17, nil})
	full := c.LastFullRoot(pk)
	if full == nil {
		t.Error("misisng last full root")
	}
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

func TestContainer_DelRootsBefore(t *testing.T) {
	//
}

func TestContainer_GC(t *testing.T) {
	//
}
