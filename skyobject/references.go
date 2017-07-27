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
	Len    uint32 // amount of non-zero elements of the tree
	Depth  uint32 // depth of the tree

	// the Depth allows to determine end of branches, that
	// is begining of leafs; leafs contains references to
	// elements of the References array instead of
	// references to RefsNode

	// depth * degree = elems (including zero elements)

	Nodes []RefsNode // branches (if Depth == 0, then this is leafs)

	sch   Schema                      `enc:"-"` // schema of the References
	index map[cipher.SHA256]Reference `enc:"-"` // hash-table index

	cache map[cipher.SHA256][]byte `enc:"-"` // insert cache

	// TODO (kostyarin): cahnges
}

// A RefsNode is internal
type RefsNode struct {
	// Len is amount of non-zero elements of the branch
	Len uint32
	// Hashes of the next branches or References of leafs
	Hashes []Reference

	// Nodes of the branch (for unpack a branch, range and walk)
	Nodes []RefsNode `enc:"-"`

	sch Schema `enc:"-"` // schema inherited from References

	// TODO (kostyarin): changes
}

// insert Reference to []RefsNode returning ok = true and new
// or the same []RefsNode slcie
func (r *References) insert(depth int, ns []RefsNode,
	ref Reference) (nn []RefsNode, ok bool) {

	if len(ns) == 0 {
		var an []RefsNode = ns
		for i := 0; i < depth; i++ {
			//
		}
	}

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
