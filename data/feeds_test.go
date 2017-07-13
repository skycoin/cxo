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

func testViewFeedsRange(t *testing.T, db DB) {

	t.Run("empty", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			var called int
			err := tx.Feeds().Range(func(pk cipher.PubKey) (_ error) {
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
			err := tx.Feeds().Range(func(pk cipher.PubKey) (_ error) {
				if called > 1 {
					t.Error("called too many times")
					return ErrStopRange
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

	t.Run("stop range", func(t *testing.T) {
		err := db.View(func(tx Tv) (_ error) {
			var called int
			err := tx.Feeds().Range(func(pk cipher.PubKey) error {
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

func TestViewFeeds_Range(t *testing.T) {
	// Range(func(pk cipher.PubKey) error) (err error)

	t.Run("memory", func(t *testing.T) {
		testViewFeedsRange(t, NewMemoryDB())
	})

	t.Run("drive", func(t *testing.T) {
		db, cleanUp := testDriveDB(t)
		defer cleanUp()
		testViewFeedsRange(t, db)
	})

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
