package skyobject

import (
	"errors"
	"fmt"

	"github.com/disiqueira/gotree"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// A Refs represetns array of Ref
type Refs struct {
	Hash cipher.SHA256 // hash of the refsNode (empty for nodess)

	depth  int `enc:"-"` // not stored
	degree int `enc:"-"` // not stored

	length   int     `enc:"-"`
	branches []*Refs `enc:"-"` // nodes
	leafs    []*Ref  `enc:"-"`

	upper *Refs                  `enc:"-"` // upper node (nil for root)
	wn    *walkNode              `enc:"-"` // wn of Refs
	index map[cipher.SHA256]*Ref `enc:"-"` // index (or reference)

	reduceLock bool `enc:"-"` // don't rebuild the tree in some cases
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
// has nested nodes (and non-zero length). Thus if lenght is zero, then
// the Hash should be cleared
//
// If HashTableIndex, EntireTree or EntireMerkleTree flags set
// the all nodes should be loaded to (3) state if possible

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
func (r *Refs) setupInternalsOfLeaf(leaf *Ref) {
	if leaf.wn != nil {
		return // already set up
	}
	leaf.wn = &walkNode{
		upper: r,
		sch:   r.wn.sch,
		pack:  r.wn.pack,
	}
}

// from (1) to (2)
func (r *Refs) load(depth int) (err error) {

	if r.isLoaded() {
		return // already loaded
	}

	if r.wn == nil {
		return errors.New("Refs is not attached to Pack")
	}

	if r.isRoot() && r.wn.pack.flags&HashTableIndex != 0 {
		r.index = make(map[cipher.SHA256]*Ref)
	}

	// the Refs can be fresh and empty
	if r.Hash == (cipher.SHA256{}) {
		// degree set by (*Pack).refs()
		// nothing to load
		return
	}

	var val []byte
	if val, err = r.wn.pack.get(r.Hash); err != nil {
		return
	}

	var er encodedRefs
	if err = encoder.DeserializeRaw(val, &er); err != nil {
		return
	}

	r.depth = int(er.Depth)   // 0 if the r is not root of Refs
	r.degree = int(er.Degree) // 0 if the r is not root of Refs
	r.length = int(er.Length) // actual value

	if r.isRoot() {
		depth = r.depth // this r is root and we're using depth from database
	}

	if depth == 0 {

		// er.Nested contains leafs

		r.leafs = make([]*Ref, 0, len(er.Nested))

		for _, hash := range er.Nested {

			leaf := new(Ref)
			leaf.Hash = hash

			r.leafs = append(r.leafs, leaf)

			if r.wn.pack.flags&HashTableIndex != 0 {
				r.index[hash] = leaf
			}

			if r.wn.pack.flags&EntireTree != 0 {

				r.setupInternalsOfLeaf(leaf) // create walkNode of the leaf

				if _, err = leaf.Value(); err != nil { // load value of the leaf
					return
				}

			}

		}

		return

	}

	// er.Nested contains branches

	r.branches = make([]*Refs, 0, len(er.Nested))

	for _, hash := range er.Nested {

		branch := new(Refs)

		branch.Hash = hash

		branch.wn = r.wn
		branch.index = r.index
		branch.upper = r

		r.branches = append(r.branches, branch)

		if r.wn.pack.flags&(EntireTree|EntireMerkleTree) != 0 {

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
// You also can set another value using this approach
func (r *Refs) RefBiIndex(i int) (ref *Ref, err error) {
	if err = r.load(0); err != nil {
		return
	}
	if err = validateIndex(i, r.length); err != nil {
		return
	}
	ref, err = r.refByIndex(r.depth, i)
	return
}

func (r *Refs) refByIndex(depth, i int) (ref *Ref, err error) {

	// the r is loaded
	// i >= 0

	if r.depth == 0 {
		// i is index in r.leafs
		if i >= len(r.leafs) {
			err = fmt.Errorf(
				"malformed tree: not enough leafs in {%s}, leafs: %d, want: %d",
				r.Hash.Hex()[:7], len(r.leafs), i)
			return
		}

		ref = r.leafs[i]
		r.setupInternalsOfLeaf(ref)
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

// RefByHash returns Ref by its hash. It returns first Ref.
// The call is O(1) if HashTableIndex flag set. Otherwise
// the call is O(n). It returns (nil, nil) if requested Ref
// not found
func (r *Refs) RefByHash(hash cipher.SHA256) (needle *Ref, err error) {

	if err = r.load(0); err != nil {
		return
	}

	if r.wn.pack.flags&HashTableIndex != 0 {
		needle = r.index[hash]
		return
	}

	_, err = r.ascend(r.depth, 0, func(_ int, ref *Ref) (_ error) {
		if ref.Hash == hash {
			needle = ref
			return ErrStopIteration
		}
		return
	})
	return
}

// reduce depth, the r is root, the r is loaded
func (r *Refs) reduceIfNeed() (err error) {
	if r.reduceLock == true {
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
type IterateRefsFunc func(i int, ref *Ref) (err error)

// Ascend iterates over Refs ascending order. See also
// docs for IterateRefsFunc
func (r *Refs) Ascend(irf IterateRefsFunc) (err error) {

	if err = r.load(0); err != nil {
		return
	}

	// we can't rebuild the tree while iterating,
	// but we can rebuild it after

	r.reduceLock = true
	defer func() { r.reduceLock = false }() // unlock

	_, err = r.ascend(r.depth, r.length-1, irf)
	if err != nil && err != ErrStopIteration {
		return
	}

	r.reduceLock = false // unlock
	err = r.reduceIfNeed()
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
			r.setupInternalsOfLeaf(leaf)
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
// docs for IterateRefsFunc
func (r *Refs) Descend(irf IterateRefsFunc) (err error) {

	if err = r.load(0); err != nil {
		return
	}

	if r.length == 0 {
		return
	}

	// we can't rebuild the tree while iterating,
	// but we can rebuild it after

	r.reduceLock = true
	defer func() { r.reduceLock = false }()

	_, err = r.descend(r.depth, r.length-1, irf)
	if err != nil && err != ErrStopIteration {
		return
	}

	r.reduceLock = false // unlock
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
		for _, leaf := range r.leafs {
			if i < 0 {
				// TODO (kostyarin): detailed error
				err = errors.New(
					"negative index during Descend: malformed tree")
				return
			}

			r.setupInternalsOfLeaf(leaf)
			if err = irf(i, leaf); err != nil {
				return
			}

			i--
			pass++
		}
		return
	}

	// branches

	for _, br := range r.branches {
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

// free spece taht we can use to append new items;
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
// the ref is not blank
// the tryInsert doesn't cahnge 'length' and 'Hash' fields,
// thus we need to walk the tree (from tail or whole tree) and
// check the length and change hashes if the length has been changed
func (r *Refs) tryInsert(degree, depth int, ref *Ref) (ok bool, err error) {

	if err = r.load(depth); err != nil {
		return
	}

	if depth == 0 {

		// we are not replacing zero elements with new

		if len(r.leafs) >= degree {
			return // can not
		}

		// setup the ref and insert
		pwn := ref.wn
		r.setupInternalsOfLeaf(ref)
		if pwn != nil && pwn.value != nil {
			ref.wn.value = pwn.value // keep the value
		}

		if r.wn.pack.flags&HashTableIndex != 0 {
			r.index[ref.Hash] = ref // insert to the index
		}

		r.leafs, ok = append(r.leafs, ref), true // inserted
		return
	}

	// 1) find last branch and try to insert, return if ok
	// 2) create new branch if possible and insert

	if len(r.branches) > 0 {

		lb := r.branches[len(r.branches)-1]
		if ok, err = lb.tryInsert(degree, depth-1, ref); err != nil || ok {
			return // success or error
		}

	}

	// is there a free space for new branch?

	if len(r.branches) >= degree {
		return // can not insert
	}

	// add new branch and insert

	nb := new(Refs) // new branch

	nb.wn = r.wn
	nb.index = r.index
	nb.upper = r

	if ok, err = nb.tryInsert(degree, depth-1, ref); err != nil {
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

	var nl int         // new length
	var er encodedRefs // encode to save and get hash

	if depth == 0 {
		er.Nested = make([]cipher.SHA256, 0, len(r.leafs))
		for _, leaf := range r.leafs {
			if leaf.Hash == (cipher.SHA256{}) {
				continue
			}
			nl++
			er.Nested = append(er.Nested, leaf.Hash)
		}
	} else {
		er.Nested = make([]cipher.SHA256, 0, len(r.branches))
		for _, br := range r.branches {
			if br.Hash == (cipher.SHA256{}) {
				continue
			}
			nl += br.length
			er.Nested = append(er.Nested, br.Hash)
		}
	}

	er.Length, r.length = uint32(nl), nl

	if r.isRoot() {
		er.Degree = uint32(r.degree)
		er.Depth = uint32(depth)
	}

	r.Hash, _ = r.wn.pack.save(er)
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

	var bset bool = true // set of branch (set to true to check first from tail)

	for i := len(r.branches) - 1; i >= 0; i-- {
		br := r.branches[i]
		if br.Hash == (cipher.SHA256{}) {
			continue
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

	if r.wn.pack.flags&HashTableIndex != 0 {
		nr.index = make(map[cipher.SHA256]*Ref)
	}

	// 1) create branches (keep 'Hash' and 'length' fields blank)
	// 2) walk throut the tree and set proper 'Hash' and 'length' fields

	_, err = r.ascend(r.depth, 0, func(_ int, ref *Ref) (err error) {
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

	*r = nr // replace the r with nr
	return
}

// Append obejcts to the Refs. Type of the objects must
// be the same as type of the Refs. Nils will be skipped,
// even if it's nil of some type or nil-interace{}
func (r *Refs) Append(objs ...interface{}) (err error) {

	if len(objs) == 0 {
		return // nothing to append
	}

	if err = r.load(0); err != nil {
		return
	}

	if r.wn.pack.flags&ViewOnly != 0 {
		return ErrViewOnlyTree
	}

	// rid out of nils

	// os - Schema of an object
	// ns - new schema of the Refs if it's nil for now

	var os, ns Schema

	var i int
	for _, obj := range objs {
		if obj == nil {
			continue // delte from the objs
		}

		if os, obj, err = r.wn.pack.initialize(obj); err != nil {
			return
		}

		// check out schemas

		if r.wn.sch != nil {
			if r.wn.sch != os {
				err = fmt.Errorf(
					"can't Append object of another type: want %q, got %q",
					r.wn.sch.String(), os.String())
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

	if r.wn.sch == nil {
		r.wn.sch = ns // this will be schema of the Refs
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
		ref := new(Ref)

		ref.wn = &walkNode{value: obj}    // keep initialized value
		ref.Hash, _ = r.wn.pack.save(obj) // set Hash field, saving object

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

// TODO (kostyarin): implement Slice(i, j int) (cut *Refs, err error)

// Clear the Refs
func (r *Refs) Clear() {
	if r.isLoaded() {
		// detach nested form the Refs for golang GC,
		// and to avoid potential memory leaks, and,
		// mainly, to protect the Refs from bubbling
		// changes
		if r.depth == 0 {
			for _, leaf := range r.leafs {
				if leaf.wn != nil {
					leaf.wn.upper = nil
				}
			}
		} else {
			for _, br := range r.branches {
				br.index = nil
				br.upper = nil
				br.wn = nil
			}
		}
	}
	r.Hash = cipher.SHA256{}
	r.length = 0
	r.depth = 0
	if r.wn.pack.flags&HashTableIndex != 0 {
		r.index = make(map[cipher.SHA256]*Ref) // clear
	}
	return
}

// utils

// unsave implements unsaver interface
func (r *Refs) unsave() (err error) {
	stack := []*Refs{}

	// push unsaved nodes to the stack
	var up *Refs
	for up = r; up != nil; up = up.upper {
		stack = append(stack, up)
	}

	// now the 'up' keeps root or pre-root node if this
	// Refs is part of detached tree (after Clear)
	if up.wn == nil {
		// this is detached part of tree
		return
	}

	// index is depth
	for i, br := range stack {
		br.save(i)
	}

	// the unsave call can be triggered by deleteing
	// and length o the tree can be changed,
	// thus we need to rebuild the tree if it
	// contains a lot of zero-elements

	// up keeps root Refs
	err = up.reduceIfNeed()
	return
}

// max possible non-zero items, the Refs must be root and loaded
// the depth is depth-1 (because it can be 0) and we're using
// depth+1 to calculate the items
func (r *Refs) items() int {
	return pow(r.degree, r.depth+1)
}

// encoded representation of Refs and nodes of the Refs
type encodedRefs struct {
	Depth  uint32          // actual only for Refs, it is zero for nodes of Refs
	Degree uint32          // actual only for Refs, it is zero for nodes of Refs
	Length uint32          // actual for Refs and nodes
	Nested []cipher.SHA256 // hashes of nested objects (leafs or ndoes)
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

// don't need to commit the Refs
func (r *Refs) commit() (_ error) {
	if r.wn == nil {
		panic("commit not initialized Refs")
	}
	return
}

// ------ ------ ------ ------ ------ ------ ------ ------ ------ ------ ------

// print debug tree

// DebugString returns string that represents the Refs as
// is. E.g. as Merkle-tree. If the tree is not loaded then
// this mehtod prints only loaded parts. To load all branches
// to print pass true
func (r *Refs) DebugString(forceLoad bool) string {
	var gt gotree.GTStructure
	gt.Name = "*(refs) " + r.Short()

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
			its = append(its, gotree.GTStructure{"*(ref) " + leaf.Short(), nil})
		}
		return
	}

	for _, br := range r.branches {
		its = append(its, gotree.GTStructure{
			Name:  br.Hash.Hex()[:7],
			Items: br.debugItems(forceLoad, depth-1),
		})
	}
	return

}
