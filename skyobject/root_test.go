package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

func TestRoot_IsReadOnly(t *testing.T) {
	c := getCont()
	pk, sk := cipher.GenerateKeyPair()
	r, err := c.NewRoot(pk, sk)
	if err != nil {
		t.Fatal(err)
	}
	if r.IsReadOnly() {
		t.Error("read only")
	}
	if _, err := r.Touch(); err != nil {
		t.Fatal(err)
	}
	r = c.LastRoot(pk)
	if !r.IsReadOnly() {
		t.Error("can edit")
	}
	r.Edit(sk)
	if r.IsReadOnly() {
		t.Error("read only")
	}
}

func TestRoot_IsAttached(t *testing.T) {
	c := getCont()
	pk, sk := cipher.GenerateKeyPair()
	r, err := c.NewRoot(pk, sk)
	if err != nil {
		t.Fatal(err)
	}
	if r.IsAttached() {
		t.Error("attached")
	}
	if _, err := r.Touch(); err != nil {
		t.Fatal(err)
	}
	if !r.IsAttached() {
		t.Error("detached")
	}
	r = c.LastRoot(pk)
	if !r.IsAttached() {
		t.Error("detached")
	}
}

func TestRoot_Edit(t *testing.T) {
	// Edit(sk cipher.SecKey)
	// implemented inside IsReadOnly test
}

func TestRoot_Registry(t *testing.T) {
	// Registry() (reg *Registry, err error)
	c := getCont()
	pk, sk := cipher.GenerateKeyPair()
	// core reg
	r, err := c.NewRoot(pk, sk)
	if err != nil {
		t.Fatal(err)
	}
	reg, err := r.Registry()
	if err != nil {
		t.Fatal(err)
	}
	if reg != c.CoreRegistry() {
		t.Error("wrong registry")
	}
	// root reg (fictive)
	r, err = c.NewRootReg(pk, sk, RegistryReference{}) // fictive reg. ref
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.Registry(); err == nil {
		t.Error("mising error")
	}
	// root reg (added)
	reg = NewRegistry()
	reg.Register("cxo.User", User{})
	reg.Done()
	r, err = c.NewRootReg(pk, sk, reg.Reference()) // not added yet
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.Registry(); err == nil {
		t.Error("mising error")
	}
	c.AddRegistry(reg)
	if gr, err := r.Registry(); err != nil {
		t.Error(err)
	} else if gr.Reference() != reg.Reference() {
		t.Error("wrong reg")
	}
}

func TestRoot_RegistryReference(t *testing.T) {
	// RegistryReference() RegistryReference
	c := getCont()
	pk, sk := cipher.GenerateKeyPair()
	// core reg
	r, err := c.NewRoot(pk, sk)
	if err != nil {
		t.Fatal(err)
	}
	if r.RegistryReference() != c.CoreRegistry().Reference() {
		t.Error("wrong reg ref")
	}
	// root reg (fictive)
	r, err = c.NewRootReg(pk, sk, RegistryReference{}) // fictive reg. ref
	if err != nil {
		t.Fatal(err)
	}
	if r.RegistryReference() != (RegistryReference{}) {
		t.Error("wrong reg ref")
	}
	// root reg (added)
	reg := NewRegistry()
	reg.Register("cxo.User", User{})
	c.AddRegistry(reg)
	r, err = c.NewRootReg(pk, sk, reg.Reference()) // not added yet
	if err != nil {
		t.Fatal(err)
	}
	if r.RegistryReference() != reg.Reference() {
		t.Error("wrong reg ref")
	}
}

func TestRoot_Touch(t *testing.T) {
	// Touch() (data.RootPack, error)
}

func TestRoot_Seq(t *testing.T) {
	// Seq() uint64
}

func TestRoot_Time(t *testing.T) {
	// Time() int64
}

func TestRoot_Pub(t *testing.T) {
	// Pub() cipher.PubKey
}

func TestRoot_Sig(t *testing.T) {
	// Sig() cipher.Sig
}

func TestRoot_Hash(t *testing.T) {
	// Hash() RootReference
}

