package data

import (
	"bytes"
	"testing"
)

//
// ViewMisc
//

func testViewMiscGet(t *testing.T, db DB) {

	value, key := []byte("yo-ho-ho"), []byte("and bottle of rum")

	t.Run("not exist", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			misc := tx.Misc()

			if misc.Get(key) != nil {
				t.Error("got unexisting value")
			}

			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	err := db.Update(func(tx Tu) (_ error) {
		return tx.Misc().Set(key, value)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("exists", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			misc := tx.Misc()

			got := misc.Get(key)

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

func TestViewMisc_Get(t *testing.T) {
	// Get(key cipher.SHA256) (value []byte)

	t.Run("memory", func(t *testing.T) {
		testViewMiscGet(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewMiscGet(t, db)
	})

}

func testViewMiscGetCopy(t *testing.T, db DB) {

	// TODO (kostyarin): how to be sure that returned slice
	//                   is long-lived and will not be modified
	//                   after transaction? fuzz? I hate fuzz tests!

	value, key := []byte("any"), []byte("and bottle of rum")

	t.Run("not exist", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			misc := tx.Misc()

			if misc.GetCopy(key) != nil {
				t.Error("got unexisting value")
			}

			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	err := db.Update(func(tx Tu) (_ error) {
		return tx.Misc().Set(key, value)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("exists", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			misc := tx.Misc()

			got := misc.GetCopy(key)

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

func TestViewMisc_GetCopy(t *testing.T) {
	// GetCopy(key cipher.SHA256) (value []byte)

	t.Run("memory", func(t *testing.T) {
		testViewMiscGetCopy(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewMiscGetCopy(t, db)
	})

}

func testViewMiscAscend(t *testing.T, db DB) {

	t.Run("empty", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			misc := tx.Misc()

			var called int

			err := misc.Ascend(func(_, _ []byte) (_ error) {
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

	to := []struct{ key, value []byte }{
		{[]byte("1"), []byte("one")},
		{[]byte("2"), []byte("two")},
		{[]byte("3"), []byte("three")},
	}

	err := db.Update(func(tx Tu) (_ error) {
		misc := tx.Misc()

		for _, o := range to {
			if err := misc.Set(o.key, o.value); err != nil {
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
			misc := tx.Misc()

			var called int

			err := misc.Ascend(func(key, value []byte) (_ error) {
				if called > len(to) {
					t.Error("called too many times")
					return ErrStopIteration
				}

				if bytes.Compare(to[called].key, key) != 0 {
					t.Error("wrong order:", string(key))
				}
				if bytes.Compare(to[called].value, value) != 0 {
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
			misc := tx.Misc()

			var called int

			err := misc.Ascend(func(key, value []byte) (_ error) {
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

func TestViewMisc_Ascend(t *testing.T) {
	// Ascend(func(key cipher.SHA256, value []byte) error) (err error)

	t.Run("memory", func(t *testing.T) {
		testViewMiscAscend(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewMiscAscend(t, db)
	})

}

//
// UpdateMisc
//

func TestUpdateMisc_Get(t *testing.T) {

	t.Skip("inherited from ViewMisc")

}

func testUpdateMiscGetCopy(t *testing.T, db DB) {

	t.Skip("inheriter from ViewMisc")

}

func TestUpdateMisc_GetCopy(t *testing.T) {

	t.Skip("inherited from ViewMisc")

}

func testUpdateMiscDel(t *testing.T, db DB) {

	value, key := []byte("ha-ha"), []byte("and bottle of rum")

	t.Run("not exist", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			if err := tx.Misc().Del(key); err != nil {
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
		return tx.Misc().Set(key, value)
	})
	if err != nil {
		t.Error(err)
	}

	t.Run("delete", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			if err := tx.Misc().Del(key); err != nil {
				t.Error(err)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestUpdateMisc_Del(t *testing.T) {
	// Del(key cipher.SHA256) (err error)

	t.Run("memory", func(t *testing.T) {
		testUpdateMiscDel(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateMiscDel(t, db)
	})

}

func testUpdateMiscSet(t *testing.T, db DB) {

	value, key := []byte("yo-h-ho"), []byte("and bottle of rum")

	t.Run("set", func(t *testing.T) {

		err := db.Update(func(tx Tu) (_ error) {
			objs := tx.Misc()

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
			objs := tx.Misc()

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

func TestUpdateMisc_Set(t *testing.T) {
	// Set(key cipher.SHA256, value []byte) (err error)

	t.Run("memory", func(t *testing.T) {
		testUpdateMiscSet(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateMiscSet(t, db)
	})

}

// not implemented yet

/*
func testUpdateMiscAscendDel(t *testing.T, db DB) {

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

func TestUpdateMisc_AscendDel(t *testing.T) {
	// AscendDel(
	//     func(key cipher.SHA256, value []byte) (del bool, err error)) error

	t.Run("memory", func(t *testing.T) {
		testUpdateMiscAscendDel(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateMiscAscendDel(t, db)
	})

}
*/
