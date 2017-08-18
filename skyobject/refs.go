package skyobject

import (
	"errors"
	"fmt"

	"github.com/disiqueira/gotree"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// A Refs represetns array of referencs.
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

	upper *Refs     `enc:"-"` // upper node (nil for root)
	rn    *refsNode `enc:"-"` // pack, schema, index
}

func (r *Refs) isInitialized() bool {
	return r.rn != nil
}

type refsNode struct {
	pack  *Pack                       // related Pack
	sch   Schema                      // shema of the Refs
	index map[cipher.SHA256]*RefsElem // index (or reference)

	// lock
	// - don't rebuild the tree in some cases
	// - don't allow Append, Ascend, Descend
	//   methods in some cases
	iterating bool
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
		r.degree = int(er.Degree) // get degree from datbase
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

// RefByHash returns Ref by its hash. It returns first Ref.
// The call is O(1) if HashTableIndex flag set. Otherwise
// the call is O(n). It returns (nil, nil) if requested RefsElem
// not found
func (r *Refs) RefByHash(hash cipher.SHA256) (needle *RefsElem, err error) {

	if err = r.load(0); err != nil {
		return
	}

	if r.rn.pack.flags&HashTableIndex != 0 {
		needle = r.rn.index[hash]
		return
	}

	if r.length == 0 {
		return // nil, nil (not found)
	}

	_, err = r.descend(r.depth, r.length-1,
		func(_ int, el *RefsElem) (_ error) {
			if el.Hash == hash {
				needle = el
				return ErrStopIteration
			}
			return
		})
	if err == ErrStopIteration {
		err = nil
	}
	return
}

// RefByHashWithIndex returns RefsElem by its hash and index of the
// element. It returns (0, nil, nil) if the needle not found
func (r *Refs) RefByHashWithIndex(hash cipher.SHA256) (i int, needle *RefsElem,
	err error) {

	if err = r.load(0); err != nil {
		return
	}

	if r.rn.pack.flags&HashTableIndex != 0 {
		var ok bool
		if needle, ok = r.rn.index[hash]; !ok {
			return // 0, nil, nil
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

	if r.length == 0 {
		return // 0, nil, nil (not found)
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
// docs for IterateRefsFunc
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
		r.Hash = cipher.SHA256{}
		return
	}

	if r.isRoot() {
		// root Refs represented as encodedRefs
		var er encodedRefs           // encode to save and get hash
		er.Degree = uint32(r.degree) // keep in database
		er.Depth = uint32(depth)     // keep in database
		er.Length = uint32(nl)       // keep in database
		er.Nested = nested           // actually, keep in database
		r.Hash, _ = r.rn.pack.save(er)
	} else {
		// non-root Refs represented as encodedRefsNode
		var ern encodedRefsNode
		ern.Nested = nested     // keep in database
		ern.Length = uint32(nl) // keep in daabase
		r.Hash, _ = r.rn.pack.save(ern)
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

	var bset bool = true // set of branch (set to true to check first from tail)

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

	*r = nr // replace the r with nr

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
// even if it's nil of some type or nil-interace{}
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
			continue // delte from the objs
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

// AppendHash to the Refs. Type of the objects must
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

// Clear the Refs
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

// don't need to commit the Refs
func (r *Refs) commit() (_ error) {
	if r.rn == nil {
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
	gt.Name = "[](refs) " + r.Short()

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

// ------ ------ ------ ------ ------ ------ ------ ------ ------ ------ ------

// RefsElem

// RefsElem is element of Refs
type RefsElem struct {
	Hash cipher.SHA256 // hash of the RefsElem

	// internals
	value interface{} // golang-value of the RefsElem
	rn    *refsNode   // related
	upper *Refs       // upper node
}

// Short string
func (r *RefsElem) Short() string {
	return r.Hash.Hex()[:7]
}

// String implements fmt.Stringer interface
func (r *RefsElem) String() string {
	return r.Hash.Hex()
}

func (r *RefsElem) SetHash(hash cipher.SHA256) (err error) {
	if r.Hash == hash {
		return
	}
	if hash == (cipher.SHA256{}) {
		return r.Delete()
	}
	r.value = nil // clear related value
	r.Hash = hash
	return
}

// Delete the RefsElem from related Refs
func (r *RefsElem) Delete() (err error) {
	r.Hash = (cipher.SHA256{})
	r.value = nil
	r.rn = nil // detach
	if r.upper != nil {
		err = r.upper.bubble()
	}
	r.upper = nil
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
		obj = r.value // already have (already tracked)
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

	r.trackChanges() // if AutoTrackChanges enabled
	return
}

// SetValue of the element. A nil removes this
// element from related Refs
func (r *RefsElem) SetValue(obj interface{}) (err error) {

	// copy-pased from Ref.SetValue wiht little changes
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
		r.Hash = key
		r.rn.pack.set(key, val) // save

		if err = r.upper.bubble(); err != nil {
			return
		}
	}

	r.trackChanges()
	return
}

func (r *RefsElem) trackChanges() {
	if r.rn.pack.flags&AutoTrackChanges != 0 {
		r.rn.pack.Push(r) // track
	}
	return
}

func (r *RefsElem) commit() (err error) {
	if r.rn == nil || r.value == nil || r.upper == nil {
		return // detached or not loaded
	}
	key, val := r.rn.pack.dsave(r.value) // don't save - get key-value pair
	if key == r.Hash {
		return
	}
	r.rn.pack.set(key, val) // save
	r.Hash = key

	err = r.upper.bubble() // bubble up
	return
}
