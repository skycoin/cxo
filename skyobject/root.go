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

	Sig cipher.Sig `enc:"-"` // signature (not part of the Root)

	Hash cipher.SHA256 `enc:"-"` // hash (not part of the Root)
	Prev cipher.SHA256 // hash of previous root
}

func (r *Root) Encode() []byte {
	return encoder.Serialize(r)
}

// Pack of the Root (IsFull field always false)
func (r *Root) Pack() (rp *data.RootPack) {
	rp = new(data.RootPack)
	rp.Root = r.Encode()
	rp.Hash = r.Hash // if set
	rp.Prev = r.Prev // if set
	rp.Seq = r.Seq   // if set
	rp.Sig = r.Sig   // if set
	return
}

// PackToRoot convertes *data.RootPack to Root decoding it. The
// pk argument used for errors and can be empty. The rp argument
// must not be nil
func (c *Container) PackToRoot(pk cipher.PubKey,
	rp *data.RootPack) (*Root, error) {

	return c.unpackRoot(pk, rp)
}

// LastFullPack returns data.RootPack to send throug network.
func (c *Container) LastFullPack(pk cipher.PubKey) (rp *data.RootPack,
	err error) {

	err = c.DB().View(func(tx data.Tv) (_ error) {
		roots := tx.Feeds().Roots(pk)
		if roots == nil {
			return fmt.Errorf("no such feed %s", pk.Hex()[:7])
		}
		return roots.Descend(func(rpd *data.RootPack) (_ error) {
			if rpd.IsFull {
				rp = rpd
				return data.ErrStopIteration // data.ErrStopIteration
			}
			return
		})
	})
	if rp == nil {
		err = fmt.Errorf("last full of %s not found", pk.Hex()[:7])
	}
	return
}

// LastFull root of given feed
func (c *Container) LastFull(pk cipher.PubKey) (r *Root, err error) {
	var rp *data.RootPack
	if rp, err = c.LastFullPack(pk); err != nil {
		return
	}
	return c.unpackRoot(pk, rp)
}

// LastPack returns data.RootPack to send throug network
func (c *Container) LastPack(pk cipher.PubKey) (rp *data.RootPack,
	err error) {

	err = c.DB().View(func(tx data.Tv) (_ error) {
		if roots := tx.Feeds().Roots(pk); roots != nil {
			rp = roots.Last()
		}
		return
	})
	if rp == nil && err == nil {
		err = fmt.Errorf("last root of %s not found", pk.Hex()[:7])
	}
	return
}

// Last root of given feed
func (c *Container) Last(pk cipher.PubKey) (r *Root, err error) {
	var rp *data.RootPack
	if rp, err = c.LastPack(pk); err != nil {
		return
	}
	return c.unpackRoot(pk, rp)
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

// AddRoot to container. The method sets rp.IsFull to false.
// The rp must not be nil.
func (c *Container) AddRoot(pk cipher.PubKey, rp *data.RootPack) (r *Root,
	err error) {

	rp.IsFull = false

	if r, err = c.unpackRoot(pk, rp); err != nil {
		return
	}

	c.cleanmx.Lock()
	defer c.cleanmx.Unlock()

	err = c.DB().Update(func(tx data.Tu) error {
		roots := tx.Feeds().Roots(pk)
		if roots == nil {
			return ErrNoSuchFeed
		}
		return roots.Add(rp)
	})
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
	return
}

// RootBySeq is the same as Root, But the method also returns "full"
// reply, that describes fullness of the Root
func (c *Container) RootBySeq(pk cipher.PubKey, seq uint64) (r *Root,
	full bool, err error) {

	var rp *data.RootPack
	err = c.DB().View(func(tx data.Tv) (_ error) {
		if roots := tx.Feeds().Roots(pk); roots != nil {
			rp = roots.Get(seq)
		}
		return
	})
	if err != nil {
		return
	}
	if rp == nil {
		err = fmt.Errorf("root %d of %s not found", seq, pk.Hex()[:7])
		return
	}
	full = rp.IsFull
	r, err = c.unpackRoot(pk, rp)
	return
}

// DelRootsBefore deletes root obejcts of given feed before given seq number
// (exclusive). It never returns "no such feed" error. The error can only
// be error of database
func (c *Container) DelRootsBefore(pk cipher.PubKey, seq uint64) error {
	return c.DB().Update(func(tx data.Tu) (_ error) {
		roots := tx.Feeds().Roots(pk)
		if roots == nil {
			return // nothing to delete
		}
		return roots.DelBefore(seq)
	})
}
