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
// terminate the finding returning any other error.
// Use the deepper reply to ectract and explore current
// object. A findRefsFunc never called with empty hash
type findRefsFunc func(key cipher.SHA256) (deepper bool, err error)

// find refs of an element
type findRefs struct {
	reg *Registry
	c   *Container
}

func (f *findRefs) Dynamic(dr Dynamic, fr findRefsFunc) (err error) {
	if dr.Object == (cipher.SHA256{}) {
		return // ignore blank
	}
	if false == dr.IsValid() {
		return ErrInvalidDynamicReference
	}
	var deepper bool
	if deepper, err = fr(dr.Object); err != nil || deepper == false {
		return
	}
	var s Schema
	if s, err = f.reg.SchemaByReference(dr.SchemaRef); err != nil {
		return
	}
	return f.Ref(s, key, fr)
}

func (f *findRefs) Ref(s Schema, key cipher.SHA256,
	fr findRefsFunc) (err error) {

	if key == (cipher.SHA256{}) {
		return
	}

	var val []byte
	if val, _, err = f.c.db.CXDS().Get(key); err != nil {
		return
	}

	//
	return
}
