package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

func TestDynamic_IsBlank(t *testing.T) {
	// IsBlank() bool

	var dr Dynamic
	if !dr.IsBlank() {
		t.Error("balnk is not blank")
	}
	dr.SchemaRef = SchemaRef{1, 2, 3}
	if dr.IsBlank() {
		t.Error("non-balnk is blank")
	}
	dr.SchemaRef, dr.Object = SchemaRef{}, cipher.SHA256{1, 2, 3}
	if dr.IsBlank() {
		t.Error("non-balnk is blank")
	}
}

func TestDynamic_IsValid(t *testing.T) {
	// IsValid() bool

	// TODO (kostyarin): low priority
}

func TestDynamic_Short(t *testing.T) {
	// Short() string

	// TODO (kostyarin): low priority
}

func TestDynamic_String(t *testing.T) {
	// String() string

	// TODO (kostyarin): low priority
}

func TestDynamic_Schema(t *testing.T) {
	// Schema() (sch Schema, err error)

	c := getCont()
	defer c.db.Close()
	defer c.Close()

	pk, sk := cipher.GenerateKeyPair()
	if err := c.AddFeed(pk); err != nil {
		t.Fatal(err)
	}

	pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
	if err != nil {
		t.Fatal(err)
	}

	// blank
	dr := pack.Dynamic(nil)
	if sch, err := dr.Schema(); err != nil {
		t.Error(err)
	} else if sch != nil {
		t.Error("blank Dynamic returns non-nil Schema")
	}

	// non-blank
	dr = pack.Dynamic(User{"Alice", 21, nil})
	if sch, err := dr.Schema(); err != nil {
		t.Error(err)
	} else if sch == nil {
		t.Error("non-blank Dynamic returns nil Schema")
	} else if sch.Name() != "cxo.User" {
		t.Error("unexpected schema:", sch.String())
	}

	// invaid
	dr = pack.Dynamic(User{"Alice", 21, nil})
	dr.SchemaRef = SchemaRef{}
	if _, err := dr.Schema(); err == nil {
		t.Error("missing error")
	}

	// not found
	dr = pack.Dynamic(User{"Alice", 21, nil})
	dr.SchemaRef = SchemaRef{1, 2, 3} // not empty
	if _, err := dr.Schema(); err == nil {
		t.Error("missing error")
	}

	// detached (blank)
	dr = Dynamic{}
	if sch, err := dr.Schema(); err != nil {
		t.Error(err)
	} else if sch != nil {
		t.Error("non nil schema")
	}

	// detached (non-blank)
	dr.SchemaRef = SchemaRef{1, 2, 3}
	if _, err := dr.Schema(); err == nil {
		t.Error("missing error")
	}

}

func TestDynamic_Value(t *testing.T) {
	// Value() (obj interface{}, err error)

	c := getCont()
	defer c.db.Close()
	defer c.Close()

	pk, sk := cipher.GenerateKeyPair()
	if err := c.AddFeed(pk); err != nil {
		t.Fatal(err)
	}

	pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
	if err != nil {
		t.Fatal(err)
	}

	// invalid
	dr := pack.Dynamic(nil)
	dr.SchemaRef = SchemaRef{1, 2, 3}
	if _, err := dr.Value(); err == nil {
		t.Error(err)
	}

	// detached
	dr = Dynamic{SchemaRef: SchemaRef{1, 2, 3}}
	if _, err := dr.Value(); err == nil {
		t.Error("missing error")
	}

	// blank
	dr = pack.Dynamic(nil)
	if valInterface, err := dr.Value(); err != nil {
		t.Error(err)
	} else if valInterface != nil {
		t.Error("unexpected result:", valInterface)
	}

	// already have
	alice := &User{"Alice", 21, nil}
	dr = pack.Dynamic(alice)
	if valInterface, err := dr.Value(); err != nil {
		t.Error(err)
	} else if valInterface == nil {
		t.Error("Value is nil")
	} else if usr, ok := valInterface.(*User); !ok {
		t.Errorf("unexpected type: %T", valInterface)
	} else if usr != alice {
		t.Error("wrong pointer")
	}

	// non-pointer
	dr = pack.Dynamic(User{"Alice", 21, nil})
	if valInterface, err := dr.Value(); err != nil {
		t.Error(err)
	} else if valInterface == nil {
		t.Error("Value is nil")
	} else if usr, ok := valInterface.(*User); !ok {
		t.Errorf("unexpected type %T", valInterface)
	} else if usr.Name != "Alice" || usr.Age != 21 {
		t.Error("wrong value")
	}

	// get from database
	pack.Append(alice)
	if _, err := pack.Save(); err != nil {
		t.Fatal(err)
	}

	pack, err = c.Unpack(pack.Root(), 0, c.CoreRegistry().Types(), sk)
	if err != nil {
		t.Fatal(err)
	}

	if valInterface, err := pack.RefByIndex(0); err != nil {
		t.Error(err)
	} else if valInterface == nil {
		t.Error("Value is nil")
	} else if usr, ok := valInterface.(*User); !ok {
		t.Errorf("unexpected type %T", valInterface)
	} else if usr.Name != "Alice" || usr.Age != 21 {
		t.Error("wrong value")
	}

}

func TestDynamic_SetValue(t *testing.T) {
	// SetValue(obj interfae{}) (err error)

	//
}

func TestDynamic_Clear(t *testing.T) {
	// Clear() (err error)

	//
}

func TestDynamic_Copy(t *testing.T) {
	// Copy() (cp Dynamic)

	//
}
