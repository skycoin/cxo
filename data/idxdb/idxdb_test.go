package idxdb

import (
	"errors"
	"os"
	"testing"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/data/tests"
)

const testFileName string = "test.db.goignore"

var errTestError = errors.New("test error")

func testNewDriveIdxDB(t *testing.T) (idx data.IdxDB) {
	var err error
	if idx, err = NewDriveIdxDB(testFileName); err != nil {
		t.Fatal(err)
	}
	return
}

func TestIdxDB_Tx(t *testing.T) {
	// Tx(func(Tx) error) error

	// TODO (kostyarin):
}

func TestIdxDB_Close(t *testing.T) {
	// Close() error

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		tests.IdxDBClose(t, idx)
	})

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
