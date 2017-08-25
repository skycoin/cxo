package idxdb

import (
	"os"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

func testAddFeed(t *testing.T, idx IdxDB, pk cipher.PubKey) {
	err := idx.Tx(func(tx Tx) (err error) {
		return tx.Feeds().Add(pk)
	})
	if err != nil {
		t.Error(err)
	}
}

func testNewRoot(seed string, sk cipher.SecKey) (r *Root) {
	r = new(Root)
	r.Vol = 500
	r.Subtree.Amount = 1000
	r.Subtree.Volume = 9000
	r.RefsCount = 888
	r.CreateTime = 111
	r.AccessTime = 222

	r.Seq = 0
	r.Prev = cipher.SHA256{}
	r.Hash = cipher.SumSHA256([]byte(seed))
	r.Sig = cipher.SignHash(r.Hash, sk)
	r.IsFull = false

	return
}

func testAddRoot(t *testing.T, idx IdxDB, pk cipher.PubKey, r *Root) {
	err := idx.Tx(func(tx Tx) (err error) {
		var rs Roots
		if rs, err = tx.Feeds().Roots(pk); err != nil {
			return
		}
		return rs.Set(r)
	})
	if err != nil {
		t.Error(err)
	}
}

func testRootsAscend(t *testing.T, idx IdxDB) {

	pk, sk := cipher.GenerateKeyPair()

	if testAddFeed(t, idx, pk); t.Failed() {
		return
	}

	t.Run("empty feed", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var called int
			var rs Roots
			if rs, err = tx.Feeds().Roots(pk); err != nil {
				return
			}
			err = rs.Ascend(func(r *Root) (err error) {
				called++
				return
			})
			if err != nil {
				return
			}
			if called != 0 {
				t.Error("called on empty feed:", called)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	r1 := testNewRoot("r", sk)
	r2 := testNewRoot("e", sk)

	r2.Seq = 1

	ra := []*Root{r1, r2}

	for _, r := range ra {
		if testAddRoot(t, idx, pk, r); t.Failed() {
			return
		}
	}

	t.Run("ascend", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var rs Roots
			if rs, err = tx.Feeds().Roots(pk); err != nil {
				return
			}
			var called int
			err = rs.Ascend(func(r *Root) (err error) {
				if called > len(ra)-1 {
					t.Error("called too many times", called)
					return ErrStopIteration
				}
				x := ra[called]
				if r.Hash != x.Hash {
					t.Error("got wrong Root")
				}
				called++
				return
			})
			if err != nil {
				return
			}
			if called != 2 {
				t.Error("called too few times", called)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("stop iteration", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var rs Roots
			if rs, err = tx.Feeds().Roots(pk); err != nil {
				return
			}
			var called int
			err = rs.Ascend(func(r *Root) error {
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

	r3 := testNewRoot("w", sk)
	r3.Seq = 2

	t.Run("mutate add", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var rs Roots
			if rs, err = tx.Feeds().Roots(pk); err != nil {
				return
			}
			var called int
			err = rs.Ascend(func(r *Root) (err error) {
				if called == 0 {
					if err = rs.Set(r3); err != nil {
						return
					}
				}
				called++
				return
			})
			if err != nil {
				return
			}
			if called != 3 {
				t.Error("wrong times called")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("mutate del", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var rs Roots
			if rs, err = tx.Feeds().Roots(pk); err != nil {
				return
			}
			var called int
			err = rs.Ascend(func(r *Root) (err error) {
				if called == 0 {
					if err = rs.Del(r3.Seq); err != nil {
						return
					}
				}
				called++
				return
			})
			if err != nil {
				return
			}
			if called != 2 {
				t.Error("wrong times called")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestRoots_Ascend(t *testing.T) {
	// Ascend(IterateRootsFunc) error

	// TODO (kostyarin): memeory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testRootsAscend(t, idx)
	})

}

func testRootsDescend(t *testing.T, idx IdxDB) {

	pk, sk := cipher.GenerateKeyPair()

	if testAddFeed(t, idx, pk); t.Failed() {
		return
	}

	t.Run("empty feed", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var called int
			var rs Roots
			if rs, err = tx.Feeds().Roots(pk); err != nil {
				return
			}
			err = rs.Ascend(func(r *Root) (err error) {
				called++
				return
			})
			if err != nil {
				return
			}
			if called != 0 {
				t.Error("called on empty feed:", called)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	r3 := testNewRoot("r", sk)
	r2 := testNewRoot("e", sk)

	r3.Seq, r2.Seq = 2, 1

	ra := []*Root{r3, r2}

	for _, r := range ra {
		if testAddRoot(t, idx, pk, r); t.Failed() {
			return
		}
	}

	t.Run("descend", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var rs Roots
			if rs, err = tx.Feeds().Roots(pk); err != nil {
				return
			}
			var called int
			err = rs.Descend(func(r *Root) (err error) {
				if called > len(ra)-1 {
					t.Error("called too many times", called)
					return ErrStopIteration
				}
				x := ra[called]
				if r.Hash != x.Hash {
					t.Error("got wrong Root")
				}
				called++
				return
			})
			if err != nil {
				return
			}
			if called != 2 {
				t.Error("called too few times", called)
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("stop iteration", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var rs Roots
			if rs, err = tx.Feeds().Roots(pk); err != nil {
				return
			}
			var called int
			err = rs.Descend(func(r *Root) error {
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

	r1 := testNewRoot("w", sk)

	t.Run("mutate add", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var rs Roots
			if rs, err = tx.Feeds().Roots(pk); err != nil {
				return
			}
			var called int
			err = rs.Descend(func(r *Root) (err error) {
				if called == 0 {
					if err = rs.Set(r1); err != nil {
						return
					}
				}
				called++
				return
			})
			if err != nil {
				return
			}
			if called != 3 {
				t.Error("wrong times called")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("mutate del", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var rs Roots
			if rs, err = tx.Feeds().Roots(pk); err != nil {
				return
			}
			var called int
			err = rs.Descend(func(r *Root) (err error) {
				if called == 0 {
					if err = rs.Del(r1.Seq); err != nil {
						return
					}
				}
				called++
				return
			})
			if err != nil {
				return
			}
			if called != 2 {
				t.Error("wrong times called")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestRoots_Descend(t *testing.T) {
	// Descend(IterateRootsFunc) error

	// TODO (kostyarin): memeory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testRootsDescend(t, idx)
	})

}

func testRootsSet(t *testing.T, idx IdxDB) {

	pk, sk := cipher.GenerateKeyPair()

	if testAddFeed(t, idx, pk); t.Failed() {
		return
	}

	r := testNewRoot("r", sk)

	t.Run("create", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var rs Roots
			if rs, err = tx.Feeds().Roots(pk); err != nil {
				return
			}
			if err = rs.Set(r); err != nil {
				return
			}
			if r.CreateTime == 111 {
				t.Error("CreateTime not updated")
			}
			if r.AccessTime != r.CreateTime {
				t.Error("access time not set")
			}
			var x *Root
			if x, err = rs.Get(r.Seq); err != nil {
				return
			}
			if *x != *r {
				t.Error("wrong")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	r.IsFull = true      // should be updated
	r.Subtree.Volume = 1 // should not be updated

	t.Run("unpdate", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var rs Roots
			if rs, err = tx.Feeds().Roots(pk); err != nil {
				return
			}
			if err = rs.Set(r); err != nil {
				return
			}
			var x *Root
			if x, err = rs.Get(r.Seq); err != nil {
				return
			}
			r.Subtree.Volume = 9000 // restore
			if *x != *r {
				t.Error("wrong")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestRoots_Set(t *testing.T) {
	// Set(*Root) error

	// TODO (kostyarin): memeory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testRootsSet(t, idx)
	})

}

func testRootsDel(t *testing.T, idx IdxDB) {

	/*	pk, sk := cipher.GenerateKeyPair()

		if testAddFeed(t, idx, pk); t.Failed() {
			return
		}*/

}

func TestRoots_Del(t *testing.T) {
	// Del(uint64) error

	// TODO (kostyarin): memeory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testRootsDel(t, idx)
	})

}

func testRootsGet(t *testing.T, idx IdxDB) {

	pk, _ := cipher.GenerateKeyPair()

	if testAddFeed(t, idx, pk); t.Failed() {
		return
	}

	t.Run("not found", func(t *testing.T) {
		err := idx.Tx(func(tx Tx) (err error) {
			var rs Roots
			if rs, err = tx.Feeds().Roots(pk); err != nil {
				return
			}
			_, err = rs.Get(0)
			return
		})
		if err == nil {
			t.Error("missing error")
		} else if err != ErrNotFound {
			t.Error("unexpected error:", err)
		}
	})

}

func TestRoots_Get(t *testing.T) {
	// Get(uint64) (*Root, error)

	// TODO (kostyarin): memeory

	t.Run("drive", func(t *testing.T) {
		idx := testNewDriveIdxDB(t)
		defer os.Remove(testFileName)
		defer idx.Close()

		testRootsGet(t, idx)
	})

}
