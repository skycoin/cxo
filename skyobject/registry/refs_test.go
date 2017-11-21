package registry

import (
	"fmt"
	"testing"

	"github.com/kr/pretty"

	"github.com/skycoin/skycoin/src/cipher"
)

const (
	testNoMeaninFlag Flags = 1 << 7
)

type testRefs struct {
	hash cipher.SHA256

	flags Flags

	depth  int
	degree int

	testNode
}

type testNode struct {
	hash cipher.SHA256

	mods   refsMod
	length int

	leafs    []cipher.SHA256
	branches []*testNode
}

func testRefsTest(t *testing.T, r *Refs, tr *testRefs) {

	if r.flags != tr.flags {
		t.Errorf("wrong flags: want %08b, got %08b", tr.flags, r.flags)
	}

	if r.depth != tr.depth {
		t.Errorf("wrong depth: want %d, got: %d", tr.depth, r.depth)
	}

	if r.degree != tr.degree {
		t.Errorf("wrong degree: want %d, got: %d", tr.degree, r.degree)
	}

	if r.Hash != tr.hash {
		t.Errorf("wrong hash: want %s, got %s", tr.hash.Hex()[:7], r.String())
	}

	testNodeTest(t, r, &r.refsNode, &tr.testNode)

}

func testNodeTest(t *testing.T, r *Refs, rn *refsNode, tn *testNode) {

	if rn.hash != tn.hash {
		t.Errorf("wrong node hash: want %s, got %s", tn.hash.Hex()[:7],
			rn.hash.Hex()[:7])
	}

	if rn.mods != tn.mods {
		t.Errorf("wrong mods: want %08b, got %08b", tn.mods, rn.mods)
	}

	if rn.length != tn.length {
		t.Errorf("wrong length: want %d, got: %d", tn.length, rn.length)
	}

	if &r.refsNode != rn && rn.upper == nil {
		t.Errorf("missing upper reference")
	}

	if len(rn.leafs) != len(tn.leafs) {
		t.Errorf("wrong leafs length: want %d, got %d", len(tn.leafs),
			len(rn.leafs))
	} else {
		for i, leaf := range tn.leafs {
			var el = rn.leafs[i]
			if el.Hash != leaf {
				t.Errorf("wrong leaf %d", i)
			}
			if r.flags&HashTableIndex != 0 {
				if res, ok := r.refsIndex[leaf]; !ok {
					t.Errorf("missing in hash table index")
				} else {
					for _, re := range res {
						if re == el {
							goto leafInIndexFound
						}
					}
					t.Errorf("missing element in index")
				leafInIndexFound:
				}
			}
			if el.upper != rn {
				t.Errorf("wron upper of leaf")
			}
		}
	}

	if len(rn.branches) != len(tn.branches) {
		t.Errorf("wrong branches length: want %d, got %d", len(tn.branches),
			len(rn.branches))
	} else {
		for i, branch := range tn.branches {
			testNodeTest(t, r, rn.branches[i], branch)
			if rn.branches[i].upper != rn {
				t.Errorf("wrong upper reference of the node")
			}
		}
	}

}

func testRefsFromRefs(r *Refs) (tr *testRefs) {
	tr = new(testRefs)
	tr.hash = r.Hash
	tr.flags = r.flags
	tr.depth = r.depth
	tr.degree = r.degree
	tr.testNode = *testNodeFromNode(&r.refsNode)
	return
}

func testNodeFromNode(rn *refsNode) (tn *testNode) {
	tn = new(testNode)
	tn.hash = rn.hash
	tn.mods = rn.mods
	tn.length = rn.length

	for _, el := range rn.leafs {
		tn.leafs = append(tn.leafs, el.Hash)
	}

	for _, br := range rn.branches {
		tn.branches = append(tn.branches, testNodeFromNode(br))
	}

	return
}

func logRefsTree(t *testing.T, r *Refs, pack Pack, forceLoad bool) {

	var tree string
	var err error

	if tree, err = r.Tree(pack, forceLoad); err != nil {
		t.Error(err)
	}

	t.Log(tree)

}

func testFillRefsWithUsers(
	t *testing.T, //        : the testing
	r *Refs, //             : the Refs to fill
	pack Pack, //           : pack to save the users in
	n int, //               : number of users
) (
	users []interface{}, // : the users
) {

	var err error

	users = testUsers(n)

	if err = r.AppendValues(pack, users...); err != nil {
		t.Error(err)
	}

	return
}

