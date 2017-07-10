package skyobject

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

func testValueDereferenceNil(t *testing.T) {
	//
}

func TestValue_Derefernece(t *testing.T) {

	type Value struct {
		Value string
	}

	type Single struct {
		Value Reference `skyobject:"schema=Value"`
	}

	type Array struct {
		Values References `skyobject:"schema=Value"`
	}

	reg := NewRegistry()
	reg.Register("Value", Value{})
	reg.Register("Single", Single{})
	reg.Register("Array", Array{})

	c := NewContainer(data.NewMemoryDB(), reg)

	pk, sk := cipher.GenerateKeyPair()

	root, err := c.NewRoot(pk, sk)

	if err != nil {
		t.Fatal(err)
	}

	hey := root.Save(Value{"hey"})

	root.Append(root.MustDynamic("Single", Single{hey}))

	t.Run("single", func(t *testing.T) {
		vals, err := root.Values()
		if err != nil {
			t.Fatal(err)
		}
		if len(vals) != 1 {
			t.Fatal("invalid values length")
		}
		val := vals[0]
		fld, err := val.FieldByName("Value")
		if err != nil {
			t.Fatal(err)
		}
		if fld.Kind() != reflect.Ptr {
			t.Fatal("invalid kind of field")
		}
		if bytes.Compare(fld.Data(), hey[:]) != 0 {
			t.Fatal("invalid reference")
		}
		der, err := fld.Dereference()
		if err != nil {
			t.Fatal(err)
		}
		valf, err := der.FieldByName("Value")
		if err != nil {
			t.Fatal(err)
		}
		if s, err := valf.String(); err != nil {
			t.Fatal(err)
		} else if s != "hey" {
			t.Fatal("wrong value:", s)
		}
	})

	t.Run("array", func(t *testing.T) {
		// TODO
	})

	t.Run("dynamic", func(t *testing.T) {
		// TODO
	})

}
