package data

import (
	"bytes"
	"io/ioutil"
	"os"
	"sort"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	// "github.com/skycoin/skycoin/src/cipher/encoder"
)

//
// helper functions
//

func shouldNotPanic(t *testing.T) {
	if err := recover(); err != nil {
		t.Error("unexpected panic:", err)
	}
}

func testPath(t *testing.T) string {
	fl, err := ioutil.TempFile("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer fl.Close()
	return fl.Name()
}

func testDriveDB(t *testing.T) (db DB, cleanUp func()) {
	dbFile := testPath(t)
	db, err := NewDriveDB(dbFile)
	if err != nil {
		os.Remove(dbFile)
		t.Fatal(err)
	}
	cleanUp = func() {
		db.Close()
		os.Remove(dbFile)
	}
	return
}

// returns RootPack that contains dummy Root field,
// the field can't be used to encode/decode
func getRootPack(seq uint64, content string) (rp RootPack) {
	rp.Seq = seq
	if seq != 0 {
		rp.Prev = cipher.SumSHA256([]byte("any"))
	}
	rp.Root = []byte(content)
	rp.Hash = cipher.SumSHA256(rp.Root)
	return
}

type testObjectKeyValue struct {
	key   cipher.SHA256
	value []byte
}

type testObjectKeyValues []testObjectKeyValue

func (t testObjectKeyValues) Len() int {
	return len(t)
}

func (t testObjectKeyValues) Less(i, j int) bool {
	return bytes.Compare(t[i].key[:], t[j].key[:]) < 0
}

func (t testObjectKeyValues) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func testSortedObjects(input ...string) (to testObjectKeyValues) {
	to = make(testObjectKeyValues, 0, len(input))
	for _, s := range input {
		to = append(to, testObjectKeyValue{
			key:   cipher.SumSHA256([]byte(s)),
			value: []byte(s),
		})
	}
	sort.Sort(to)
	return
}

//
// Tests
//

//
// DB
//

func testDBView(t *testing.T, db DB) {
	t.Skip("(TODO) not implemenmted yet")
}

func TestDB_View(t *testing.T) {
	// View(func(t Tv) error) (err error)

	t.Run("memory", func(t *testing.T) {
		testDBView(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testDBView(t, db)
	})

}

func testDBUpdate(t *testing.T, db DB) {
	t.Skip("(TODO) not implemented yet")
}

func TestDB_Update(t *testing.T) {
	// Update(func(t Tu) error) (err error)

	t.Run("memory", func(t *testing.T) {
		testDBUpdate(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testDBUpdate(t, db)
	})

}

func testDBStat(t *testing.T, db DB) {
	t.Skip("(TODO) not implemented yet")
}

func TestDB_Stat(t *testing.T) {
	// Stat() (s Stat)

	t.Run("memory", func(t *testing.T) {
		testDBStat(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testDBStat(t, db)
	})

}

func testDBClose(t *testing.T, db DB) {
	if err := db.Close(); err != nil {
		t.Error("closing error:", err)
	}
	// Close can be called many times
	defer shouldNotPanic(t)
	db.Close()
}

func TestDB_Close(t *testing.T) {
	// Close() (err error)

	t.Run("memory", func(t *testing.T) {
		testDBClose(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testDBClose(t, db)
	})

}

//
// Tv
//

func testTvObjects(t *testing.T, db DB) {
	err := db.View(func(tx Tv) (_ error) {
		if tx.Objects() == nil {
			t.Error("Tv.Objects returns nil")
		}
		return
	})
	if err != nil {
		t.Error(err)
	}
	return
}

func TestTv_Objects(t *testing.T) {
	// Objects() ViewObjects

	t.Run("memory", func(t *testing.T) {
		testTvObjects(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testTvObjects(t, db)
	})

}

func testTvFeeds(t *testing.T, db DB) {
	err := db.View(func(tx Tv) (_ error) {
		if tx.Feeds() == nil {
			t.Error("Tv.Feeds returns nil")
		}
		return
	})
	if err != nil {
		t.Error(err)
	}
}

func TestTv_Feeds(t *testing.T) {
	// Feeds() ViewFeeds

	t.Run("memory", func(t *testing.T) {
		testTvFeeds(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testTvFeeds(t, db)
	})

}

//
// Tu
//

func testTuObjects(t *testing.T, db DB) {
	err := db.Update(func(tx Tu) (_ error) {
		if tx.Objects() == nil {
			t.Error("Tu.Objects returns nil")
		}
		return
	})
	if err != nil {
		t.Error(err)
	}
	return
}

func TestTu_Objects(t *testing.T) {
	// Objects() UpdateObjects

	t.Run("memory", func(t *testing.T) {
		testTuObjects(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testTuObjects(t, db)
	})

}

func testTuFeeds(t *testing.T, db DB) {
	err := db.Update(func(tx Tu) (_ error) {
		if tx.Feeds() == nil {
			t.Error("Tu.Feeds returns nil")
		}
		return
	})
	if err != nil {
		t.Error(err)
	}
}

func TestTu_Feeds(t *testing.T) {
	// Feeds() UpdateFeeds

	t.Run("memory", func(t *testing.T) {
		testTuFeeds(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testTuFeeds(t, db)
	})

}

//
// ViewObjects
//

func testViewObjectsGet(t *testing.T, db DB) {

	value := []byte("any")
	key := cipher.SumSHA256(value)

	t.Run("not exists", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			objs := tx.Objects()

			if objs.Get(key) != nil {
				t.Error("got unexisting value")
			}

			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	err := db.Update(func(tx Tu) (_ error) {
		return tx.Objects().Set(key, value)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("exists", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			objs := tx.Objects()

			got := objs.Get(key)

			if got == nil {
				t.Error("missing value")
				return
			}

			if bytes.Compare(got, value) != 0 {
				t.Error("wrong value")
			}

			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestViewObjects_Get(t *testing.T) {
	// Get(key cipher.SHA256) (value []byte)

	t.Run("memory", func(t *testing.T) {
		testViewObjectsGet(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewObjectsGet(t, db)
	})

}

func testViewObjectsGetCopy(t *testing.T, db DB) {

	// TODO (kostyarin): how to be sure that returned slice
	//                   is long-lived and will not be modified
	//                   after transaction? fuzz? I hate fuzz tests!

	value := []byte("any")
	key := cipher.SumSHA256(value)

	t.Run("not exists", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			objs := tx.Objects()

			if objs.GetCopy(key) != nil {
				t.Error("got unexisting value")
			}

			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	err := db.Update(func(tx Tu) (_ error) {
		return tx.Objects().Set(key, value)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("exists", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			objs := tx.Objects()

			got := objs.GetCopy(key)

			if got == nil {
				t.Error("missing value")
				return
			}

			if bytes.Compare(got, value) != 0 {
				t.Error("wrong value")
			}

			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestViewObjects_GetCopy(t *testing.T) {
	// GetCopy(key cipher.SHA256) (value []byte)

	t.Run("memory", func(t *testing.T) {
		testViewObjectsGetCopy(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewObjectsGetCopy(t, db)
	})

}

func testViewObjectsIsExists(t *testing.T, db DB) {

	value := []byte("any")
	key := cipher.SumSHA256(value)

	t.Run("not exists", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			objs := tx.Objects()

			if objs.IsExist(key) == true {
				t.Error("has unexisting value")
			}

			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	err := db.Update(func(tx Tu) (_ error) {
		return tx.Objects().Set(key, value)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("exists", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			objs := tx.Objects()

			if objs.IsExist(key) == false {
				t.Error("hasn't got existing value")
			}

			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestViewObjects_IsExist(t *testing.T) {
	// IsExist(key cipher.SHA256) (ok bool)

	t.Run("memory", func(t *testing.T) {
		testViewObjectsIsExists(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewObjectsIsExists(t, db)
	})

}

func testViewObjectsRange(t *testing.T, db DB) {

	t.Run("empty", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			objs := tx.Objects()

			var called int

			err := objs.Range(func(cipher.SHA256, []byte) (_ error) {
				called++
				return
			})
			if err != nil {
				t.Error(err)
			}
			if called != 0 {
				t.Error("ranges over empty objects bucket")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	// fill

	to := testSortedObjects("one", "two", "three")

	err := db.Update(func(tx Tu) (_ error) {
		objs := tx.Objects()

		for _, o := range to {
			if err := objs.Set(o.key, o.value); err != nil {
				return err // rollback and bubble the err up
			}
		}

		return
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("full", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			objs := tx.Objects()

			var called int

			err := objs.Range(func(key cipher.SHA256, value []byte) (_ error) {
				if called > len(to) {
					t.Error("called too many times")
					return ErrStopRange
				}

				tObj := to[called]

				if bytes.Compare(tObj.key[:], key[:]) != 0 {
					t.Error("wrong order",
						shortHex(tObj.key.Hex()),
						shortHex(key.Hex()))
				}
				if bytes.Compare(tObj.value, value) != 0 {
					t.Error("wrong or missing value")
				}

				called++
				return
			})
			if err != nil {
				t.Error(err)
			}
			if called != len(to) {
				t.Error("called wrong times")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("stop range", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			objs := tx.Objects()

			var called int

			err := objs.Range(func(key cipher.SHA256, value []byte) (_ error) {
				called++
				return ErrStopRange
			})
			if err != nil {
				t.Error(err)
			}
			if called != 1 {
				t.Error("ErrStopRange doesn't stop the Range")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestViewObjects_Range(t *testing.T) {
	// Range(func(key cipher.SHA256, value []byte) error) (err error)

	t.Run("memory", func(t *testing.T) {
		testViewObjectsRange(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewObjectsRange(t, db)
	})

}

//
// UpdateObjects
//

// inherited from ViewObjects

func TestUpdateObjects_Get(t *testing.T) {
	// Get(key cipher.SHA256) (value []byte)

	//

}

func TestUpdateObjects_GetCopy(t *testing.T) {
	// GetCopy(key cipher.SHA256) (value []byte)

	//

}

func TestUpdateObjects_IsExist(t *testing.T) {
	// IsExist(key cipher.SHA256) (ok bool)

	//

}

func TestUpdateObjects_Range(t *testing.T) {
	// Range(func(key cipher.SHA256, value []byte) error) (err error)

	//

}

// UpdateObjects

func TestUpdateObjects_Del(t *testing.T) {
	// Del(key cipher.SHA256) (err error)

	//

}

func TestUpdateObjects_Set(t *testing.T) {
	// Set(key cipher.SHA256, value []byte) (err error)

	//

}

func TestUpdateObjects_Add(t *testing.T) {
	// Add(value []byte) (key cipher.SHA256, err error)

	//

}

func TestUpdateObjects_SetMap(t *testing.T) {
	// SetMap(map[cipher.SHA256][]byte) (err error)

	//

}

func TestUpdateObjects_RangeDel(t *testing.T) {
	// RangeDel(
	//     func(key cipher.SHA256, value []byte) (del bool, err error)) error

	//

}

//
// ViewFeeds
//

func TestViewFeeds_IsExist(t *testing.T) {
	// IsExist(pk cipher.PubKey) (ok bool)

	//

}

func TestViewFeeds_List(t *testing.T) {
	// List() (list []cipher.PubKey)

	//

}

func TestViewFeeds_Range(t *testing.T) {
	// Range(func(pk cipher.PubKey) error) (err error)

	//

}

func TestViewFeeds_Roots(t *testing.T) {
	// Roots(pk cipher.PubKey) ViewRoots

	//

}

//
// UpdateFeeds
//

func TestUpdateFeeds_IsExist(t *testing.T) {
	// IsExist(pk cipher.PubKey) (ok bool)

	//

}

func TestUpdateFeeds_List(t *testing.T) {
	// List() (list []cipher.PubKey)

	//

}

func TestUpdateFeeds_Range(t *testing.T) {
	// Range(func(pk cipher.PubKey) error) (err error)

	//

}

func TestUpdateFeeds_Add(t *testing.T) {
	// Add(pk cipher.PubKey) (err error)

	//

}

func TestUpdateFeeds_Del(t *testing.T) {
	// Del(pk cipher.PubKey) (err error)

	//

}

func TestUpdateFeeds_RangeDel(t *testing.T) {
	// RangeDel(func(pk cipher.PubKey) (del bool, err error)) error

	//

}

func TestUpdateFeeds_Roots(t *testing.T) {
	// Roots(pk cipher.PubKey) UpdateRoots

	//

}

//
// ViewRoots
//

func TestViewRoots_Feed(t *testing.T) {
	// Feed() cipher.PubKey

	//

}

func TestViewRoots_Last(t *testing.T) {
	// Last() (rp *RootPack)

	//

}

func TestViewRoots_Get(t *testing.T) {
	// Get(seq uint64) (rp *RootPack)

	//

}

func TestViewRoots_Range(t *testing.T) {
	// Range(func(rp *RootPack) (err error)) error

	//

}

func TestViewRoots_Reverse(t *testing.T) {
	// Reverse(fn func(rp *RootPack) (err error)) error

	//

}

//
// UpdateRoots
//

// inherited from ViewRoots

func TestUpdateRoots_Feed(t *testing.T) {
	// Feed() cipher.PubKey

	//

}

func TestUpdateRoots_Last(t *testing.T) {
	// Last() (rp *RootPack)

	//

}

func TestUpdateRoots_Get(t *testing.T) {
	// Get(seq uint64) (rp *RootPack)

	//

}

func TestUpdateRoots_Range(t *testing.T) {
	// Range(func(rp *RootPack) (err error)) error

	//

}

func TestUpdateRoots_Reverse(t *testing.T) {
	// Reverse(fn func(rp *RootPack) (err error)) error

	//

}

// UpdateRoots

func TestUpdateRoots_Add(t *testing.T) {
	// Add(rp *RootPack) (err error)

	//

}

func TestUpdateRoots_Del(t *testing.T) {
	// Del(seq uint64) (err error)

	//

}

func TestUpdateRoots_RangeDel(t *testing.T) {
	// RangeDel(fn func(rp *RootPack) (del bool, err error)) error

	//

}

func TestUpdateRoots_DelBefore(t *testing.T) {
	// DelBefore(seq uint64) (err error)

	//

}