func TestRefs_String(t *testing.T) {
	// String() string

	// TODO (kostyarin): lowest priority

}

func TestRefs_Short(t *testing.T) {
	// Short() string

	// TODO (kostyarin): lowest priority

}

func TestRefs_Init(t *testing.T) {
	// Init(pack Pack) (err error)

	var (
		pack = testPack()

		refs Refs
		err  error
	)

	pack.AddFlags(testNoMeaninFlag)

	t.Run("blank", func(t *testing.T) {

		// (1) load and (2) use already loaded Refs
		for i := 0; i < 2; i++ {

			if err = refs.Init(pack); err != nil {
				t.Fatal("can't initialize blank Refs:", err)
			}

		}

		testRefsTest(t, &refs, &testRefs{
			testNode: testNode{mods: loadedMod},
			flags:    testNoMeaninFlag,
			degree:   Degree,
		})

		logRefsTree(t, &refs, pack, false)

	})

	// fill
	//   (1) only leafs
	//   (2) leafs and branches
	//   (3) branches with branches with leafs

	for _, length := range []int{
		Degree,            // only leafs
		Degree + 1,        // leafs and branches
		Degree*Degree + 1, // branches with branches with leafs
	} {

		t.Logf("Refs with %d elements", length)

		refs.Clear()

		pack.ClearFlags(^0)
		pack.AddFlags(testNoMeaninFlag)

		if testFillRefsWithUsers(t, &refs, pack, length); t.Failed() {
			t.FailNow()
		}

		logRefsTree(t, &refs, pack, false)

		var trFull = testRefsFromRefs(&refs) // keep the refs
		var trHead = testRefsFromRefs(&refs) // to cut

		for _, br := range trHead.branches {
			br.length = 0     // only
			br.mods = 0       // hash
			br.branches = nil // and
			br.leafs = nil    // upper
		}

		trFull.mods &^= originMod
		trHead.mods &^= originMod

		// to be sure
		// ----

		for _, tr := range []*testRefs{
			trFull,
			trHead,
		} {

			tr.length = length
			tr.mods = loadedMod
			tr.flags = pack.Flags()
			tr.degree = Degree

		}

		// ----

		// load from pack (no flags)

		t.Run(fmt.Sprintf("load %d", length), func(t *testing.T) {

			refs.Reset() // reset the refs

			// (1) load and (2) use already loaded Refs
			for i := 0; i < 2; i++ {

				if err = refs.Init(pack); err != nil {
					t.Fatal(err)
				}

			}

			testRefsTest(t, &refs, trHead)
			logRefsTree(t, &refs, pack, false)

		})

		t.Run(fmt.Sprintf("load entire %d", length), func(t *testing.T) {

			refs.Reset() // reset the refs

			pack.AddFlags(EntireRefs) // set the flag to load entire Refs

			// (1) load and (2) use already loaded Refs
			for i := 0; i < 2; i++ {

				if err = refs.Init(pack); err != nil {
					t.Fatal(err)
				}

			}

			trFull.flags |= EntireRefs

			testRefsTest(t, &refs, trFull)
			logRefsTree(t, &refs, pack, false)

		})

		t.Run(fmt.Sprintf("hash table index %d", length), func(t *testing.T) {

			refs.Reset()

			pack.ClearFlags(EntireRefs)
			pack.AddFlags(HashTableIndex)

			// (1) load and (2) use already loaded Refs
			for i := 0; i < 2; i++ {

				if err = refs.Init(pack); err != nil {
					t.Fatal(err)
				}

			}

			trFull.flags &^= EntireRefs
			trFull.flags |= HashTableIndex

			testRefsTest(t, &refs, trFull)
			logRefsTree(t, &refs, pack, false)

		})

	}

	_ = pretty.Print

}

