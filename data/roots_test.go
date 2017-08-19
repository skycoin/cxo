package data

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

//
// helper
//

func testFillWithExampleFeed(t *testing.T, pk cipher.PubKey, db DB) {
	// add feed and root
	err := db.Update(func(tx Tu) (err error) {
		feeds := tx.Feeds()
		if err = feeds.Add(pk); err != nil {
			return
		}
		roots := feeds.Roots(pk)
		for _, rp := range []RootPack{
			getRootPack(0, "hey"),
			getRootPack(1, "hoy"),
			getRootPack(2, "gde kon' moy voronoy"),
		} {
			if err = roots.Add(&rp); err != nil {
				return
			}
		}
		return
	})
	if err != nil {
		t.Error(err)
	}
}

//
// ViewRoots
//

func testViewRootsFeed(t *testing.T, db DB) {
	pk, _ := cipher.GenerateKeyPair()

	if testFillWithExampleFeed(t, pk, db); t.Failed() {
		return
	}

	err := db.View(func(tx Tv) (_ error) {
		if tx.Feeds().Roots(pk).Feed() != pk {
			t.Error("wrong feed of Roots")
		}
		return
	})
	if err != nil {
		t.Error(err)
	}

}

func TestViewRoots_Feed(t *testing.T) {
	// Feed() cipher.PubKey

	t.Run("memory", func(t *testing.T) {
		testViewRootsFeed(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewRootsFeed(t, db)
	})

}

func testViewRootsLast(t *testing.T, db DB) {
	pk, _ := cipher.GenerateKeyPair()

	// add empty feed
	err := db.Update(func(tx Tu) error {
		return tx.Feeds().Add(pk)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("empty", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			if tx.Feeds().Roots(pk).Last() != nil {
				t.Error("got last root of empty feed")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	if testFillWithExampleFeed(t, pk, db); t.Failed() {
		return
	}

	t.Run("full", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			rp := tx.Feeds().Roots(pk).Last()
			if rp == nil {
				t.Error("misisng last root")
				return
			}
			if rp.Seq != 2 {
				t.Error("wrong last root")
			}

			return
		})
		if err != nil {
			t.Error(err)
		}
	})
}

func TestViewRoots_Last(t *testing.T) {
	// Last() (rp *RootPack)

	t.Run("memory", func(t *testing.T) {
		testViewRootsLast(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewRootsLast(t, db)
	})

}

func testViewRootsGet(t *testing.T, db DB) {
	pk, _ := cipher.GenerateKeyPair()

	// add empty feed
	err := db.Update(func(tx Tu) error {
		return tx.Feeds().Add(pk)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("empty", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			roots := tx.Feeds().Roots(pk)
			if roots.Get(0) != nil {
				t.Error("got a root of empty feed")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	if testFillWithExampleFeed(t, pk, db); t.Failed() {
		return
	}

	t.Run("full", func(t *testing.T) {
		err = db.View(func(tx Tv) (_ error) {
			roots := tx.Feeds().Roots(pk)
			for _, u := range []uint64{0, 1, 2} {
				if rp := roots.Get(u); rp == nil {
					t.Error("missing root")
				} else if rp.Seq != u {
					t.Error("got with wrong seq")
				}
			}
			if roots.Get(1050) != nil {
				t.Error("got unexisting root")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestViewRoots_Get(t *testing.T) {
	// Get(seq uint64) (rp *RootPack)

	t.Run("memory", func(t *testing.T) {
		testViewRootsGet(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewRootsGet(t, db)
	})

}

func testViewRootsAscend(t *testing.T, db DB) {

	pk, _ := cipher.GenerateKeyPair()

	// add empty feed
	err := db.Update(func(tx Tu) error {
		return tx.Feeds().Add(pk)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("empty", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			roots := tx.Feeds().Roots(pk)

			var called int
			err := roots.Ascend(func(rp *RootPack) (_ error) {
				called++
				return
			})
			if err != nil {
				t.Error(err)
			}
			if called != 0 {
				t.Error("ranges over empty feed (no roots)")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	if testFillWithExampleFeed(t, pk, db); t.Failed() {
		return
	}

	t.Run("full", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			roots := tx.Feeds().Roots(pk)

			var called int
			err := roots.Ascend(func(rp *RootPack) (_ error) {
				if called > 2 {
					t.Error("called too many times")
					return ErrStopIteration
				}
				if rp.Seq != uint64(called) {
					t.Error("wrong order")
				}
				called++
				return
			})
			if err != nil {
				t.Error(err)
			}
			if called != 3 {
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
			roots := tx.Feeds().Roots(pk)

			var called int
			err := roots.Ascend(func(rp *RootPack) error {
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

func TestViewRoots_Ascend(t *testing.T) {
	// Ascend(func(rp *RootPack) (err error)) error

	t.Run("memory", func(t *testing.T) {
		testViewRootsAscend(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewRootsAscend(t, db)
	})

}

func testViewRootsDescend(t *testing.T, db DB) {

	pk, _ := cipher.GenerateKeyPair()

	// add empty feed
	err := db.Update(func(tx Tu) error {
		return tx.Feeds().Add(pk)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("empty", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			roots := tx.Feeds().Roots(pk)

			var called int
			err := roots.Descend(func(rp *RootPack) (_ error) {
				called++
				return
			})
			if err != nil {
				t.Error(err)
			}
			if called != 0 {
				t.Error("ranges over empty feed (no roots)")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	if testFillWithExampleFeed(t, pk, db); t.Failed() {
		return
	}

	t.Run("full", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			roots := tx.Feeds().Roots(pk)

			var called int
			err := roots.Descend(func(rp *RootPack) (_ error) {
				if called > 2 {
					t.Error("called too many times")
					return ErrStopIteration
				}
				if rp.Seq != uint64(2-called) {
					t.Error("wrong order")
				}
				called++
				return
			})
			if err != nil {
				t.Error(err)
			}
			if called != 3 {
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
			roots := tx.Feeds().Roots(pk)

			var called int
			err := roots.Descend(func(rp *RootPack) error {
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

func TestViewRoots_Descend(t *testing.T) {
	// Reverse(fn func(rp *RootPack) (err error)) error

	t.Run("memory", func(t *testing.T) {
		testViewRootsDescend(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewRootsDescend(t, db)
	})

}

//
// UpdateRoots
//

// inherited from ViewRoots

func TestUpdateRoots_Feed(t *testing.T) {
	// Feed() cipher.PubKey

	t.Skip("inherited from ViewRoots")

}

func TestUpdateRoots_Last(t *testing.T) {
	// Last() (rp *RootPack)

	t.Skip("inherited from ViewRoots")

}

func TestUpdateRoots_Get(t *testing.T) {
	// Get(seq uint64) (rp *RootPack)

	t.Skip("inherited from ViewRoots")

}

func TestUpdateRoots_Ascend(t *testing.T) {
	// Acsending(func(rp *RootPack) (err error)) error

	t.Skip("inherited from ViewRoots")

}

func TestUpdateRoots_Reverse(t *testing.T) {
	// Reverse(fn func(rp *RootPack) (err error)) error

	t.Skip("inherited from ViewRoots")

}

// UpdateRoots

func testUpdateRootsAdd(t *testing.T, db DB) {

	pk, _ := cipher.GenerateKeyPair()

	// add empty feed
	err := db.Update(func(tx Tu) error {
		return tx.Feeds().Add(pk)
	})
	if err != nil {
		t.Error(err)
		return
	}

	// don't test Hash/Prev/Seq etc (seems to be depricated)

	rp := getRootPack(0, "yo-ho-ho")

	t.Run("add", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			roots := tx.Feeds().Roots(pk)

			if err := roots.Add(&rp); err != nil {
				t.Error(err)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	if t.Failed() {
		return
	}

	t.Run("already exists", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			roots := tx.Feeds().Roots(pk)

			if err := roots.Add(&rp); err == nil {
				t.Error("misisng error")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestUpdateRoots_Add(t *testing.T) {
	// Add(rp *RootPack) (err error)

	t.Run("memory", func(t *testing.T) {
		testUpdateRootsAdd(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateRootsAdd(t, db)
	})

}

func testUpdateRootsDel(t *testing.T, db DB) {
	pk, _ := cipher.GenerateKeyPair()

	if testFillWithExampleFeed(t, pk, db); t.Failed() {
		return
	}

	err := db.Update(func(tx Tu) (_ error) {
		roots := tx.Feeds().Roots(pk)
		if err := roots.Del(0); err != nil {
			t.Error(err)
		}
		if err := roots.Del(0); err != nil {
			t.Error(err)
		}
		return
	})
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateRoots_Del(t *testing.T) {
	// Del(seq uint64) (err error)

	t.Run("memory", func(t *testing.T) {
		testUpdateRootsDel(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateRootsDel(t, db)
	})

}

func testUpdateRootsUpdate(t *testing.T, db DB) {

	// TODO (kostyarin): implememt
	t.Skip("not implemented")

	/*
		pk, _ := cipher.GenerateKeyPair()

		if testFillWithExampleFeed(t, pk, db); t.Failed() {
			return
		}

		err := db.Update(func(tx Tu) (_ error) {
			roots := tx.Feeds().Roots(pk)
			if err := roots.Update(0); err != nil {
				t.Error(err)
			}
			if rp := roots.Get(0); rp == nil {
				t.Error("missing Root after MarkFull")
			} else if rp.IsFull == false {
				t.Error("not marked as full")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	*/
}

func TestUpdateRoots_Update(t *testing.T) {
	// Del(seq uint64) (err error)

	t.Run("memory", func(t *testing.T) {
		testUpdateRootsUpdate(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateRootsUpdate(t, db)
	})

}

func testUpdateRootsAscendDel(t *testing.T, db DB) {
	pk, _ := cipher.GenerateKeyPair()

	// add empty feed
	err := db.Update(func(tx Tu) error {
		return tx.Feeds().Add(pk)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("empty", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			roots := tx.Feeds().Roots(pk)
			var called int
			err := roots.AscendDel(func(*RootPack) (_ bool, _ error) {
				called++
				return
			})
			if err != nil {
				t.Error(err)
			}
			if called != 0 {
				t.Error("ranges over empty feed")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	if testFillWithExampleFeed(t, pk, db); t.Failed() {
		return
	}

	t.Run("full", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			roots := tx.Feeds().Roots(pk)
			var called int
			err := roots.AscendDel(func(rp *RootPack) (del bool, err error) {
				if called > 2 {
					t.Error("called too many times")
					return false, ErrStopIteration
				}
				if rp.Seq != uint64(called) {
					t.Error("wrong order")
				}
				del = (rp.Seq == 1)
				called++
				return
			})
			if err != nil {
				t.Error(err)
			}
			if called != 3 {
				t.Error("wrong times called")
			}
			if roots.Get(1) != nil {
				t.Error("not deleted")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("stop iteration", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			roots := tx.Feeds().Roots(pk)
			var called int
			err := roots.AscendDel(func(*RootPack) (bool, error) {
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

func TestUpdateRoots_AscendDel(t *testing.T) {
	// AscendDel(fn func(rp *RootPack) (del bool, err error)) error

	t.Run("memory", func(t *testing.T) {
		testUpdateRootsAscendDel(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateRootsAscendDel(t, db)
	})

}

func testUpdateRootsDelBefore(t *testing.T, db DB) {
	pk, _ := cipher.GenerateKeyPair()

	if testFillWithExampleFeed(t, pk, db); t.Failed() {
		return
	}

	err := db.Update(func(tx Tu) (_ error) {
		roots := tx.Feeds().Roots(pk)

		if err := roots.DelBefore(2); err != nil {
			t.Error(err)
			return
		}
		if roots.Get(0) != nil {
			t.Error("not deleted")
		}
		if roots.Get(1) != nil {
			t.Error("not deleted")
		}

		return
	})
	if err != nil {
		t.Error(err)
	}

}

func TestUpdateRoots_DelBefore(t *testing.T) {
	// DelBefore(seq uint64) (err error)

	t.Run("memory", func(t *testing.T) {
		testUpdateRootsDelBefore(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateRootsDelBefore(t, db)
	})

}
