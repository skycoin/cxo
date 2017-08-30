package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data/idxdb"
)

// decrementAll references of given *idxdb.Root
// (do it before deleting the Root)
func (c *Container) decrementAll(ir *idxdb.Root) (err error) {
	// ----
	// (0) get encoded Root by hash from CXDS
	// (1) decode the root turning it *skyobejct.Root
	// (2) get registry (and decrement it)
	// (3) range over Refs decrementing them
	// (4) if a Ref of the Refs deleted decode it and
	//     decrement its branches
	// (5) and so on
	// (6) Profit!
	// ----

	var val []byte

	// (0)
	if val, _, err = c.db.CXDS().Get(ir.Hash); err != nil {
		return
	}

	// (1)
	var r *Root
	if r, err = decodeRoot(val); err != nil {
		return
	}
	r.Hash = ir.Hash
	r.Sig = ir.Sig
	r.IsFull = true // but it doesn't make sence

	// (2)
	if r.Reg == (RegistryRef{}) {
		return ErrEmptyRegistryRef
	}

	var reg *Registry
	if r, err = c.Registry(r.Reg); err != nil {
		return
	}

	if _, err = c.db.CXDS().Dec(cipher.SHA256(r.Reg)); err != nil {
		return
	}

	// (3)
	//

	return
}

// A findRefsFunc used by findRefs. Error reply used
// to stop finding (use ErrStopIteration) and to
// terminate the finding returning any other error
type findRefsFunc func(key cipher.SHA256) (err error)

// find refs of an element
type findRefs struct {
	reg *Registry
	c   *Container
}

func (f *findRefs) Dynamic(dr Dynamic, fr findRefsFunc) (err error) {
	//
}