func TestRefs_Len(t *testing.T) {
	// Len(pack Pack) (ln int, err error)

	var (
		pack = testPack()

		refs Refs

		ln  int
		err error
	)

	pack.AddFlags(testNoMeaninFlag)

	t.Run("blank", func(t *testing.T) {

		// (1) load and (2) use already loaded Refs
		for i := 0; i < 2; i++ {

			if ln, err = refs.Len(pack); err != nil {
				t.Fatal("can't initialize blank Refs:", err)
			} else if ln != 0 {
				t.Errorf("wrong length, want 0, got %d", ln)
			}

		}

		testRefsTest(t, &refs, &testRefs{
			testNode: testNode{mods: loadedMod},
			flags:    testNoMeaninFlag,
			degree:   Degree,
		})

		logRefsTree(t, &refs, pack, false)

	})

	// fill
	//   (1) only leafs
	//   (2) leafs and branches
	//   (3) branches with branches with leafs

	for _, length := range []int{
		Degree,            // only leafs
		Degree + 1,        // leafs and branches
		Degree*Degree + 1, // branches with branches with leafs
	} {

		t.Logf("Refs with %d elements", length)

		refs.Clear()

		pack.ClearFlags(^0)
		pack.AddFlags(testNoMeaninFlag)

		if testFillRefsWithUsers(t, &refs, pack, length); t.Failed() {
			t.FailNow()
		}

		logRefsTree(t, &refs, pack, false)

		var trFull = testRefsFromRefs(&refs) // keep the refs
		var trHead = testRefsFromRefs(&refs) // to cut

		for _, br := range trHead.branches {
			br.length = 0     // only
			br.mods = 0       // hash
			br.branches = nil // and
			br.leafs = nil    // upper
		}

		trFull.mods &^= originMod
		trHead.mods &^= originMod

		// to be sure
		// ----

		for _, tr := range []*testRefs{
			trFull,
			trHead,
		} {

			tr.length = length
			tr.mods = loadedMod
			tr.flags = pack.Flags()
			tr.degree = Degree

		}

		// ----

		// load from pack (no flags)

		t.Run(fmt.Sprintf("load %d", length), func(t *testing.T) {

			refs.Reset() // reset the refs

			// (1) laod and (2) use already loaded Refs
			for i := 0; i < 2; i++ {

				if ln, err = refs.Len(pack); err != nil {
					t.Fatal(err)
				} else if ln != length {
					t.Error("wrong length: want %d, got %d", length, ln)
				}

			}

			testRefsTest(t, &refs, trHead)
			logRefsTree(t, &refs, pack, false)

		})

		t.Run(fmt.Sprintf("load entire %d", length), func(t *testing.T) {

			refs.Reset() // reset the refs

			pack.AddFlags(EntireRefs) // set the flag to load entire Refs

			// (1) laod and (2) use already loaded Refs
			for i := 0; i < 2; i++ {

				if ln, err = refs.Len(pack); err != nil {
					t.Fatal(err)
				} else if ln != length {
					t.Error("wrong length: want %d, got %d", length, ln)
				}

			}

			trFull.flags |= EntireRefs

			testRefsTest(t, &refs, trFull)
			logRefsTree(t, &refs, pack, false)

		})

		t.Run(fmt.Sprintf("hash table index %d", length), func(t *testing.T) {

			refs.Reset()

			pack.ClearFlags(EntireRefs)
			pack.AddFlags(HashTableIndex)

			// (1) laod and (2) use already loaded Refs
			for i := 0; i < 2; i++ {

				if ln, err = refs.Len(pack); err != nil {
					t.Fatal(err)
				} else if ln != length {
					t.Error("wrong length: want %d, got %d", length, ln)
				}

			}

			trFull.flags &^= EntireRefs
			trFull.flags |= HashTableIndex

			testRefsTest(t, &refs, trFull)
			logRefsTree(t, &refs, pack, false)

		})

	}

}

