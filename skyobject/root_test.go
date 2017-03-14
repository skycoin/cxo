package skyobject

import (
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

func TestContainer_NewRoot(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if NewContainer(data.NewDB()).NewRoot() == nil {
			t.Error("returns nil")
		}
	})
	t.Run("filling down", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		tb := time.Now().UnixNano() // time point
		r := c.NewRoot()
		te := time.Now().UnixNano() // time point
		if r.container != c {
			t.Error("wrong container field")
		}
		if r.Schema != (cipher.SHA256{}) {
			t.Error("non-empty schema")
		}
		if r.Root != (cipher.SHA256{}) {
			t.Error("non-empty root")
		}
		if r.Time < tb || r.Time > te {
			t.Error("invalid timestamp")
		}
		if r.Seq != 0 {
			t.Error("invalid seq: ", r.Seq)
		}
	})
}

func TestRoot_Set(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		root := c.NewRoot()
		if err := root.Set(nil); err == nil {
			t.Error("missing error")
		}
	})
	t.Run("value", func(t *testing.T) {
		c := NewContainer(data.NewDB())
		root := c.NewRoot()
		val := User{"Vasya Pupkin", 34, "businessman"}
		if err := root.Set(val); err != nil {
			t.Error("unexpected error: ", err)
		}
		if stat := c.db.Stat(); stat.Total != 2 {
			t.Error("wrong object count: want 2, got ", stat.Total)
		}
		if _, ok := c.db.Get(getHash(getSchema(User{}))); !ok {
			t.Error("wrong or misisng schema in db")
		}
		if _, ok := c.db.Get(getHash(val)); !ok {
			t.Error("wrong or misisng value in db")
		}
	})
}

func TestRoot_Touch(t *testing.T) {
	root := new(Root)
	tb := time.Now().UnixNano()
	root.Touch()
	te := time.Now().UnixNano()
	if root.Time < tb || root.Time > te {
		t.Error("invalid timestamp")
	}
	if root.Seq != 1 {
		t.Error("invalid seq: ", root.Seq)
	}
}

func TestRoot_initialize(t *testing.T) {
	c := NewContainer(data.NewDB())
	root := new(Root)
	root.initialize(c)
	if root.container != c {
		t.Error("initialization failed")
	}
}

func TestRoot_Encode(t *testing.T) {
	c := NewContainer(data.NewDB())
	root := c.NewRoot()
	root.Set(Man{"Bob", 182, 82})
	data := root.Encode()
	if data == nil || len(data) == 0 {
		t.Error("empty encoding")
		return
	}
	// decoding
	r2, err := decodeRoot(data)
	if err != nil {
		t.Error("unexpectd error:", err)
	}
	// compare
	if root.Schema != r2.Schema {
		t.Error("different shemas")
	}
	if root.Root != r2.Root {
		t.Error("different roots")
	}
	if root.Time != r2.Time {
		t.Error("different time")
	}
	if root.Seq != r2.Seq {
		t.Error("different seq")
	}
}

func Test_decodeRoot(t *testing.T) {
	// TODO or not to do
	// see TestRoot_Encode
}
