package registry

import (
	"fmt"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

func TestRefs_Append(t *testing.T) {
	// Append(pack Pack, refs *Refs) (err error)

	//

}

func TestRefs_AppendValues(t *testing.T) {
	// AppendValues(pack Pack, values ...interface{}) (err error)

	// TODO (kostyarin): the AppendValues method based on the AppendHashes
	//                   method and this test case is not important a lot,
	//                   thus I mark it as low priority

}

func testRefsAppendHashes(
	t *testing.T, //           :
	r *Refs, //                :
	pack Pack, //              :
	hashes []cipher.SHA256, // :
) (
	nl int, //                 : new length
) {

	var ln int
	var err error

	if ln, err = r.Len(pack); err != nil {
		t.Error(err)
		return
	}

	if err = r.AppendHashes(pack, hashes...); err != nil {
		t.Error(err)
		return
	}

	nl = testRefsAppendHashesCheck(t, r, pack, ln, hashes)
	return

}

func testRefsAppendHashesCheck(
	t *testing.T, //           : the testing
	r *Refs, //                : the Refs
	pack Pack, //              : the Pack
	shift int, //              : length of the Refs before appending
	hashes []cipher.SHA256, // : appended values
) (
	nl int, //                 : new length
) {

	var err error

	if nl, err = r.Len(pack); err != nil {
		t.Error(err)
		return
	}

	if nl != shift+len(hashes) {
		t.Error("wrong new length", nl, shift+len(hashes))
	}

	var h cipher.SHA256
	for i, hash := range hashes {
		if h, err = r.HashByIndex(pack, shift+i); err != nil {
			t.Error(err)
		} else if h != hash {
			t.Error("wrong hash of %d: %s, want %s",
				shift+i,
				h.Hex()[:7],
				hash.Hex()[:7])
		}
	}

	return

}

func TestRefs_AppendHashes(t *testing.T) {
	// AppendHashes(pack Pack, hashes ...cipher.SHA256) (err error)

	t.Skip("test HashByIndex first")

	var (
		pack = testPack()

		refs Refs
		ln   int // length of the Refs before appending
		err  error

		users []cipher.SHA256 // hashes of the users

		// clear and set degree
		clear = func(t *testing.T, r *Refs, degree int) {
			pack.ClearFlags(^0)   // clear all flags
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

		t.Run(fmt.Sprintf("append nothing to blank (degree %d)", degree),
			func(t *testing.T) {
				clear(t, &refs, degree)
				if ln = testRefsAppendHashes(t, &refs, pack, nil); t.Failed() {
					logRefsTree(t, &refs, pack, false)
					t.FailNow()
				}
			})

		// fill
		//   (1) only leafs
		//   (2) leafs and branches
		//   (3) branches with branches with leafs

		// so, since we adds this users twice, then we have to
		// redue the length to test only leafs and so on

		for _, length := range []int{
			degree,            // only leafs
			degree + 1,        // leafs and branches
			degree*degree + 1, // branches with branches with leafs
		} {

			t.Logf("Refs with %d elements (degree %d)", length, degree)

			clear(t, &refs, degree)

			// generate users
			users = getHashList(testUsers(length))

			if ln = testRefsAppendHashes(t, &refs, pack, users); t.Failed() {
				logRefsTree(t, &refs, pack, false)
				t.FailNow()
			}

			t.Run(fmt.Sprintf("load %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset() // reset the refs

					testRefsAppendHashesCheck(t, &refs, pack, ln, users)
					logRefsTree(t, &refs, pack, false)

				})

			t.Run(fmt.Sprintf("load entire %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset()              // reset the refs
					pack.AddFlags(EntireRefs) // load entire Refs

					testRefsAppendHashesCheck(t, &refs, pack, ln, users)
					logRefsTree(t, &refs, pack, false)

				})

			t.Run(fmt.Sprintf("hash table index %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset()

					pack.ClearFlags(EntireRefs)
					pack.AddFlags(HashTableIndex)

					testRefsAppendHashesCheck(t, &refs, pack, ln, users)
					logRefsTree(t, &refs, pack, false)

				})

		}

	}

	t.Run("blank hash", func(t *testing.T) {

		refs.Clear()
		pack.ClearFlags(^0) // all

		var hashes = []cipher.SHA256{
			{}, // the blank one
			{}, // the blank two
		}

		if err = refs.AppendHashes(pack, hashes...); err != nil {
			t.Fatal(err)
		}

		var is []int
		if is, err = refs.IndicesByHash(pack, cipher.SHA256{}); err != nil {
			t.Error(err)
		} else if len(is) != 2 {
			t.Errorf("wron number of indices: want 2, got %d", len(is))
		} else {
			for _, idx := range is {
				if (idx == 0 || idx == 1) == false {
					t.Error("got wrong index:", idx)
				}
			}
		}

	})

}
