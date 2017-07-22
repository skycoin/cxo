package skyobject

import (
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

// A Root represents root object of a feed
type Root struct {
	Refs []Dynamic // main branches

	Reg RegistryReference // registry
	Pub cipher.PubKey     // feed

	Sig  cipher.Sig // signature
	Seq  uint64     // seq number
	Time int64      // timestamp (unix nano)

	Hash cipher.SHA256 // hash
	Prev cipher.SHA256 // hash of previous root

	/*
		c *Container `enc:"-"` // back reference

		r   *Registry  `enc:"-"` // registry short hand
		rmx sync.Mutex `enc:"-"` // mutex for the r field

		fmx  sync.Mutex
		full bool // is full*/

}

/*
// Registry of the Root. Result can be nil
// if related Container doesn't contain
// required Registry
func (r *Root) Registry() *Registry {
	r.rmx.Lock()
	defer r.rmx.Unlock()

	if r.r != nil {
		return r.r
	}
	r.r = r.c.Registry(r.Reg)
	return r.r
}

func (r *Root) IsFull() (full bool) {
	r.fmx.Lock()
	defer r.fmx.Unlock()

	if r.full == true {
		return true
	}

	if r.Registry() == nil {
		return false
	}

	r.c.db.View(func(tx data.Tv) (_ error) {
		full = r.isFull(tx.Objects())
		return
	})

	if full {
		r.full = full
	}

	return

}

// isFull inside transaction (must have regsitry)
func (r *Root) isFull(objs isExistor) (full bool) {
	// TODO
	return
}
*/
