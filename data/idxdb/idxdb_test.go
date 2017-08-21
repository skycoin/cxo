package idxdb

import (
	"errors"
	"os"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
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
	o.AccessTime = 786
	o.CreateTime = 987
	o.RefsCount = 0
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
