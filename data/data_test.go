package data

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	// "github.com/skycoin/skycoin/src/cipher/encoder"
)

func testPath(t *testing.T) string {
	fl, err := ioutil.TempFile("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer fl.Close()
	return fl.Name()
}

//
// objects
//

func testDataDel(t *testing.T, db DB) {
	key := db.Add([]byte("hey ho"))
	if got, ok := db.Get(key); !ok {
		t.Error("not added")
	} else if string(got) != "hey ho" {
		t.Error("wrong value returned", string(got))
	} else {
		db.Del(key)
		if _, ok := db.Get(key); ok {
			t.Error("not deleted")
		}
	}
}

func TestData_Del(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// Del
		testDataDel(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// Del
		//
		testDataDel(t, db)
		//
	})
}

func testDataGet(t *testing.T, db DB) {
	key := db.Add([]byte("hey ho"))
	if got, ok := db.Get(key); !ok {
		t.Error("not added")
	} else if string(got) != "hey ho" {
		t.Error("wrong value returned", string(got))
	}
	if _, ok := db.Get(cipher.SumSHA256([]byte("ho hey"))); ok {
		t.Error("got unexisting value")
	}
}

func TestData_Get(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// Get
		//
		testDataGet(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// Get
		//
		testDataGet(t, db)
		//
	})
}

func testDataSet(t *testing.T, db DB) {
	val := []byte("hey ho")
	key := cipher.SumSHA256(val)
	db.Set(key, val)
	if got, ok := db.Get(key); !ok {
		t.Error("not added")
	} else if string(got) != string(val) {
		t.Error("wrong value returned", string(got))
	}
}

func TestData_Set(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// Set
		//
		testDataSet(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// Set
		//
		testDataSet(t, db)
		//
	})
}

func testDataAdd(t *testing.T, db DB) {
	val := []byte("hey ho")
	key := db.Add(val)
	if key != cipher.SumSHA256(val) {
		t.Error("wrong key calculated")
	}
	if got, ok := db.Get(key); !ok {
		t.Error("not added")
	} else if string(got) != string(val) {
		t.Error("wrong value returned", string(got))
	}
}

func TestData_Add(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// Add
		//
		testDataAdd(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// Add
		//
		testDataAdd(t, db)
		//
	})
}

func testDataIsExists(t *testing.T, db DB) {
	val := []byte("hey ho")
	key := db.Add(val)
	if ok := db.IsExist(key); !ok {
		t.Error("not added")
	}
	if ok := db.IsExist(cipher.SumSHA256([]byte("ho hey"))); ok {
		t.Error("have unexisting value")
	}
}

func TestData_IsExist(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// IsExist
		//
		testDataIsExists(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// IsExist
		//
		testDataIsExists(t, db)
		//
	})
}

func testDataRange(t *testing.T, db DB) {
	var vals [][]byte = [][]byte{
		[]byte("one"),
		[]byte("two"),
		[]byte("othree"),
		[]byte("four"),
	}
	for _, val := range vals {
		db.Add(val)
	}
	var collect map[cipher.SHA256][]byte = make(map[cipher.SHA256][]byte)
	db.Range(func(key cipher.SHA256, value []byte) (stop bool) {
		collect[key] = value
		return
	})
	if len(collect) != len(vals) {
		t.Error("wong amount of values given")
		return
	}
	for _, v := range vals {
		if string(collect[cipher.SumSHA256(v)]) != string(v) {
			t.Error("wrong value")
		}
	}
	var i int
	db.Range(func(key cipher.SHA256, value []byte) (stop bool) {
		i++
		return true
	})
	if i != 1 {
		t.Error("can't stop")
	}
}

func TestData_Range(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// Range
		//
		testDataRange(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// Range
		//
		testDataRange(t, db)
		//
	})
}

//
// feeds
//

func testDataFeeds(t *testing.T, db DB) {
	// empty
	if len(db.Feeds()) != 0 {
		t.Error("wrong feeds length")
	}
	// prepare
	var rp RootPack
	rp.Hash = cipher.SumSHA256(rp.Root)
	// one
	pk1, _ := cipher.GenerateKeyPair()
	if err := db.AddRoot(pk1, &rp); err != nil {
		t.Error(err)
		return
	}
	if fs := db.Feeds(); len(fs) != 1 {
		t.Error("wrong feeds length")
	} else if fs[0] != pk1 {
		t.Error("wrong feed content")
	}
	// two
	pk2, _ := cipher.GenerateKeyPair()
	if err := db.AddRoot(pk2, &rp); err != nil {
		t.Error(err)
		return
	}
	if fs := db.Feeds(); len(fs) != 2 {
		t.Error("wrong feeds length")
	} else {
		pks := map[cipher.PubKey]struct{}{
			pk1: struct{}{},
			pk2: struct{}{},
		}
		for _, pk := range fs {
			if _, ok := pks[pk]; !ok {
				t.Error("missing feed")
			}
		}
	}
}

func TestData_Feeds(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// Feeds
		//
		testDataFeeds(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// Feeds
		//
		testDataFeeds(t, db)
		//
	})
}

func testDataDelFeed(t *testing.T, db DB) {
	// prepare
	var rp RootPack
	rp.Hash = cipher.SumSHA256(rp.Root) // nil
	pk, _ := cipher.GenerateKeyPair()
	if err := db.AddRoot(pk, &rp); err != nil {
		t.Error(err) // fatal
		return
	}
	// go
	db.DelFeed(pk)
	if len(db.Feeds()) != 0 {
		t.Error("feed is not deleted")
	}
	if _, ok := db.GetRoot(rp.Hash); ok {
		t.Error("root is not deleted")
	}
}

func TestData_DelFeed(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// DelFeed
		//
		testDataDelFeed(t, db)
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// DelFeed
		//
		testDataDelFeed(t, db)
		//
	})
}

func TestData_AddRoot(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// AddRoot
		//
		_ = db
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// AddRoot
		//
		_ = db
		//
	})
}

func TestData_LastRoot(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// LastRoot
		//
		_ = db
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// LastRoot
		//
		_ = db
		//
	})
}

func TestData_RangeFeed(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// RangeFeed
		//
		_ = db
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// RangeFeed
		//
		_ = db
		//
	})
}

func TestData_RangeFeedReverse(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// RangeFeedReverse
		//
		_ = db
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// RangeFeedReverse
		//
		_ = db
		//
	})
}

//
// roots
//

func TestData_GetRoot(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// GetRoot
		//
		_ = db
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// GetRoot
		//
		_ = db
		//
	})
}

func TestData_DelRootsBefore(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// DelRootsBefore
		//
		_ = db
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// DelRootsBefore
		//
		_ = db
		//
	})
}

//
// stat and close
//

func TestData_Stat(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// Stat
		//
		_ = db
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// Stat
		//
		_ = db
		//
	})
}

func TestData_Close(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// Close
		//
		_ = db
		//
	})
	t.Run("drive", func(t *testing.T) {
		dbFile := testPath(t)
		defer os.Remove(dbFile)
		db, err := NewDriveDB(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		// Close
		//
		_ = db
		//
	})
}