func TestRefs_Depth(t *testing.T) {
	// Depth(pack Pack) (depth int, err error)

	var (
		pack = testPack()

		refs Refs

		dp  int // depth
		err error
	)

	pack.AddFlags(testNoMeaninFlag)

	t.Run("blank", func(t *testing.T) {

		// (1) load and (2) use already loaded Refs
		for i := 0; i < 2; i++ {

			if dp, err = refs.Depth(pack); err != nil {
				t.Fatal("can't initialize blank Refs:", err)
			} else if dp-1 != 0 {
				t.Errorf("wrong depth, want 0, got %d", dp)
			}

		}

		testRefsTest(t, &refs, &testRefs{
			testNode: testNode{mods: loadedMod},
			flags:    testNoMeaninFlag,
			degree:   Degree,
		})

		logRefsTree(t, &refs, pack, false)

	})

	// fill
	//   (1) only leafs
	//   (2) leafs and branches
	//   (3) branches with branches with leafs

	for _, length := range []int{
		Degree,            // only leafs
		Degree + 1,        // leafs and branches
		Degree*Degree + 1, // branches with branches with leafs
	} {

		var depth = depthToFit(Degree, 0, length)

		t.Logf("Refs with %d elements (depth %d)", length, depth)

		refs.Clear()

		pack.ClearFlags(^0)
		pack.AddFlags(testNoMeaninFlag)

		if testFillRefsWithUsers(t, &refs, pack, length); t.Failed() {
			t.FailNow()
		}

		logRefsTree(t, &refs, pack, false)

		var trFull = testRefsFromRefs(&refs) // keep the refs
		var trHead = testRefsFromRefs(&refs) // to cut

		for _, br := range trHead.branches {
			br.length = 0     // only
			br.mods = 0       // hash
			br.branches = nil // and
			br.leafs = nil    // upper
		}

		trFull.mods &^= originMod
		trHead.mods &^= originMod

		// to be sure
		// ----

		for _, tr := range []*testRefs{
			trFull,
			trHead,
		} {

			tr.length = length
			tr.mods = loadedMod
			tr.flags = pack.Flags()
			tr.degree = Degree

		}

		// ----

		// load from pack (no flags)

		t.Run(fmt.Sprintf("load %d", length), func(t *testing.T) {

			refs.Reset() // reset the refs

			// (1) laod and (2) use already loaded Refs
			for i := 0; i < 2; i++ {

				if dp, err = refs.Depth(pack); err != nil {
					t.Fatal(err)
				} else if dp-1 != depth {
					t.Error("wrong depth: want %d, got %d", depth, dp)
				}

			}

			testRefsTest(t, &refs, trHead)
			logRefsTree(t, &refs, pack, false)

		})

		t.Run(fmt.Sprintf("load entire %d", length), func(t *testing.T) {

			refs.Reset() // reset the refs

			pack.AddFlags(EntireRefs) // set the flag to load entire Refs

			// (1) laod and (2) use already loaded Refs
			for i := 0; i < 2; i++ {

				if dp, err = refs.Depth(pack); err != nil {
					t.Fatal(err)
				} else if dp-1 != depth {
					t.Error("wrong depth: want %d, got %d", depth, dp)
				}

			}

			trFull.flags |= EntireRefs

			testRefsTest(t, &refs, trFull)
			logRefsTree(t, &refs, pack, false)

		})

		t.Run(fmt.Sprintf("hash table index %d", length), func(t *testing.T) {

			refs.Reset()

			pack.ClearFlags(EntireRefs)
			pack.AddFlags(HashTableIndex)

			// (1) laod and (2) use already loaded Refs
			for i := 0; i < 2; i++ {

				if dp, err = refs.Depth(pack); err != nil {
					t.Fatal(err)
				} else if dp-1 != depth {
					t.Error("wrong depth: want %d, got %d", depth, dp)
				}

			}

			trFull.flags &^= EntireRefs
			trFull.flags |= HashTableIndex

			testRefsTest(t, &refs, trFull)
			logRefsTree(t, &refs, pack, false)

		})

	}

}

