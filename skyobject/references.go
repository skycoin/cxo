package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

const (
	RefsDegree int32 = 16 // default degree of References' Merkle tree
)

// A Changer represents a changable object
type Changer interface {
	Cahgne()         // mark as changed
	IsChanged() bool // true if has been changed
}

type changer struct {
	isChanged
	upper Changer
}

func (c *changer) Change() {
	c.isChanged = true
	if c.upper != nil {
		c.upper.Cahgne()
	}
}

func (c *changer) IsChanged() bool {
	return c.isChanged
}

//
// References
//

// A References represents list of references to
// objects. A References is not thread safe
type References struct {
	Degree uint32 // degree (hashes per leaf, hashes per branch)
	Len    uint32 // amount of non-zero elements of the tree
	Depth  uint32 // depth of the tree

	// the Depth allows to determine end of branches, that
	// is begining of leafs; leafs contains references to
	// elements of the References array instead of
	// references to RefsNode

	// depth * degree = items (including zeroes)

	Nodes []RefsNode // branches (if Depth == 0, then this is leafs)

	// internals

	walkNode *WalkNode   `enc:"-"` // walking
	place    *References `enc:"-"` // palce of the References

	refsPackNode *refsPackNode `enc:"-"` // unpack
}

// A RefsNode is internal
type RefsNode struct {
	// Len is amount of non-zero elements of the branch
	Len uint32

	// Hashes of the next branches or References of leafs
	Hashes []Reference

	// Nodes of the branch (for unpack a branch, range and walk)
	Nodes []RefsNode `enc:"-"`

	// internals

	refsPackNode *refsPackNode `enc:"-"`
}

func (r *RefsNode) insert(deep, degree int, ref Reference) (ok bool) {
	if deep == 0 {
		// the r is leaf
		if len(r.Hashes) < degree { // can insert easily
			r.Hashes = append(r.Hashes, ref)
			r.Len++
			//
		}
	}
	// TODO
	return
}

// insert Reference to []RefsNode returning ok = true and new
// or the same []RefsNode slcie
func (r *References) insert(depth int, ns []RefsNode,
	ref Reference) (nn []RefsNode, ok bool) {

	// if depth == 0, then the ns is leafs
	// and  contains references to objects

	if depth == 0 {
		return r.insertToLeafs(ns, ref)
	}

	// (1) no elements in the ns, create the branch
	//     (depth length) and insert the ref as the
	//     first element

	if len(ns) == 0 {
		var nodes []RefsNode
		depth--
		nodes, ok = r.insert(depth, nil, ref)
		if !ok {
			return // never happens
		}
		// ok is already true
		nn = []RefsNode{{
			Len:    1,
			Hashes: nodes,
		}}
	}

	// (2) try to find free space for the ref

	//

}

func (r *References) insertToLeafs(ns []RefsNode, ref Reference) (nn []RefsNode,
	ok bool) {

	// may be there aren't RefsNodes

	if len(ns) == 0 {
		nn, ok = []RefsNode{{
			Len:    1,
			Hashes: []Reference{ref},
		}}, true
		return
	}

	// get last leaf

	ll := ns[len(ns)-1]

	// let's look at the last RefsNode (leaf)

	if len(ll.Hashes) == int(r.Degree) {

		// last RefsNode is full

		// is len(ns) less then Degree
		if len(ns) < int(r.Degree) {

			// okay there is the place we need
			nn, ok = append(ns, RefsNode{
				Len:    1,
				Hashes: []Reference{ref},
			}), true
			return

		}

		// no, we can't insert here (into the leaf) anymore
		return // nil, false
	}

	// there is a place we can to use in the last leaf

	ll.Hashes = append(ll.Hashes, ref)

	nn, ok = ns, true
	return

}

// increase depth + 1
func (r *References) deeper() {
	ndepth := int(r.Depth) + 1 // new depth
	//
}

func (r *References) rangeNodes(d int, ns []RefsNode,
	rangeFunc func(ref Reference)) {

	if d == 0 { // leafs
		for _, n := range ns {
			for _, ref := range n.Hashes {
				rangeFunc(ref)
			}
		}
		return
	}

	//

	return

}

// IsValid returns false if the References is nto valid.
// Anyway, the Referenes can be invalid in deep, even if
// the method returns true
func (r *References) IsValid() bool {
	return r.Degree >= 2 && (r.Degree*r.Depth <= r.Len)
}

// IsBlank returns true if the reference is blank
func (r *References) IsBlank() bool {
	return r.Len == 0
}

// String implements fmt.Stringer interface
func (r *References) String() (s string) {
	// TODO
	return
}

type refsPackNode struct {
	upper     *refsPackNode // upper node o nil for References
	pack      *Pack         // related Pack
	isChanged bool          // true if the node has been changed
	unpacked  bool          // true if Nodes contains unpacked Hashes
}

// mark the ndoe as changed
func (r *refsPackNode) change() {
	// using non-recursive algorithm
	for up := r; up != nil; up = up.upper {
		up.isChanged = true
	}
}

// TODO (kostyarin)
func (r *refsPackNode) attach(upper *refsPackNode) {
	r.pack = upper.pack
	r.upper = upper
}

// TODO (kostyarin): detach pack?
func (r *refsPackNode) detach() {
	r.upper = nil
}
