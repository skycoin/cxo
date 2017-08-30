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

func testRoot(s string) (r *Root) {

	_, sk := cipher.GenerateDeterministicKeyPair([]byte("test"))

	r = new(Root)
	r.AccessTime = 996
	r.CreateTime = 998

	r.Seq = 0
	r.Prev = cipher.SHA256{}

	r.Hash = cipher.SumSHA256([]byte(s))
	r.Sig = cipher.SignHash(r.Hash, sk)
	return
}

func TestIdxDB_Tx(t *testing.T) {
	// Tx(func(Tx) error) error

	// TODO (kostyarin):
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

	p := encoder.Serialize(r)

	x := new(Root)
	if err := x.Decode(p); err != nil {
		t.Fatal(err)
	}

	if *x != *r {
		t.Error("wrong")
	}

	if err := x.Decode(p[:12]); err == nil {
		t.Error("misisng error")
	}

}

func TestRoot_UpdateAccessTime(t *testing.T) {
	// UpdateAccessTime()

	r := testRoot("r1")
	start := time.Now().UnixNano()
	r.UpdateAccessTime()
	end := time.Now().UnixNano()
	if r.AccessTime < start {
		t.Error("not updated")
	} else if r.AccessTime > end {
		t.Error("wrong time", r.AccessTime, end)
	}
}

func TestRoot_Validate(t *testing.T) {

	r := testRoot("seed")

	if err := r.Validate(); err != nil {
		t.Error(err)
	}

	// empty hash
	var hash cipher.SHA256
	hash, r.Hash = r.Hash, cipher.SHA256{}
	if err := r.Validate(); err == nil {
		t.Error("missing error")
	}
	r.Hash = hash

	// unexpected prev
	r.Prev = cipher.SumSHA256([]byte("random"))

	if err := r.Validate(); err == nil {
		t.Error("missing error")
	}

	// misisng prev
	r.Seq, r.Prev = 1, cipher.SHA256{}
	if err := r.Validate(); err == nil {
		t.Error("missing error")
	}

	r.Seq = 0
	r.Sig = (cipher.Sig{})

	if err := r.Validate(); err == nil {
		t.Error("missing error")
	}

}

/*
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
*/