func TestRefs_Degree(t *testing.T) {
	// Degree(pack Pack) (degree int, err error)

	// degree saved only if the Refs is not blank

	var (
		pack = testPack()

		refs Refs

		dg  int // degree
		err error

		clear = func(t *testing.T, r *Refs, degree int) {
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

			pack.ClearFlags(^0)
			pack.AddFlags(testNoMeaninFlag)

			clear(t, &refs, degree)

			if dg, err = refs.Degree(pack); err != nil {
				t.Fatal("can't initialize blank Refs:", err)
			} else if dg != degree {
				t.Errorf("wrong degree, want %d, got %d", degree, dg)
			}

			testRefsTest(t, &refs, &testRefs{
				testNode: testNode{mods: loadedMod},
				flags:    testNoMeaninFlag,
				degree:   degree,
			})

			logRefsTree(t, &refs, pack, false)

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

			pack.ClearFlags(^0)
			pack.AddFlags(testNoMeaninFlag)

			clear(t, &refs, degree)

			if testFillRefsWithUsers(t, &refs, pack, length); t.Failed() {
				t.FailNow()
			}

			logRefsTree(t, &refs, pack, false)

			var trFull = testRefsFromRefs(&refs) // keep the refs
			var trHead = testRefsFromRefs(&refs) // to cut

			for _, br := range trHead.branches {
				br.length = 0     // only
				br.mods = 0       // hash
				br.branches = nil // and
				br.leafs = nil    // upper
			}

			trFull.mods &^= originMod
			trHead.mods &^= originMod

			// to be sure
			// ----

			for _, tr := range []*testRefs{
				trFull,
				trHead,
			} {

				tr.length = length
				tr.mods = loadedMod
				tr.flags = pack.Flags()
				tr.degree = degree

			}

			// ----

			// load from pack (no flags)

			t.Run(fmt.Sprintf("load %d", length), func(t *testing.T) {

				refs.Reset() // reset the refs

				// (1) laod and (2) use already loaded Refs
				for i := 0; i < 2; i++ {

					if dg, err = refs.Degree(pack); err != nil {
						t.Fatal(err)
					} else if dg != degree {
						t.Error("wrong degree: want %d, got %d", degree, dg)
					}

				}

				testRefsTest(t, &refs, trHead)
				logRefsTree(t, &refs, pack, false)

			})

			t.Run(fmt.Sprintf("load entire %d", length), func(t *testing.T) {

				refs.Reset() // reset the refs

				pack.AddFlags(EntireRefs) // set the flag to load entire Refs

				// (1) laod and (2) use already loaded Refs
				for i := 0; i < 2; i++ {

					if dg, err = refs.Degree(pack); err != nil {
						t.Fatal(err)
					} else if dg != degree {
						t.Error("wrong degree: want %d, got %d", degree, dg)
					}

				}

				trFull.flags |= EntireRefs

				testRefsTest(t, &refs, trFull)
				logRefsTree(t, &refs, pack, false)

			})

			t.Run(fmt.Sprintf("hash table index %d", length),
				func(t *testing.T) {

					refs.Reset()

					pack.ClearFlags(EntireRefs)
					pack.AddFlags(HashTableIndex)

					// (1) laod and (2) use already loaded Refs
					for i := 0; i < 2; i++ {

						if dg, err = refs.Degree(pack); err != nil {
							t.Fatal(err)
						} else if dg != degree {
							t.Error("wrong degree: want %d, got %d", degree, dg)
						}

					}

					trFull.flags &^= EntireRefs
					trFull.flags |= HashTableIndex

					testRefsTest(t, &refs, trFull)
					logRefsTree(t, &refs, pack, false)

				})

		}

	}

}

func TestRefs_Flags(t *testing.T) {
	// Flags() (flags Flags)

	// flags are not saved in DB

	// So, the Flags tested inside another tests
	// let's mark this test case low priority

	// TODO (kostyarin): low priority

}

func TestRefs_Reset(t *testing.T) {
	// Reset() (err error)

	// TODO (kostyarin): lowest priority

}

func getHashList(users []interface{}) (has []cipher.SHA256) {

	has = make([]cipher.SHA256, 0, len(users))

	for _, user := range users {
		has = append(has, getHash(user))
	}

	return

}

func testRefsHasHash(
	t *testing.T, //        : the testing
	r *Refs, //             : the Refs to test
	pack Pack, //           : the pack
	not cipher.SHA256, //   : has not this hash
	has []cipher.SHA256, // : has this hashes
) {

	var (
		ok  bool
		err error
	)

	// check the "not" first

	// (1) init and (2) use initialized Refs
	for i := 0; i < 2; i++ {

		if ok, err = r.HasHash(pack, not); err != nil {
			t.Error(err)
		} else if ok == true {
			t.Error("the Refs has hash that it should not have")
		}

	}

	// check all users

	for _, hash := range has {

		if ok, err = r.HasHash(pack, hash); err != nil {
			t.Error(err)
		} else if ok == false {
			t.Error("missing hash:", hash.Hex()[:7])
		}

	}

}

