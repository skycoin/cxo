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
	mods   refsMod       // unsaved modifications

	leafsBranches // leafs and branches

	upper *refsNode // upper node
}

// DB representation of the refsNode
type encodedRefsNode struct {
	Length   uint32          // length of the node
	Elements []cipher.SHA256 // if empty then all elements are nils
}

//
// loading
//

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

// loadNode that already contains hash and upper fields;
// the method loads the node anyway (no flags affect it)
func (r *Refs) loadNode(
	pack Pack, //    : pack to get
	rn *refsNode, // : the branch to load
	depth int, //    : depth of the rn (> 0)
) (
	err error, //    : get, decoding or 'invalid refs' error
) {

	var ern encodedRefsNode // encoded branch
	if err = get(pack, rn.hash, &ern); err != nil {
		return // get or decoding error
	}

	rn.length = int(ern.Length)

	return r.loadSubtree(pack, rn, ern.Elements, depth) // depth is the same
}

// laod branch, setting hash and upper fields and loading
// deeppre if HashTableIndex or (or and) EntireRefs flags
// set
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

	return r.loadNode(pack, rn, depth) // load deepper
}

// 'hash' and 'upper' fields of the rn are already set;
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

// 'hash' and 'upper' fields of the rn are already set;
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

// load given node if it's not loaded yet;
// e.g. if lazy loading used
func (r *Refs) loadNodeIfNeed(
	pack Pack, //    : pack to get
	rn *refsNode, // : the branch to load
	depth int, //    : depth of the rn (> 0)
) (
	err error, //    : get, decoding or 'invalid refs' error
) {

	if rn.length > 0 {
		return
	}

	return r.loadNode(pack, rn, depth)
}

//
// encoding
//

// encode a the refsNode as is
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

//
// index of element in Refs
//

// indexInRefs finds index of the leaf in Refs
func (r *refsElement) indexInRefs() (i int, err error) {

	if i, err = r.indexInUpper(); err != nil {
		return // invalid state
	}

	// upper node (can not be nil, but can be &Refs.refsNode)
	var j int
	if j, err = r.upper.indexInRefs(); err != nil {
		return
	}

	return i + j, nil
}

// index of the element in upper refsNode
func (r *refsElement) indexInUpper() (i int, err error) {
	var el *refsElement
	for i, el = range r.upper.leafs {
		if el == r {
			return // found
		}
	}
	return 0, ErrInvalidRefs // can't find element in upper leafs
}

// indexInRefs returns index of first element of the
// node in Refs; e.g. a refsNode has not an index, but
// it contains elements with indicies, and this mehtod
// returns index of first element
func (r *refsNode) indexInRefs() (i int, err error) {

Upper:
	for up, down := r.upper, r; up != nil; up, down = up.upper, up {
		for _, br := range up.branches {
			if br != down {
				i += br.length
			}
			continue Upper
		}
		return 0, ErrInvalidRefs // can't find in upper.branches
	}

	return
}

//
// find element by index
//

// refsElementByIndex finds *refsElemet by given index;
func (r *Refs) refsElementByIndex(
	pack Pack, //       : pack to load
	rn *refsNode, //    : the node to find inside (should be loaded)
	i int, //           : index of the needle
	depth int, //       : depth of the rn
) (
	el *refsElement, // : element if found
	err error, //       : error if any
) {

	// so, the rn is already loaded

	if depth == 0 { // take a look at leafs

		for j, el := range rn.leafs {
			if j == i {
				return el, nil // found
			}
		}

		return nil, ErrInvalidRefs // can't find the element
	}

	// else, take a look at branches

	var br *refsNode
	for _, br = range rn.branches {

		if err = r.loadNodeIfNeed(pack, br, depth-1); err != nil {
			return
		}

		if i < br.length {
			i -= br.length // subtract length of the skipped branch
			continue       // and skip the branch
		}

		break // the branch that contains the needle has been found
	}

	return r.refsElementByIndex(pack, br, i, depth-1)
}

//
// change hash of refsElement
//

// updateNodeHash updates hash of given node
func (r *Refs) updateNodeHash(
	pack Pack, //    : pack to save
	rn *refsNode, // : the node to update hash
	depth int, //    : depth of the node
) (
	err error, //    : saving error
) {

	// encode
	val := rn.encode(depth)
	// get hash
	hash := cipher.SumSHA256(val)
	// compare with previous one
	if err = pack.Set(hash, val); err != nil {
		return
	}

	rn.mods &^= contentMod // clear the flag if it has been set
	return
}

// bubbleContentChanges change hashes of
// all upper nodes if LazyUpdating flag
// is not set
func (r *Refs) bubbleContentChanges(
	pack Pack, //       : pack to save
	el *refsElement, // : element to start bubbling
) (
	err error, //       : error if any
) {

	var lazy = r.flags&LazyUpdating != 0
	var depth int

	for up := el.upper; up != nil; up = up.upper {
		if lazy == true {
			up.mods |= contentMod // modified but not saved
		} else {
			if err = r.updateNodeHash(pack, rn, depth); err != nil {
				return // saving error
			}
		}
		depth++ // the depth grows
	}

	r.Hash = r.refsNode.hash // set actual value of the Refs.Hash
	r.mods |= originMod      // origin has been modified

	return
}

// setElementHash replaces element hash
func (r *Refs) setElementHash(
	pack Pack, //          : pack to save
	el *refsElement, //    : element to change
	hash cipher.SHA256, // : new hash
) (
	err error, //          : error if any
) {

	if el.Hash == hash {
		return // nothing to change
	}

	if r.flags&HashTableIndex != 0 {
		r.delElementFromIndex(el)     // delete old
		r.addElementToIndex(hash, el) // add new
	}

	el.Hash = hash

	// so, length of the Refs is still the same
	// but content has been changed

	return r.bubbleContentChanges(pack, el)
}

// --- --- --- --- --- --- --- --- --- --- --- --- --- --- --- --- --- --- --- -

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
