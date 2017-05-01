package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

func getCont() *Container {
	return NewContainer(data.NewDB())
}

func TestNewContainer(t *testing.T) {
	t.Run("nil db", func(t *testing.T) {
		defer shouldPanic(t)
		NewContainer(nil)
	})
	t.Run("norm", func(t *testing.T) {
		db := data.NewDB()
		c := NewContainer(db)
		if c.roots == nil {
			t.Error("nil roots map")
		}
		if c.reg == nil {
			t.Error("nil regitry")
		}
		if c.db != db {
			t.Error("wrong db")
		}
	})
}

func TestContainer_NewRoot(t *testing.T) {
	c := getCont()
	pk, sk := cipher.GenerateKeyPair()
	root := c.NewRoot(pk, sk)
	if root.cnt != c {
		t.Error("wrong back reference")
	}
	if root.reg != c.reg {
		t.Error("wrong registry reference")
	}
	if root.Pub != pk {
		t.Error("wrong pub key")
	}
}

func TestContainer_Root(t *testing.T) {
	c := getCont()
	pub, sec := cipher.GenerateKeyPair()
	root := c.NewRoot(pub, sec)
	if c.Root(pub) != root {
		t.Error("wrong root by pk")
	}
	if pk := pubKey(); pk != pub && c.Root(pk) != nil {
		t.Error("expected nil, got a root")
	}
}

func TestContainer_AddEncodedRoot(t *testing.T) {
	c1 := getCont()
	pub, sec := cipher.GenerateKeyPair()
	root := c1.NewRoot(pub, sec)
	root.Touch()
	root.Sign(sec)
	p := root.Encode()
	c2 := getCont()
	ok, err := c2.AddEncodedRoot(p, root.Pub, root.Sig)
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Error("don't set")
	}
}

func TestContainer_SchemaByReference(t *testing.T) {
	//
}

func TestContainer_Save(t *testing.T) {
	//
}

func TestContainer_SaveArray(t *testing.T) {
	//
}

func TestContainer_SaveSchema(t *testing.T) {
	//
}

func TestContainer_Dynamic(t *testing.T) {
	//
}

func TestContainer_Register(t *testing.T) {
	t.Run("complex recursive", func(t *testing.T) {
		type W struct {
			Z Reference `skyobject:"schema=X"`
		}
		type G struct {
			W W
		}
		type X struct {
			G G
		}
		cnt := getCont()
		defer shouldNotPanic(t)
		cnt.Register(
			"W", W{},
			"G", G{},
			"X", X{},
		)
	})
}
