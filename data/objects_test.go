package data

import (
	"bytes"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

//
// ViewObjects
//

func testViewObjectsGet(t *testing.T, db DB) {

	value := []byte("any")
	key := cipher.SumSHA256(value)

	t.Run("not exist", func(t *testing.T) {
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

	t.Run("not exist", func(t *testing.T) {
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

	t.Run("not exist", func(t *testing.T) {
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

func testViewObjectsAscend(t *testing.T, db DB) {

	t.Run("empty", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			objs := tx.Objects()

			var called int

			err := objs.Ascend(func(cipher.SHA256, []byte) (_ error) {
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

			err := objs.Ascend(func(key cipher.SHA256,
				value []byte) (_ error) {

				if called > len(to) {
					t.Error("called too many times")
					return ErrStopIteration
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

	t.Run("stop iteration", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			objs := tx.Objects()

			var called int

			err := objs.Ascend(func(key cipher.SHA256,
				value []byte) (_ error) {

				called++
				return ErrStopIteration
			})
			if err != nil {
				t.Error(err)
			}
			if called != 1 {
				t.Error("ErrStopIteration doesn't stop the iteration")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestViewObjects_Ascend(t *testing.T) {
	// Ascend(func(key cipher.SHA256, value []byte) error) (err error)

	t.Run("memory", func(t *testing.T) {
		testViewObjectsAscend(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewObjectsAscend(t, db)
	})

}

//
// UpdateObjects
//

// inherited from ViewObjects

func testUpdateObjectsGet(t *testing.T, db DB) {

	value := []byte("any")
	key := cipher.SumSHA256(value)

	t.Run("not exist", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
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
		err := db.Update(func(tx Tu) (_ error) {
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

func TestUpdateObjects_Get(t *testing.T) {
	// Get(key cipher.SHA256) (value []byte)

	t.Run("memory", func(t *testing.T) {
		testUpdateObjectsGet(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateObjectsGet(t, db)
	})

}

func testUpdateObjectsGetCopy(t *testing.T, db DB) {

	// TODO (kostyarin): how to be sure that returned slice
	//                   is long-lived and will not be modified
	//                   after transaction? fuzz? I hate fuzz tests!

	value := []byte("any")
	key := cipher.SumSHA256(value)

	t.Run("not exist", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
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
		err := db.Update(func(tx Tu) (_ error) {
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

func TestUpdateObjects_GetCopy(t *testing.T) {
	// GetCopy(key cipher.SHA256) (value []byte)

	t.Run("memory", func(t *testing.T) {
		testUpdateObjectsGetCopy(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateObjectsGetCopy(t, db)
	})

}

func testUpdateObjectsIsExists(t *testing.T, db DB) {

	value := []byte("any")
	key := cipher.SumSHA256(value)

	t.Run("not exist", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
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
		err := db.Update(func(tx Tu) (_ error) {
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

func TestUpdateObjects_IsExist(t *testing.T) {
	// IsExist(key cipher.SHA256) (ok bool)

	t.Run("memory", func(t *testing.T) {
		testUpdateObjectsIsExists(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateObjectsIsExists(t, db)
	})

}

func testUpdateObjectsAscend(t *testing.T, db DB) {

	t.Run("empty", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			objs := tx.Objects()

			var called int

			err := objs.Ascend(func(cipher.SHA256, []byte) (_ error) {
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
		err := db.Update(func(tx Tu) (_ error) {
			objs := tx.Objects()

			var called int

			err := objs.Ascend(func(key cipher.SHA256,
				value []byte) (_ error) {

				if called > len(to) {
					t.Error("called too many times")
					return ErrStopIteration
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

	t.Run("stop iteration", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			objs := tx.Objects()

			var called int

			err := objs.Ascend(func(key cipher.SHA256,
				value []byte) (_ error) {

				called++
				return ErrStopIteration
			})
			if err != nil {
				t.Error(err)
			}
			if called != 1 {
				t.Error("ErrStopIteration doesn't stop the iteration")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestUpdateObjects_Ascend(t *testing.T) {
	// Ascend(func(key cipher.SHA256, value []byte) error) (err error)

	t.Run("memory", func(t *testing.T) {
		testUpdateObjectsAscend(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateObjectsAscend(t, db)
	})

}

// UpdateObjects

func testUpdateObjectsDel(t *testing.T, db DB) {

	value := []byte("ha-ha")
	key := cipher.SumSHA256(value)

	t.Run("not exist", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			if err := tx.Objects().Del(key); err != nil {
				t.Error(err)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	// fill
	err := db.Update(func(tx Tu) error {
		return tx.Objects().Set(key, value)
	})
	if err != nil {
		t.Error(err)
	}

	t.Run("delete", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			if err := tx.Objects().Del(key); err != nil {
				t.Error(err)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestUpdateObjects_Del(t *testing.T) {
	// Del(key cipher.SHA256) (err error)

	t.Run("memory", func(t *testing.T) {
		testUpdateObjectsDel(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateObjectsDel(t, db)
	})

}

func testUpdateObjectsSet(t *testing.T, db DB) {

	value := []byte("yo-h-ho")
	key := cipher.SumSHA256(value)

	t.Run("set", func(t *testing.T) {

		err := db.Update(func(tx Tu) (_ error) {
			objs := tx.Objects()

			if err := objs.Set(key, value); err != nil {
				t.Error(err)
				return
			}

			if bytes.Compare(objs.Get(key), value) != 0 {
				t.Error("wrong of missing value")
			}

			return

		})

		if err != nil {
			t.Error(err)
		}

	})

	t.Run("overwrite", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			objs := tx.Objects()

			replace := []byte("zorro!")

			if err := objs.Set(key, replace); err != nil {
				t.Error(err)
				return
			}

			if bytes.Compare(objs.Get(key), replace) != 0 {
				t.Error("wrong of missing value")
			}

			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestUpdateObjects_Set(t *testing.T) {
	// Set(key cipher.SHA256, value []byte) (err error)

	t.Run("memory", func(t *testing.T) {
		testUpdateObjectsSet(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateObjectsSet(t, db)
	})

}

func testUpdateObjectsAdd(t *testing.T, db DB) {

	value := []byte("value")
	key := cipher.SumSHA256(value)

	t.Run("name", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			objs := tx.Objects()

			if rk, err := objs.Add(value); err != nil {
				t.Error(err)
				return
			} else if bytes.Compare(rk[:], key[:]) != 0 {
				t.Error("wrong key returned")
				return
			}

			if bytes.Compare(objs.Get(key), value) != 0 {
				t.Error("wrong of missing value")
			}

			// adding many times should not produce an error
			if _, err := objs.Add(value); err != nil {
				t.Error(err)
			}

			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestUpdateObjects_Add(t *testing.T) {
	// Add(value []byte) (key cipher.SHA256, err error)

	t.Run("memory", func(t *testing.T) {
		testUpdateObjectsAdd(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateObjectsAdd(t, db)
	})

}

func testUpdateObjectsSetMap(t *testing.T, db DB) {

	mp := make(map[cipher.SHA256][]byte)

	for _, s := range []string{
		"one",
		"two",
		"three",
		"c4",
	} {
		value := []byte(s)
		key := cipher.SumSHA256(value)
		mp[key] = value
	}

	err := db.Update(func(tx Tu) (_ error) {
		objs := tx.Objects()

		if err := objs.SetMap(mp); err != nil {
			t.Error(err)
			return
		}

		for k, v := range mp {
			if bytes.Compare(objs.Get(k), v) != 0 {
				t.Error("missing or wrong value")
			}
		}

		return
	})
	if err != nil {
		t.Error(err)
	}

}

func TestUpdateObjects_SetMap(t *testing.T) {
	// SetMap(map[cipher.SHA256][]byte) (err error)

	t.Run("memory", func(t *testing.T) {
		testUpdateObjectsSetMap(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateObjectsSetMap(t, db)
	})

}

func testUpdateObjectsAscendDel(t *testing.T, db DB) {

	t.Run("empty", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			objs := tx.Objects()

			var called int

			err := objs.AscendDel(func(cipher.SHA256, []byte) (_ bool,
				_ error) {

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
		err := db.Update(func(tx Tu) (_ error) {
			objs := tx.Objects()

			var called int

			err := objs.AscendDel(func(key cipher.SHA256,
				value []byte) (del bool, _ error) {

				if called > len(to) {
					t.Error("called too many times")
					return false, ErrStopIteration
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

				if string(value) == "two" {
					del = true // delete
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

			// check deleting
			for _, o := range to {
				if string(o.value) == "two" {
					// should be deleted
					if objs.Get(o.key) != nil {
						t.Error("undeleted")
					}
					continue
				}
				if objs.Get(o.key) == nil {
					t.Error("deleted")
				}
			}

			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("stop iteration", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			objs := tx.Objects()

			var called int

			err := objs.AscendDel(func(cipher.SHA256, []byte) (bool, error) {
				called++
				return false, ErrStopIteration
			})
			if err != nil {
				t.Error(err)
			}
			if called != 1 {
				t.Error("ErrStopIteration doesn't stop the iteration")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestUpdateObjects_AscendDel(t *testing.T) {
	// AscendDel(
	//     func(key cipher.SHA256, value []byte) (del bool, err error)) error

	t.Run("memory", func(t *testing.T) {
		testUpdateObjectsAscendDel(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateObjectsAscendDel(t, db)
	})

}
