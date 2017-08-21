package idxdb

import (
	"bytes"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

const testFileName string = "test.db.goignore"

var errTestError = errors.New("test error")

func testNewDriveIdxDB(t *testing.T) (idx IdxDB) {
	var err error
	if idx, err = NewDriveIdxDB(testFileName); err != nil {
		t.Fatal(err)
	}
	return
}

func testKeyObject(s string) (key cipher.SHA256, o *Object) {
	key = cipher.SumSHA256([]byte(s))
	o = new(Object)
	o.Vol = Volume(len(s))
	o.Subtree.Amount = 1
	o.Subtree.Volume = 100
	o.AccessTime = 776
	o.CreateTime = 778
	o.RefsCount = 0
	return
}

func testRoot(s string) (r *Root) {

	_, sk := cipher.GenerateDeterministicKeyPair([]byte("test"))

	r = new(Root)
	r.Vol = Volume(len(s))
	r.Subtree.Amount = 1000
	r.Subtree.Volume = 8000
	r.AccessTime = 996
	r.CreateTime = 998
	r.RefsCount = 1

	r.Seq = 0
	r.Prev = cipher.SHA256{}

	r.Hash = cipher.SumSHA256([]byte(s))
	r.Sig = cipher.SignHash(r.Hash, sk)

	r.IsFull = true
	return
}

func testIdxDBTx(t *testing.T, idx IdxDB) {

	key, o := testKeyObject("ha-ha")

	t.Run("commit", func(t *testing.T) {

		err := idx.Tx(func(tx Tx) (err error) {
			objs := tx.Objects()
			if err = objs.Set(key, o); err != nil {
				return
			}
			feeds := tx.Feeds()
			// TODO (kostyarin): feeds
			_ = feeds
			return
		})

		if err != nil {
			t.Error(err)
			return
		}

		err = idx.Tx(func(tx Tx) (err error) {
			objs := tx.Objects()
			var x *Object
			if x, err = objs.Get(key); err != nil {
				return
			}
			if x == nil {
				t.Error("misisng saved object")
			}
			feeds := tx.Feeds()
			// TODO (kostyarin): feeds
			_ = feeds
			return
		})

		if err != nil {
			t.Error(err)
		}

	})

	key, o = testKeyObject("ho-ho") // another obejct

	t.Run("rollback", func(t *testing.T) {

		err := idx.Tx(func(tx Tx) (err error) {
			objs := tx.Objects()
			if err = objs.Set(key, o); err != nil {
				return
			}
			feeds := tx.Feeds()
			// TODO (kostyarin): feeds
			_ = feeds
			return errTestError
		})

		if err == nil {
			t.Error("mising error")
			return
		} else if err != errTestError {
			t.Error("unexpected error:", err)
			return
		}

		err = idx.Tx(func(tx Tx) (err error) {
			objs := tx.Objects()
			var x *Object
			if x, err = objs.Get(key); err != nil {
				return
			}
			if x != nil {
				t.Error("has not been rolled back")
			}
			feeds := tx.Feeds()
			// TODO (kostyarin): feeds
			_ = feeds
			return
		})

		if err != nil {
			t.Error(err)
		}

	})

}

func TestIdxDB_Tx(t *testing.T) {
	// Tx(func(Tx) error) error

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testIdxDBTx(t, idx)
	})
}

func testIdxDBClose(t *testing.T, idx IdxDB) {
	if err := idx.Close(); err != nil {
		t.Error(err)
	}
	if err := idx.Close(); err != nil {
		t.Error(err)
	}
}

func TestIdxDB_Close(t *testing.T) {
	// Close() error

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testIdxDBClose(t, idx)
	})

}

func TestRoot_Encode(t *testing.T) {
	// Encode() (p []byte)

	r := testRoot("ha-ha")
	if bytes.Compare(r.Encode(), encoder.Serialize(r)) != 0 {
		t.Error("wrong")
	}
}

func TestRoot_Decode(t *testing.T) {
	// Decode(p []byte) (err error)

	r := testRoot("ha-ha")

	x := new(Root)
	if err := x.Decode(encoder.Serialize(r)); err != nil {
		t.Fatal(err)
	}

	if *x != *r {
		t.Error("wrong")
	}

}

func TestObject_Volume(t *testing.T) {
	// Volume() (vol Volume)

	_, o := testKeyObject("obj")
	if o.Volume() != o.Vol+o.Subtree.Volume {
		t.Error("wrong")
	}
}

func TestObject_Amount(t *testing.T) {
	// Amount() (amnt Amount)

	_, o := testKeyObject("obj")
	if o.Amount() != 1+o.Subtree.Amount {
		t.Error("wrong")
	}
}

func TestObject_UpdateAccessTime(t *testing.T) {
	// UpdateAccessTime()

	_, o := testKeyObject("obj")
	start := time.Now().UnixNano()
	o.UpdateAccessTime()
	end := time.Now().UnixNano()
	if o.AccessTime < start {
		t.Error("not updated")
	} else if o.AccessTime > end {
		t.Error("wrong time", o.AccessTime, end)
	}
}

func TestObject_Encode(t *testing.T) {
	// Encode() (p []byte)

	_, o := testKeyObject("obj")

	if bytes.Compare(o.Encode(), encoder.Serialize(o)) != 0 {
		t.Error("wrong")
	}
}

func TestObject_EncodeTo(t *testing.T) {
	// EncodeTo(p []byte) (err error)

	_, o := testKeyObject("obj")

	p := make([]byte, 32)
	if err := o.EncodeTo(p); err != nil {
		t.Error(err)
	} else if bytes.Compare(p, encoder.Serialize(o)) != 0 {
		t.Error("wrong")
	}

	p = make([]byte, 64)
	if err := o.EncodeTo(p); err != nil {
		t.Error(err)
	} else if bytes.Compare(p[:32], encoder.Serialize(o)) != 0 {
		t.Error("wrong")
	}

	if err := o.EncodeTo(p[:31]); err == nil {
		t.Error("misisng error")
	}
}

func TestObject_Decode(t *testing.T) {
	// Decode(p []byte) (err error)

	_, o := testKeyObject("obj")
	p := encoder.Serialize(o)

	x := new(Object)
	if err := x.Decode(p); err != nil {
		t.Error(err)
	}
	if *x != *o {
		t.Error("wrong")
	}

	if err := x.Decode(p[:31]); err == nil {
		t.Error("misisng error")
	}

}

func TestVolume_String(t *testing.T) {
	// String() (s string)

	type vs struct {
		vol Volume
		s   string
	}

	for i, vs := range []vs{
		{0, "0B"},
		{1023, "1023B"},
		{1024, "1kB"},
		{1030, "1.01kB"},
		{1224, "1.2kB"},
		{1424, "1.39kB"},
		{10241024, "9.77MB"},
	} {
		if vs.vol.String() != vs.s {
			t.Errorf("wrong %d: %d - %s", i, vs.vol, vs.vol.String())
		} else {
			t.Logf("      %d: %d - %s", i, vs.vol, vs.vol.String())
		}
	}
}
