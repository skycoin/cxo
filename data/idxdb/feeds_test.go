package idxdb

import (
	"os"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

func testFeedsAdd(t *testing.T, idx IdxDB) {
	pk, _ := cipher.GenerateKeyPair()

	t.Run("add", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			feeds := tx.Feeds()
			if err = feeds.Add(pk); err != nil {
				return
			}
			if feeds.HasFeed(pk) == false {
				t.Error("missing feed")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	if t.Failed() == true {
		return
	}

	t.Run("twice", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			feeds := tx.Feeds()
			if err = feeds.Add(pk); err != nil {
				return
			}
			if feeds.HasFeed(pk) == false {
				t.Error("missing feed")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestFeeds_Add(t *testing.T) {
	// Add(cipher.PubKey) error

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()
		testFeedsAdd(t, idx)
	})

}

func testFeedsDel(t *testing.T, idx IdxDB) {

	pk, _ := cipher.GenerateKeyPair()

	// not found (should not return NotFoundError)
	t.Run("not found", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) error {
			return tx.Feeds().Del(pk)
		})
		if err != nil {
			t.Error(err)
		}
	})

	err := idx.Tx(func(tx Tx) error {
		return tx.Feeds().Add(pk)
	})
	if err != nil {
		t.Error(err)
		return
	}

	// delete
	t.Run("delete", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			feeds := tx.Feeds()
			if err = feeds.Del(pk); err != nil {
				return
			}
			if feeds.HasFeed(pk) == true {
				t.Error("feed was not deleted")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})
}

func TestFeeds_Del(t *testing.T) {
	// Del(cipher.PubKey) error

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()
		testFeedsDel(t, idx)
	})

}

func testFeedsIterate(t *testing.T, idx IdxDB) {

	t.Run("no feeds", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var called int
			err = tx.Feeds().Iterate(func(pk cipher.PubKey) (err error) {
				called++
				return
			})
			if err != nil {
				return
			}
			if called != 0 {
				t.Error("called", called)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	pk1, _ := cipher.GenerateKeyPair()
	pk2, _ := cipher.GenerateKeyPair()

	pks := make(map[cipher.PubKey]struct{})

	err := idx.Tx(func(tx Tx) (err error) {
		feeds := tx.Feeds()
		for _, pk := range []cipher.PubKey{pk1, pk2} {
			if err = feeds.Add(pk); err != nil {
				return
			}
			pks[pk] = struct{}{}
		}
		return
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("itterate", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var called int
			err = tx.Feeds().Iterate(func(pk cipher.PubKey) (err error) {
				if _, ok := pks[pk]; !ok {
					t.Error("wrong pk given", pk.Hex())
				} else {
					delete(pks, pk)
				}
				called++
				return
			})
			if err != nil {
				return
			}
			if called != 2 {
				t.Error("called wrong times", called)
			}
			if len(pks) != 0 {
				t.Error("existsing feeds was not iterated")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("stop iteration", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var called int
			err = tx.Feeds().Iterate(func(pk cipher.PubKey) (err error) {
				called++
				return ErrStopIteration
			})
			if err != nil {
				return
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

	// mutating during iteration

	pk3, _ := cipher.GenerateKeyPair()

	t.Run("mutate add", func(t *testing.T) {
		var called int
		err := idx.Tx(func(tx Tx) (err error) {
			feeds := tx.Feeds()
			err = feeds.Iterate(func(pk cipher.PubKey) (err error) {
				called++
				if called == 1 {
					return feeds.Add(pk3)
				}
				return
			})
			if err != nil {
				return
			}
			if false == (called == 2 || called == 3) {
				t.Error("wrong times called", called)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("mutate delete", func(t *testing.T) {
		var called int
		err := idx.Tx(func(tx Tx) (err error) {
			feeds := tx.Feeds()
			err = feeds.Iterate(func(pk cipher.PubKey) (err error) {
				called++
				if called == 1 {
					return feeds.Del(pk3)
				}
				return
			})
			if err != nil {
				return
			}
			if false == (called == 2 || called == 3) {
				t.Error("wrong times called", called)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestFeeds_Iterate(t *testing.T) {
	// Iterate(IterateFeedsFunc) error

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()
		testFeedsIterate(t, idx)
	})

}

func testFeedsHasFeed(t *testing.T, idx IdxDB) {

	pk, _ := cipher.GenerateKeyPair()

	t.Run("not exist", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			if tx.Feeds().HasFeed(pk) == true {
				t.Error("has unexisting feed")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	err := idx.Tx(func(tx Tx) error {
		return tx.Feeds().Add(pk)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("exists", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (_ error) {
			if tx.Feeds().HasFeed(pk) == false {
				t.Error("has not existing feed")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestFeeds_HasFeed(t *testing.T) {
	// HasFeed(cipher.PubKey) bool

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()
		testFeedsHasFeed(t, idx)
	})

}

func testFeedsRoots(t *testing.T, idx IdxDB) {

	pk, _ := cipher.GenerateKeyPair()

	t.Run("no such feed", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var rs Roots
			if rs, err = tx.Feeds().Roots(pk); err == nil {
				t.Error("missng ErrNoSuchFeed")
			} else if err != ErrNoSuchFeed {
				return // bubble the err up
			} else if rs != nil {
				t.Error("NoSuchFeed but Roots are not nil")
			}
			return nil
		})
		if err != nil {
			t.Error(err)
		}
	})

	err := idx.Tx(func(tx Tx) error {
		return tx.Feeds().Add(pk)
	})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("empty", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var rs Roots
			if rs, err = tx.Feeds().Roots(pk); err != nil {
				return // bubble the err up
			}
			if rs == nil {
				t.Error("got nil")
			}
			return // ok mutherfucekr
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestFeeds_Roots(t *testing.T) {
	// Roots(cipher.PubKey) (Roots, error)

	// TODO (kostyarin): memory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()
		testFeedsRoots(t, idx)
	})

}
