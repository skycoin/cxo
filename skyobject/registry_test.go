package skyobject

import (
	"bytes"
	"testing"
)

func shouldPanic(t *testing.T) {
	if recover() == nil {
		t.Error("missing panic")
	}
}

func shouldNotPanic(t *testing.T) {
	if recover() != nil {
		t.Error("unexpected panic")
	}
}

type User struct {
	Name     string
	Age      uint32
	Hidden   []byte    `enc:"-"`
	MemberOf Reference `skyobject:"schema=cxo.Group"`
}

type Group struct {
	Name    string
	Leader  Reference  `skyobject:"schema=cxo.User"`
	Members References `skyobject:"schema=cxo.User"`
	Curator Dynamic
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r.done != false {
		t.Error("unexpected done")
	}
	if r.ref != (RegistryReference{}) {
		t.Error("non-empty reference")
	}
}

func TestDecodeRegistry(t *testing.T) {
	e := NewRegistry()
	e.Regsiter("cxo.User", User{})
	e.Regsiter("cxo.Group", Group{})
	e.Done()
	d, err := DecodeRegistry(e.Encode())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(d.Encode(), e.Encode()) != 0 {
		t.Error("different")
	}
	if d.reg["cxo.User"] == nil {
		t.Error("misisng schema")
	}
	if d.reg["cxo.Group"] == nil {
		t.Error("missing schema")
	}
}

func TestRegistry_Regsiter(t *testing.T) {
	//
}

func TestRegistry_Done(t *testing.T) {
	t.Run("panic SchemaByName", func(t *testing.T) {
		defer shouldPanic(t)
		r := NewRegistry()
		r.SchemaByName("cxo.User")
	})
	t.Run("panic SchemaByReference", func(t *testing.T) {
		defer shouldPanic(t)
		r := NewRegistry()
		r.SchemaByReference(SchemaReference{})
	})
	t.Run("panic SchemaByInterface", func(t *testing.T) {
		defer shouldPanic(t)
		r := NewRegistry()
		r.SchemaByInterface(User{})
	})
	t.Run("panic Encode", func(t *testing.T) {
		defer shouldPanic(t)
		r := NewRegistry()
		r.Encode()
	})
	t.Run("panic Reference", func(t *testing.T) {
		defer shouldPanic(t)
		r := NewRegistry()
		r.Reference()
	})

	t.Run("panic Done", func(t *testing.T) {
		defer shouldPanic(t)
		r := NewRegistry()
		r.Regsiter("User", User{})
		r.Regsiter("Group", Group{})
		r.Done() // must panic with misisng
	})

	t.Run("Done", func(t *testing.T) {
		defer shouldNotPanic(t)
		r := NewRegistry()
		r.Regsiter("cxo.User", User{})
		r.Regsiter("cxo.Group", Group{})
		r.Done()
	})

	t.Run("panic Register", func(t *testing.T) {
		r := NewRegistry()
		r.Done()
		defer shouldPanic(t)
		type Age struct{}
		r.Regsiter("internal.Age", Age{})
	})

	t.Run("reference", func(t *testing.T) {
		r := NewRegistry()
		r.Done()
		if r.Reference() == (RegistryReference{}) {
			t.Error("empty reference")
		}
	})

	t.Run("identity", func(t *testing.T) {
		r1 := NewRegistry()
		r1.Regsiter("cxo.User", User{})
		r1.Regsiter("cxo.Group", Group{})
		r1.Done()
		r2 := NewRegistry()
		r2.Regsiter("cxo.User", User{})
		r2.Regsiter("cxo.Group", Group{})
		r2.Done()
		if r1.Reference() != r2.Reference() {
			t.Error("not equal")
		}
	})
}

func TestRegistry_SchemaByName(t *testing.T) {
	r := NewRegistry()
	r.Regsiter("cxo.User", User{})
	r.Regsiter("cxo.Group", Group{})
	r.Done()
	u, err := r.SchemaByName("cxo.User")
	if err != nil {
		t.Error(err)
	} else if u.Name() != "cxo.User" {
		t.Error("name is ", u.Name())
	}
	if _, err := r.SchemaByName("nothing"); err == nil {
		t.Error("missing error")
	}
	g, err := r.SchemaByName("cxo.Group")
	if err != nil {
		t.Error(err)
	} else if g.Name() != "cxo.Group" {
		t.Error("name is ", g.Name())
	}
	if len(g.Fields()) != 4 {
		t.Error("wrong fields count", len(g.Fields()))
	} else if uf := g.Fields()[2].Schema().Elem(); uf.Name() != "cxo.User" {
		t.Error("wrong field schema:")
	} else if uf != u {
		t.Error("not the same") // must be the same instance of Schema
	}
}

func TestRegistry_SchemaByReference(t *testing.T) {
	//
}

func TestRegistry_SchemaByInterface(t *testing.T) {
	//
}

func TestRegistry_Encode(t *testing.T) {
	//
}

func TestRegistry_Reference(t *testing.T) {
	//
}
