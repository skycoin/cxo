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
		if c.db == nil {
			t.Error("missing database")
		} else if c.db != db {
			t.Error("wrong database")
		} else if c.coreRegistry != reg {
			t.Error("wrong core registry")
		} else {
			if reg.done != true {
				t.Error("missing (*Registry).Done in NewContainer")
			}
			if _, ok := c.db.Get(cipher.SHA256(reg.Reference())); !ok {
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
	} else if _, ok := c.db.Get(cipher.SHA256(reg.Reference())); !ok {
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
		alice, err := r.Dynamic("cxo.User", User{"Alice", 20, nil})
		if err != nil {
			t.Fatal(err)
		}
		rp, err := r.Append(alice)
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
	t.Run("panic", func(t *testing.T) {
		// unable to test
	})
	t.Run("many registries", func(t *testing.T) {
		reg1 := NewRegistry()
		reg1.Register("User", User{})
		reg2 := getRegisty()
		s := NewContainer(data.NewMemoryDB(), nil)
		s.AddRegistry(reg1)
		s.AddRegistry(reg2)
		pk, sk := cipher.GenerateKeyPair()
		r1, err := s.NewRootReg(pk, sk, reg1.Reference())
		if err != nil {
			t.Fatal(err)
		}
		rp1, err := r1.Touch()
		if err != nil {
			t.Fatal(err)
		}
		r2, err := s.NewRootReg(pk, sk, reg2.Reference())
		if err != nil {
			t.Fatal(err)
		}
		rp2, err := r2.Touch()
		if err != nil {
			t.Fatal(err)
		}
		d := NewContainer(data.NewMemoryDB(), nil)
		if _, err := d.AddRootPack(&rp1); err != nil {
			t.Fatal(err)
		}
		if _, err := d.AddRootPack(&rp2); err != nil {
			t.Fatal(err)
		}
		if !d.WantRegistry(reg1.Reference()) {
			t.Error("don't want registry")
		}
		if !d.WantRegistry(reg2.Reference()) {
			t.Error("don't want registry")
		}
	})
}

func TestContainer_Registries(t *testing.T) {
	t.Run("no", func(t *testing.T) {
		c := NewContainer(data.NewMemoryDB(), nil)
		if c.Registries() != nil {
			t.Fatal("unexpected registries")
		}
	})
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
	if c.db != db {
		t.Error("wrong db")
	}
}

func TestContainer_Get(t *testing.T) {
	db := data.NewMemoryDB()
	c := NewContainer(db, nil)
	value := []byte("hey ho")
	key := cipher.SumSHA256(value)
	db.Set(key, value)
	data, ok := c.Get(Reference(key))
	if !ok {
		t.Error("misisng object")
	} else if string(data) != string(value) {
		t.Error("wrong value")
	}
}

func TestContainer_Set(t *testing.T) {
	db := data.NewMemoryDB()
	c := NewContainer(db, nil)
	value := []byte("hey ho")
	key := cipher.SumSHA256(value)
	c.Set(Reference(key), value)
	data, ok := db.Get(key)
	if !ok {
		t.Error("misisng object")
	} else if string(data) != string(value) {
		t.Error("wrong value")
	}
}

func TestContainer_NewRoot(t *testing.T) {
	t.Run("no core", func(t *testing.T) {
		c := NewContainer(data.NewMemoryDB(), nil)
		pk, sk := cipher.GenerateKeyPair()
		if _, err := c.NewRoot(pk, sk); err == nil {
			t.Error("misisng error")
		} else if err != ErrNoCoreRegistry {
			t.Error("wrong error")
		}
	})
	t.Run("invalid pk sk", func(t *testing.T) {
		c := NewContainer(data.NewMemoryDB(), NewRegistry())
		pk, sk := cipher.GenerateKeyPair()
		if _, err := c.NewRoot(cipher.PubKey{}, sk); err == nil {
			t.Error("misisng error")
		}
		if _, err := c.NewRoot(pk, cipher.SecKey{}); err == nil {
			t.Error("misisng error")
		}
	})
	t.Run("first", func(t *testing.T) {
		reg := NewRegistry()
		db := data.NewMemoryDB()
		c := NewContainer(db, reg)
		pk, sk := cipher.GenerateKeyPair()
		r, err := c.NewRoot(pk, sk)
		if err != nil {
			t.Fatal(err)
		}
		if r.RegistryReference() != reg.Reference() {
			t.Error("wrong reg ref")
		}
		if r.Seq() != 0 {
			t.Error("wrong seq")
		}
		if r.IsReadOnly() {
			t.Error("read only")
		}
		if r.IsAttached() {
			t.Error("attached")
		}
		if r.PrevHash() != (RootReference{}) {
			t.Error("wrong prev. hash")
		}
	})
	t.Run("non first", func(t *testing.T) {
		reg := NewRegistry()
		db := data.NewMemoryDB()
		c := NewContainer(db, reg)
		pk, sk := cipher.GenerateKeyPair()
		r, err := c.NewRoot(pk, sk)
		if err != nil {
			t.Fatal(err)
		}
		rp, err := r.Touch()
		if err != nil {
			t.Fatal(err)
		}
		r, err = c.NewRoot(pk, sk)
		if err != nil {
			t.Fatal(err)
		}
		if r.RegistryReference() != reg.Reference() {
			t.Error("wrong reg ref")
		}
		if r.Seq() != 1 {
			t.Error("wrong seq")
		}
		if r.IsReadOnly() {
			t.Error("read only")
		}
		if r.IsAttached() {
			t.Error("attached")
		}
		if r.PrevHash() != RootReference(rp.Hash) {
			t.Error("wrong prev. hash")
		}
	})
}

func TestContainer_NewRootReg(t *testing.T) {
	t.Run("invalid pk sk", func(t *testing.T) {
		reg := NewRegistry()
		reg.Done()
		rr := reg.Reference()
		c := NewContainer(data.NewMemoryDB(), nil)
		pk, sk := cipher.GenerateKeyPair()
		if _, err := c.NewRootReg(cipher.PubKey{}, sk, rr); err == nil {
			t.Error("misisng error")
		}
		if _, err := c.NewRootReg(pk, cipher.SecKey{}, rr); err == nil {
			t.Error("misisng error")
		}
	})
	t.Run("first", func(t *testing.T) {
		reg := NewRegistry()
		reg.Done()
		rr := reg.Reference()
		db := data.NewMemoryDB()
		c := NewContainer(db, nil)
		pk, sk := cipher.GenerateKeyPair()
		r, err := c.NewRootReg(pk, sk, rr)
		if err != nil {
			t.Fatal(err)
		}
		if r.RegistryReference() != rr {
			t.Error("wrong reg ref")
		}
		if r.Seq() != 0 {
			t.Error("wrong seq")
		}
		if r.IsReadOnly() {
			t.Error("read only")
		}
		if r.IsAttached() {
			t.Error("attached")
		}
		if r.PrevHash() != (RootReference{}) {
			t.Error("wrong prev. hash")
		}
	})
	t.Run("non first", func(t *testing.T) {
		reg := NewRegistry()
		reg.Done()
		rr := reg.Reference()
		db := data.NewMemoryDB()
		c := NewContainer(db, nil)
		pk, sk := cipher.GenerateKeyPair()
		r, err := c.NewRootReg(pk, sk, rr)
		if err != nil {
			t.Fatal(err)
		}
		rp, err := r.Touch()
		if err != nil {
			t.Fatal(err)
		}
		r, err = c.NewRootReg(pk, sk, rr)
		if err != nil {
			t.Fatal(err)
		}
		if r.RegistryReference() != rr {
			t.Error("wrong reg ref")
		}
		if r.Seq() != 1 {
			t.Error("wrong seq")
		}
		if r.IsReadOnly() {
			t.Error("read only")
		}
		if r.IsAttached() {
			t.Error("attached")
		}
		if r.PrevHash() != RootReference(rp.Hash) {
			t.Error("wrong prev. hash")
		}
	})
}

func isRefsEqual(a, b []Dynamic) (equal bool) {
	if len(a) != len(b) {
		return false
	}
	for i, r := range a {
		if r != b[i] {
			return false
		}
	}
	return true
}

func TestContainer_AddRootPack(t *testing.T) {
	t.Run("add", func(t *testing.T) {
		// create
		c := NewContainer(data.NewMemoryDB(), NewRegistry())
		pk, sk := cipher.GenerateKeyPair()
		cr, err := c.NewRoot(pk, sk)
		if err != nil {
			t.Fatal(err)
		}
		rp, err := cr.Touch()
		if err != nil {
			t.Fatal(err)
		}
		// add
		a := NewContainer(data.NewMemoryDB(), nil)
		ar, err := a.AddRootPack(&rp)
		if err != nil {
			t.Fatal(err)
		}
		if ar.Seq() != cr.Seq() {
			t.Error("wrong seq")
		}
		if ar.Hash() != cr.Hash() {
			t.Error("wrong hash")
		}
		if ar.PrevHash() != cr.PrevHash() {
			t.Error("wrong prev reference")
		}
		if ar.RegistryReference() != cr.RegistryReference() {
			t.Error("wrong reg. ref")
		}
		if !isRefsEqual(ar.Refs(), cr.Refs()) {
			t.Error("wrong refs")
		}
		if !ar.IsAttached() {
			t.Error("not attached")
		}
		if !ar.IsReadOnly() {
			t.Error("can edit")
		}
	})
	t.Run("deserialize error", func(t *testing.T) {
		var rp data.RootPack
		_, err := NewContainer(data.NewMemoryDB(), nil).AddRootPack(&rp)
		if err == nil {
			t.Error("missing error")
		}
	})
	t.Run("invalid signature", func(t *testing.T) {
		// create
		c := NewContainer(data.NewMemoryDB(), NewRegistry())
		pk, sk := cipher.GenerateKeyPair()
		cr, err := c.NewRoot(pk, sk)
		if err != nil {
			t.Fatal(err)
		}
		rp, err := cr.Touch()
		if err != nil {
			t.Fatal(err)
		}
		wrongHashForSign := cipher.SumSHA256([]byte("hey ho"))
		rp.Sig = cipher.SignHash(wrongHashForSign, sk)
		// add
		a := NewContainer(data.NewMemoryDB(), nil)
		_, err = a.AddRootPack(&rp)
		if err == nil {
			t.Error("missing error")
		}
	})
	t.Run("wrong hash", func(t *testing.T) {
		// create
		c := NewContainer(data.NewMemoryDB(), NewRegistry())
		pk, sk := cipher.GenerateKeyPair()
		cr, err := c.NewRoot(pk, sk)
		if err != nil {
			t.Fatal(err)
		}
		rp, err := cr.Touch()
		if err != nil {
			t.Fatal(err)
		}
		rp.Hash = cipher.SumSHA256([]byte("hey ho"))
		// add
		a := NewContainer(data.NewMemoryDB(), nil)
		_, err = a.AddRootPack(&rp)
		if err == nil {
			t.Error("missing error")
		}
	})
}

func TestContainer_LastRoot(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		c := NewContainer(data.NewMemoryDB(), nil)
		pk, _ := cipher.GenerateKeyPair()
		if c.LastRoot(pk) != nil {
			t.Error("got root")
		}
	})
	t.Run("found", func(t *testing.T) {
		c := NewContainer(data.NewMemoryDB(), NewRegistry())
		pk, sk := cipher.GenerateKeyPair()
		r, err := c.NewRoot(pk, sk)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = r.Touch(); err != nil {
			t.Fatal(err)
		}
		if lr := c.LastRoot(pk); lr == nil {
			t.Error("can't get root")
		} else if lr.Seq() != r.Seq() {
			t.Error("invalid last root")
		} else if !lr.IsReadOnly() {
			t.Error("can edit")
		}
		if _, err = r.Touch(); err != nil {
			t.Fatal(err)
		}
		if lr := c.LastRoot(pk); lr == nil {
			t.Error("can't get root")
		} else if lr.Seq() != r.Seq() {
			t.Error("invalid last root")
		} else if !lr.IsReadOnly() {
			t.Error("can edit")
		}
	})
}

