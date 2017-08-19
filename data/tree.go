package data

import (
	"github.com/skycoin/skycoin/src/cipher"
)

// A Tree used to build tree of
// elements to save
type Tree struct {
	Key   cipher.SHA256 // key
	Value []byte        // encoded value (not Object)

	Subtree // info about subtree

	Nested []*Tree // nested objects
}

// Insert subtree
func (t *Tree) Insert(n *Tree) {
	t.Nested = append(t.Nested, t)
}

// Save the tree returnin amount and volume of the tree
// excluding root element. After this call Subtree of
// the Tree will contain actual values
func (t *Tree) Save(objs UpdateObjects) (err error) {
	for _, no := range t.Nested {
		if err = no.Save(objs); err != nil {
			return
		}
		t.Subtree.Amount += no.Amount + 1                     // + the no
		t.Subtree.Volume += no.Volume + Volume(len(no.Value)) // + the no
	}
	return
}
