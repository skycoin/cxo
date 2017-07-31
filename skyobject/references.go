package skyobject

import (
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// References represents list of references to
// objects. The References is not thread safe
type References struct {
	Hash cipher.SHA256

	walkNode *walkNode `enc:"-"`
	refs     *Refs     `enc:"-"`
}

// IsBlank returns true if the References represent nil
func (r *References) IsBlank() bool {
	return r.Hash == (cipher.SHA256{})
}

// Short string
func (r *References) Short() string {
	return r.Hash.Hex()[:7]
}

// String implements fmt.Stringer interface. The
// method returns References.Hash.Hex()
func (r *References) String() string {
	return r.Hash.Hex()
}

// Eq returns true if these References equal to given
func (r *References) Eq(x *References) bool {
	return r.Hash == x.Hash
}

// Schema of the Referenes. It returns nil
// if the References are not unpacked
func (r *References) Schema() Schema {
	if r.walkNode != nil {
		return r.walkNode.sch
	}
	return nil
}

// Len returns length of the References
func (r *References) Len() (ln int, err error) {
	if r.refs == nil {
		//
	}
	return
}

// RefByIndex returns Reference by index
func (r *References) RefByIndex(i int) (ref Reference, err error) {
	// TODO (kostyarin): implement
	return
}

// DelByIndex delete element by index. You can also
// get element using any method and call SetValue(nil)
// to delete an element
func (r *References) DelByIndex(i int) (err error) {
	// TODO (kostyarin): implement
	return
}

// RefByHash returns first Reference by hash if these References contain it
func (r *References) RefByHash(hash cipher.SHA256) (ref Reference, err error) {
	// TODO (kostyarin): implement
	return
}

// LastRefByHash returns last Reference by hash if these References contain it
func (r *References) LastRefByHash(hash cipher.SHA256) (ref Reference,
	err error) {

	// TODO (kostyarin): implement
	return
}

// Append to tail. Arguments must be type/schema of the References.
// Arguments can be golagn values if related Pack created with Types
// or Reference(s). It's impossible to append nil or empty Reference
// (it will be skipped silenty)
func (r *References) Append(obj ...interface{}) (err error) {
	// TODO (kostyarin): implement
	return
}

// Slice returns another References instance that keep values
// of these References from i to j
func (r *References) Slice(i, j int) (refs References, err error) {
	// TODO (kostyarin): implement
	return
}

// Clear the References making them blank.
func (r *References) Clear() {
	if r.Hash != (cipher.SHA256{}) {
		r.Hash = (cipher.SHA256{})
		if wn := r.walkNode; wn != nil {
			wn.unsave()
		}
	}
}

// internal methods

// loadFull tree
func (r *References) loadFull(action string) (err error) {
	var wn *walkNode
	if wn, err = r.getWalkNode(action); err != nil {
		return
	}

	//

	return
}

// loadHead, e.g. load Refs
func (r *References) loadHead(action string) (err error) {
	var wn *walkNode
	if wn, err = r.getWalkNode(action); err != nil {
		return
	}

	if r.Hash == (cipher.SHA256{}) {
		r.refs = new(Refs)   // create empty Refs
		r.refs.walkNode = wn // walk node
		return
	}

	var val []byte
	if val, err = wn.pack.get(r.Hash); err != nil {
		err = fmt.Errorf("can't %s: %v", action, err)
		return
	}

	var refs Refs
	if err = encoder.DeserializeRaw(val, &refs); err != nil {
		err = fmt.Errorf("can't %s: error decoding head: %v", action, err)
		return
	}

	refs.walkNode = wn
	r.refs = &refs

	return
}

func (r *References) getWalkNode(action string) (wn *walkNode, err error) {
	if wn = r.walkNode; wn == nil {
		err = fmt.Errorf("can't %s: References detached from Pack", action)
	}
	return
}

// References.Hash -> Refs -> Refs.Nodes -> ...

// Refs is internal
type Refs struct {
	Degree uint32 // degree (hashes per leaf, hashes per branch)
	Len    uint32 // amount of non-zero elements of the tree
	Depth  uint32 // depth of the tree

	// the Depth allows to determine end of branches, that
	// is begining of leafs; leafs contains references to
	// elements of the References array instead of
	// references to RefsNode

	// depth * degree = items (including zeroes)

	Nodes []RefsNode // branches (if Depth == 0, then these is leafs)

	// internals

	// reference to walkNode of References
	walkNode *walkNode               `enc:"-"`
	index    map[cipher.SHA256][]int `enc:"-"` // hash-table index
}

// A RefsNode is internal
type RefsNode struct {
	// Len is amount of non-zero elements of the branch
	Len uint32

	// Hashes of the next branches or References of leafs.
	// If the RefsNode is not a leaf then these Hashes
	// points to another RefsNode(s). In this (non-leaf)
	// case, only Hash field of these Refrence(s) has
	// meaning
	Hashes []Reference

	// internals

	nodes        []RefsNode    `enc:"-"` // if it's a branch
	refsWalkNode *refsWalkNode `enc:"-"` // track changes
}

//
//
//

// track changes
type refsWalkNode struct {
	upper    *refsWalkNode
	unsaved  bool
	walkNode *walkNode
}

func (r *refsWalkNode) unsave() {
	for up := r; up != nil; up = r.upper {
		up.unsaved = true
	}
	if wn := r.walkNode; wn != nil {
		wn.unsaved = true
	}
}