func TestRoot_PrevHash(t *testing.T) {
	// PrevHash() RootReference
}

func TestRoot_IsFull(t *testing.T) {
	// IsFull() bool
	c := getCont()
	pk, sk := cipher.GenerateKeyPair()
	r, err := c.NewRoot(pk, sk)
	if err != nil {
		t.Fatal(err)
	}
	if r.IsFull() {
		t.Error("detached root is full")
	}
	if alice, err := r.Dynamic("cxo.User", User{"Alice", 20, nil}); err != nil {
		t.Fatal(err)
	} else if _, err = r.Append(alice); err != nil {
		t.Fatal(err)
	}
	if !r.IsFull() {
		t.Error("full root is not full")
	}
	lr := c.LastRoot(pk)
	if lr == nil {
		t.Fatal("missing last root")
	}
	if !lr.IsFull() {
		t.Error("full root is not full")
	}
	// todo: non-full roots
}

func TestRoot_Encode(t *testing.T) {
	// Encode() (rp data.RootPack)
	c := getCont()
	pk, sk := cipher.GenerateKeyPair()
	r, err := c.NewRoot(pk, sk)
	if err != nil {
		t.Fatal(err)
	}
	rp := r.Encode()
	var x encodedRoot
	if err := encoder.DeserializeRaw(rp.Root, &x); err != nil {
		t.Fatal(err)
	}
	if _, err := c.unpackRoot(&rp); err != nil {
		t.Fatal(err)
	}
	if alice, err := r.Dynamic("cxo.User", User{"Alice", 20, nil}); err != nil {
		t.Fatal(err)
	} else if rp, err = r.Append(alice); err != nil {
		t.Fatal(err)
	}
	t.Log(rp)
	if _, err := c.unpackRoot(&rp); err != nil {
		t.Fatal(err)
	}
	//
}

func TestRoot_Sign(t *testing.T) {
	// Sign() (sig cipher.Sig)
}

func TestRoot_HasRegistry(t *testing.T) {
	// HasRegistry() bool
	t.Run("has", func(t *testing.T) {
		c := getCont()
		pk, sk := cipher.GenerateKeyPair()
		r, err := c.NewRoot(pk, sk)
		if err != nil {
			t.Fatal(err)
		}
		if !r.HasRegistry() {
			t.Error("missing registry")
		}
	})
	t.Run("has not", func(t *testing.T) {
		c := NewContainer(data.NewMemoryDB(), nil)
		pk, sk := cipher.GenerateKeyPair()
		reg := NewRegistry()
		reg.Done()
		r, err := c.NewRootReg(pk, sk, reg.Reference())
		if err != nil {
			t.Fatal(err)
		}
		if r.HasRegistry() {
			t.Error("unexpected registry")
		}
		c.AddRegistry(reg)
		if !r.HasRegistry() {
			t.Error("missing registry")
		}
	})
}

func TestRoot_Get(t *testing.T) {
	// Get(ref Reference) ([]byte, bool)
}

