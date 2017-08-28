package skyobject

import (
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data/idxdb"
)

// A Root represents root object of a feed
type Root struct {
	Refs []Dynamic // main branches

	Reg RegistryRef   // registry
	Pub cipher.PubKey // feed

	Seq  uint64 // seq number
	Time int64  // timestamp (unix nano)

	// sig and hash are anot parts of the Root

	Sig cipher.Sig `enc:"-"` // signature (not part of the Root)

	Hash cipher.SHA256 `enc:"-"` // hash (not part of the Root)
	Prev cipher.SHA256 // hash of previous root

	// machine local fields, not parts of the Root

	// IsFull set to true if DB contains all objects
	// required by this Root
	IsFull bool `enc:"-"`
}

func (r *Root) Encode() []byte {
	return encoder.Serialize(r)
}

// Short retusn string like "[1a2ef33:2]" ({pub_key:seq})
func (r *Root) Short() string {
	return fmt.Sprintf("{%s:%d}",
		r.Pub.Hex()[:7],
		r.Seq)
}

// String implements fmt.Stringer interface
func (r *Root) String() string {
	return fmt.Sprintf("Root{%s:%d:%s}",
		r.Pub.Hex()[:7],
		r.Seq,
		r.Hash.Hex()[:7])
}

func decodeRoot(val []byte) (r *Root, err error) {
	r = new(Root)
	if err = encoder.DeserializeRaw(val, r); err != nil {
		r = nil
	}
	return
}

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

	err = c.db.IdxDB().Tx(func(feeds idxdb.Feeds) (err error) {
		var rs idxdb.Roots
		if rs, err = feeds.Roots(r.Pub); err != nil {
			return
		}
		var ir *idxdb.Root
		if ir, err = rs.Get(r.Seq); err != nil {
			return
		}
		r.IsFull = ir.IsFull
		return
	})
	if err != nil {
		r = nil
	}

	return
}

// LastFull Root of given feed
func (c *Container) LastFull(pk cipher.PubKey) (r *Root, err error) {
	err = c.DB().IdxDB().Tx(func(feeds idxdb.Feeds) (err error) {
		var rs idxdb.Roots
		if rs, err = feeds.Roots(pk); err != nil {
			return
		}
		return rs.Descend(func(ir *idxdb.Root) (err error) {
			if ir.IsFull {
				var val []byte
				if val, _, err = c.DB().CXDS().Get(ir.Hash); err != nil {
					return
				}
				r, err = decodeRoot(val)
				return
			}
			return
		})
	})
	if err != nil {
		r = nil
		return
	}
	if r == nil {
		err = idxdb.ErrNotFound
	}
	return
}
