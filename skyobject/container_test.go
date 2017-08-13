package skyobject

import (
	"math/big"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

func TestNewContainer(t *testing.T) {

	t.Run("missing db", func(t *testing.T) {
		defer shouldPanic(t)
		NewContainer(nil, nil)
	})

	t.Run("nil config", func(t *testing.T) {
		c := NewContainer(data.NewMemoryDB(), nil)
		c.Close()
		c.db.Close()
	})

	t.Run("invalid config", func(t *testing.T) {
		db := data.NewMemoryDB()
		defer db.Close()
		defer shouldPanic(t)
		NewContainer(db, &Config{MerkleDegree: -1000})
	})

	t.Run("add core registry error", func(t *testing.T) {
		db := data.NewMemoryDB()
		conf := NewConfig()
		conf.Registry = NewRegistry(func(r *Reg) {
			r.Register("test.User", User{})
		})
		db.Close()
		defer shouldPanic(t)
		NewContainer(db, conf)
	})

}

func TestContainer_Set(t *testing.T) {
	c := getCont()
	defer c.db.Close()

	val := []byte("value")
	hash := cipher.SumSHA256(val)

	t.Run("set", func(t *testing.T) {
		if err := c.Set(hash, val); err != nil {
			t.Error(err)
		}
		c.DB().View(func(tx data.Tv) (_ error) {
			if string(tx.Objects().Get(hash)) != "value" {
				t.Error("not set")
			}
			return
		})
	})

	t.Run("error", func(t *testing.T) {
		c.db.Close()
		if err := c.Set(hash, val); err == nil {
			t.Error("misisng error")
		}
	})

}

func TestContainer_Get(t *testing.T) {

	c := getCont()
	defer c.db.Close()
	defer c.Close()

	val := []byte("value")
	hash := cipher.SumSHA256(val)
	if err := c.Set(hash, val); err != nil {
		t.Fatal(err)
	}

	t.Run("get", func(t *testing.T) {
		if c.Get(cipher.SHA256{}) != nil {
			t.Error("go unexisting vlaue")
		}
		if string(c.Get(hash)) != string(val) {
			t.Error("missing or wrong value")
		}
	})

	t.Run("fatal get", func(t *testing.T) {
		c.db.Close()
		defer shouldPanic(t)
		c.Get(hash)
	})

}

func TestContainer_Root(t *testing.T) {
	c := getCont()
	defer c.db.Close()
	defer c.Close()

	pk, sk := cipher.GenerateKeyPair()

	// no such feed
	if _, err := c.Root(pk, 0); err == nil {
		t.Error("misisng error")
	} else if err != ErrNoSuchFeed {
		t.Error("unexpected error:", err)
	}

	if err := c.AddFeed(pk); err != nil {
		t.Fatal(err)
	}

	// not found
	if _, err := c.Root(pk, 0); err == nil {
		t.Error("missing error")
	}

	pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pack.Save(); err != nil {
		t.Fatal(err)
	}

	root, err := c.Root(pk, 0)
	if err != nil {
		t.Error(err)
	}

	if root.Seq != 0 {
		t.Error("wrong root")
	}

}

func TestContainer_Unpack(t *testing.T) {

	c := getCont()
	defer c.db.Close()
	defer c.Close()

	pk, sk := cipher.GenerateKeyPair()

	if err := c.AddFeed(pk); err != nil {
		t.Fatal(err)
	}

	// nil root
	if _, err := c.Unpack(nil, 0, nil, cipher.SecKey{}); err == nil {
		t.Error("missing error")
	}

	// invalid public key
	pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pack.Save(); err != nil {
		t.Fatal(err)
	}
	root := pack.Root()
	root.Pub = cipher.PubKey{}
	if _, err := c.Unpack(root, 0, nil, cipher.SecKey{}); err == nil {
		t.Error("missing error")
	}
	root.Pub = pk // restore

	// TODO (kostyarin): how to get invlid secret key?
	// invalid secret key
	_ = big.NewInt(0)
	/*
		iskn := big.NewInt(0)
		iskn.SetBytes(sk[:])
		iskn.Neg(iskn)
		var isk cipher.SecKey
		copy(isk[:], iskn.Bytes())
		if isk.Verify() == nil {
			t.Fatal("VALID", iskn.Sign(), isk.Hex())
		}
		if _, err := c.Unpack(root, 0, nil, isk); err == nil {
			t.Error("missing error")
		}
	*/

	// nil types
	if _, err := c.Unpack(root, 0, nil, cipher.SecKey{}); err == nil {
		t.Error("missing error")
	}

	// nil direct
	types := c.CoreRegistry().Types()
	types.Direct = nil
	if _, err := c.Unpack(root, 0, types, cipher.SecKey{}); err == nil {
		t.Error("missing error")
	}

	// nil inverse
	types = c.CoreRegistry().Types()
	types.Inverse = nil
	if _, err := c.Unpack(root, 0, types, cipher.SecKey{}); err == nil {
		t.Error("missing error")
	}
	types = c.CoreRegistry().Types() // store

	// empty regsitry reference
	root.Reg = RegistryRef{}
	if _, err := c.Unpack(root, 0, types, cipher.SecKey{}); err == nil {
		t.Error("missing error")
	}

	// registry not found
	root.Reg = RegistryRef{1, 2, 3}
	if _, err := c.Unpack(root, 0, types, cipher.SecKey{}); err == nil {
		t.Error("missing error")
	}

	// TODO: (*Pack).init error

	// got it
	root.Reg = c.CoreRegistry().Reference()
	pack, err = c.Unpack(root, 0, types, sk)
	if err != nil {
		t.Error(err)
	}
	if pack == nil {
		t.Error("nil")
	}

}

func TestContainer_NewRoot(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()

	conf := getConf()
	conf.Registry = nil

	// no core regsitry
	c := NewContainer(data.NewMemoryDB(), conf)
	defer c.db.Close()
	defer c.Close()

	if _, err := c.NewRoot(pk, sk, 0, nil); err == nil {
		t.Error("missing error")
	}
}

func TestContainer_NewRootReg(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()

	conf := getConf()
	reg := conf.Registry
	conf.Registry = nil

	// no core regsitry
	c := NewContainer(data.NewMemoryDB(), conf)
	defer c.db.Close()
	defer c.Close()

	if err := c.AddRegistry(reg); err != nil {
		t.Fatal(err)
	}

	// registry not foung
	_, err := c.NewRootReg(pk, sk, reg.Reference(), 0, reg.Types())
	if err != nil {
		t.Error(err)
	}
}

func testHasFeed(db data.DB, pk cipher.PubKey) (ok bool) {
	db.View(func(tx data.Tv) (_ error) {
		ok = tx.Feeds().Roots(pk) != nil
		return
	})
	return
}

func TestContainer_AddFeed(t *testing.T) {
	c := getCont()
	defer c.db.Close()
	defer c.Close()

	pk, _ := cipher.GenerateKeyPair()
	if err := c.AddFeed(pk); err != nil {
		t.Error(err)
	}
	if !testHasFeed(c.db, pk) {
		t.Error("not added feed")
	}
}

func TestContainer_DelFeed(t *testing.T) {
	c := getCont()
	defer c.db.Close()
	defer c.Close()

	pk, _ := cipher.GenerateKeyPair()
	if err := c.AddFeed(pk); err != nil {
		t.Error(err)
	}
	if err := c.DelFeed(pk); err != nil {
		t.Error(err)
	}
	if testHasFeed(c.db, pk) {
		t.Error("undeleted feed")
	}
}