func TestContainer_LastRootSk(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		c := NewContainer(data.NewMemoryDB(), nil)
		pk, sk := cipher.GenerateKeyPair()
		if c.LastRootSk(pk, sk) != nil {
			t.Error("got root")
		}
	})
	t.Run("found", func(t *testing.T) {
		c := NewContainer(data.NewMemoryDB(), NewRegistry())
		pk, sk := cipher.GenerateKeyPair()
		r, err := c.NewRoot(pk, sk)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = r.Touch(); err != nil {
			t.Fatal(err)
		}
		if lr := c.LastRootSk(pk, sk); lr == nil {
			t.Error("can't get root")
		} else if lr.Seq() != r.Seq() {
			t.Error("invalid last root")
		} else if lr.IsReadOnly() {
			t.Error("can't edit")
		}
		if _, err = r.Touch(); err != nil {
			t.Fatal(err)
		}
		if lr := c.LastRootSk(pk, sk); lr == nil {
			t.Error("can't get root")
		} else if lr.Seq() != r.Seq() {
			t.Error("invalid last root")
		} else if lr.IsReadOnly() {
			t.Error("can't edit")
		}
	})
}

func TestContainer_LastFullRoot(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		c := NewContainer(data.NewMemoryDB(), nil)
		pk, sk := cipher.GenerateKeyPair()
		if c.LastRootSk(pk, sk) != nil {
			t.Error("got root")
		}
	})
	t.Run("found", func(t *testing.T) {
		defer shouldNotPanic(t) // for insurance

		c := getCont()
		pk, sk := cipher.GenerateKeyPair()
		r, err := c.NewRoot(pk, sk)
		if err != nil {
			t.Error(err)
			return
		}
		r.Append(r.MustDynamic("cxo.User", User{"Alice", 15, nil}))
		r.Append(r.MustDynamic("cxo.User", User{"Eva", 16, nil}))
		r.Append(r.MustDynamic("cxo.User", User{"Ammy", 17, nil}))
		full := c.LastFullRoot(pk)
		if full == nil {
			t.Error("misisng last full root")
		}
	})
}

