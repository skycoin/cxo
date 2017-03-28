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
	pk := pubKey()
	root := c.NewRoot(pk)
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
	root := c.NewRoot(pub)
	c.AddRoot(root, sec)
	if c.Root(pub) != root {
		t.Error("wrong root by pk")
	}
	if c.Root(pubKey()) != nil {
		t.Error("expected nil, got a root")
	}
}

func TestContainer_AddRoot(t *testing.T) {
	t.Run("aside", func(t *testing.T) {
		c := getCont()
		defer shouldPanic(t)
		c.AddRoot(&Root{}, cipher.SecKey{})
	})
	t.Run("newer", func(t *testing.T) {
		c := getCont()
		pk, sk := cipher.GenerateKeyPair()
		r1 := c.NewRoot(pk)
		if !c.AddRoot(r1, sk) {
			t.Error("can't add root")
		}
		if c.AddRoot(r1, sk) {
			t.Error("add with same time")
		}
		r2 := c.NewRoot(pk)
		r2.Touch()
		if !c.AddRoot(r2, sk) {
			t.Error("can't add newer root")
		}
	})
}

func TestContainer_AddEncodedRoot(t *testing.T) {
	c := getCont()
	pub, sec := cipher.GenerateKeyPair()
	root := c.NewRoot(pub)
	root.Touch()
	root.Sign(sec)
	p := root.Encode()
	ok, err := c.AddEncodedRoot(p, root.Pub, root.Sig)
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
