package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func TestRoot_Encode(t *testing.T) {
	type User struct{ Name string }
	reg := NewRegistry()
	reg.Register("test.User", User{})
	c := NewContainer(reg)
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
	if _, err := c.unpackRoot(rp); err != nil {
		t.Fatal(err)
	}
	_, rp, err = r.Inject("test.User", User{"Alice"})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(rp)
	if _, err := c.unpackRoot(rp); err != nil {
		t.Fatal(err)
	}
}
