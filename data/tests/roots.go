package tests

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

func addFeed(t *testing.T, idx data.IdxDB, pk cipher.PubKey) {
	err := idx.Tx(func(feeds data.Feeds) (err error) {
		return feeds.Add(pk)
	})
	if err != nil {
		t.Error(err)
	}
}

func newRoot(seed string, sk cipher.SecKey) (r *data.Root) {
	r = new(data.Root)
	r.CreateTime = 111
	r.AccessTime = 222

	r.Seq = 0
	r.Prev = cipher.SHA256{}
	r.Hash = cipher.SumSHA256([]byte(seed))
	r.Sig = cipher.SignHash(r.Hash, sk)
	return
}

func addRoot(t *testing.T, idx data.IdxDB, pk cipher.PubKey, r *data.Root) {
	err := idx.Tx(func(feeds data.Feeds) (err error) {
		var rs data.Roots
		if rs, err = feeds.Roots(pk); err != nil {
			return
		}
		return rs.Set(r)
	})
	if err != nil {
		t.Error(err)
	}
}

// RootsAscend is test case for Roots.Ascend
func RootsAscend(t *testing.T, idx data.IdxDB) {

	pk, sk := cipher.GenerateKeyPair()

	if addFeed(t, idx, pk); t.Failed() {
		return
	}

	t.Run("empty feed", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var called int
			var rs data.Roots
			if rs, err = feeds.Roots(pk); err != nil {
				return
			}
			err = rs.Ascend(func(r *data.Root) (err error) {
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

	r1 := newRoot("r", sk)
	r2 := newRoot("e", sk)

	r2.Seq, r2.Prev = 1, cipher.SumSHA256([]byte("random"))

	ra := []*data.Root{r1, r2}

	for _, r := range ra {
		if addRoot(t, idx, pk, r); t.Failed() {
			return
		}
	}

	t.Run("ascend", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var rs data.Roots
			if rs, err = feeds.Roots(pk); err != nil {
				return
			}
			var called int
			err = rs.Ascend(func(r *data.Root) (err error) {
				if called > len(ra)-1 {
					t.Error("called too many times", called)
					return data.ErrStopIteration
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
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var rs data.Roots
			if rs, err = feeds.Roots(pk); err != nil {
				return
			}
			var called int
			err = rs.Ascend(func(r *data.Root) error {
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

	r3 := newRoot("w", sk)
	r3.Seq, r3.Prev = 2, cipher.SumSHA256([]byte("random"))

	t.Run("mutate add", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var rs data.Roots
			if rs, err = feeds.Roots(pk); err != nil {
				return
			}
			var called int
			err = rs.Ascend(func(r *data.Root) (err error) {
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
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var rs data.Roots
			if rs, err = feeds.Roots(pk); err != nil {
				return
			}
			var called int
			err = rs.Ascend(func(r *data.Root) (err error) {
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

// RootsDescend is test case for Roots.Descend
func RootsDescend(t *testing.T, idx data.IdxDB) {

	pk, sk := cipher.GenerateKeyPair()

	if addFeed(t, idx, pk); t.Failed() {
		return
	}

	t.Run("empty feed", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var called int
			var rs data.Roots
			if rs, err = feeds.Roots(pk); err != nil {
				return
			}
			err = rs.Descend(func(r *data.Root) (err error) {
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

	r3 := newRoot("r", sk)
	r2 := newRoot("e", sk)

	r3.Seq, r3.Prev = 2, cipher.SumSHA256([]byte("random"))
	r2.Seq, r2.Prev = 1, cipher.SumSHA256([]byte("random"))

	ra := []*data.Root{r3, r2}

	for _, r := range ra {
		if addRoot(t, idx, pk, r); t.Failed() {
			return
		}
	}

	t.Run("descend", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var rs data.Roots
			if rs, err = feeds.Roots(pk); err != nil {
				return
			}
			var called int
			err = rs.Descend(func(r *data.Root) (err error) {
				if called > len(ra)-1 {
					t.Error("called too many times", called)
					return data.ErrStopIteration
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
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var rs data.Roots
			if rs, err = feeds.Roots(pk); err != nil {
				return
			}
			var called int
			err = rs.Descend(func(r *data.Root) error {
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

	r1 := newRoot("w", sk)

	t.Run("mutate add", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var rs data.Roots
			if rs, err = feeds.Roots(pk); err != nil {
				return
			}
			var called int
			err = rs.Descend(func(r *data.Root) (err error) {
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
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var rs data.Roots
			if rs, err = feeds.Roots(pk); err != nil {
				return
			}
			var called int
			err = rs.Descend(func(r *data.Root) (err error) {
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

// RootsSet is test case for Roots.Set
func RootsSet(t *testing.T, idx data.IdxDB) {

	pk, sk := cipher.GenerateKeyPair()

	if addFeed(t, idx, pk); t.Failed() {
		return
	}

	r := newRoot("r", sk)

	t.Run("create", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var rs data.Roots
			if rs, err = feeds.Roots(pk); err != nil {
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
			var x *data.Root
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

	t.Run("update", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var rs data.Roots
			if rs, err = feeds.Roots(pk); err != nil {
				return
			}
			if err = rs.Set(r); err != nil {
				return
			}
			var x *data.Root
			if x, err = rs.Get(r.Seq); err != nil {
				return
			}
			r.AccessTime = x.AccessTime
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

// RootsDel is test case for Roots.Del
func RootsDel(t *testing.T, idx data.IdxDB) {

	/*	pk, sk := cipher.GenerateKeyPair()

		if addFeed(t, idx, pk); t.Failed() {
			return
		}*/

}

// RootsGet is test case for Roots.Get
func RootsGet(t *testing.T, idx data.IdxDB) {

	pk, _ := cipher.GenerateKeyPair()

	if addFeed(t, idx, pk); t.Failed() {
		return
	}

	t.Run("not found", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var rs data.Roots
			if rs, err = feeds.Roots(pk); err != nil {
				return
			}
			_, err = rs.Get(0)
			return
		})
		if err == nil {
			t.Error("missing error")
		} else if err != data.ErrNotFound {
			t.Error("unexpected error:", err)
		}
	})

}

// RootsHas is test case for Roots.Has
func RootsHas(t *testing.T, idx data.IdxDB) {

	pk, sk := cipher.GenerateKeyPair()

	if addFeed(t, idx, pk); t.Failed() {
		return
	}

	t.Run("not found", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var rs data.Roots
			if rs, err = feeds.Roots(pk); err != nil {
				return
			}
			if true == rs.Has(0) {
				t.Error("has unexisting root")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

	addRoot(t, idx, pk, newRoot("seed", sk))

	t.Run("has", func(t *testing.T) {
		err := idx.Tx(func(feeds data.Feeds) (err error) {
			var rs data.Roots
			if rs, err = feeds.Roots(pk); err != nil {
				return
			}
			if true != rs.Has(0) {
				t.Error("has not existing root")
			}
			return
		})
		if err != nil {
			t.Error(err)
		}
	})

}
