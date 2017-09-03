package tests

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

func FeedsAdd(t *testing.T, idx data.IdxDB) {
	pk, _ := cipher.GenerateKeyPair()

	t.Run("add", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			if err = feeds.Add(pk); err != nil {
				return
			}
			if feeds.Has(pk) == false {
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
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			if err = feeds.Add(pk); err != nil {
				return
			}
			if feeds.Has(pk) == false {
				t.Error("missing feed")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func FeedsDel(t *testing.T, idx data.IdxDB) {

	pk, sk := cipher.GenerateKeyPair()

	// not found (should not return NotFoundError)
	t.Run("not found", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) error {
			return feeds.Del(pk)
		})
		if err != nil {
			t.Error(err)
		}
	})

	if testAddFeed(t, idx, pk); t.Failed() {
		return
	}

	// delete
	t.Run("delete", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			if err = feeds.Del(pk); err != nil {
				return
			}
			if feeds.Has(pk) == true {
				t.Error("feed was not deleted")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	if testAddFeed(t, idx, pk); t.Failed() {
		return
	}
	if testAddRoot(t, idx, pk, testNewRoot("root", sk)); t.Failed() {
		return
	}

	t.Run("not empty", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) error {
			return feeds.Del(pk)
		})
		if err == nil {
			t.Error("missing error")
		} else if err != data.ErrFeedIsNotEmpty {
			t.Error("unexpected error:", err)
		}
	})
}

func FeedsIterate(t *testing.T, idx data.IdxDB) {

	t.Run("no feeds", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var called int
			err = feeds.Iterate(func(pk cipher.PubKey) (err error) {
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

	for _, pk := range []cipher.PubKey{pk1, pk2} {
		if testAddFeed(t, idx, pk); t.Failed() {
			return
		}
		pks[pk] = struct{}{}
	}

	t.Run("itterate", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var called int
			err = feeds.Iterate(func(pk cipher.PubKey) (err error) {
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
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var called int
			err = feeds.Iterate(func(pk cipher.PubKey) (err error) {
				called++
				return data.ErrStopIteration
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
		err := idx.Tx(func(feeds data.Feeds) (err error) {
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
		err := idx.Tx(func(feeds data.Feeds) (err error) {
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

func FeedsHas(t *testing.T, idx data.IdxDB) {

	pk, _ := cipher.GenerateKeyPair()

	t.Run("not exist", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (_ error) {
			if feeds.Has(pk) == true {
				t.Error("has unexisting feed")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	if testAddFeed(t, idx, pk); t.Failed() {
		return
	}

	t.Run("exists", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (_ error) {
			if feeds.Has(pk) == false {
				t.Error("has not existing feed")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func FeedsRoots(t *testing.T, idx data.IdxDB) {

	pk, _ := cipher.GenerateKeyPair()

	t.Run("no such feed", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var rs data.Roots
			if rs, err = feeds.Roots(pk); err == nil {
				t.Error("missng data.ErrNoSuchFeed")
			} else if err != data.ErrNoSuchFeed {
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

	if testAddFeed(t, idx, pk); t.Failed() {
		return
	}

	t.Run("empty", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var rs data.Roots
			if rs, err = feeds.Roots(pk); err != nil {
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
