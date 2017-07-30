package skyobject

import (
	"errors"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

// common Root-related errors
var (
	ErrNoSuchFeed = errors.New("no such feed")
)

// A Root represents root object of a feed
type Root struct {
	Refs []Dynamic // main branches

	Reg RegistryReference // registry
	Pub cipher.PubKey     // feed

	Seq  uint64 // seq number
	Time int64  // timestamp (unix nano)

	Sig cipher.Sig // signature

	Hash cipher.SHA256 // hash
	Prev cipher.SHA256 // hash of previous root
}

// PH retusn string like "[1a2ef33:98a7ec5]"
// ([pub_key:root_hash]). First 7 characters
func (r *Root) PH() string {
	return "[" + r.Pub.Hex()[:7] + ":" + r.Hash.Hex()[:7] + "]"
}

// AddRoot to container. The method sets rp.IsFull to false
func (c *Container) AddRoot(pk cipher.PubKey, rp *data.RootPack) (r *Root,
	err error) {

	rp.IsFull = false

	if r, err = c.unpackRoot(rp); err != nil {
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
		//
	}
	return
}
