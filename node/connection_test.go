package node

import (
	"fmt"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

func genPubKey() (pub cipher.PubKey) {
	pub, _ = cipher.GenerateKeyPair()
	return
}

func TestConnection_MarshalJSON(t *testing.T) {
	pub := genPubKey()
	a := "127.0.0.1:8970"
	c := Connection{
		Addr: a,
		Pub:  pub,
	}
	data, err := c.MarshalJSON()
	if err != nil {
		t.Error(err)
		return
	}
	if data == nil {
		t.Error("mising data")
		return
	}
	want := fmt.Sprintf(`{"pub":%q,"addr":%q}`, pub.Hex(), a)
	got := string(data)
	if got != want {
		t.Errorf("wrong json-value: want %s, got %s", want, got)
	}
}

func TestConnection_UnmarshalJSON(t *testing.T) {
	pub := genPubKey()
	a := "127.0.0.1:8970"
	c := Connection{
		Addr: a,
		Pub:  pub,
	}
	data, err := c.MarshalJSON()
	if err != nil {
		t.Error(err)
		return
	}
	c.Addr = ""
	c.Pub = cipher.PubKey{}
	if err = c.UnmarshalJSON(data); err != nil {
		t.Error(err)
		return
	}
	if c.Addr != a {
		t.Error("wrong address")
	}
	if c.Pub != pub {
		t.Error("wrong pub key")
	}
}
