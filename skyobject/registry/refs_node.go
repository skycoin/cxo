package registry

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// an element of the Refs
type refsElement struct {
	Hash  cipher.SHA256 ``        // hash (blank if nil)
	upper *refsNode     `enc:"-"` // upper node or nil if the node is the Refs
}

// leafs and branches,
// actually it contains
// leafs or branches
type leafsBranches struct {
	leafs    []*refsElement
	branches []*refsNode
}

// a branch of the Refs
type refsNode struct {
	hash   cipher.SHA256 // hash of this node
	length int           // length of this subtree

	leafsBranches // leafs and branches

	upper *refsNode // upper node
}

// DB representation of the refsNode
type encodedRefsNode struct {
	Length   uint32          // length of the node
	Elements []cipher.SHA256 // if empty then all elements are nils
}

// create leafs by given elements
func (r *Refs) loadLeafs(
	rn *refsNode, // node that contains this leafs
	elements []cipher.SHA256, // elemets
) {

	rn.leafs = make([]*refsElement, 0, len(elements))

	for _, hash := range elements {
		rn.leafs = append(r.leafs, r.loadLeaf(hash, rn))
	}

	return
}

// create leaf by given hash, the name starts with
// load by analogy of loadBranch
func (r *Refs) loadLeaf(
	hash cipher.SHA256, // : hash of the leaf
	upper *refsNode, //    : upper node
) (
	re *refsElement, //    : the "loaded" leaf
) {

	re = new(refsElement)

	re.Hash = hash
	re.upper = upper

	if r.flags&HashTableIndex != 0 {
		r.addElementToIndex(hash, re)
	}

	return
}

func (r *Refs) loadBranch(
	pack Pack, //          : pack to get the elemet
	hash cipher.SHA256, // : hash of the branch
	depth int, //          : depth of the branch
	upper *refsNode, //    : upper node
) (
	br *refsNode, //        : the loaded branch
	err error, //           : error if any
) {

	if hash == (cipher.SHA256{}) {
		return nil, ErrInvalidRefs
	}

	br = new(refsNode)

	br.hash = hash
	br.upper = upper

	if r.flags&(HashTableIndex|EntireRefs) == 0 {
		return // don't load deepper (lazy loading)
	}

	var ern encodedRefsNode // encoded branch
	if err = get(pack, hash, &ern); err != nil {
		return // Pack related error (get or decoding)
	}

	br.length = int(ern.Length)

	return r.loadSubtree(pack, br, ern.Elements, depth-1) // go deepper
}

// 'length' and 'upper' fields of the rn are already set;
// the depth if not 0 (e.g. the elements point to refsNode)
//
// the loadBranches doesn't load entire tree if it's not
// necessary; e.g. it depends on flags HashTableIndex and
// EntireRefs
func (r *Refs) loadBranches(
	pack Pack, //                : pack to load
	rn *refsNode, //             : the node
	depth int, //                : depth of the branches (> 0)
	elements []cipher.SHA256, // : elements of the branches
) (
	err error, //                : pack/decoding related error
) {

	rn.branches = make([]*refsNode, 0, len(elements))

	var br *refsNode
	for _, hash := range elements {
		if br, err = r.loadBranch(pack, hash, depth, rn); err != nil {
			return
		}
		rn.branches = append(rn.branches, br)
	}

	return
}

// 'length' and 'upper' fields of the rn are already set;
// the loadSubtree doesn't load entire tree if it's not
// necessary; e.g. the loading depends on falgs
// HashTableIndex and EntireRefs
func (r *Refs) loadSubtree(
	pack Pack, //                : pack to load
	rn *refsNode, //             : load subtree of this
	elements []cipher.SHA256, // : elements to load
	depth int, //                : depth of the rn (of the elements)
) (
	err error, //                : pack/decoding related error
) {

	if depth == 0 {
		r.loadLeafs(rn, elements)
		return // if the depth is 0, then the node contains leafs
	}

	// else if depth > 0, then the node contains branches
	return r.loadBranches(pack, rn, depth, elements)
}

