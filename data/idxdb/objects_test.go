package idxdb

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func testObjectsInc(t *testing.T, idx IdxDB) {

	key, o := testKeyObject("ha")

	t.Run("not exist", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if rc, err := objs.Inc(key); err != nil {
				t.Error(err)
			} else if rc != 0 {
				t.Error("wrong rc", rc)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	err := idx.Tx(func(tx Tx) error { return tx.Objects().Set(key, o) })
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("exist", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			objs := tx.Objects()
			if rc, err := objs.Inc(key); err != nil {
				t.Error(err)
			} else if rc != 2 {
				t.Error("wrong rc", rc)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestObjects_Inc(t *testing.T) {
	// Inc(key cipher.SHA256) (rc uint32, err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsInc(t, idx)
	})
}

func testObjectsGet(t *testing.T, idx IdxDB) {
	//
}

func TestObjects_Get(t *testing.T) {
	// Get(key cipher.SHA256) (o *Object, err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsGet(t, idx)
	})
}

func testObjectsMultiGet(t *testing.T, idx IdxDB) {
	//
}

func TestObjects_MultiGet(t *testing.T) {
	// MultiGet(keys []cipher.SHA256) (os []*Object, err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsMultiGet(t, idx)
	})
}

func testObjectsMultiInc(t *testing.T, idx IdxDB) {
	//
}

func TestObjects_MultiInc(t *testing.T) {
	// MultiInc(keys []cipher.SHA256) (err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsMultiInc(t, idx)
	})
}

func testObjectsIterate(t *testing.T, idx IdxDB) {
	//
}

func TestObjects_Iterate(t *testing.T) {
	// Iterate(IterateObjectsFunc) (err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsIterate(t, idx)
	})
}

func testObjectsDec(t *testing.T, idx IdxDB) {
	//
}

func TestObjects_Dec(t *testing.T) {
	// Dec(key cipher.SHA256) (rc uint32, err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsDec(t, idx)
	})
}

func testObjectsSet(t *testing.T, idx IdxDB) {
	//
}

func TestObjects_Set(t *testing.T) {
	// Set(key cipher.SHA256, o *Object) (err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsSet(t, idx)
	})
}

func testObjectsMultiSet(t *testing.T, idx IdxDB) {
	//
}

func TestObjects_MultiSet(t *testing.T) {
	// MultiSet(ko []KeyObject) (err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsMultiSet(t, idx)
	})
}

func testObjectsMulitDec(t *testing.T, idx IdxDB) {
	//
}

func TestObjects_MulitDec(t *testing.T) {
	// MulitDec(keys []cipher.SHA256) (err error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsMulitDec(t, idx)
	})
}

func testObjectsAmount(t *testing.T, idx IdxDB) {
	//
}

func TestObjects_Amount(t *testing.T) {
	// Amount() Amount

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsAmount(t, idx)
	})
}

func testObjectsVolume(t *testing.T, idx IdxDB) {
	//
}

func TestObjects_Volume(t *testing.T) {
	// Volume() Volume

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testObjectsVolume(t, idx)
	})
}
