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

func TestData_Del(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// Del
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
		//
	})
}

func TestData_Get(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// Get
		//
		key := db.Add([]byte("hey ho"))
		if got, ok := db.Get(key); !ok {
			t.Error("not added")
		} else if string(got) != "hey ho" {
			t.Error("wrong value returned", string(got))
		}
		if _, ok := db.Get(cipher.SumSHA256([]byte("ho hey"))); ok {
			t.Error("got unexisting value")
		}
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
		key := db.Add([]byte("hey ho"))
		if got, ok := db.Get(key); !ok {
			t.Error("not added")
		} else if string(got) != "hey ho" {
			t.Error("wrong value returned", string(got))
		}
		if _, ok := db.Get(cipher.SumSHA256([]byte("ho hey"))); ok {
			t.Error("got unexisting value")
		}
		//
	})
}

func TestData_Set(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// Set
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
		// Set
		//
		_ = db
		//
	})
}

func TestData_Add(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// Add
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
		// Add
		//
		_ = db
		//
	})
}

func TestData_IsExist(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// IsExist
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
		// IsExist
		//
		_ = db
		//
	})
}

func TestData_Range(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// Range
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
		// Range
		//
		_ = db
		//
	})
}

func TestData_Feeds(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// Feeds
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
		// Feeds
		//
		_ = db
		//
	})
}

func TestData_DelFeed(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		db := NewMemoryDB()
		// DelFeed
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
		// DelFeed
		//
		_ = db
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
