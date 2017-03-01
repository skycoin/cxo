package node

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

func genPubKey() (pub cipher.PubKey) {
	pub, _ = cipher.GenerateKeyPair()
	return
}

func TestConnection_MarshalJSON(t *testing.T) {
	t.Run("MarshalJSON", func(t *testing.T) {
		pub := genPubKey()
		a := "127.0.0.1:8970"
		c := Connection{
			Addr: a,
			Pub:  pub,
		}
		got, err := c.MarshalJSON()
		if err != nil {
			t.Error(err)
			return
		}
		if got == nil {
			t.Error("mising encode value")
			return
		}
		t.Logf("json-encoded public key: %s", string(got))
		want := fmt.Sprintf(`{"pub":%q,"addr":%q}`, pub.Hex(), a)
		if string(got) != want {
			t.Errorf("wrong json-value: want %s, got %s", want, string(got))
		}
	})
	t.Run("encoding/json.Marshal", func(t *testing.T) {
		pub := genPubKey()
		a := "127.0.0.1:8970"
		c := &Connection{
			Addr: a,
			Pub:  pub,
		}
		got, err := json.Marshal(c)
		if err != nil {
			t.Error(err)
			return
		}
		if got == nil {
			t.Error("mising encoded value")
			return
		}
		want := fmt.Sprintf(`{"pub":%q,"addr":%q}`, pub.Hex(), a)
		if string(got) != want {
			t.Errorf("wrong json-value: want %s, got %s", want, string(got))
		}
	})
}

func TestConnection_UnmarshalJSON(t *testing.T) {
	t.Run("UnmarshalJSON", func(t *testing.T) {
		wpub := genPubKey()
		waddr := "127.0.0.1:8497"
		data := []byte(fmt.Sprintf(`{"pub":%q,"addr":%q}`, wpub.Hex(), waddr))
		var c Connection
		if err := c.UnmarshalJSON(data); err != nil {
			t.Error("unexpected error: ", err)
			return
		}
		if c.Pub != wpub {
			t.Error("wrong public key")
		}
		if c.Addr != waddr {
			t.Error("wrong address")
		}
	})
	t.Run("encoding.json.Unmarshal", func(t *testing.T) {
		wpub := genPubKey()
		waddr := "127.0.0.1:8497"
		data := []byte(fmt.Sprintf(`{"pub":%q,"addr":%q}`, wpub.Hex(), waddr))
		var c Connection
		if err := json.Unmarshal(data, &c); err != nil {
			t.Error("unexpected error: ", err)
			return
		}
		if c.Pub != wpub {
			t.Error("wrong public key")
		}
		if c.Addr != waddr {
			t.Error("wrong address")
		}
	})
	// invalid key
	t.Run("UnmarshalJSON invalid", func(t *testing.T) {
		data := []byte(fmt.Sprintf(`{"pub":%q,"addr":%q}`,
			"malformed key",
			"127.0.0.1:8497"))
		var c Connection
		if err := c.UnmarshalJSON(data); err == nil {
			t.Error("missing error")
		}
	})
	t.Run("encoding.json.Unmarshal invalid", func(t *testing.T) {
		data := []byte(fmt.Sprintf(`{"pub":%q,"addr":%q}`,
			"invalid data",
			"127.0.0.1:8497"))
		var c Connection
		if err := json.Unmarshal(data, &c); err == nil {
			t.Error("missing error")
			return
		}
	})
}
