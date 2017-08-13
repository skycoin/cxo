package data

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

//
// ViewFeeds
//

func testViewFeedsIsExist(t *testing.T, db DB) {

	pk, _ := cipher.GenerateKeyPair()

	t.Run("not exist", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			if tx.Feeds().IsExist(pk) == true {
				t.Error("got unexisting feed")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	// add
	err := db.Update(func(tx Tu) error {
		return tx.Feeds().Add(pk)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("exist", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			if tx.Feeds().IsExist(pk) != true {
				t.Error("missing feed")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestViewFeeds_IsExist(t *testing.T) {
	// IsExist(pk cipher.PubKey) (ok bool)

	t.Run("memory", func(t *testing.T) {
		testViewFeedsIsExist(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewFeedsIsExist(t, db)
	})

}

func testViewFeedsList(t *testing.T, db DB) {
	t.Run("empty", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			if tx.Feeds().List() != nil {
				t.Error("got non-nil feeds list")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	// add feeds
	pks := testOrderedPublicKeys()

	err := db.Update(func(tx Tu) (_ error) {
		for _, pk := range pks {
			if err := tx.Feeds().Add(pk); err != nil {
				return err
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
			list := tx.Feeds().List()

			testComparePublicKeyLists(t, pks, list)

			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestViewFeeds_List(t *testing.T) {
	// List() (list []cipher.PubKey)

	t.Run("memory", func(t *testing.T) {
		testViewFeedsList(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewFeedsList(t, db)
	})

}

func testViewFeedsAscend(t *testing.T, db DB) {

	t.Run("empty", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			var called int
			err := tx.Feeds().Ascend(func(pk cipher.PubKey) (_ error) {
				called++
				return
			})
			if err != nil {
				t.Error(err)
			}
			if called != 0 {
				t.Error("ranges over empty feeds")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	pks := testOrderedPublicKeys()

	err := db.Update(func(tx Tu) (_ error) {
		for _, pk := range pks {
			if err := tx.Feeds().Add(pk); err != nil {
				return err
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
			var called int
			lsit := []cipher.PubKey{}
			err := tx.Feeds().Ascend(func(pk cipher.PubKey) (_ error) {
				if called > 1 {
					t.Error("called too many times")
					return ErrStopIteration
				}
				lsit = append(lsit, pk)
				called++
				return
			})
			if err != nil {
				t.Error(err)
			}
			if called != 2 {
				t.Error("called wrong times")
			}
			testComparePublicKeyLists(t, pks, lsit)
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("stop iteration", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			var called int
			err := tx.Feeds().Ascend(func(pk cipher.PubKey) error {
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

func TestViewFeeds_Ascend(t *testing.T) {
	// Ascend(func(pk cipher.PubKey) error) (err error)

	t.Run("memory", func(t *testing.T) {
		testViewFeedsAscend(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewFeedsAscend(t, db)
	})

}

func testViewFeedsRoots(t *testing.T, db DB) {

	pk, _ := cipher.GenerateKeyPair()

	t.Run("no feed", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			if tx.Feeds().Roots(pk) != nil {
				t.Error("got Roots of unexisting feed")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	// add
	err := db.Update(func(tx Tu) error {
		return tx.Feeds().Add(pk)
	})
	if err != nil {
		t.Error(err)
	}

	t.Run("got", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			roots := tx.Feeds().Roots(pk)
			if roots == nil {
				t.Error("missing roots")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestViewFeeds_Roots(t *testing.T) {
	// Roots(pk cipher.PubKey) ViewRoots

	t.Run("memory", func(t *testing.T) {
		testViewFeedsRoots(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewFeedsRoots(t, db)
	})

}

//
// UpdateFeeds
//

func TestUpdateFeeds_IsExist(t *testing.T) {
	// IsExist(pk cipher.PubKey) (ok bool)

	t.Skip("inherited from ViewFeeds")

}

func TestUpdateFeeds_List(t *testing.T) {
	// List() (list []cipher.PubKey)

	t.Skip("inherited from ViewFeeds")

}

func TestUpdateFeeds_Ascend(t *testing.T) {
	// Ascend(func(pk cipher.PubKey) error) (err error)

	t.Skip("inherited from ViewFeeds")

}

func testUpdateFeedsAdd(t *testing.T, db DB) {

	pk, _ := cipher.GenerateKeyPair()

	t.Run("add", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			if err := tx.Feeds().Add(pk); err != nil {
				t.Error(err)
			}
			if tx.Feeds().IsExist(pk) == false {
				t.Error("can't add a feed")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("twice", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			if err := tx.Feeds().Add(pk); err != nil {
				t.Error(err)
			}
			if tx.Feeds().IsExist(pk) == false {
				t.Error("can't add a feed twice")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestUpdateFeeds_Add(t *testing.T) {
	// Add(pk cipher.PubKey) (err error)

	t.Run("memory", func(t *testing.T) {
		testUpdateFeedsAdd(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateFeedsAdd(t, db)
	})

}

func testUpdateFeedsDel(t *testing.T, db DB) {

	pk, _ := cipher.GenerateKeyPair()

	t.Run("not exist", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			feeds := tx.Feeds()
			if err := feeds.Del(pk); err != nil {
				t.Error(err)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	// add
	err := db.Update(func(tx Tu) (_ error) {
		feeds := tx.Feeds()
		return feeds.Add(pk)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("delete", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			feeds := tx.Feeds()
			if err := feeds.Del(pk); err != nil {
				t.Error(err)
			}
			if feeds.IsExist(pk) == true {
				t.Error("can't delete a feed")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	// add feed and roots
	if testFillWithExampleFeed(t, pk, db); t.Failed() {
		return
	}

	t.Run("roots", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			feeds := tx.Feeds()
			if err := feeds.Del(pk); err != nil {
				t.Error(err)
			}
			if feeds.IsExist(pk) == true {
				t.Error("can't delete a feed with roots")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestUpdateFeeds_Del(t *testing.T) {
	// Del(pk cipher.PubKey) (err error)

	t.Run("memory", func(t *testing.T) {
		testUpdateFeedsDel(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateFeedsDel(t, db)
	})

}

func testUpdateFeedsAscendDel(t *testing.T, db DB) {

	t.Run("no feeds", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			feeds := tx.Feeds()

			var called int
			feeds.AscendDel(func(cipher.PubKey) (_ bool, _ error) {
				called++
				return
			})

			if called != 0 {
				t.Error("ranges without feeds")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	// add feeds
	pks := testOrderedPublicKeys()
	err := db.Update(func(tx Tu) (err error) {
		feeds := tx.Feeds()
		for _, pk := range pks {
			if err = feeds.Add(pk); err != nil {
				return
			}
		}
		return
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("delete", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			feeds := tx.Feeds()

			var called int
			order := []cipher.PubKey{}

			err = feeds.AscendDel(func(pk cipher.PubKey) (del bool,
				err error) {

				if called > 2 {
					t.Error("wront times called")
					return false, ErrStopIteration
				}
				order = append(order, pk)
				del = (pk == pks[1])
				return
			})
			if err != nil {
				t.Error(err)
			}
			testComparePublicKeyLists(t, pks, order)

			if feeds.IsExist(pks[1]) == true {
				t.Error("AscendDel doesn't delete a feed")
			}

			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	// add removed feed
	err = db.Update(func(tx Tu) (err error) {
		return tx.Feeds().Add(pks[1])
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("stop iteration", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			feeds := tx.Feeds()

			var called int
			feeds.AscendDel(func(cipher.PubKey) (_ bool, _ error) {
				called++
				return false, ErrStopIteration
			})

			if called != 1 {
				t.Error("ErrStopIteration doesn't stop the AscendDel")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestUpdateFeeds_AscendDel(t *testing.T) {
	// AscendDel(func(pk cipher.PubKey) (del bool, err error)) error

	t.Run("memory", func(t *testing.T) {
		testUpdateFeedsAscendDel(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateFeedsAscendDel(t, db)
	})

}

func testUpdateFeedsRoots(t *testing.T, db DB) {

	pk, _ := cipher.GenerateKeyPair()

	t.Run("no feed", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			if tx.Feeds().Roots(pk) != nil {
				t.Error("got Roots of unexisting feed")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	// add
	err := db.Update(func(tx Tu) error {
		return tx.Feeds().Add(pk)
	})
	if err != nil {
		t.Error(err)
	}

	t.Run("got", func(t *testing.T) {
		err := db.Update(func(tx Tu) (_ error) {
			roots := tx.Feeds().Roots(pk)
			if roots == nil {
				t.Error("missing roots")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestUpdateFeeds_Roots(t *testing.T) {
	// Roots(pk cipher.PubKey) UpdateRoots

	t.Run("memory", func(t *testing.T) {
		testUpdateFeedsRoots(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testUpdateFeedsRoots(t, db)
	})

}