// --- --- --- --- --- --- --- --- --- --- --- --- --- --- --- --- --- --- --- -

func (r *Refs) loadRefsNodeIfNeed(pack *Pack, rn *refsNode, depth int,
	upper *refsNode) (err error) {

	if rn.hash != (cipher.SHA256{}) && rn.length == 0 {
		var ln *refsNode // loaded
		if ln, err = r.loadRefsNode(pack, depth, upper, rn.hash); err != nil {
			return
		}
		*rn = *ln // set
	}

	return
}

// encode as is
func (r *refsNode) encode(depth int) []byte {
	var ern encodedRefsNode

	ern.Length = uint32(r.length)

	if depth == 0 {
		ern.Elements = make([]cipher.SHA256, 0, len(r.leafs))
		for _, el := range r.leafs {
			ern.Elements = append(ern.Elements, el.Hash)
		}
	} else {
		ern.Elements = make([]cipher.SHA256, 0, len(r.branches))
		for _, br := range r.branches {
			ern.Elements = append(ern.Elements, br.hash)
		}
	}

	return encoder.Serialize(ern)
}

// without deep exploring
func (r *refsNode) updateHash(pack Pack, depth int) (err error) {

	if r.length == 0 {
		r.hash = cipher.SHA256{}
		return // it's enough
	}

	val := r.encode(depth)

	if hash := cipher.SumSHA256(val); r.hash != hash {
		err = pack.Set(hash, val)
	}

	return
}

func (r *Refs) updateNodeHashIfNeed(pack Pack, rn *refsNode,
	depth int) (err error) {

	if r.flags&LazyUpdating != 0 {
		return // doesn't need
	}

	err = rn.updateHash(pack, depth)
	return
}

// deleteLeafByIndex removes element from leafs of
// the leafsBranches by index
func (l *leafsBranches) deleteLeafByIndex(k int) {
	copy(l.leafs[k:], l.leafs[k+1:])
	l.leafs[len(l.leafs)-1] = nil
	l.leafs = l.leafs[:len(l.leafs)-1]
}

func (l *leafsBranches) deleteBranchByIndex(k int) {
	copy(l.branches[k:], l.branches[k+1:])      // move
	l.branches[len(l.branches)-1] = nil         // free
	l.branches = l.branches[:len(l.branches)-1] // reduce length
}

func (l *leafsBranches) deleteLeaf(el *refsElement) (err error) {
	for k, lf := range l.leafs {
		if lf == el {
			l.deleteLeafByIndex(k)
			return
		}
	}
	return ErrInvalidRefs // can't find (invalid state)
}

func (l *leafsBranches) deleteBranch(br *refsNode) (err error) {
	for k, rn := range l.branches {
		if br == rn {
			l.deleteBranchByIndex(k)
			return
		}
	}
	return ErrInvalidRefs // can't find (invalid state)
}

// deleteLeafByIndex remove element from hash-table
// index of the Refs and from given leafsBranches;
// the index is index of the element in the leafsBranches
func (r *Refs) deleteLeafByIndex(lb *leafsBranches, k int, el *refsElement) {

	if r.flags&HashTableIndex != 0 {
		r.delElementFromIndex(el.Hash, el)
	}

	lb.deleteLeafByIndex(k)

	r.mods |= (lengthMod | originMod)

	if len(r.iterators) > 0 {
		// force iterators to find next index from root of the Refs
		r.iterators[len(r.iterators)-1] = true
	}
}

// index of given leaf or error
func (l *leafsBranches) indexOfLeaf(re *refsElement) (i int, err error) {
	var el *refsElement
	for i, el = range l.leafs {
		if el == re {
			return // found
		}
	}
	err = ErrInvalidRefs // not found (invalid state)
	return
}

func (l *leafsBranches) indicesOfLeaf(hash cipher.SHA256) (is []int,
	err error) {

	for i, el := range l.leafs {
		if el.Hash == hash {
			is = append(is, i) // found, but let's look for another one
		}
	}

	if len(is) == 0 {
		err = ErrInvalidRefs // not found (invalid state)
	}

	return
}