func TestContainer_Feeds(t *testing.T) {
	c := NewContainer(data.NewMemoryDB(), nil)
	t.Run("no", func(t *testing.T) {
		if len(c.Feeds()) != 0 {
			t.Error("got feeds")
		}
	})
	opk, osk := cipher.GenerateKeyPair()
	t.Run("one", func(t *testing.T) {
		reg := NewRegistry()
		reg.Register("cxo.User", User{})
		c.AddRegistry(reg)
		r, err := c.NewRootReg(opk, osk, reg.Reference())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.Touch(); err != nil {
			t.Fatal(err)
		}
		if fs := c.Feeds(); len(fs) != 1 {
			t.Fatal("missing feed")
		} else if fs[0] != opk {
			t.Fatal("wrong feed")
		}
	})
	tpk, tsk := cipher.GenerateKeyPair()
	t.Run("two", func(t *testing.T) {
		reg := NewRegistry()
		reg.Register("cxo.User", User{})
		c.AddRegistry(reg)
		r, err := c.NewRootReg(tpk, tsk, reg.Reference())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.Touch(); err != nil {
			t.Fatal(err)
		}
		if fs := c.Feeds(); len(fs) != 2 {
			t.Fatal("missing feed")
		} else {
			if !((fs[0] == opk && fs[1] == tpk) ||
				(fs[0] == tpk && fs[1] == opk)) {
				t.Error("wrong feeds")
			}
		}
	})
	t.Run("delete one", func(t *testing.T) {
		c.DelRootsBefore(opk, 100500)
		if fs := c.Feeds(); len(fs) != 2 {
			t.Fatal("wrong amount of feeds")
		}
	})
	t.Run("delete two", func(t *testing.T) {
		c.DelRootsBefore(tpk, 100500)
		if len(c.Feeds()) != 2 {
			t.Fatal("wrong amount of feeds")
		}
	})
}

