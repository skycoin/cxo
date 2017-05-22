package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func TestRoot_Encode(t *testing.T) {
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
	_, rp, err = r.Inject("cxo.User", User{"Alice", 20, nil})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(rp)
	if _, err := c.unpackRoot(&rp); err != nil {
		t.Fatal(err)
	}
}

func TestRoot_IsFull(t *testing.T) {
	c := getCont()
	pk, sk := cipher.GenerateKeyPair()
	r, err := c.NewRoot(pk, sk)
	if err != nil {
		t.Fatal(err)
	}
	if r.IsFull() {
		t.Error("detached root is full")
	}
	_, _, err = r.Inject("cxo.User", User{"Alice", 20, nil})
	if err != nil {
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
