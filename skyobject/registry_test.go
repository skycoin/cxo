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
	if err := recover(); err != nil {
		t.Error("unexpected panic:", err)
	}
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
	e.Register("cxo.User", User{})
	e.Register("cxo.Group", Group{})
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

func TestRegistry_Register(t *testing.T) {
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
		r.SchemaReferenceByName("cxo.User")
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
		r.Register("User", User{})
		r.Register("Group", Group{})
		r.Done() // must panic with misisng
	})

	t.Run("Done", func(t *testing.T) {
		defer shouldNotPanic(t)
		r := NewRegistry()
		r.Register("cxo.User", User{})
		r.Register("cxo.Group", Group{})
		r.Done()
	})

	t.Run("panic Register", func(t *testing.T) {
		r := NewRegistry()
		r.Done()
		defer shouldPanic(t)
		type Age struct{}
		r.Register("internal.Age", Age{})
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
		r1.Register("cxo.User", User{})
		r1.Register("cxo.Group", Group{})
		r1.Done()
		r2 := NewRegistry()
		r2.Register("cxo.User", User{})
		r2.Register("cxo.Group", Group{})
		r2.Done()
		if r1.Reference() != r2.Reference() {
			t.Error("not equal")
		}
	})
}

func TestRegistry_SchemaByName(t *testing.T) {
	r := NewRegistry()
	r.Register("cxo.User", User{})
	r.Register("cxo.Group", Group{})
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

func TestRegistry_slice(t *testing.T) {
	type Post struct {
		Name    string
		Content string
	}
	type Vote struct {
		Up   uint32
		Down uint32
	}
	type PostVotePage struct {
		Post  Reference  `skyobject:"schema=Post"`
		Votes References `skyobject:"schema=Vote"`
	}
	type PostVoteContainer struct {
		Posts []PostVotePage
	}
	r := NewRegistry()
	r.Register("Post", Post{})
	r.Register("Vote", Vote{})
	r.Register("PostVotePage", PostVotePage{})
	r.Register("PostVoteContainer", PostVoteContainer{})
	t.Run("done", func(t *testing.T) {
		defer shouldNotPanic(t)
		r.Done()
	})
	t.Run("encode decode", func(t *testing.T) {
		defer shouldNotPanic(t)
		_, err := DecodeRegistry(r.Encode())
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestRegistry_userProvidedName(t *testing.T) {

	type Info struct {
		About string
	}
	type Brief struct {
		Note string
	}
	type Any struct {
		Info
		Brief Brief
	}
	reg := NewRegistry()
	reg.Register("test.Info", Info{})
	reg.Register("test.Brief", Brief{})
	reg.Register("test.Any", Any{})

	defer shouldNotPanic(t)

	reg.Done()

	// TODO
}
