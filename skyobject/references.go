package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

const (
	RefsDegree int32 = 8 // default degree of References' Merkle tree
)

//
// References
//

// A References represents list of references to
// objects. A References is not thread safe
type References struct {
	Degree uint32 // degree (hashes per leaf, hashes per branch)
	Len    uint32 // amount of non-zero elements

	Nodes []RefsNode // branches

	sch Schema `enc:"-"` // schema of the References
}

// A RefsNode is internal
type RefsNode struct {
	// IsLeaf is true if the RefsNode is a leaf
	IsLeaf bool
	// Nodes of the branch
	Nodes []RefsNode `enc:"-"`
	// Hashes of the next branches or References of leafs
	Hashes []Reference
}

// Insert appends given reference to the end
// of the References
func (r *References) Insert(ref Reference) {
	//
}

// Delete removes first reference
func (r *References) Delete(ref Reference) {
	//
}

// RangeReferencesFunc is itterator over a References.
// Use ErrStopRnge to terminate itteration
type RangeReferencesFunc func(i int, ref References) error

// Range over the References from first element to last.
// It's not safe to modify the References inside the Range
func (r *References) Range(rrf RangeReferencesFunc) (err error) {
	if r.Len == 0 {
		return
	}
	if _, err = rangeOverRefsNodes(0, r.Nodes, rff); err == ErrStopRange {
		err = nil
	}
	return
}

// rangeOverRefsNode itterates over []RefsNode and calls
// given RangeReferencesFunc function for every non-zero
// reference. It returns error if any. If error is nil
// then j keeps advancing of index
func rangeOverRefsNodes(i int, rn []RefsNode,
	rff RangeReferencesFunc) (j int, err error) {

	for _, node := range rn {

		if node.Len == 0 {
			continue
		}

		if len(node.Hashes) > 0 { // leaf

			for _, hash := range node.Hashes {
				if hash == (Reference{}) { // zero
					continue
				}
				if err = rff(i, hash); err != nil {
					return
				}
				i++
			}

		} else { // branch

			if j, err = rangeOverRefsNodes(i, node.Nodes, rff); err != nil {
				return
			}
			i += j

		}

	}

	j = i

	return
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
