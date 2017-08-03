package skyobject

import (
	"errors"
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

// common Root-related errors
var (
	ErrNoSuchFeed = errors.New("no such feed")
)

// A Root represents root object of a feed
type Root struct {
	Refs []Dynamic // main branches

	Reg RegistryRef   // registry
	Pub cipher.PubKey // feed

	Seq  uint64 // seq number
	Time int64  // timestamp (unix nano)

	Sig cipher.Sig // signature

	Hash cipher.SHA256 // hash
	Prev cipher.SHA256 // hash of previous root
}

func (r *Root) Encode() []byte {
	return encoder.Serialize(r)
}

func DecodeRoot(val []byte) (r *Root, err error) {
	r = new(Root)
	if err = encoder.DeserializeRaw(val, r); err != nil {
		r = nil
	}
	return
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

// AddRoot to container. The method sets rp.IsFull to false
func (c *Container) AddRoot(pk cipher.PubKey, rp *data.RootPack) (r *Root,
	err error) {

	rp.IsFull = false

	if r, err = c.unpackRoot(pk, rp); err != nil {
		return
	}

	err = c.DB().Update(func(tx data.Tu) (_ error) {
		roots := tx.Feeds().Roots(pk)
		if roots == nil {
			return ErrNoSuchFeed
		}
		return roots.Add(rp)
	})
	if err == nil {
		// track seq number
		c.trackSeq.addSeq(r.Pub, r.Seq, false)
	}
	return
}

// MarkFull marks given Root as full in DB
func (c *Container) MarkFull(r *Root) (err error) {
	err = c.DB().Update(func(tx data.Tu) error {
		roots := tx.Feeds().Roots(r.Pub)
		if roots == nil {
			return ErrNoSuchFeed
		}
		return roots.MarkFull(r.Seq)
	})
	if err == nil {
		// track last full
		c.trackSeq.addSeq(r.Pub, r.Seq, true)
	}
	return
}
