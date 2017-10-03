package registry

import (
	"errors"
	"fmt"

	"github.com/disiqueira/gotree"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

type Refs struct {
	// Hash that represnts the Refs. If Refs is
	// blank then this Hash is blank too
	Hash cipher.SHA256

	depth  int `enc:"-"` // depth - 1
	degree int `enc:"-"` // degree

	// hash -> node that keeps element
	//         to find index. It points to
	//         nils if the depth is 0
	index map[cipher.SHA256]*refsNode `enc:"-"`

	length int `enc:"-"` // length of Refs

	leafs    []refsElement `enc:"-"` //
	branches []*refsNode   `enc:"-"` //

	flags Flags `enc:"-"` // first use (load) flags

	modified  bool `enc:"-"` // actual state
	iterating int  `enc"-"`  // is iterating now
}

type refsElement struct {
	Deleted bool          // deleted if true
	Hash    cipher.SHA256 // hash (blank if nil)
}

type refsNode struct {
	hash cipher.SHA256 // hash of this node

	length int // length of this subtree

	leafs    []refsElement //
	branches []*refsNode   //

	upper *refsNode // upper node
	root  *Refs     // head (root)
}

func (r *Refs) initialize(pack Pack) (err error) {

	if r.iterating > 0 {
		return // already initialized
	}

	if r.flags == 0 {
		r.flags = pack.Flags() | flagIsSet // keep current flags
	}

	if r.Hash == (cipher.SHA256{}) {
		return // blank Refs
	}

	if r.length == 0 {
		err = r.load(pack)
	}
	return
}

// Len returns length of the Refs
func (r *Refs) Len(pack Pack) (ln int, err error) {
	if err = r.initialize(pack); err != nil {
		return
	}

	ln = r.length
	return
}

type encodedRefs struct {
	Depth    uint32
	Degree   uint32
	Length   uint32
	Elements []refsElement
}

func (r *Refs) load(pack Pack) (err error) {

	var er encodedRefs
	if err = get(pack, r.Hash, &er); err != nil {
		return
	}

	r.depth = int(er.Depth)
	r.degree = int(er.Degree)

	r.length = int(er.Length)

	if r.length == 0 || r.degree < 2 {
		return ErrInvalidEncodedRefs
	}

	if r.flags&HashTableIndex != 0 {
		r.index = make(map[cipher.SHA256]*refsNode)
	}

	if r.depth == 0 {

		r.leafs = er.Elements

		if r.flags&HashTableIndex != 0 {
			for _, el := range r.leafs {
				if el.Deleted == false {
					r.index[el.Hash] = nil
				}
			}
		}

	} else {

		err = r.loadRootRefsNodes(pack, er.Elements)

	}

	return
}

// 1. only 'hash' of refsNode has meaning
// 2. refsNode loaded
// 3. refsNode loaded with all subtrees

type encodedRefsNode struct {
	Length   uint32        // length of the node
	Elements []refsElement // if empty then all are elements nils
}

func (r *Refs) loadRootRefsNodes(pack Pack,
	elements []refsElement) (err error) {

	var depth = r.depth // the depth is not 0
	var rn *refsNode

	r.branches = make([]*refsNode, 0, len(elements))

	for _, el := range elements {

		if el.Deleted == true {
			continue // deleted
		}

		if rn, err = r.loadRefsNodes(pack, depth-1, nil, el.Hash); err != nil {
			return
		}

		r.branches = append(r.branches, rn)

	}

	return
}

// load fucking recursive
func (r *Refs) loadRefsNodes(pack Pack, depth int, upper *refsNode,
	hash cipher.SHA256) (rn *refsNode, err error) {

	var val []byte

	rn = new(refsNode)

	rn.hash = hash
	rn.upper = upper
	rn.root = r

	if hash == (cipher.SHA256{}) {
		return // empty branch or leaf
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

	if depth == 0 {
		rn.leafs = ern.Elements

		if r.flags&HashTableIndex != 0 {
			for _, el := range rn.leafs {
				if el.Deleted == false {
					r.index[hash] = rn // parent node of the leafs
				}
			}
		}

	} else {

		var sn *refsNode

		rn.branches = make([]*refsNode, 0, len(ern.Elements))

		for _, el := range ern.Elements {

			if el.Deleted == true {
				continue
			}

			//                        pack, depth, upper, hash
			sn, err = r.loadRefsNodes(pack, depth-1, rn, el.Hash)
			if err != nil {
				return
			}

			rn.branches = append(rn.branches, sn)

		}

	}

	return
}

// HasHash returns false if the Refs doesn't have given hash. It returns
// true if contains, or first error. It never returns ErrNotFound
func (r *Refs) HasHash(pack Pack, hash cipher.SHA256) (ok bool, err error) {

	if err = r.initialize(pack); err != nil {
		return
	}

	if r.flags&HashTableIndex != 0 {
		_, ok = r.index[hash]
		return
	}

	// else

	// TODO (kostyarin): iterate

	return
}

// ValueByHash decodes value by given hash. If HashTableIndex
// is set, then hash table used. If given hash is blank then
// the ValueByHash returns ErrNotFound if the Refs doesn't
// contan blank elements. Or ErrRefsElementIsNil if contains.
// If the HashTableIndex is set and the Refs contains many
// elements with the same hash, then the ValueByHash use one
// of this elements (undefined).
func (r *Refs) ValueByHash(pack Pack, hash cipher.SHA256,
	obj interface{}) (err error) {

	// initilize() inside the HashHash

	var ok bool
	if ok, err = r.HasHash(pack, hash); err != nil {
		return
	}
	if !ok {
		return ErrNotFound
	}
	if hash == (cipher.SHA256{}) {
		return ErrRefsElementIsNil
	}

	err = get(pack, hash, obj)
	return
}

// IndexByHash returns index of element by hash. It returns ErrNotfound
// if the Refs doesn't have given hash
func (r *Refs) IndexByHash(pack Pack, hash cipher.SHA256) (i int, err error) {

	if err = r.initialize(pack); err != nil {
		return
	}

	if r.flags&HashTableIndex != 0 {

		var ok bool
		var pn *refsNode // parent node

		if pn, ok = r.index[hash]; !ok {
			err = ErrNotFound
			return
		}

		// find index
		if pn == nil {
			return r.indexInLeafs(hash, r.leafs)
		}

		return r.indexOfLeaf(pn, hash)
	}

	// TODO (kostyarin): iterate descending

	// descend

	return
}

func (r *Refs) indexOfLeaf(rn *refsNode, hash cipher.SHA256) (i int,
	err error) {

	if i, err = r.indexInLeafs(hash, rn.leafs); err != nil {
		return
	}

	down, up := rn, rn.upper

	for ; up != nil; down, up = up, up.upper {
		for _, br := range up.branches {
			if br == down {
				break
			}
			i += br.length
		}
	}

	for _, br := range r.branches {
		if br == down {
			break
		}
		i += br.length
	}

	return
}

func (r *Refs) indexInLeafs(hash cipher.SHA256,
	leafs []refsElement) (i int, err error) {

	// it is member of Refs.leafs
	for _, el := range r.leafs {
		if el.Deleted == true {
			continue // skip deleted
		}
		if el.Hash == hash {
			return // found
		}
		i++
	}

	err = ErrInvalidRefs // not found
	return
}

// ValueByHashWithIndex decodes value by given hash and returns
// index of the value. It returns actual index and ErrRefsElementIsNil
// if given hash is blank but exists in the Refs
func (r *Refs) ValueByHashWithIndex(pack Pack, hash cipher.SHA256,
	obj interface{}) (i int, err error) {

	// initialize() inside the IndxByHash

	if i, err = r.IndexByHash(pack, hash); err != nil {
		return
	}

	if hash == (cipher.SHA256{}) {
		err = ErrRefsElementIsNil
		return
	}

	err = get(pack, hash, obj)
	return
}

func validateIndex(i int, length int) (err error) {
	if i < 0 || i >= length {
		err = ErrIndexOutOfRange
	}
	return
}

func (r *Refs) hashByIndex(pack Pack, i int, depth int,
	leafs []cipher.SHA256, branches []*refsNode) (hash cipher.SHA256,
	err error) {

	if depth == 0 {

		for _, hash = range leafs {
			if hash == (cipher.SHA256{}) {
				continue
			}
			if i == 0 {
				return // got it
			}
			i--
		}
		err = ErrInvalidRefs
		return

	}

	// else if depth > 0

	for _, br := range branches {

		// TODO (kostyarin): load the branch if it's not loaded yet

		if i > br.length {
			i -= br.length
			continue
		}
		return r.hashByIndex(pack, i, depth-1, br.leafs, br.branches)
	}

	err = ErrInvalidRefs
	return
}

// HashByIndex returns hash by index
func (r *Refs) HashByIndex(pack Pack, i int) (hash cipher.SHA256, err error) {

	if err = r.initialize(pack); err != nil {
		return
	}

	if err = validateIndex(i, r.length); err != nil {
		return
	}

	hash, err = r.hashByIndex(pack, i, r.depth, r.leafs, r.branches)
	return
}

// ValueByIndex returns value by index or ErrNotFound
// or another error. It also returns hash of the value
func (r *Refs) ValueByIndex(pack Pack, i int,
	obj interface{}) (hash cipher.SHA256, err error) {

	// initialize() inside the HashByIndex

	if hash, err = r.HashByIndex(pack, i); err != nil {
		return
	}
	if hash == (cipher.SHA256{}) {
		err = ErrRefsElementIsNil
		return
	}
	err = get(pack, hash, obj)
	return
}

// SetHashByIndex replaces hash of element with given index with
// given hash
func (r *Refs) SetHashByIndex(pack Pack, i int,
	hash cipher.SHA256) (err error) {

	if err = r.initialize(pack); err != nil {
		return
	}

	if err = validateIndex(i, r.length); err != nil {
		return
	}

	return r.setHashByIndex(pack, i, hash, r.depth, r.leafs, r.branches)
}

func (r *Refs) setHashByIndex(pack Pack, i int, hash cipher.SHA256, depth int,
	leafs []refsElement, branches []*refsNode) (err error) {

	if depth == 0 {
		for k, el := range leafs {
			if el.Deleted == true {
				continue // skip deleted
			}
			if i == 0 {
				leafs[k].Hash = hash // set
				return
			}
			i--
		}
		return ErrInvalidRefs // can't find
	}

	// else if depth > 0

	for _, br := range branches {

		// TODO (kostyarin): load br if it's not loaded yet

		if i > br.length {
			i -= br.length
			continue
		}

		return r.setHashByIndex(pack, i, hash, depth-1, br.leafs, br.branches)
	}

	return ErrInvalidRefs // can't find
}

// SetValueByIndex saves given value calculating its hash and sets this
// hash to given index. You must be sure that schema of given element is
// schema of the Refs. Otherwise, Refs will be broken. Use nil-interface{}
// to set blank hash
func (r *Refs) SetValueByIndex(pack Pack, i int, obj interface{}) (err error) {

	// initialize() inside the SetHashByIndex

	var hash cipher.SHA256

	if obj != nil {
		if hash, err = pack.Add(encoder.Serialize(obj)); err != nil {
			return
		}
	}

	return r.SetHashByIndex(pack, i, hash)
}

func (r *Refs) cutElementByIndex(pack Pack, i, depth int, leafs []refsElement,
	branches []*refsNode) (err error) {

	if depth == 0 {
		for k, el := range leafs {
			if el.Deleted == true {
				continue // skip deleted
			}
			if i == 0 {
				// delete (cut)
				leafs[k] = refsElement{
					Deleted: true,
					Hash:    cipher.SHA256{},
				}
				r.modified = true // mark as modified
				return
			}
			i--
		}
		return ErrInvalidRefs // can't find
	}

	// else if depth > 0

	for _, br := range branches {

		// TODO (kostyarin): load br if it's not loaded yet

		if i > br.length {
			i -= br.length
			continue
		}

		err = r.cutElementByIndex(pack, i, depth-1, br.leafs, br.branches)
		if err != nil {
			return
		}
		br.length-- // reduce length (has been deleted (cut))
	}

	return ErrInvalidRefs // can't find
}

// CutElementByIndex cuts single element from the Refs changing
// the Refs
func (r *Refs) CutElementByIndex(pack Pack, i int) (err error) {

	if err = r.initialize(pack); err != nil {
		return
	}

	if err = validateIndex(i, r.length); err != nil {
		return
	}

	err = r.cutElementByIndex(pack, i, r.depth, r.leafs, r.branches)
	if err != nil {
		return
	}
	r.length-- // reduce length (has been deleted (cut))
	return
}

// TODO (kostyarin): implement the CutSliceByIndices
//
// // CutSliceByIndices cuts slice from the 'from' to the 'to' arguments.
// // Like a golang [a:b] but cut
// func (r *Refs) CutSliceByIndices(pack Pack, from, to int) (err error) {
//
//	//
//
//	return
// }

func validateSliceIndices(i, j, length int) (err error) {
	if i < 0 || j < 0 || i > length || j > length {
		err = ErrIndexOutOfRange
	} else if i > j {
		err = ErrInvalidSliceIndex
	}
	return
}

// Slice returns new Refs that contains values of this Refs from
// given i (inclusive) to given j (exclusive). If i and j are valid
// and equal, then the Slcie return new empty Refs
func (r *Refs) Slice(pack Pack, i, j int) (slice *Refs, err error) {

	// https://play.golang.org/p/4tP7_MuCN9

	if err = r.initialize(pack); err != nil {
		return
	}

	if err = validateSliceIndices(i, j, r.length); err != nil {
		return
	}

	slice = new(Refs)

	slice.degree = r.degree
	if r.flags&HashTableIndex != 0 {
		slice.index = make(map[cipher.SHA256]*refsNode)
	}
	slice.flags = r.flags

	if i == j {
		return // done
	}

	// TODO (kostyarin): ascend from i to j appending to the slice

	return
}

// for defer
func (r *Refs) decrementIterating() {
	r.iterating--
}

// Ascend iterates over all values ascending order until
// first error or the end. Use ErrStopIteration to
// break iteration. Any error is returned by given function
// (except the ErrStopIteration) is returned by the Ascend()
// method
func (r *Refs) Ascend(pack Pack,
	ascendFunc func(i int, hash cipher.SHA256) (err error)) (err error) {

	if err = r.initialize(pack); err != nil {
		return
	}

	if r.length == 0 {
		return // empty Refs
	}

	r.iterating++
	defer r.decrementIterating()

	_, err = r.ascend(pack, 0, ascendFunc, r.depth, r.leafs, r.branches)
	if err == ErrStopIteration {
		err = nil // clear the ErrStopIteration
	}
	return
}

func (r *Refs) ascend(pack Pack, i int,
	ascendFunc func(i int, hash cipher.SHA256) (err error), depth int,
	leafs []refsElement, branches []*refsNode) (pass int, err error) {

	if depth == 0 {
		return r.ascendLeafs(i, ascendFunc, leafs)
	}

	// else if depth > 0

	return r.ascendBranches(pack, i, ascendFunc, depth, branches)
}

func (r *Refs) ascendLeafs(i int,
	ascendFunc func(i int, hash cipher.SHA256) (err error),
	leafs []refsElement) (pass int, err error) {

	for _, el := range leafs {
		if el.Deleted == true {
			continue
		}

		if err = ascendFunc(i, el.Hash); err != nil {
			return
		}

		i++
		pass++
	}

	return
}

func (r *Refs) ascendBranches(pack Pack, i int,
	ascendFunc func(i int, hash cipher.SHA256) (err error), depth int,
	branches []*refsNode) (pass int, err error) {

	var j int

	for _, br := range branches {

		// TODO (kostyarin): load the br if it's not loaded yet

		if br.length == 0 {
			continue // skip empty branches
		}

		j, err = r.ascend(pack, i, ascendFunc, depth-1, br.leafs, br.branches)
		if err != nil {
			return
		}

		i += j
		pass += j

	}

	return
}

// Descend iterates over all values descending order until
// first error or the end. Use ErrStopIteration to
// break iteration. Any error is returned by given function
// (except the ErrStopIteration) is returned by the Descend()
// method
func (r *Refs) Descend(pack Pack,
	descendFunc func(i int, hash cipher.SHA256) (err error)) (err error) {

	if err = r.initialize(pack); err != nil {
		return
	}

	if r.length == 0 {
		return // empty Refs
	}

	r.iterating++
	defer r.decrementIterating()

	_, err = r.descend(pack, r.length-1, descendFunc, r.depth, r.leafs,
		r.branches)
	if err == ErrStopIteration {
		err = nil // clear the ErrStopIteration
	}
	return
}

func (r *Refs) descend(pack Pack, i int,
	descendFunc func(i int, hash cipher.SHA256) (err error), depth int,
	leafs []refsElement, branches []*refsNode) (pass int, err error) {

	if depth == 0 {
		return r.descendLeafs(i, descendFunc, leafs)
	}

	// else if depth > 0

	return r.descendBranches(pack, i, descendFunc, depth, branches)
}

func (r *Refs) descendLeafs(i int,
	descendFunc func(i int, hash cipher.SHA256) (err error),
	leafs []refsElement) (pass int, err error) {

	for k := len(leafs) - 1; k >= 0; k-- {

		el := leafs[k]

		if el.Deleted == true {
			continue
		}

		if err = descendFunc(i, el.Hash); err != nil {
			return
		}

		i--
		pass++
	}

	return
}

func (r *Refs) descendBranches(pack Pack, i int,
	descendFunc func(i int, hash cipher.SHA256) (err error), depth int,
	branches []*refsNode) (pass int, err error) {

	var j int

	for k := len(branches) - 1; k >= 0; k-- {

		br := branches[k]

		// TODO (kostyarin): load the br if it's not loaded yet

		if br.length == 0 {
			continue // skip empty branches
		}

		j, err = r.descend(pack, i, descendFunc, depth-1, br.leafs, br.branches)
		if err != nil {
			return
		}

		i -= j
		pass += j

	}

	return
}

// AscendFrom iterates over all values ascending order starting
// from the 'from' element until first error or the end. Use
// ErrStopIteration to break iteration. Any error is returned
// by given function (except the ErrStopIteration) is returned
// by the AscendFrom() method. If the 'from' argument exceeds
// the Refs bounds, then call does nothing (e.g. no error returned)
func (r *Refs) AscendFrom(pack Pack, from int,
	ascendFunc func(i int, hash cipher.SHA256) (err error)) (err error) {

	if err = r.initialize(pack); err != nil {
		return
	}

	if r.length == 0 || from >= r.length {
		return // empty Refs or from is too big
	}

	r.iterating++
	defer r.decrementIterating()

	_, err = r.ascendFrom(pack, from, 0, ascendFunc, r.depth, r.leafs,
		r.branches)
	if err == ErrStopIteration {
		err = nil // clear the ErrStopIteration
	}
	return
}

func (r *Refs) ascendFrom(pack Pack, from, i int,
	ascendFunc func(i int, hash cipher.SHA256) (err error), depth int,
	leafs []refsElement, branches []*refsNode) (pass int, err error) {

	if depth == 0 {
		return r.ascendFromLeafs(from, i, ascendFunc, leafs)
	}

	// else if depth > 0

	return r.ascendFromBranches(pack, from, i, ascendFunc, depth, branches)
}

func (r *Refs) ascendFromLeafs(from, i int,
	ascendFunc func(i int, hash cipher.SHA256) (err error),
	leafs []refsElement) (pass int, err error) {

	for _, el := range leafs {

		if el.Deleted == true {
			continue
		}

		if i >= from {
			if err = ascendFunc(i, el.Hash); err != nil {
				return
			}
		}

		i++
		pass++
	}

	return
}

func (r *Refs) ascendFromBranches(pack Pack, from, i int,
	ascendFunc func(i int, hash cipher.SHA256) (err error), depth int,
	branches []*refsNode) (pass int, err error) {

	var j int

	for _, br := range branches {

		// TODO (kostyarin): load the br if it's not loaded yet

		if br.length == 0 {
			continue // skip empty branches
		}

		if from < i+br.length {

			// just add length of the br
			i += br.length
			pass += br.length

			continue
		}

		j, err = r.ascendFrom(pack, from, i, ascendFunc, depth-1, br.leafs,
			br.branches)
		if err != nil {
			return
		}

		i += j
		pass += j

	}

	return
}

// DescendFrom iterates over all values descending order starting
// from the 'from' element until first error or the end. Use
// ErrStopIteration to break iteration. Any error is returned
// by given function (except the ErrStopIteration) is returned
// by the DescendFrom() method
func (r *Refs) DescendFrom(pack Pack, from int,
	descendFunc func(i int, hash cipher.SHA256) (err error)) (err error) {

	if err = r.initialize(pack); err != nil {
		return
	}

	if r.length == 0 || from >= r.length {
		return // empty Refs or from is too big
	}

	r.iterating++
	defer r.decrementIterating()

	_, err = r.descendFrom(pack, from, r.length-1, descendFunc, r.depth,
		r.leafs, r.branches)
	if err == ErrStopIteration {
		err = nil // clear the ErrStopIteration
	}
	return
}

func (r *Refs) descendFrom(pack Pack, from, i int,
	descendFunc func(i int, hash cipher.SHA256) (err error), depth int,
	leafs []refsElement, branches []*refsNode) (pass int, err error) {

	if depth == 0 {
		return r.descendFromLeafs(from, i, descendFunc, leafs)
	}

	// else if depth > 0

	return r.descendFromBranches(pack, from, i, descendFunc, depth, branches)
}

func (r *Refs) descendFromLeafs(from, i int,
	descendFunc func(i int, hash cipher.SHA256) (err error),
	leafs []refsElement) (pass int, err error) {

	for k := len(leafs) - 1; k >= 0; k-- {

		el := leafs[k]

		if el.Deleted == true {
			continue
		}

		if i <= from {
			if err = descendFunc(i, el.Hash); err != nil {
				return
			}
		}

		i--
		pass++
	}

	return
}

func (r *Refs) descendFromBranches(pack Pack, from, i int,
	descendFunc func(i int, hash cipher.SHA256) (err error), depth int,
	branches []*refsNode) (pass int, err error) {

	var j int

	for k := len(branches) - 1; k >= 0; k-- {

		br := branches[k]

		// TODO (kostyarin): load the br if it's not loaded yet

		if br.length == 0 {
			continue // skip empty branches
		}

		if from > i-br.length {

			// just subtract length of the br
			i -= br.length
			pass += br.length

			continue
		}

		j, err = r.descendFrom(pack, from, i, descendFunc, depth-1, br.leafs,
			br.branches)
		if err != nil {
			return
		}

		i -= j
		pass += j

	}

	return
}

// Append given objects to the Refs. You must be sure that type of this
// obejcts is type of this Refs. All nil-interfaces will be treated as
// blank references (cipher.SHA256{})
func (r *Refs) AppendValues(pack Pack, objs ...interface{}) (err error) {

	if len(objs) == 0 {
		return // nothing ot append
	}

	hashes := make([]cipher.SHA256, 0, len(objs))

	var hash cipher.SHA256

	for _, obj := range objs {
		if obj == nil {
			hashes == append(hashes, cipher.SHA256{}) // nil = blank
			continue
		}
		if hash, err = pack.Add(encoder.Serialize(obj)); err != nil {
			return
		}
		hashes = append(hashes, hash)
	}

	return r.AppendHashes(pack, hashes)
}

// AppendRefs appends another Refs to this one
func (r *Refs) AppendRefs(pack Pack, refs *Refs) (err error) {

	if err = r.initialize(pack); err != nil {
		return
	}

	if err = refs.initialize(pack); err != nil {
		return
	}

	//

	return
}

// AppendHashes appends given hashes to Refs. You must be sure that
// type of this obejcts is type of this Refs. Degree of the Refs will
// be increased if necessary
func (r *Refs) AppendHashes(pack Pack, hash ...cipher.SHA256) (err error) {

	if err = r.initialize(pack); err != nil {
		return
	}

	//

	return
}

// Clear the Refs making them blank
func (r *Refs) Clear() (err error) {

	// TODO (kostyarin): iterating

	r.Hash = cipher.SHA256{}

	r.depth = 0
	r.degree = 0

	r.index = nil

	r.length = 0

	r.leafs = nil
	r.branches = nil

	r.flags = 0

	r.modified = false

	return nil
}

// Rebuild the Refs if need. It's impossible to rebuild
// while the Refs is iterating. In most cases you should
// not call the Rebuild, beacuse it's called automatically
// by Pack during saving. The Rebuild rebuilds the Refs tree
// to choose best (compact and fast) depth removing deleted
// elements (all deleted elements still present in the tree
// after CutElementByIndex and CutSliceByIndex that not
// implemented yet). Also, the Refs tree can grows horizontally
// in some cases to speed up some operations. But after the
// grow the tree is not to fast to access elements and should
// be rebuilt. The Rebuild solves all this issues. If the Refs
// tree has not been changed, then the Rebuild does nothing.
// Also, the Rebuild can keep some dleted elements if if their
// removal will not change depth of the Refs tree.
//
// Also, the depth is the height of the Refs tree, but root
// of the tree is above all. Ha-ha
func (r *Refs) Rebuild(pack Pack) (err error) {

	if r.iterating > 0 {
		return ErrRefsIsIterating // can't rebuild while the Refs is iterating
	}

	// (degree + 1) ^ depth == real length

	// TODO (kostyarin): rebuild

	return
}

// Len(pack Pack) (ln int, err error)
//
// VaueByHash(pack Pack, hash cipher.SHA256, obj interface{}) (err error)
// ValueByHashWithIndex(pack Pack, hash cipher.SHA256, obj interface{}) (i int, err error)
// ValueByIndex(pack Pack, i int, obj interface{}) (hash cipher.SHA256, err error)
//
// DelByIndex(pack Pack, i int) (err error)
// Cut(pack Pack, i, j int) (err error)
//
// Slice(pack Pack, i, j int) (err error)
//
// Ascend(pack Pack, func(i int, hash cipher.SHA256)) (err error)
// Descend(pack Pack, func(i int, hash cipher.SHA256)) (err error)
//
// AscendFrom(pack Pack, from int, func(i int, hash cipher.SHA256)) (err error)
// DescendFrom(pack Pack, from int, func(i int, hash cipher.SHA256)) (err error)
//
// Append(pack Pack, obj ...interface{}) (err error)
// AppendRefs(pack Pack, refs *Refs) (err error)
// AppendHashes(pack Pack, hash ...cipher.SHA256) (err error)
//
// Clear()
// Rebuild(pack Pack) (err error)

/*

// A Refs represetns array of references.
// The Refs contains RefsElem(s). To delete
// an element use Delete method of the
// RefsElem. It's impossible to reborn
// deleted RefsElem. To cahgne Hash field
// of the Refs use SetHash method
type Refs struct {
	Hash cipher.SHA256 // hash of the Refs

	// the Hash is also hash of nested nodes

	depth  int `enc:"-"` // not stored
	degree int `enc:"-"` // not stored

	length   int         `enc:"-"`
	branches []*Refs     `enc:"-"` // branches
	leafs    []*RefsElem `enc:"-"` // leafs

	upper *Refs                       `enc:"-"` // upper node (nil for root)
	index map[cipher.SHA256]*RefsElem `enc:"-"` // index (or reference)

	// lock
	// - don't rebuild the tree in some cases
	// - don't allow Append, Ascend, Descend
	//   methods in some cases
	iterating bool `enc:"-"`

	ch bool `enc:"-"` // true if the Refs has been changed or fresh
}

func (r *Refs) encode(root bool, depth int) (val []byte) {
	if root {
		var ers encodedRefs
		ers.Degree = uint32(r.degree)
		ers.Length = uint32(r.length)
		ers.Depth = uint32(r.depth)
		if depth == 0 {
			ers.Nested = make([]cipher.SHA256, 0, len(r.leafs))
			for _, leaf := range r.leafs {
				if leaf.Hash == (cipher.SHA256{}) {
					continue
				}
				ers.Nested = append(ers.Nested, leaf.Hash)
			}
		} else {
			ers.Nested = make([]cipher.SHA256, 0, len(r.branches))
			for _, br := range r.branches {
				if br.Hash == (cipher.SHA256{}) {
					continue
				}
				ers.Nested = append(ers.Nested, br.Hash)
			}
		}
		val = encoder.Serialize(&ers)
	} else {
		var ern encodedRefsNode
		ern.Length = uint32(r.length)
		if depth == 0 {
			ern.Nested = make([]cipher.SHA256, 0, len(r.leafs))
			for _, leaf := range r.leafs {
				if leaf.Hash == (cipher.SHA256{}) {
					continue
				}
				ern.Nested = append(ern.Nested, leaf.Hash)
			}
		} else {
			ern.Nested = make([]cipher.SHA256, 0, len(r.branches))
			for _, br := range r.branches {
				if br.Hash == (cipher.SHA256{}) {
					continue
				}
				ern.Nested = append(ern.Nested, br.Hash)
			}
		}
		val = encoder.Serialize(&ern)
	}
	return
}

// IsBlank returns true if the Refs is blank
func (r *Refs) IsBlank() bool {
	return r.Hash == (cipher.SHA256{})
}

// Eq returns true if given Refs is the same
// as this one (ccompare hashes only)
func (r *Refs) Eq(x *Refs) bool {
	return r.Hash == x.Hash
}

// Short string
func (r *Refs) Short() string {
	return r.Hash.Hex()[:7]
}

// String implements fmt.Stringer interface
func (r *Refs) String() string {
	return r.Hash.Hex()
}

// Developer:
//
// There are 3 states of Refs.
// 1) Only Hash field has meaning
// 2) Branches or leafs contains (1) state nodes
// 3) Branches or leafs are loaded
//
// So, if Hash is blank. Then nothing to load. In this case all (1), (2)
// and (3) states are equal. If the Hash is not blank, then the node
// has nested nodes (and non-zero length). Thus, if length is zero, then
// the Hash should be cleared
//
// If HashTableIndex, EntireTree or EntireMerkleTree flags set,
// then all nodes should be loaded to (3) state if possible

func (r *Refs) isLoaded() bool {
	if r.Hash == (cipher.SHA256{}) {
		return true
	}
	return r.length > 0
}

func (r *Refs) isRoot() bool {
	return r.upper == nil
}

// setup wn field of leaf-Ref
func (r *Refs) setupInternalsOfLeaf(leaf *RefsElem) {
	leaf.rn = r.rn
	leaf.upper = r
}

// the r is root
func (r *Refs) attach() {
	if r.depth == 0 {
		if len(r.leafs) == 0 {
			return
		}
		if up := r.leafs[0].upper; up == r {
			return // everything is ok already
		} else if up != nil {
			up.leafs = nil // clear
			up.rn = nil    // detach
		}
		for _, leaf := range r.leafs {
			leaf.upper = r // re-attach
		}
		return
	}
	if len(r.branches) == 0 {
		return
	}
	if up := r.branches[0].upper; up == r {
		return // everything is ok already
	} else if up != nil {
		up.branches = nil // clear
		up.rn = nil       // detach
	}
	for _, br := range r.branches {
		br.upper = r // re-attach
	}
}

// from (1) to (2)
func (r *Refs) load(depth int) (err error) {

	if r.rn == nil {
		return errors.New("Refs is not attached to Pack")
	}

	// since we are using non-pinter values we must to know,
	// are nested nodes points to the r
	r.attach()

	if r.isLoaded() {
		return // already loaded
	}

	if r.isRoot() && r.rn.pack.flags&HashTableIndex != 0 {
		r.rn.index = make(map[cipher.SHA256]*RefsElem)
	}

	var val []byte
	if val, err = r.rn.pack.get(r.Hash); err != nil {
		return
	}

	var nested []cipher.SHA256 // nested nodes

	if r.isRoot() {
		var er encodedRefs
		if err = encoder.DeserializeRaw(val, &er); err != nil {
			return
		}
		r.depth = int(er.Depth)   // get depth from database
		r.degree = int(er.Degree) // get degree from database
		r.length = int(er.Length) // get length from database

		depth = r.depth // it's root and we are using depth from database

		nested = er.Nested // nested nodes or leafs
	} else {
		var ern encodedRefsNode
		if err = encoder.DeserializeRaw(val, &ern); err != nil {
			return
		}
		r.length = int(ern.Length) // length of the node

		nested = ern.Nested // nested nodes
	}

	if depth == 0 {

		// the nested contains leafs

		r.leafs = make([]*RefsElem, 0, len(nested))

		for _, hash := range nested {

			leaf := new(RefsElem)
			leaf.Hash = hash
			r.setupInternalsOfLeaf(leaf)

			r.leafs = append(r.leafs, leaf)

			if r.rn.pack.flags&HashTableIndex != 0 {
				r.rn.index[hash] = leaf
			}

			if r.rn.pack.flags&EntireTree != 0 {
				if _, err = leaf.Value(); err != nil { // load value of the leaf
					return
				}
			}

		}

		return
	}

	// the nested contains branches

	r.branches = make([]*Refs, 0, len(nested))

	for _, hash := range nested {

		branch := new(Refs)

		branch.Hash = hash

		branch.rn = r.rn
		branch.upper = r

		r.branches = append(r.branches, branch)

		if r.rn.pack.flags&(EntireTree|EntireMerkleTree|HashTableIndex) != 0 {
			// go deepper
			if err = branch.load(depth - 1); err != nil {
				return
			}
		}

	}

	return
}

// load a Refs and get it length (or error)
func (r *Refs) len(depth int) (ln int, err error) {
	if err = r.load(depth); err != nil {
		return
	}
	return r.length, nil
}

// Len returns length of the Refs. The length is amount of
// non-blank elements. It returns error if the Refs can't
// be loaded from database
func (r *Refs) Len() (ln int, err error) {
	return r.len(0)
}

// RefByIndex returns element by index. To delete an element
// by index use RefByIndex and then Clear result (if found).
// You also can set another value using this approach.
func (r *Refs) RefByIndex(i int) (ref *RefsElem, err error) {
	if err = r.load(0); err != nil {
		return
	}
	if err = validateIndex(i, r.length); err != nil {
		return
	}
	ref, err = r.refByIndex(r.depth, i)
	return
}

func (r *Refs) refByIndex(depth, i int) (ref *RefsElem, err error) {

	// the r is loaded
	// i >= 0

	if depth == 0 {
		// i is index in r.leafs
		if i >= len(r.leafs) {
			err = fmt.Errorf(
				"malformed tree: not enough leafs in {%s}, leafs: %d, want: %d",
				r.Hash.Hex()[:7], len(r.leafs), i)
			return
		}

		ref = r.leafs[i]
		return
	}

	// find branch subtracting length

	// so, we need to load branches from database to
	// get actual lengths

	for _, br := range r.branches {

		var ln int // length
		if ln, err = br.len(depth - 1); err != nil {
			return
		}

		// so if i less then the ln, then the br is branch we are looking for
		if i < ln {
			// the br is loaded by the len call above
			ref, err = br.refByIndex(depth-1, i) //
			return                               // done
		}

		i -= ln // subtract and go further
	}

	// TODO (kostyarin): explain the error, add hash and any other information
	//                   to find broken branch
	err = errors.New("malformed tree: index not found in branches")
	return
}

func (r *Refs) refByHashWihtoutHashTable(hash cipher.SHA256) (i int,
	needle *RefsElem, err error) {

	if r.length == 0 {
		err = ErrNotFound
		return // 0, nil, {not found}
	}

	_, err = r.descend(r.depth, r.length-1,
		func(k int, el *RefsElem) (_ error) {
			if el.Hash == hash {
				needle, i = el, k
				return ErrStopIteration
			}
			return
		})
	if err == ErrStopIteration {
		err = nil
	}
	if err == nil && needle == nil {
		err = ErrNotFound
	}
	return
}

// RefByHash returns Ref by its hash. If HashTableIndex flag
// is set, and there are many values with the same hash,
// then which value will be retuned is not defined. If the
// flag is not set, then it returns last value
func (r *Refs) RefByHash(hash cipher.SHA256) (needle *RefsElem, err error) {

	if err = r.load(0); err != nil {
		return
	}

	if r.rn.pack.flags&HashTableIndex != 0 {
		if needle = r.rn.index[hash]; needle == nil {
			err = ErrNotFound
		}
		return
	}

	_, needle, err = r.refByHashWihtoutHashTable(hash)
	return
}

// RefByHashWithIndex returns RefsElem by its hash and index of the
// element. It returns ErrNotFound if the needle has not been found
func (r *Refs) RefByHashWithIndex(hash cipher.SHA256) (i int, needle *RefsElem,
	err error) {

	if err = r.load(0); err != nil {
		return
	}

	if r.rn.pack.flags&HashTableIndex != 0 {
		var ok bool
		if needle, ok = r.rn.index[hash]; !ok {
			err = ErrNotFound
			return // 0, nil, ErrNotFound
		}
		// walk up
		if needle.upper == nil {
			err = errors.New("the RefsElem is detached from Refs-tree" +
				" (it shoud not happen, please contact CXO developer)")
			return
		}
		// upper of the needle contans leafs,
		// all nodes above contains branches

		// check the needle.upper
		for _, leaf := range needle.upper.leafs {
			if leaf == needle {
				break
			}
			i++
		}

		cur := needle.upper // the node that contains leafs (the needle holder)
		if cur.upper == nil {
			return // we're done
		}

		// the cur already checked out, take a look above

		for up := cur.upper; up != nil; cur, up = up, up.upper {
			// so, we need to check out branches only,
			// because the needle.upper contains leafs, but
			// all nodes above contains brances

			for _, br := range up.branches {
				if br == cur {
					break
				}
				i += br.length
			}

		}
		return
	}

	i, needle, err = r.refByHashWihtoutHashTable(hash)
	return
}

// reduce depth, the r is root, the r is loaded
func (r *Refs) reduceIfNeed() (err error) {
	if r.rn.iterating == true {
		return // don't reduce we are inside Ascend or Descend
	}
	var depth = r.depth
	for ; depth > 0 && pow(r.degree, depth-1) >= r.length; depth-- {
	}
	if depth != r.depth {
		err = r.changeDepth(depth)
	}
	return
}

// An IterateRefsFunc represents iterator over Refs.
// Feel free to delete and modify elements inside the
// function. Don't modify the Refs using Append, Clear
// or similar inside the function. Use ErrStopIteration
// to stop the iteration
type IterateRefsFunc func(i int, ref *RefsElem) (err error)

// Ascend iterates over Refs ascending order. See also
// docs for IterateRefsFunc. It's safe to delete an element
// of the Refs inside the Ascend, but it's unsafe to
// append something to the Refs while the Append ecxecutes
func (r *Refs) Ascend(irf IterateRefsFunc) (err error) {

	if err = r.load(0); err != nil {
		return
	}

	// we can't rebuild the tree while iterating,
	// but we can rebuild it after

	r.rn.iterating = true                     // lock
	defer func() { r.rn.iterating = false }() // unlock

	_, err = r.ascend(r.depth, 0, irf)
	if err != nil && err != ErrStopIteration {
		return
	}

	r.rn.iterating = false // unlock
	err = r.reduceIfNeed() // overwrite the ErrStopIteration if so
	return
}

func (r *Refs) ascend(depth, i int, irf IterateRefsFunc) (pass int, err error) {

	// root already loaded
	if err = r.load(depth); err != nil {
		return
	}

	if r.length == 0 {
		return // nothing to iterate
	}

	if depth == 0 { // leafs
		for _, leaf := range r.leafs {
			if leaf.Hash == (cipher.SHA256{}) {
				continue
			}
			if err = irf(i, leaf); err != nil {
				return
			}
			i++
			pass++
		}
		return
	}

	// branches
	for _, br := range r.branches {
		if br.Hash == (cipher.SHA256{}) {
			continue
		}
		var subpass int
		if subpass, err = br.ascend(depth-1, i, irf); err != nil {
			return
		}
		i += subpass
		pass += subpass
	}

	return
}

// Descend iterates over Refs descending order. See also
// docs for IterateRefsFunc. It's safe to delete an element
// of the Refs inside the Ascend, but it's unsafe to
// append something to the Refs while the Append ecxecutes
func (r *Refs) Descend(irf IterateRefsFunc) (err error) {

	if err = r.load(0); err != nil {
		return
	}

	if r.length == 0 {
		return
	}

	// we can't rebuild the tree while iterating,
	// but we can rebuild it after

	r.rn.iterating = true
	defer func() { r.rn.iterating = false }()

	_, err = r.descend(r.depth, r.length-1, irf)
	if err != nil && err != ErrStopIteration {
		return
	}

	r.rn.iterating = false // unlock
	err = r.reduceIfNeed()

	return
}

func (r *Refs) descend(depth, i int, irf IterateRefsFunc) (pass int,
	err error) {

	// root already loaded

	if err = r.load(depth); err != nil {
		return
	}

	if r.length == 0 {
		return // nothing to iterate
	}

	if depth == 0 { // leafs
		for k := len(r.leafs) - 1; k >= 0; k-- {
			leaf := r.leafs[k]
			if leaf.Hash == (cipher.SHA256{}) {
				continue
			}
			if i < 0 {
				// TODO (kostyarin): detailed error
				err = errors.New(
					"negative index during Descend: malformed tree")
				return
			}

			if err = irf(i, leaf); err != nil {
				return
			}

			i--
			pass++
		}
		return
	}

	// branches

	for k := len(r.branches) - 1; k >= 0; k-- {
		br := r.branches[k]
		if br.Hash == (cipher.SHA256{}) {
			continue
		}
		if i < 0 {
			// TODO (kostyarin): detailed error
			err = errors.New(
				"negative index during Descend: malformed tree")
			return
		}

		var subpass int
		if subpass, err = br.descend(depth-1, i, irf); err != nil {
			return
		}

		i -= subpass
		pass += subpass
	}

	return
}

// free spece that we can use to append new items;
// for example the tree can contain 1000 elements
// and contains only 980, and the tree has 10
// zero elements; and this zero-elements not on tail;
// thus we can insert 10 elements only; if we need
// to insert more, then we need to rebuild the tree
// removing zero elements (and possible increasing
// depth); but if we have enough place on tail then
// we can aviod unnecessary rebuilding
//
// the fsot is free space on tail of the node
func (r *Refs) freeSpaceOnTail(degree, depth int) (fsot int, err error) {

	if err = r.load(depth); err != nil {
		return
	}

	// r.degree is amount o branches or leafs per node
	// thus every node can contain (r.degree ** (depth+1))
	// (we are using depth+1, because the depth is 'actual depth'-1)

	if depth == 0 {
		fsot = degree - len(r.leafs)
		return
	}

	if len(r.branches) == 0 {
		fsot = pow(degree, depth+1)
		return
	}

	// empty branches
	if eb := degree - len(r.branches); eb > 0 {
		fsot = eb * pow(degree, (depth-1)+1)
	}

	// check out last branch
	lastBranch := r.branches[len(r.branches)-1]

	var lb int
	if lb, err = lastBranch.freeSpaceOnTail(degree, depth-1); err != nil {
		return
	}

	fsot += lb
	return
}

// try to insert given reference to the node;
//
// the el is not blank
// the tryInsert doesn't cahnge 'length' and 'Hash' fields,
// thus we need to walk the tree (from tail or whole tree) and
// check the length and change hashes if the length has been changed
func (r *Refs) tryInsert(degree, depth int, el *RefsElem) (ok bool,
	err error) {

	if err = r.load(depth); err != nil {
		return
	}

	if depth == 0 {

		// we are not replacing zero elements with new

		if len(r.leafs) >= degree {
			return // can not
		}

		// setup the el and insert
		r.setupInternalsOfLeaf(el)

		if r.rn.pack.flags&HashTableIndex != 0 {
			r.rn.index[el.Hash] = el // insert to the index
		}

		r.leafs, ok = append(r.leafs, el), true // inserted
		return
	}

	// 1) find last branch and try to insert, return if ok
	// 2) create new branch if possible and insert

	if len(r.branches) > 0 {

		lb := r.branches[len(r.branches)-1]
		if ok, err = lb.tryInsert(degree, depth-1, el); err != nil || ok {
			return // success or error
		}

	}

	// is there a free space for new branch?

	if len(r.branches) >= degree {
		return // can not insert
	}

	// add new branch and insert

	nb := new(Refs) // new branch

	nb.rn = r.rn
	nb.upper = r

	if ok, err = nb.tryInsert(degree, depth-1, el); err != nil {
		return // never happens (a precaution)
	}

	if !ok {
		panic("fatality: bug in the code")
	}

	r.branches = append(r.branches, nb)
	return
}

// save the Refs (not recursive)
func (r *Refs) save(depth int) {

	var nl int                 // new length
	var nested []cipher.SHA256 // nested elements

	if depth == 0 {
		nested = make([]cipher.SHA256, 0, len(r.leafs))
		for _, leaf := range r.leafs {
			if leaf.Hash == (cipher.SHA256{}) {
				continue
			}
			nl++
			nested = append(nested, leaf.Hash)
		}
	} else {
		nested = make([]cipher.SHA256, 0, len(r.branches))
		for _, br := range r.branches {
			if br.Hash == (cipher.SHA256{}) {
				continue
			}
			nl += br.length
			nested = append(nested, br.Hash)
		}
	}

	r.length = nl // store actual (possible new) length
	if nl == 0 {
		// clear (don't save blank Refs)
		r.depth = 0
		r.leafs, r.branches = nil, nil
		if r.Hash != (cipher.SHA256{}) {
			r.Hash, r.ch = cipher.SHA256{}, true // changed
		}
		return
	}

	if r.isRoot() {
		// root Refs represented as encodedRefs
		var er encodedRefs            // encode to save and get hash
		er.Degree = uint32(r.degree)  // keep in database
		er.Depth = uint32(depth)      // keep in database
		er.Length = uint32(nl)        // keep in database
		er.Nested = nested            // actually, keep in database
		hash, _ := r.rn.pack.save(er) // save into cache
		if hash != r.Hash {
			r.Hash, r.ch = hash, true
		}
	} else {
		// non-root Refs represented as encodedRefsNode
		var ern encodedRefsNode
		ern.Nested = nested            // keep in database
		ern.Length = uint32(nl)        // keep in daabase
		hash, _ := r.rn.pack.save(ern) // save into cache
		if hash != r.Hash {
			r.Hash, r.ch = hash, true
		}
	}
	return
}

// walk from tail to fild nodes that has incorrect 'Hash' and 'length'
// fields to set proper; the method doesn't check hashes of nested
// nodes or leafs, it checks only lengths; e.g. the method is useful
// after insert (Append or cahngeDepth); it can't be used after Ascend
// or Descend (if someone cahnges or removes elements while iterating),
// because it walks from tail and stops on first correct (unchanged) node
func (r *Refs) updateHashLengthFieldsFromTail(depth int) (set bool, err error) {

	if err = r.load(depth); err != nil {
		return
	}

	var al int // actual length

	if depth == 0 {
		for _, leaf := range r.leafs {
			if leaf.Hash == (cipher.SHA256{}) {
				continue
			}
			al++
		}
		if al != r.length { // has been changed
			r.length, set = al, true
			r.save(depth)
		}
		return
	}

	var bset = true // set of branch (set to true to check first from tail)

	for i := len(r.branches) - 1; i >= 0; i-- {
		br := r.branches[i]
		if br.Hash == (cipher.SHA256{}) {
			if len(br.branches) == 0 && len(br.leafs) == 0 {
				continue
			}
		}
		if bset {
			// we here only if this branch is first from tail or
			// previous (from tail) branch has been updated
			bset, err = br.updateHashLengthFieldsFromTail(depth - 1)
			if err != nil {
				return
			}
			set = set || bset
		}
		al += br.length // actual length
	}

	if al != r.length || set { // has been changed
		r.length, set = al, true
		r.save(depth)
	}

	return
}

// change depth of the tree or remove zero elements
//
// the r is loaded
func (r *Refs) changeDepth(depth int) (err error) {

	if depth < 0 {
		return errors.New("negative depth: malformd tree or bug")
	}

	var nr Refs // new Refs

	nr.depth = depth
	nr.degree = r.degree
	nr.rn = &refsNode{
		pack: r.rn.pack,
		sch:  r.rn.sch,
	}

	if r.rn.pack.flags&HashTableIndex != 0 {
		nr.rn.index = make(map[cipher.SHA256]*RefsElem)
	}

	// 1) create branches (keep 'Hash' and 'length' fields blank)
	// 2) walk throut the tree and set proper 'Hash' and 'length' fields

	_, err = r.ascend(r.depth, 0, func(_ int, ref *RefsElem) (err error) {
		var ok bool
		if ok, err = nr.tryInsert(nr.degree, nr.depth, ref); err != nil {
			return
		}
		if !ok {
			panic("can't insert to freshly created tree: bug")
		}
		return
	})

	if err != nil {
		return
	}

	// let's walk through entire nr to set proper 'length' and 'Hash' fields,
	// saving new elements (encoded nodes and root-Refs)
	if _, err = nr.updateHashLengthFieldsFromTail(nr.depth); err != nil {
		return
	}

	*r = nr     // replace the r with nr
	r.ch = true // has been changed

	// set proper upper
	if r.depth == 0 {
		for _, leaf := range r.leafs {
			leaf.upper = r
		}
		return
	}
	for _, br := range r.branches {
		br.upper = r
	}

	return
}

// Schema of the Refs. The method can returns nil if
// the Refs are not attached
func (r *Refs) Schema() (sch Schema) {
	if r.rn == nil {
		return
	}
	return r.rn.sch
}

// Append obejcts to the Refs. Type of the objects must
// be the same as type of the Refs. Nils will be skipped,
// even if it's nil of some type or nil-interface{}
func (r *Refs) Append(objs ...interface{}) (err error) {

	if len(objs) == 0 {
		return // nothing to append
	}

	if err = r.load(0); err != nil {
		return
	}

	if r.rn.pack.flags&ViewOnly != 0 {
		return ErrViewOnlyTree
	}

	// rid out of nils

	// os - Schema of an object
	// ns - new schema of the Refs if it's nil for now

	var os, ns Schema

	var i int
	for _, obj := range objs {
		if obj == nil {
			continue // delete from the objs
		}

		if os, obj, err = r.rn.pack.initialize(obj); err != nil {
			return
		}

		// check out schemas

		if r.rn.sch != nil {
			if r.rn.sch != os {
				err = fmt.Errorf(
					"can't Append object of another type: want %q, got %q",
					r.rn.sch.String(), os.String())
				return
			}
		} else if ns != nil && ns != os {
			err = fmt.Errorf("can't append obejct of differetn types: %q, %q",
				ns.String(), os.String())
			return
		} else {
			// r.wn.sch == nil
			// ns == nil
			// so, let's initialize the ns
			ns = os
		}

		if obj == nil {
			continue // delete from the objs
		}

		objs[i] = obj
		i++
	}
	objs = objs[:i] // delete all deleted

	if len(objs) == 0 {
		return // only nils
	}

	if r.rn.sch == nil {
		r.rn.sch = ns // this will be schema of the Refs
	}

	// do we need to rebuild the tree to place new elements
	var fsot int
	if fsot, err = r.freeSpaceOnTail(r.degree, r.depth); err != nil {
		return
	}

	if fsot < len(objs) {
		// so, we need to rebuild the tree removing zero elements or
		// increasing depth (and removing the zero elements, actually)

		// new depth, required space
		var depth, required int = r.depth, r.length + len(objs)
		for ; pow(r.degree, depth) < required; depth++ {
		}

		// rebuild the tree, depth can be the same
		if err = r.changeDepth(depth); err != nil {
			return
		}

	}

	// append
	var ok bool
	for _, obj := range objs {
		ref := new(RefsElem)

		ref.rn = r.rn                     //
		ref.value = obj                   // keep initialized value
		ref.Hash, _ = r.rn.pack.save(obj) // set Hash field, saving object
		ref.ch = true                     // fresh non-blank

		if ok, err = r.tryInsert(r.degree, r.depth, ref); err != nil {
			return
		}
		if !ok {
			panic("can't insert inside Append: bug")
		}
	}

	// walk from tail to update Hash and length fields of the
	// tree after appending, saving new and changed elements
	_, err = r.updateHashLengthFieldsFromTail(r.depth)
	return
}

// AppendHashes to the Refs. Type of the objects must
// be the same as type of the Refs. It never checked
// Blank hashes will be skipped
func (r *Refs) AppendHashes(hashes ...cipher.SHA256) (err error) {

	if len(hashes) == 0 {
		return // nothing to append
	}

	if err = r.load(0); err != nil {
		return
	}

	if r.rn.pack.flags&ViewOnly != 0 {
		return ErrViewOnlyTree
	}

	// rid out of blank hashes
	var i int
	for _, hash := range hashes {
		if hash == (cipher.SHA256{}) {
			continue // delte from the hashes
		}
		hashes[i] = hash
		i++
	}
	hashes = hashes[:i] // delete all deleted

	if len(hashes) == 0 {
		return // only blank
	}

	// do we need to rebuild the tree to place new elements
	var fsot int
	if fsot, err = r.freeSpaceOnTail(r.degree, r.depth); err != nil {
		return
	}

	if fsot < len(hashes) {
		// so, we need to rebuild the tree removing zero elements or
		// increasing depth (and removing the zero elements, actually)

		// new depth, required space
		var depth, required int = r.depth, r.length + len(hashes)
		for ; pow(r.degree, depth) < required; depth++ {
		}

		// rebuild the tree, depth can be the same
		if err = r.changeDepth(depth); err != nil {
			return
		}

	}

	// append
	var ok bool
	for _, hash := range hashes {
		el := new(RefsElem)

		el.rn = r.rn   //
		el.Hash = hash //

		if ok, err = r.tryInsert(r.degree, r.depth, el); err != nil {
			return
		}
		if !ok {
			panic("can't insert inside Append: bug")
		}
	}

	// walk from tail to update Hash and length fields of the
	// tree after appending, saving new and changed elements
	_, err = r.updateHashLengthFieldsFromTail(r.depth)
	return
}

// TODO (kostyarin): implement Slice(i, j int) (cut *Refs, err error)

// Clear the Refs making them blank
func (r *Refs) Clear() {
	if r.isLoaded() {
		// detach nested form the Refs for golang GC,
		// and to avoid potential memory leaks, and,
		// mainly, to protect the Refs from bubbling
		// changes
		if r.depth == 0 {
			for _, leaf := range r.leafs {
				leaf.upper = nil // detach from Refs
				leaf.rn = nil
			}
		} else {
			for _, br := range r.branches {
				br.upper = nil
				br.rn = nil
			}
		}
	}
	r.Hash = cipher.SHA256{}
	r.ch = true
	r.length = 0
	r.depth = 0
	if r.rn.pack.flags&HashTableIndex != 0 {
		r.rn.index = make(map[cipher.SHA256]*RefsElem) // clear
	}
	return
}

// utils

// bubble changes up
func (r *Refs) bubble() (err error) {
	stack := []*Refs{}

	// push unsaved nodes to the stack
	var up *Refs
	for up = r; up != nil; up = up.upper {
		stack = append(stack, up)
	}

	root := stack[len(stack)-1]
	if root.depth != len(stack)-1 {
		// it's detached branch of Refs
		return
	}

	// index is depth
	for i, br := range stack {
		br.save(i)
	}

	// the bubble can be triggered by deleteing
	// and length o the tree can be changed,
	// thus we need to rebuild the tree if it
	// contains a lot of zero-elements

	// up keeps root Refs
	err = root.reduceIfNeed()
	return
}

// max possible non-zero items, the Refs must be root and loaded
// the depth is depth-1 (because it can be 0) and we're using
// depth+1 to calculate the items
func (r *Refs) items() int {
	return pow(r.degree, r.depth+1)
}

// encoded representation of Refs
type encodedRefs struct {
	Depth  uint32          // depth of the Refs (depth-1)
	Degree uint32          // deger of the Refs
	Length uint32          // total elements
	Nested []cipher.SHA256 // hashes of nested objects (leafs or ndoes)
}

type encodedRefsNode struct {
	Length uint32 // total elements
	Nested []cipher.SHA256
}

// a**b
func pow(a, b int) (p int) {
	p = 1
	for b > 0 {
		if b&1 != 0 {
			p *= a
		}
		b >>= 1
		a *= a
	}
	return p
}

// ------ ------ ------ ------ ------ ------ ------ ------ ------ ------ ------

// print debug tree

// DebugString returns string that represents the Refs as
// is. E.g. as Merkle-tree. If the tree is not loaded then
// this mehtod prints only loaded parts. To load all branches
// to print pass true
func (r *Refs) DebugString(forceLoad bool) string {
	var gt gotree.GTStructure
	gt.Name = "[](refs) " + r.Short() + " " + fmt.Sprint(r.length)

	if !forceLoad && !r.isLoaded() {
		gt.Name += " (not loaded)"
		return gotree.StringTree(gt)
	}

	if err := r.load(0); err != nil {
		gt.Items = []gotree.GTStructure{{Name: "(error): " + err.Error()}}
		return gotree.StringTree(gt)
	}

	gt.Items = r.debugItems(forceLoad, r.depth)
	return gotree.StringTree(gt)
}

func (r *Refs) debugItems(forceLoad bool,
	depth int) (its []gotree.GTStructure) {

	if forceLoad {
		if err := r.load(depth); err != nil {
			return []gotree.GTStructure{{Name: "(error): " + err.Error()}}
		}
	}

	if depth == 0 {
		for _, leaf := range r.leafs {
			its = append(its, gotree.GTStructure{
				Name: "*(ref) " + leaf.Short(),
			})
		}
		return
	}

	for _, br := range r.branches {
		its = append(its, gotree.GTStructure{
			Name:  br.Hash.Hex()[:7] + " " + fmt.Sprint(br.length),
			Items: br.debugItems(forceLoad, depth-1),
		})
	}
	return

}

// ------ ------ ------ ------ ------ ------ ------ ------ ------ ------ ------

// RefsElem

// RefsElem is element of Refs
type RefsElem struct {
	Hash cipher.SHA256 // hash of the RefsElem

	// internals
	value interface{} // golang-value of the RefsElem
	rn    *refsNode   // related
	upper *Refs       // upper node
	ch    bool        // has been changed
}

// Short string
func (r *RefsElem) Short() string {
	return r.Hash.Hex()[:7]
}

// String implements fmt.Stringer interface
func (r *RefsElem) String() string {
	return r.Hash.Hex()
}

// SetHash replaces hash of the RefsElem with given one
func (r *RefsElem) SetHash(hash cipher.SHA256) (err error) {
	if r.Hash == hash {
		return
	}
	if hash == (cipher.SHA256{}) {
		return r.Delete()
	}

	if r.rn != nil && r.rn.pack.flags&HashTableIndex != 0 {
		delete(r.rn.index, r.Hash)
		r.rn.index[hash] = r
	}

	r.value = nil // clear related value
	r.Hash = hash
	r.ch = true
	return
}

// Delete the RefsElem from related Refs
func (r *RefsElem) Delete() (err error) {
	if r.Hash == (cipher.SHA256{}) {
		return
	}

	if r.rn != nil && r.rn.pack.flags&HashTableIndex != 0 {
		delete(r.rn.index, r.Hash)
	}

	r.Hash = (cipher.SHA256{})

	if r.upper != nil {
		err = r.upper.bubble()
	}

	r.value = nil
	r.rn = nil // detach
	r.upper = nil
	r.ch = true // changed
	return
}

// Value of the RefsElem. If err is nil, then obj is not
func (r *RefsElem) Value() (obj interface{}, err error) {

	// copy pasted from Ref.Value with little changes
	//
	// TODO (kostyarin): DRY with the Ref.Value

	if r.rn == nil {
		err = errors.New(
			"can't get value: the RefsElem is not attached to Pack")
		return
	}

	if r.value != nil {
		obj = r.value // already have
		return
	}

	if r.rn.sch == nil {
		err = errors.New("can't get value: Schema of related Refs is nil")
		return
	}

	// obtain encoded object
	var val []byte
	if val, err = r.rn.pack.get(r.Hash); err != nil {
		return
	}

	// unpack and setup
	if obj, err = r.rn.pack.unpackToGo(r.rn.sch.Name(), val); err != nil {
		return
	}
	r.value = obj // keep

	return
}

// SetValue of the element. A nil removes this
// element from related Refs
func (r *RefsElem) SetValue(obj interface{}) (err error) {

	// copy-pasted from Ref.SetValue with little changes
	//
	// TODO (kostyarin): DRY with Ref.SetValue

	if obj == nil {
		return r.Delete()
	}

	if r.rn == nil {
		return errors.New(
			"can't set value: the RefsElem is not attached to Pack")
	}

	if r.upper == nil {
		return errors.New(
			"can't set value: the RefsElem is not attached to Refs")
	}

	if r.rn.pack.flags&ViewOnly != 0 {
		return ErrViewOnlyTree
	}

	var sch Schema
	if sch, obj, err = r.rn.pack.initialize(obj); err != nil {
		return fmt.Errorf("can't set value to RefsElem: %v", err)
	}

	if r.rn.sch != nil && r.rn.sch != sch {
		return fmt.Errorf(`can't set value: type of given object "%T"`+
			" is not type of the Ref %q", obj, r.rn.sch.String())
	}

	if obj == nil {
		return r.Delete() // the obj was a nil pointer of some type
	}

	r.rn.sch = sch // keep schema (if it was nil or the same)
	r.value = obj  // keep

	if key, val := r.rn.pack.dsave(obj); key != r.Hash {

		if r.rn.pack.flags&HashTableIndex != 0 {
			if r.Hash != (cipher.SHA256{}) {
				delete(r.rn.index, r.Hash)
			}
			r.rn.index[key] = r
		}

		r.Hash = key
		r.ch = true
		r.rn.pack.set(key, val) // save

		if err = r.upper.bubble(); err != nil {
			return
		}
	}

	return
}

// Save updates hash using underlying value.
// The vaue can be changd from outside, this
// method encodes it and updates hash
func (r *RefsElem) Save() (err error) {
	if r.rn == nil || r.value == nil || r.upper == nil {
		return // detached or not loaded
	}

	key, val := r.rn.pack.dsave(r.value) // don't save - get key-value pair
	if key == r.Hash {
		return // exactly the same
	}

	if r.rn.pack.flags&HashTableIndex != 0 {
		if r.Hash != (cipher.SHA256{}) {
			delete(r.rn.index, r.Hash)
		}
		r.rn.index[key] = r
	}

	r.rn.pack.set(key, val) // save
	r.Hash, r.ch = key, false

	err = r.upper.bubble() // bubble up
	return
}


*/
