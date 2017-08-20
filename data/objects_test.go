package data

import (
	"bytes"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

//
// ViewObjects
//

func testViewObjectsGetObject(t *testing.T, db DB) {

	obj := testObjectOf("any")
	key := cipher.SumSHA256(obj.Value)

	t.Run("not exist", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			objs := tx.Objects()

			if objs.GetObject(key) != nil {
				t.Error("got unexisting value")
			}

			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	err := db.Update(func(tx Tu) (_ error) {
		return tx.Objects().Set(key, obj)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("exists", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			objs := tx.Objects()

			got := objs.GetObject(key)

			if got == nil {
				t.Error("missing value")
				return
			}

			if bytes.Compare(got.Value, obj.Value) != 0 {
				t.Error("wrong value")
			}

			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestViewObjects_GetObject(t *testing.T) {
	// Get(key cipher.SHA256) (value []byte)

	t.Run("memory", func(t *testing.T) {
		testViewObjectsGetObject(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewObjectsGetObject(t, db)
	})

}

func testViewObjectsGet(t *testing.T, db DB) {

	obj := testObjectOf("any")
	key := cipher.SumSHA256(obj.Value)

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
		return tx.Objects().Set(key, obj)
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

			if bytes.Compare(got, obj.Value) != 0 {
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
			if err := objs.Set(o.key, o.obj); err != nil {
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
				if bytes.Compare(tObj.obj.Value, value) != 0 {
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

func TestViewObjects_MultiGet(t *testing.T) {
	// MultiGet(keys ...cipher.SHA256) (vals [][]byte)

	// TODO (kostyarin): implement
	t.Skip("not implemented yet")
}

func TestViewObjects_Amount(t *testing.T) {
	// Amount() uint32

	// TODO (kostyarin): implement
	t.Skip("not implemented yet")
}

func TestViewObjects_Volume(t *testing.T) {
	// Volume() (vol Volume)

	// TODO (kostyarin): implement
	t.Skip("not implemented yet")
}

//
// UpdateObjects
//

// inherited from ViewObjects

func TestUpdateObjects_Get(t *testing.T) {
	// Get(key cipher.SHA256) (value []byte)

	t.Skip("inherited from ViewObjects")
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
			if err := objs.Set(o.key, o.obj); err != nil {
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
				if bytes.Compare(tObj.obj.Value, value) != 0 {
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

	obj := testObjectOf("ho-ho-ho")
	key := cipher.SumSHA256(obj.Value)

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
		return tx.Objects().Set(key, obj)
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

	obj := testObjectOf("ha-ha-ha")
	key := cipher.SumSHA256(obj.Value)

	t.Run("set", func(t *testing.T) {

		err := db.Update(func(tx Tu) (_ error) {
			objs := tx.Objects()

			if err := objs.Set(key, obj); err != nil {
				t.Error(err)
				return
			}

			if bytes.Compare(objs.Get(key), obj.Value) != 0 {
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

			replace := testObjectOf("zorro!")

			if err := objs.Set(key, replace); err != nil {
				t.Error(err)
				return
			}

			if bytes.Compare(objs.Get(key), replace.Value) != 0 {
				t.Error("wrong or missing value")
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

	obj := testObjectOf("any")
	key := cipher.SumSHA256(obj.Value)

	t.Run("name", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			objs := tx.Objects()

			if rk, err := objs.Add(obj); err != nil {
				t.Error(err)
				return
			} else if bytes.Compare(rk[:], key[:]) != 0 {
				t.Error("wrong key returned")
				return
			}

			if bytes.Compare(objs.Get(key), obj.Value) != 0 {
				t.Error("wrong of missing value")
			}

			// adding many times should not produce an error
			if _, err := objs.Add(obj); err != nil {
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
			if err := objs.Set(o.key, o.obj); err != nil {
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
				if bytes.Compare(tObj.obj.Value, value) != 0 {
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
				if string(o.obj.Value) == "two" {
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

func TestUpdateObjects_MultiAdd(t *testing.T) {
	// MultiAdd(vals ...[]byte) (err error)

	// TODO (kostyarin): implement
	t.Skip("not implemented yet")
}