func TestRefs_HasHash(t *testing.T) {
	// HasHsah(pack Pack, hash cipher.SHA256) (ok bool, err error)

	var (
		pack = testPack()

		refs Refs
		err  error

		has []cipher.SHA256 // the users
		not = cipher.SumSHA256([]byte("any Refs doesn't contain this hash"))

		users []interface{}

		clear = func(t *testing.T, r *Refs, degree int) {
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

			pack.ClearFlags(^0)
			pack.AddFlags(testNoMeaninFlag)

			clear(t, &refs, degree)
			testRefsHasHash(t, &refs, pack, not, nil)

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

			pack.ClearFlags(^0)
			pack.AddFlags(testNoMeaninFlag)

			clear(t, &refs, degree)

			// generate users
			users = testFillRefsWithUsers(t, &refs, pack, length)

			if t.Failed() {
				t.FailNow()
			}

			logRefsTree(t, &refs, pack, false)

			has = getHashList(users)

			t.Run(fmt.Sprintf("load %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset() // reset the refs

					testRefsHasHash(t, &refs, pack, not, has)
					logRefsTree(t, &refs, pack, false)

				})

			t.Run(fmt.Sprintf("load entire %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset()              // reset the refs
					pack.AddFlags(EntireRefs) // load entire Refs

					testRefsHasHash(t, &refs, pack, not, has)
					logRefsTree(t, &refs, pack, false)

				})

			t.Run(fmt.Sprintf("hash table index %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset()

					pack.ClearFlags(EntireRefs)
					pack.AddFlags(HashTableIndex)

					testRefsHasHash(t, &refs, pack, not, has)
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

		var ok bool
		if ok, err = refs.HasHash(pack, cipher.SHA256{}); err != nil {
			t.Error(err)
		} else if ok == false {
			t.Error("missing blank hash")
		}

	})

}

// don't compare Hidden field, only Name and Age
func testUserEq(u1, u2 *TestUser) (equal bool) {
	return u1.Name == u2.Name && u1.Age == u2.Age
}

func testRefsValueByHash(
	t *testing.T, //         : the testing
	r *Refs, //              : the Refs to test
	pack Pack, //            : the pack
	not cipher.SHA256, //    : has not this hash
	has []cipher.SHA256, //  : has this hashes
	valuse []interface{}, // : expected values
) {

	var (
		usr TestUser
		err error
	)

	// check the "not" first

	// (1) init and (2) use initialized Refs
	for i := 0; i < 2; i++ {

		if err = r.ValueByHash(pack, not, &usr); err == nil {
			t.Error("missing error")
		} else if err != ErrNotFound {
			t.Errorf("wrong error: want ErrNotFound, got %q", err)
		}

	}

	// check all users

	for i, hash := range has {

		if err = r.ValueByHash(pack, hash, &usr); err != nil {
			t.Error(err)
		} else {

			var want = valuse[i].(TestUser)

			if testUserEq(&usr, &want) == false {
				t.Error("got wrong value")
			}
		}

	}

}

func TestRefs_ValueByHash(t *testing.T) {
	// ValueByHash(pack Pack, hash cipher.SHA256, obj interface{}) (err error)

	var (
		pack = testPack()

		refs Refs
		err  error

		has []cipher.SHA256 // the users
		not = cipher.SumSHA256([]byte("any Refs doesn't contain this hash"))

		users []interface{}

		clear = func(t *testing.T, r *Refs, degree int) {
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

			pack.ClearFlags(^0)
			pack.AddFlags(testNoMeaninFlag)

			clear(t, &refs, degree)
			testRefsValueByHash(t, &refs, pack, not, nil, nil)

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

			pack.ClearFlags(^0)
			pack.AddFlags(testNoMeaninFlag)

			clear(t, &refs, degree)

			// generate users
			users = testFillRefsWithUsers(t, &refs, pack, length)

			if t.Failed() {
				t.FailNow()
			}

			logRefsTree(t, &refs, pack, false)

			has = getHashList(users)

			t.Run(fmt.Sprintf("load %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset() // reset the refs

					testRefsValueByHash(t, &refs, pack, not, has, users)
					logRefsTree(t, &refs, pack, false)

				})

			t.Run(fmt.Sprintf("load entire %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset()              // reset the refs
					pack.AddFlags(EntireRefs) // load entire Refs

					testRefsValueByHash(t, &refs, pack, not, has, users)
					logRefsTree(t, &refs, pack, false)

				})

			t.Run(fmt.Sprintf("hash table index %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset()

					pack.ClearFlags(EntireRefs)
					pack.AddFlags(HashTableIndex)

					testRefsValueByHash(t, &refs, pack, not, has, users)
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

		var usr TestUser
		if err = refs.ValueByHash(pack, cipher.SHA256{}, &usr); err == nil {
			t.Error("missing error")
		} else if err != ErrRefsElementIsNil {
			t.Errorf("wrong error: want ErrRefsElementIsNil, got %q", err)
		}

	})

}

func testRefsIndexOfHash(
	t *testing.T, //         : the testing
	r *Refs, //              : the Refs to test
	pack Pack, //            : the pack
	not cipher.SHA256, //    : has not this hash
	has []cipher.SHA256, //  : has this hashes
) {

	var (
		idx int
		err error
	)

	// check the "not" first

	// (1) init and (2) use initialized Refs
	for i := 0; i < 2; i++ {

		if _, err = r.IndexOfHash(pack, not); err == nil {
			t.Error("missing error")
		} else if err != ErrNotFound {
			t.Errorf("wrong error: want ErrNotFound, got %q", err)
		}

	}

	// check all users

	for i, hash := range has {

		if idx, err = r.IndexOfHash(pack, hash); err != nil {
			t.Error(err)
		} else if idx < 0 || idx >= len(has) {
			t.Error("got index out of range")
		} else if has[idx] != hash {
			t.Errorf("wrong index, want %d, got %d", i, idx)
		}

	}

}

func TestRefs_IndexOfHash(t *testing.T) {
	// IndexOfHash(pack Pack, hash cipher.SHA256) (i int, err error)

	var (
		pack = testPack()

		refs Refs
		err  error

		has []cipher.SHA256 // the users
		not = cipher.SumSHA256([]byte("any Refs doesn't contain this hash"))

		users []interface{}

		clear = func(t *testing.T, r *Refs, degree int) {
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

			pack.ClearFlags(^0)
			pack.AddFlags(testNoMeaninFlag)

			clear(t, &refs, degree)
			testRefsIndexOfHash(t, &refs, pack, not, nil)

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

			pack.ClearFlags(^0)
			pack.AddFlags(testNoMeaninFlag)

			clear(t, &refs, degree)

			// generate users
			users = testFillRefsWithUsers(t, &refs, pack, length)

			if t.Failed() {
				t.FailNow()
			}

			logRefsTree(t, &refs, pack, false)

			has = getHashList(users)

			t.Run(fmt.Sprintf("load %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset() // reset the refs

					testRefsIndexOfHash(t, &refs, pack, not, has)
					logRefsTree(t, &refs, pack, false)

				})

			t.Run(fmt.Sprintf("load entire %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset()              // reset the refs
					pack.AddFlags(EntireRefs) // load entire Refs

					testRefsIndexOfHash(t, &refs, pack, not, has)
					logRefsTree(t, &refs, pack, false)

				})

			t.Run(fmt.Sprintf("hash table index %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset()

					pack.ClearFlags(EntireRefs)
					pack.AddFlags(HashTableIndex)

					testRefsIndexOfHash(t, &refs, pack, not, has)
					logRefsTree(t, &refs, pack, false)

				})

		}

	}

	t.Run("blank hash", func(t *testing.T) {

		refs.Clear()
		pack.ClearFlags(^0) // all

		var hashes = []cipher.SHA256{
			{1, 2, 3}, //
			{4, 5, 6}, //
			{},        // the blank one
		}

		if err = refs.AppendHashes(pack, hashes...); err != nil {
			t.Fatal(err)
		}

		var idx int
		if idx, err = refs.IndexOfHash(pack, cipher.SHA256{}); err != nil {
			t.Error(err)
		} else if idx != len(hashes)-1 {
			t.Errorf("wrong index of blank hash: want %d, got %d",
				len(hashes)-1, idx)
		}

	})

}

func testRefsIndicesByHash(
	t *testing.T, //         : the testing
	r *Refs, //              : the Refs to test
	pack Pack, //            : the pack
	not cipher.SHA256, //    : has not this hash
	has []cipher.SHA256, //  : has this hashes
) {

	var (
		is  []int
		err error
	)

	// check the "not" first

	// (1) init and (2) use initialized Refs
	for i := 0; i < 2; i++ {

		if _, err = r.IndicesByHash(pack, not); err == nil {
			t.Error("missing error")
		} else if err != ErrNotFound {
			t.Errorf("wrong error: want ErrNotFound, got %q", err)
		}

	}

	// check all users

	for _, hash := range has {

		if is, err = r.IndicesByHash(pack, hash); err != nil {
			t.Error(err)
		} else if len(is) != 2 {
			t.Errorf("wrong number of indices: want 2, got %d", len(is))
		} else {
			for _, idx := range is {
				if idx >= len(has) {
					idx -= len(has)
				}
				if has[idx] != hash {
					t.Error("got wrong index", idx)
				}
			}
		}

	}

}

// generate once, append twice
func testFillRefsWithUsersTwice(
	t *testing.T,
	r *Refs,
	pack Pack,
	n int,
) (
	users []interface{},
) {

	users = testFillRefsWithUsers(t, r, pack, n)

	if err := r.AppendValues(pack, users...); err != nil {
		t.Error(err)
	}

	return

}

func logRefsIndex(t *testing.T, r *Refs) {
	for hash, res := range r.refsIndex {
		t.Logf("{%s: %d}", hash.Hex()[:7], len(res))
	}
}

func TestRefs_IndicesByHash(t *testing.T) {
	// IndicesByHash(pack Pack, hash cipher.SHA256) (is []int, err error)

	t.Skip("test Append first")

	var (
		pack = testPack()

		refs Refs
		err  error

		has []cipher.SHA256 // the users
		not = cipher.SumSHA256([]byte("any Refs doesn't contain this hash"))

		users []interface{}

		clear = func(t *testing.T, r *Refs, degree int) {
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

			pack.ClearFlags(^0)
			pack.AddFlags(testNoMeaninFlag)

			clear(t, &refs, degree)
			testRefsIndicesByHash(t, &refs, pack, not, nil)

		})

		// fill
		//   (1) only leafs
		//   (2) leafs and branches
		//   (3) branches with branches with leafs

		// so, since we adds this users twice, then we have to
		// redue the length to test only leafs and so on

		for _, length := range []int{
			degree / 2,        // only leafs
			degree,            // leafs and branches
			degree*degree + 1, // branches with branches with leafs
		} {

			t.Logf("Refs with %d elements (degree %d)", length, degree)

			pack.ClearFlags(^0)
			pack.AddFlags(testNoMeaninFlag)

			clear(t, &refs, degree)

			// generate users
			users = testFillRefsWithUsersTwice(t, &refs, pack, length)

			if t.Failed() {
				t.FailNow()
			}

			if refs.length != length*2 {
				t.Fatal("WRONG LENGTH", length, refs.length)
			}

			logRefsTree(t, &refs, pack, false)
			logRefsIndex(t, &refs)

			has = getHashList(users)

			t.Run(fmt.Sprintf("load %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset() // reset the refs

					testRefsIndicesByHash(t, &refs, pack, not, has)
					logRefsTree(t, &refs, pack, false)
					logRefsIndex(t, &refs)

				})

			t.Run(fmt.Sprintf("load entire %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset()              // reset the refs
					pack.AddFlags(EntireRefs) // load entire Refs

					testRefsIndicesByHash(t, &refs, pack, not, has)
					logRefsTree(t, &refs, pack, false)
					logRefsIndex(t, &refs)

				})

			t.Run(fmt.Sprintf("hash table index %d:%d", length, degree),
				func(t *testing.T) {

					refs.Reset()

					pack.ClearFlags(EntireRefs)
					pack.AddFlags(HashTableIndex)

					testRefsIndicesByHash(t, &refs, pack, not, has)
					logRefsTree(t, &refs, pack, false)
					logRefsIndex(t, &refs)

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

func TestRefs_ValueOfHashWithIndex(t *testing.T) {
	// ValueOfHashWithIndex(pack Pack, hash cipher.SHA256,
	//     obj interface{}) (i int, err error)

	//

}

func TestRefs_HashByIndex(t *testing.T) {
	// HashByIndex(pack Pack, i int) (hash cipher.SHA256, err error)

	//

}

func TestRefs_ValueByIndex(t *testing.T) {
	// ValueByIndex(pack Pack, i int, obj interface{}) (hash cipher.SHA256,
	//     err error)

	//

}

func TestRefs_SetHashByIndex(t *testing.T) {
	// SetHashByIndex(pack Pack, i int, hash cipher.SHA256) (err error)

	//

}

func TestRefs_SetValueByIndex(t *testing.T) {
	// SetValueByIndex(pack Pack, i int, obj interface{}) (err error)

	//

}

func TestRefs_DeleteByIndex(t *testing.T) {
	// DeleteByIndex(pack Pack, i int) (err error)

	//

}

func TestRefs_DeleteByHash(t *testing.T) {
	// DeleteByHash(pack Pack, hash cipher.SHA256) (err error)

	//

}

// see 'refs_iterate_test.go' for
//  - Ascend
//  - AscendFrom
//  - Descend
//  - DescendFrom

func TestRefs_Slice(t *testing.T) {
	// Slice(pack Pack, i int, j int) (slice *Refs, err error)

	//

}

func TestRefs_Append(t *testing.T) {
	// Append(pack Pack, refs *Refs) (err error)

	//

}

func TestRefs_AppendValues(t *testing.T) {
	// AppendValues(pack Pack, values ...interface{}) (err error)

	//

}

func TestRefs_AppendHashes(t *testing.T) {
	// AppendHashes(pack Pack, hashes ...cipher.SHA256) (err error)

	//

}

func TestRefs_Clear(t *testing.T) {
	// Clear()

	//

}

func TestRefs_Rebuild(t *testing.T) {
	// Rebuild(pack Pack) (err error)

	//

}

func TestRefs_Tree(t *testing.T) {
	// Tree() (tree string)

	//

}
