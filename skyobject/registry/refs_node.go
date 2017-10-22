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

// from given []cipher.SHA256 create []*refsElement
func (r *Refs) makeLeafs(elements []cipher.SHA256,
	upper *refsNode) (leafs []*refsElement) {

	leafs = make([]*refsElement, 0, len(elements))

	for _, hash := range elements {

		re := &refsElement{
			Hash:  hash,
			upper: upper,
		}

		if r.flags&HashTableIndex != 0 {
			r.addElementToIndex(hash, re)
		}

		leafs = append(leafs, re)

	}

	return
}

// the depth is current depth, e.g. depth-1 will be used for subnodes
func (r *Refs) makeBranches(pack Pack, elements []cipher.SHA256, depth int,
	upper *refsNode) (branches []*refsNode, err error) {

	var sn *refsNode

	branches = make([]*refsNode, 0, len(elements))

	for _, hash := range elements {
		sn, err = r.loadRefsNode(pack, depth-1, upper, hash)
		if err != nil {
			return
		}
		branches = append(branches, sn)
	}

	return
}

func (r *Refs) makeLeafsBranches(pack Pack, lb *leafsBranches,
	elements []cipher.SHA256, depth int, upper *refsNode) (err error) {

	if depth == 0 {
		lb.leafs = r.makeLeafs(elements, upper)
	} else {
		lb.branches, err = r.makeBranches(pack, elements, depth, upper)
	}

	return
}

// load fucking recursive
func (r *Refs) loadRefsNode(pack Pack, depth int, upper *refsNode,
	hash cipher.SHA256) (rn *refsNode, err error) {

	var val []byte

	rn = new(refsNode)

	rn.hash = hash
	rn.upper = upper

	if hash == (cipher.SHA256{}) {
		err = ErrInvalidEncodedRefs
		return // blank refs node
	}

	if r.flags&(EntireRefs|HashTableIndex) == 0 {
		return // don't load deepper
	}

	// load deepper

	var ern encodedRefsNode
	if err = get(pack, hash, &ern); err != nil {
		return
	}

	rn.length = int(ern.Length)

	err = r.makeLeafsBranches(pack, &rn.leafsBranches, ern.Elements, depth,
		upper)

	return
}

func (r *Refs) loadRefsNodeIfNeed(pack *Pack, rn *refsNode, depth int,
	upper *refsNode) (err error) {

	if rn.hash != (cipher.SHA256{}) && rn.length == 0 {
		var ln *refsNode // loaded
		if ln, err = r.loadRefsNodes(pack, depth, upper, rn.hash); err != nil {
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

func (l *leafsBranches) deleteLeaf(el *refsElement) (err error) {
	for k, lf := range l.leafs {
		if lf == el {
			l.deleteLeafByIndex(k)
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

	r.modified = true // mark as modified (length has been changed)

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
