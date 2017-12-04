package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

// pack for feed preview

type getter interface {
	Get(key cipher.SHA256) (val []byte, err error)
}

// A Preview implements registry.Pack for
// feeds preview. The Preview get obejct
// of a Root from database if possible,
// otherwise it request this obejcts using
// provided getter. The Preview used by
// the node package for feeds preview
type Preview struct {
	m map[cipher.SHA256][]byte //
	g getter                   //

	*Pack
}

// Received returns map with objects received
// from peer (but not from local DB)
func (p *Preview) Received() (got map[cipher.SHA256][]byte) {
	return p.m
}

// Get from DB or from remote peer
func (p *Preview) Get(key cipher.SHA256) (val []byte, err error) {

	// check out map first
	var ok bool
	if val, ok = p.m[key]; ok {
		return // alrady received
	}

	if val, err = p.Pack.Get(key); err != nil && err != data.ErrNotFound {
		return // db failure
	}

	// not found

	if val, err = p.g.Get(key); err == nil {
		p.m[key] = val
	}

	return
}
