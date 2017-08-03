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
	r := NewRegistry(func(reg *Reg) {})
	if r.Reference() == (RegistryRef{}) {
		t.Error("empty reference")
	}
}

func TestDecodeRegistry(t *testing.T) {
	e := NewRegistry(func(r *Reg) {
		r.Register("cxo.User", User{})
		r.Register("cxo.Group", Group{})
	})
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

func TestRegistry_identity(t *testing.T) {
	r1 := NewRegistry(func(r *Reg) {
		r.Register("cxo.User", User{})
		r.Register("cxo.Group", Group{})
	})
	r2 := NewRegistry(func(r *Reg) {
		r.Register("cxo.Group", Group{})
		r.Register("cxo.User", User{})
	})
	if r1.Reference() != r2.Reference() {
		t.Error("not equal")
	}
}

func TestRegistry_SchemaByName(t *testing.T) {
	r := NewRegistry(func(r *Reg) {
		r.Register("cxo.User", User{})
		r.Register("cxo.Group", Group{})
	})
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
		Post  Ref  `skyobject:"schema=Post"`
		Votes Refs `skyobject:"schema=Vote"`
	}
	type PostVoteContainer struct {
		Posts []PostVotePage
	}
	r := NewRegistry(func(r *Reg) {
		r.Register("Post", Post{})
		r.Register("Vote", Vote{})
		r.Register("PostVotePage", PostVotePage{})
		r.Register("PostVoteContainer", PostVoteContainer{})
	})

	defer shouldNotPanic(t)
	dr, err := DecodeRegistry(r.Encode())
	if err != nil {
		t.Fatal(err)
	}
	if dr.Reference() != r.Reference() {
		t.Error("different decoder reference")
	}
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
	reg := NewRegistry(func(r *Reg) {
		r.Register("test.Info", Info{})
		r.Register("test.Brief", Brief{})
		r.Register("test.Any", Any{})
	})

	// TODO

	_ = reg
}
