package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

func testValueDereferenceNil(t *testing.T) {
	//
}

func TestValue_Derefernece(t *testing.T) {

	type Single struct {
		Ref Reference `skyobject:"schema=Single"`
	}

	type Array struct {
		Refs References `skyobject:"schema=Single"`
	}

	reg := NewRegistry()
	reg.Register("Single", Single{})
	reg.Register("Array", Array{})

	c := NewContainer(data.NewMemoryDB(), reg)

	pk, sk := cipher.GenerateKeyPair()

	root, err := c.NewRoot(pk, sk)

	if err != nil {
		t.Fatal(err)
	}

	_ = root // TODO

	t.Run("invalid type", func(t *testing.T) {
		// val := &Value{root}- // TODO
	})

	t.Run("static", func(t *testing.T) {
		// TODO
	})

	t.Run("dynamic", func(t *testing.T) {
		// TODO
	})

}