func TestContainer_WantFeed(t *testing.T) {
	// WantFunc(wf WantFunc) (err error)
	reg := getRegisty()
	// sender
	sender := NewContainer(data.NewMemoryDB(), reg)
	pk, sk := cipher.GenerateKeyPair()
	r, err := sender.NewRoot(pk, sk)
	if err != nil {
		t.Fatal(err)
	}
	//
	var aliceUser, evaUser, ammyUser User = User{"Alice", 20, nil},
		User{"Eva", 21, nil}, User{"Ammy", 16, nil}
	//
	var alice, eva, ammy Dynamic
	if alice, err = r.Dynamic("cxo.User", aliceUser); err != nil {
		t.Fatal(err)
	}
	if eva, err = r.Dynamic("cxo.User", evaUser); err != nil {
		t.Fatal(err)
	}
	if ammy, err = r.Dynamic("cxo.User", ammyUser); err != nil {
		t.Fatal(err)
	}
	//
	rp, err := r.Append(alice, eva, ammy)
	if err != nil {
		t.Fatal(err)
	}
	// receiver
	receiver := NewContainer(data.NewMemoryDB(), nil)
	if _, err = receiver.AddRootPack(&rp); err != nil {
		t.Fatal(err)
	}
	// missing registry
	var i int
	receiver.WantFeed(pk, func(Reference) (_ error) { i++; return })
	if i != 0 {
		t.Fatal("called")
	}
	// alice, eva, ammy
	receiver.AddRegistry(reg)
	var want []Reference
	receiver.WantFeed(pk, func(ref Reference) (_ error) {
		want = append(want, ref)
		return
	})
	if len(want) != 3 {
		t.Fatal("wrong wants")
	}
	set := map[Reference]struct{}{
		alice.Object: {},
		eva.Object:   {},
		ammy.Object:  {},
	}
	for _, w := range want {
		if _, ok := set[w]; !ok {
			t.Fatal("wrong ref")
		}
	}
	// alice
	if aliceData, ok := sender.Get(alice.Object); !ok {
		t.Fatal("misisng object")
	} else {
		receiver.db.Set(cipher.SHA256(alice.Object), aliceData)
	}
	want = []Reference{} // reset
	receiver.WantFeed(pk, func(ref Reference) (_ error) {
		want = append(want, ref)
		return
	})
	if len(want) != 2 {
		t.Fatal("wrong wants")
	}
	delete(set, alice.Object)
	for _, w := range want {
		if _, ok := set[w]; !ok {
			t.Fatal("wrong ref")
		}
	}
	// eva
	if evaData, ok := sender.Get(eva.Object); !ok {
		t.Fatal("misisng object")
	} else {
		receiver.db.Set(cipher.SHA256(eva.Object), evaData)
	}
	want = []Reference{} // reset
	receiver.WantFeed(pk, func(ref Reference) (_ error) {
		want = append(want, ref)
		return
	})
	if len(want) != 1 {
		t.Fatal("wrong wants")
	}
	delete(set, eva.Object)
	for _, w := range want {
		if _, ok := set[w]; !ok {
			t.Fatal("wrong ref")
		}
	}
	// ammy
	if ammyData, ok := sender.Get(ammy.Object); !ok {
		t.Fatal("misisng object")
	} else {
		receiver.db.Set(cipher.SHA256(ammy.Object), ammyData)
	}
	want = []Reference{} // reset
	receiver.WantFeed(pk, func(ref Reference) (_ error) {
		i++
		return
	})
	if i != 0 {
		t.Fatal("wrong wants")
	}
}

