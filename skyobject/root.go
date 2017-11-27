package skyobject

import (
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

// AddEncodedRoot or not to add. The method checks given encoded Root
// and returns full Root if DB aleady contains the Root and non full
// Root (that is not stored in DB) if the Root is fresh. The node
// package must care about filling one Root at the same time, this
// method only checks signature, hash, etc. Thus, if err is nil, then
// the Root reply can be full Root, or it can be unsaved non-full Root
// object, that node required to fill (or drop for some reasons).
// Actually, this method doesn't add to DB anything
func (c *Container) AddEncodedRoot(sig cipher.Sig, val []byte) (r *Root,
	err error) {

	hash := cipher.SumSHA256(val)

	if r, err = decodeRoot(val); err != nil {
		return
	}

	if err = cipher.VerifySignature(r.Pub, sig, hash); err != nil {
		return
	}

	r.Hash = hash
	r.Sig = sig

	err = c.db.IdxDB().Tx(func(feeds data.Feeds) (err error) {
		var rs data.Roots
		if rs, err = feeds.Roots(r.Pub); err != nil {
			return
		}
		if true == rs.Has(r.Seq) {
			r.IsFull = true
		}
		return
	})
	if err != nil {
		r = nil // can't determine
	}
	return
}

// UnholdRoot holdeds given Root object
func (c *Container) UnholdRoot(r *Root) { c.Unhold(r.Pub, r.Seq) }

// LastRoot of given feed. It receive Root object from DB, thus the Root
// can only be full. E.g. the method is "give me last full Root of this feed".
// This method returns holded Root object and it can't be removed from
// database. You have to unhold it later using Unhold or UnholdRoot method
func (c *Container) LastRoot(pk cipher.PubKey) (r *Root, err error) {

	var holded bool

	err = c.DB().IdxDB().Tx(func(feeds data.Feeds) (err error) {
		var rs data.Roots
		if rs, err = feeds.Roots(pk); err != nil {
			return
		}
		return rs.Descend(func(ir *data.Root) (err error) {
			var val []byte
			if val, _, err = c.DB().CXDS().Get(ir.Hash); err != nil {
				return
			}
			if r, err = decodeRoot(val); err != nil {
				return
			}

			c.Hold(pk, r.Seq) // hold the Root
			holded = true

			r.Hash = ir.Hash
			r.Sig = ir.Sig
			r.IsFull = true

			return data.ErrStopIteration // break
		})
	})
	if err != nil {
		if holded {
			c.Unhold(pk, r.Seq)
		}
		r = nil
	} else if r == nil {
		// this occurs if feed is empty and the Descend function
		// above doesn't call given function, returning nil
		err = data.ErrNotFound
	}
	return
}

// Root returns Root object by feed and seq numebr. It gets the Root object from
// DB, thus the Root can only be full. This method returns holded Root object
// and it can't be removed from database. You have to unhold it later using
// Unhold or UnholdRoot method
func (c *Container) Root(pk cipher.PubKey, seq uint64) (r *Root, err error) {

	var holded bool

	err = c.DB().IdxDB().Tx(func(feeds data.Feeds) (err error) {
		var rs data.Roots
		if rs, err = feeds.Roots(pk); err != nil {
			return
		}
		var ir *data.Root
		if ir, err = rs.Get(seq); err != nil {
			return
		}
		var val []byte
		if val, _, err = c.DB().CXDS().Get(ir.Hash); err != nil {
			return
		}
		if r, err = decodeRoot(val); err != nil {
			return
		}

		c.Hold(pk, r.Seq) // hold the Root
		holded = true

		r.Hash = ir.Hash
		r.Sig = ir.Sig
		r.IsFull = true

		return
	})
	if err != nil {
		if holded {
			c.Unhold(pk, r.Seq)
		}
		r = nil
	}
	return
}