// func TestRoot_DB(t *testing.T) {
// 	// DB() data.DB
// 	db := data.NewMemoryDB()
// 	c := NewContainer(db, nil)
// 	pk, sk := cipher.GenerateKeyPair()
// 	r, err := c.NewRootReg(pk, sk, RegistryReference{})
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if r.DB() != db {
// 		t.Error("wrong db")
// 	}
// }

func TestRoot_Save(t *testing.T) {
	// Save(i interface{}) Reference
}

func TestRoot_SaveArray(t *testing.T) {
	// SaveArray(i ...interface{}) References
}

func TestRoot_Dynamic(t *testing.T) {
	// Dynamic(schemaName string, i interface{}) (dr Dynamic,
}

func TestRoot_MustDynamic(t *testing.T) {
	// MustDynamic(schemaName string, i interface{}) (dr Dynamic)
}

func TestRoot_Inject(t *testing.T) {
	// Inject(schemaName string, i interface{}) (inj Dynamic,
}

func TestRoot_InjectMany(t *testing.T) {
	// InjectMany(schemaName string, i ...interface{}) (injs []Dynamic,
}

func TestRoot_Refs(t *testing.T) {
	// Refs() (refs []Dynamic)
}

func TestRoot_Replace(t *testing.T) {
	// Replace(refs []Dynamic) (prev []Dynamic, rp data.RootPack,
}

func TestRoot_ValueByDynamic(t *testing.T) {
	// ValueByDynamic(dr Dynamic) (val *Value, err error)
}

func TestRoot_ValueByStatic(t *testing.T) {
	// ValueByStatic(schemaName string, ref Reference) (val *Value,
}

func TestRoot_Values(t *testing.T) {
	// Values() (vals []*Value, err error)
}

func TestRoot_SchemaByName(t *testing.T) {
	// SchemaByName(name string) (s Schema, err error)
}

func TestRoot_SchemaByReference(t *testing.T) {
	// SchemaByReference(sr SchemaReference) (s Schema, err error)
}

func TestRoot_SchemaReferenceByName(t *testing.T) {
	// SchemaReferenceByName(name string) (sr SchemaReference,
}

func TestRoot_WantFunc(t *testing.T) {
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
	//

	// receiver
	receiver := NewContainer(data.NewMemoryDB(), nil)
	if r, err = receiver.AddRootPack(&rp); err != nil {
		t.Fatal(err)
	}
	// missing registry
	var i int
	err = r.WantFunc(func(Reference) (_ error) { i++; return })
	if err == nil {
		t.Fatal("missing error")
	}
	if i != 0 {
		t.Fatal("called")
	}
	// alice, eva, ammy
	receiver.AddRegistry(reg)
	var want []Reference
	err = r.WantFunc(func(ref Reference) (_ error) {
		want = append(want, ref)
		return
	})
	if err != nil {
		t.Fatal(err)
	}
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
		receiver.Set(alice.Object, aliceData)
	}
	want = []Reference{} // reset
	err = r.WantFunc(func(ref Reference) (_ error) {
		want = append(want, ref)
		return
	})
	if err != nil {
		t.Fatal(err)
	}
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
		receiver.Set(eva.Object, evaData)
	}
	want = []Reference{} // reset
	err = r.WantFunc(func(ref Reference) (_ error) {
		want = append(want, ref)
		return
	})
	if err != nil {
		t.Fatal(err)
	}
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
		receiver.Set(ammy.Object, ammyData)
	}
	want = []Reference{} // reset
	err = r.WantFunc(func(ref Reference) (_ error) {
		i++
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	if i != 0 {
		t.Fatal("wrong wants")
	}
}

func TestRoot_GotFunc(t *testing.T) {
	// GotFunc(gf GotFunc) (err error)
	reg := getRegisty()
	// sender
	sender := NewContainer(data.NewMemoryDB(), reg)
	pk, sk := cipher.GenerateKeyPair()
	r, err := sender.NewRoot(pk, sk)
	if err != nil {
		t.Fatal(err)
	}

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
	if r, err = receiver.AddRootPack(&rp); err != nil {
		t.Fatal(err)
	}
	// missing registry
	var i int
	err = r.GotFunc(func(Reference) (_ error) { i++; return })
	if err == nil {
		t.Fatal("missing error")
	}
	if i != 0 {
		t.Fatal("called")
	}
	// alice, eva, ammy
	receiver.AddRegistry(reg)
	err = r.GotFunc(func(ref Reference) (_ error) {
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
		receiver.Set(alice.Object, aliceData)
	}
	var got []Reference // reset
	err = r.GotFunc(func(ref Reference) (_ error) {
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
		receiver.Set(eva.Object, evaData)
	}
	got = []Reference{} // reset
	err = r.GotFunc(func(ref Reference) (_ error) {
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
		receiver.Set(ammy.Object, ammyData)
	}
	got = []Reference{} // reset
	err = r.GotFunc(func(ref Reference) (_ error) {
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

func TestRoot_GotOfFunc(t *testing.T) {
	// GotOfFunc(dr Dynamic, gf GotFunc) (err error)
	//
}