func TestContainer_GotFeed(t *testing.T) {
	// GotFunc(gf GotFunc) (err error)
	reg := getRegisty()
	// sender
	sender := NewContainer(data.NewMemoryDB(), reg)
	pk, sk := cipher.GenerateKeyPair()
	r, err := sender.NewRoot(pk, sk)
	if err != nil {
		t.Fatal(err)
	}

	//
	var aliceUser, evaUser, ammyUser User = User{"Alice", 20, nil},
		User{"Eva", 21, nil}, User{"Ammy", 16, nil}
	//
	var alice, eva, ammy Dynamic
	if alice, err = r.Dynamic("cxo.User", aliceUser); err != nil {
		t.Fatal(err)
	}
	if eva, err = r.Dynamic("cxo.User", evaUser); err != nil {
		t.Fatal(err)
	}
	if ammy, err = r.Dynamic("cxo.User", ammyUser); err != nil {
		t.Fatal(err)
	}
	//
	rp, err := r.Append(alice, eva, ammy)
	if err != nil {
		t.Fatal(err)
	}
	//

	// receiver
	receiver := NewContainer(data.NewMemoryDB(), nil)
	if r, err = receiver.AddRootPack(&rp); err != nil {
		t.Fatal(err)
	}
	// alice, eva, ammy
	var i int
	receiver.AddRegistry(reg)
	receiver.GotFeed(pk, func(ref Reference) (_ error) {
		i++
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	if i != 0 {
		t.Fatal("called")
	}
	// alice
	if aliceData, ok := sender.Get(alice.Object); !ok {
		t.Fatal("misisng object")
	} else {
		receiver.db.Set(cipher.SHA256(alice.Object), aliceData)
	}
	var got []Reference // reset
	receiver.GotFeed(pk, func(ref Reference) (_ error) {
		got = append(got, ref)
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatal("wrong gots", got)
	}
	set := map[Reference]struct{}{alice.Object: {}}
	for _, w := range got {
		if _, ok := set[w]; !ok {
			t.Fatal("wrong ref")
		}
	}
	// eva
	if evaData, ok := sender.Get(eva.Object); !ok {
		t.Fatal("misisng object")
	} else {
		receiver.db.Set(cipher.SHA256(eva.Object), evaData)
	}
	got = []Reference{} // reset
	receiver.GotFeed(pk, func(ref Reference) (_ error) {
		got = append(got, ref)
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatal("wrong gots")
	}
	set[eva.Object] = struct{}{}
	for _, w := range got {
		if _, ok := set[w]; !ok {
			t.Fatal("wrong ref")
		}
	}
	// ammy
	if ammyData, ok := sender.Get(ammy.Object); !ok {
		t.Fatal("misisng object")
	} else {
		receiver.db.Set(cipher.SHA256(ammy.Object), ammyData)
	}
	got = []Reference{} // reset
	receiver.GotFeed(pk, func(ref Reference) (_ error) {
		got = append(got, ref)
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatal("wrong gots")
	}
	set[ammy.Object] = struct{}{}
	for _, w := range got {
		if _, ok := set[w]; !ok {
			t.Fatal("wrong ref")
		}
	}
}

func TestContainer_DelFeed(t *testing.T) {
	c := getCont()
	pk, sk := cipher.GenerateKeyPair()
	r, err := c.NewRoot(pk, sk)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Touch(); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Touch(); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Touch(); err != nil {
		t.Fatal(err)
	}
	c.DelFeed(pk)
	if c.LastRoot(pk) != nil {
		t.Error("got feed")
	}
}

func TestContainer_DelRootsBefore(t *testing.T) {
	c := getCont()
	pk, sk := cipher.GenerateKeyPair()
	r, err := c.NewRoot(pk, sk)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Touch(); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Touch(); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Touch(); err != nil {
		t.Fatal(err)
	}
	fs, ok := c.db.Stat().Feeds[pk]
	if !ok {
		t.Error("missing feed")
	} else if fs.Roots != 3 {
		t.Error("wrong amount of roots")
	}
	c.DelRootsBefore(pk, r.Seq())
	fs, ok = c.db.Stat().Feeds[pk]
	if !ok {
		t.Error("missing feed")
	} else if fs.Roots != 1 {
		t.Error("wrong amount of roots")
	}
	c.DelRootsBefore(pk, r.Seq()+1)
	if len(c.db.Stat().Feeds) != 1 {
		t.Error("wrong amount of feeds")
	}
}

func TestContainer_GC(t *testing.T) {
	c := getCont()
	pk, sk := cipher.GenerateKeyPair()
	r, err := c.NewRoot(pk, sk)
	if err != nil {
		t.Fatal(err)
	}
	alice, err := r.Dynamic("cxo.User", User{"Alice", 20, nil})
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Append(alice)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Replace([]Dynamic{
		r.MustDynamic("cxo.User", User{"Eva", 21, nil}),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Replace([]Dynamic{
		r.MustDynamic("cxo.User", User{"Ammy", 16, nil}),
	})
	if err != nil {
		t.Fatal(err)
	}
	c.GC(false)
	stat := c.db.Stat()
	if f := stat.Feeds[pk]; f.Roots != 1 {
		t.Error("not clear", f.Roots)
	}
	// ammy + registry
	if stat.Objects != 2 {
		t.Error("not clear", stat.Objects)
	}
	// clean up (except core)
	c.DelRootsBefore(pk, 1005000)
	c.GC(false)
	stat = c.db.Stat()
	if len(stat.Feeds) != 1 {
		t.Error("looks like a feed was deleted")
	}
	// registry
	if stat.Objects != 1 {
		t.Error("not clear", stat.Objects)
	}
	if len(c.registries) != 1 {
		t.Error("core registry removed")
	}
}

func TestContainer_RangeFeed(t *testing.T) {
	c := getCont()
	pk, sk := cipher.GenerateKeyPair()

	t.Run("empty feed", func(t *testing.T) {
		var called int
		err := c.RangeFeed(pk, func(*Root) (_ error) {
			called++
			return
		})
		if err != nil {
			t.Error(err)
		}
		if called != 0 {
			t.Error("unexpected calling of RageFeedFunc")
		}
	})

	root, err := c.NewRoot(pk, sk)
	if err != nil {
		t.Fatal(err)
	}

	alice := root.MustDynamic("cxo.User", User{"Alice", 19, nil})
	eva := root.MustDynamic("cxo.User", User{"Eva", 21, nil})
	ammy := root.MustDynamic("cxo.User", User{"Ammy", 23, nil})

	root.Append(alice)
	root.Replace([]Dynamic{eva})
	root.Replace([]Dynamic{ammy})

	t.Run("stop range", func(t *testing.T) {
		err := c.RangeFeed(pk, func(*Root) error {
			return ErrStopRange
		})
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("order", func(t *testing.T) {
		var called int
		order := []Dynamic{alice, eva, ammy}
		err := c.RangeFeed(pk, func(r *Root) (_ error) {
			if dr, err := r.Index(0); err != nil {
				return err
			} else if called >= len(order) {
				t.Error("called too many times")
				return ErrStopRange
			} else if dr != order[called] {
				t.Error("wrong order")
				return ErrStopRange
			}
			called++
			return
		})
		if err != nil {
			t.Error(err)
		}
		if called != 3 {
			t.Error("too few/many times called")
		}
	})

}

func TestContainer_RootByHash(t *testing.T) {
	r := getRoot(t)
	c := r.cnt // container of the root

	r.Append(r.MustDynamic("cxo.User", User{}))

	gr, ok := c.RootByHash(r.Hash())
	if !ok {
		t.Fatal("misisng")
	}

	if gr.Hash() != r.Hash() {
		t.Error("wrong root given")
	}

}
