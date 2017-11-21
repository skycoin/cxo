package registry

import (
	"fmt"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

func testRefsAscend(t *testing.T, r *Refs, pack Pack, users []cipher.SHA256) {

	var err error
	var called int

	err = r.Ascend(pack, func(i int, hash cipher.SHA256) (err error) {

		if i < 0 {
			t.Error("got negative index", i)
		} else if i >= len(users) {
			t.Error("got index out of rage", i)
		} else if users[i] != hash {
			t.Error("wrong hash, want %s, got %s",
				users[i].Hex()[:7],
				hash.Hex()[:7])
		}

		called++

		return // continue
	})

	if err != nil {
		t.Error(err)
	} else if called != len(users) {
		t.Errorf("wrong times called %d, but want %d", called, len(users))
	}

}

func TestRefs_Ascend(t *testing.T) {
	// Ascend(pack Pack, ascendFunc IterateFunc) (err error)

	var (
		pack = testPack()

		refs Refs
		err  error

		users []cipher.SHA256

		clear = func(t *testing.T, r *Refs, degree int) {
			pack.ClearFlags(^0)   // clear flags of pack
			refs.Clear()          // clear the Refs making it Refs{}
			if degree != Degree { // if it's not default
				if err = refs.SetDegree(pack, degree); err != nil { // change it
					t.Fatal(err)
				}
			}
		}
	)

	for _, degree := range []int{
		Degree,     // default
		Degree + 7, // changed
	} {

		t.Run(fmt.Sprintf("blank (degree %d)", degree), func(t *testing.T) {

			clear(t, &refs, degree)

			var called int

			err = refs.Ascend(pack, func(int, cipher.SHA256) (_ error) {
				called++
				return
			})

			if err != nil {
				t.Error(err)
			} else if called > 0 {
				t.Errorf("called %d times, but should not be called at all",
					called)
			}

		})

		// fill
		//   (1) only leafs
		//   (2) leafs and branches
		//   (3) branches with branches with leafs

		for _, length := range []int{
			degree,            // only leafs
			degree + 1,        // leafs and branches
			degree*degree + 1, // branches with branches with leafs
		} {

			t.Logf("Refs with %d elements (degree %d)", length, degree)

			clear(t, &refs, degree)

			// generate users
			users = getHashList(
				testFillRefsWithUsers(t, &refs, pack, length),
			)

			if t.Failed() {
				t.FailNow()
			}

			logRefsTree(t, &refs, pack, false)

			t.Run(fmt.Sprintf("fresh %d:%d", length, degree),
				func(t *testing.T) {

					testRefsAscend(t, &refs, pack, users)
					logRefsTree(t, &refs, pack, false)

				})

			t.Run(fmt.Sprintf("break %d:%d", length, degree),
				func(t *testing.T) {

					var called int

					err = refs.Ascend(pack, func(int, cipher.SHA256) (_ error) {
						called++
						return ErrStopIteration
					})

					if err != nil {
						t.Error(err)
					} else if called > 1 {
						t.Error("called too many times", called)
					}

					logRefsTree(t, &refs, pack, false)

				})

			t.Run(fmt.Sprintf("load %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset() // reset the refs

					testRefsAscend(t, &refs, pack, users)
					logRefsTree(t, &refs, pack, false)

				})

			t.Run(fmt.Sprintf("load entire %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset()              // reset the refs
					pack.AddFlags(EntireRefs) // load entire Refs

					testRefsAscend(t, &refs, pack, users)
					logRefsTree(t, &refs, pack, false)

				})

			t.Run(fmt.Sprintf("hash table index %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset()

					pack.ClearFlags(EntireRefs)
					pack.AddFlags(HashTableIndex)

					testRefsAscend(t, &refs, pack, users)
					logRefsTree(t, &refs, pack, false)

				})

		}

	}

	t.Run("blank hash", func(t *testing.T) {

		refs.Clear()
		pack.ClearFlags(^0) // all

		if err = refs.AppendHashes(pack, cipher.SHA256{}); err != nil {
			t.Fatal(err)
		}

		err = refs.Ascend(pack, func(i int, hash cipher.SHA256) (err error) {
			if i != 0 {
				t.Error("wrong index given")
			}
			if hash != (cipher.SHA256{}) {
				t.Error("wrong hash given")
			}
			return
		})

	})

	// -------------------------------------------------------------------------

	// TODO (kostyaring): extended testing of the Ascend

}

func TestRefs_AscendFrom(t *testing.T) {
	// AscendFrom(pack Pack, from int, ascendFunc IterateFunc) (err error)

	//
}

func testRefsDescend(t *testing.T, r *Refs, pack Pack, users []cipher.SHA256) {

	var err error
	var called int

	err = r.Descend(pack, func(i int, hash cipher.SHA256) (err error) {

		if i < 0 {
			t.Error("got negative index", i)
		} else if i >= len(users) {
			t.Error("got index out of rage", i)
		} else if users[i] != hash {
			t.Error("wrong hash, want %s, got %s",
				users[i].Hex()[:7],
				hash.Hex()[:7])
		}

		called++

		return // continue
	})

	if err != nil {
		t.Error(err)
	} else if called != len(users) {
		t.Errorf("wrong times called %d, but want %d", called, len(users))
	}

}

func TestRefs_Descend(t *testing.T) {
	// Descend(pack Pack, descendFunc IterateFunc) (err error)

	var (
		pack = testPack()

		refs Refs
		err  error

		users []cipher.SHA256

		clear = func(t *testing.T, r *Refs, degree int) {
			pack.ClearFlags(^0)   // clear flags of pack
			refs.Clear()          // clear the Refs making it Refs{}
			if degree != Degree { // if it's not default
				if err = refs.SetDegree(pack, degree); err != nil { // change it
					t.Fatal(err)
				}
			}
		}
	)

	for _, degree := range []int{
		Degree,     // default
		Degree + 7, // changed
	} {

		t.Run(fmt.Sprintf("blank (degree %d)", degree), func(t *testing.T) {

			clear(t, &refs, degree)

			var called int

			err = refs.Ascend(pack, func(int, cipher.SHA256) (_ error) {
				called++
				return
			})

			if err != nil {
				t.Error(err)
			} else if called > 0 {
				t.Errorf("called %d times, but should not be called at all",
					called)
			}

		})

		// fill
		//   (1) only leafs
		//   (2) leafs and branches
		//   (3) branches with branches with leafs

		for _, length := range []int{
			degree,            // only leafs
			degree + 1,        // leafs and branches
			degree*degree + 1, // branches with branches with leafs
		} {

			t.Logf("Refs with %d elements (degree %d)", length, degree)

			clear(t, &refs, degree)

			// generate users
			users = getHashList(
				testFillRefsWithUsers(t, &refs, pack, length),
			)

			if t.Failed() {
				t.FailNow()
			}

			logRefsTree(t, &refs, pack, false)

			t.Run(fmt.Sprintf("fresh %d:%d", length, degree),
				func(t *testing.T) {

					testRefsDescend(t, &refs, pack, users)
					logRefsTree(t, &refs, pack, false)

				})

			t.Run(fmt.Sprintf("break %d:%d", length, degree),
				func(t *testing.T) {

					var called int

					err = refs.Descend(pack, func(int, cipher.SHA256) (_ error) {
						called++
						return ErrStopIteration
					})

					if err != nil {
						t.Error(err)
					} else if called > 1 {
						t.Error("called too many times", called)
					}

					logRefsTree(t, &refs, pack, false)

				})

			t.Run(fmt.Sprintf("load %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset() // reset the refs

					testRefsDescend(t, &refs, pack, users)
					logRefsTree(t, &refs, pack, false)

				})

			t.Run(fmt.Sprintf("load entire %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset()              // reset the refs
					pack.AddFlags(EntireRefs) // load entire Refs

					testRefsDescend(t, &refs, pack, users)
					logRefsTree(t, &refs, pack, false)

				})

			t.Run(fmt.Sprintf("hash table index %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset()

					pack.ClearFlags(EntireRefs)
					pack.AddFlags(HashTableIndex)

					testRefsDescend(t, &refs, pack, users)
					logRefsTree(t, &refs, pack, false)

				})

		}

	}

	t.Run("blank hash", func(t *testing.T) {

		refs.Clear()
		pack.ClearFlags(^0) // all

		if err = refs.AppendHashes(pack, cipher.SHA256{}); err != nil {
			t.Fatal(err)
		}

		err = refs.Descend(pack, func(i int, hash cipher.SHA256) (err error) {
			if i != 0 {
				t.Error("wrong index given")
			}
			if hash != (cipher.SHA256{}) {
				t.Error("wrong hash given")
			}
			return
		})

	})

	// -------------------------------------------------------------------------

	// TODO (kostyaring): extended testing of the Descend

}

func TestRefs_DescendFrom(t *testing.T) {
	// DescendFrom(pack Pack, from int, descendFunc IterateFunc) (err error)

	//

}
