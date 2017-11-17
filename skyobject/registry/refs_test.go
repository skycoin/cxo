package registry

import (
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

	leafs   []cipher.SHA256
	brances []*testNode
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

	if len(rn.branches) != len(tn.brances) {
		t.Errorf("wrong branches length: want %d, got %d", len(tn.brances),
			len(rn.branches))
	} else {
		for i, branch := range tn.brances {
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
		tn.brances = append(tn.brances, testNodeFromNode(br))
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

		if err = refs.Init(pack); err != nil {
			t.Fatal("can't initialize blank Refs:", err)
		}

		testRefsTest(t, &refs, &testRefs{
			testNode: testNode{mods: loadedMod},
			flags:    testNoMeaninFlag,
			degree:   Degree,
		})

		logRefsTree(t, &refs, pack, false)

	})

	// fill

	var users = testUsers(Degree + 1)

	if err = refs.AppendValues(pack, users...); err != nil {
		t.Fatal(err)
	}

	//logRefsTree(t, &refs, pack, false)

	var trFull = testRefsFromRefs(&refs) // keep the refs
	var trHead = testRefsFromRefs(&refs) // to cut

	for _, br := range trHead.brances {
		br.length = 0    // only
		br.mods = 0      // hash
		br.brances = nil // and
		br.leafs = nil   // upper
	}

	trFull.mods &^= originMod
	trHead.mods &^= originMod

	// load from pack (no flags)

	t.Run("load", func(t *testing.T) {

		refs.Reset() // reset the refs

		if err = refs.Init(pack); err != nil {
			t.Fatal(err)
		}

		testRefsTest(t, &refs, trHead)
		logRefsTree(t, &refs, pack, false)

	})

	t.Run("load entire", func(t *testing.T) {

		refs.Reset() // reset the refs

		pack.AddFlags(EntireRefs) // set the flag to load entire Refs

		if err = refs.Init(pack); err != nil {
			t.Fatal(err)
		}

		trFull.flags |= EntireRefs

		testRefsTest(t, &refs, trFull)
		logRefsTree(t, &refs, pack, false)

		for _, br := range refs.branches {
			t.Log(br.hash.Hex())
			t.Log(br.length)
			t.Logf("%#v", br.leafs)
		}

	})

	_ = pretty.Print

}

func TestRefs_Len(t *testing.T) {
	// Len(pack Pack) (ln int, err error)

	//

}

func TestRefs_Depth(t *testing.T) {
	// Depth(pack Pack) (depth int, err error)

	//

}

func TestRefs_Degree(t *testing.T) {
	// Degree(pack Pack) (degree int, err error)

	//

}

func TestRefs_Flags(t *testing.T) {
	// Flags() (flags Flags)

	//

}

func TestRefs_Reset(t *testing.T) {
	// Reset() (err error)

	//

}

func TestRefs_HasHash(t *testing.T) {
	// HasHsah(pack Pack, hash cipher.SHA256) (ok bool, err error)

	//

}

func TestRefs_ValueByHash(t *testing.T) {
	// ValueByHash(pack Pack, hash cipher.SHA256, obj interface{}) (err error)

	//

}

func TestRefs_IndexOfHash(t *testing.T) {
	// IndexOfHash(pack Pack, hash cipher.SHA256) (i int, err error)

	//

}

func TestRefs_IndicesByHash(t *testing.T) {
	// IndicesByHash(pack Pack, hash cipher.SHA256) (is []int, err error)

	//

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

func TestRefs_Ascend(t *testing.T) {
	// Ascend(pack Pack, ascendFunc IterateFunc) (err error)

	//

}

func TestRefs_AscendFrom(t *testing.T) {
	// AscendFrom(pack Pack, from int, ascendFunc IterateFunc) (err error)

	//
}

func TestRefs_Descend(t *testing.T) {
	// Descend(pack Pack, descendFunc IterateFunc) (err error)

	//

}

func TestRefs_DescendFrom(t *testing.T) {
	// DescendFrom(pack Pack, from int, descendFunc IterateFunc) (err error)

	//

}

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
