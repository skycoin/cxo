package data

import (
	"github.com/skycoin/skycoin/src/cipher"
)

// A Tree represents objects tree. The tree
// can be used to save new obejcts and
// set appropriate Subtree for this objects.
//
type Tree struct {
	upper  *Tree
	nested []*Tree

	key   cipher.SHA256
	value []byte  // can be nil
	obj   *Object // can be nil

	// if the already is true, then we need to get object
	// by key from database and use its values, otherwise
	// we are using 'value' field (its unsaved obejct)
	already bool
}

// NewTree creates new Tree by gven key and optional vlaue.
// If value is nil or empty, then the Tree will represent existsing object
func NewTree(key cipher.SHA256, val []byte) (t *Tree) {
	t = new(Tree)
	t.key = key
	t.value = val
	t.already = (len(val) == 0)
	return

}

// Insert subtree
func (t *Tree) Insert(it *Tree) {
	it.upper = t
	t.nested = append(t.nested, it)
	return
}

// Save the Tree, returning its total amount and volume
func (t *Tree) Save(objs UpdateObjects) (amnt uint32, vol Volume, err error) {

	if t.already {

		// already saved obejct, check out DB

		if t.obj == nil {
			if t.obj = objs.GetObject(t.key); t.obj == nil {
				err = ErrNotFound
				return
			}
		}
		amnt, vol = t.obj.Amount(), t.obj.Volume()
		return
	}

	// but the object can be alreaady saved even if we
	// think that it's new
	if obj := objs.GetObject(t.key); obj != nil {
		t.obj, t.already = obj, true
		amnt, vol = t.obj.Amount(), t.obj.Volume()
		return
	}

	// object is not saved yet and can have nested

	var na uint32
	var nv Volume
	for _, no := range t.nested {
		if na, nv, err = no.Save(objs); err != nil {
			return
		}
		amnt += na
		vol += nv
	}

	// save object

	obj := &Object{
		Subtree: Subtree{amnt, vol},
		Value:   t.value,
	}

	err = objs.Set(t.key, obj)
	if err != nil {
		return
	}
	t.already = true // already saved
	t.nested = nil   // clear
	t.obj = obj

	amnt, vol = t.obj.Amount(), t.obj.Volume()
	return
}
