package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject/registry"
)

// A Pack implements registry.Pack interface
type Pack struct {
	reg   *registry.Registry
	c     *Container
	deg   registry.Degree
	flags registry.Flags
}

// Registry returns related registry
func (p *Pack) Registry() (reg *registry.Registry) {
	return p.reg
}

// Get value by hash
func (p *Pack) Get(key cipher.SHA256) (val []byte, err error) {
	val, _, err = p.c.Get(key, 0)
	return
}

// Set key-value pair
func (p *Pack) Set(key cipher.SHA256, val []byte) (err error) {
	err = p.c.Set(key, val, 1)
	return
}

// Add is Set that calculates hash inside
func (p *Pack) Add(val []byte) (key cipher.SHA256, err error) {
	key = cipher.SumSHA256(val)
	err = p.Set(key, val)
	return
}

// Dgree of the Pack
func (p *Pack) Degree() registry.Degree {
	return p.deg
}

// SetDegree to the Pack
func (p *Pack) SetDegree(degree registry.Degree) (err error) {
	if err = degree.Validate(); err == nil {
		p.deg = degree
	}
	return
}

// Flags of the Pack
func (p *Pack) Flags() registry.Flags {
	return p.flags
}

// AddFlags adds given flags to flags of the Pack (|)
func (p *Pack) AddFlags(flags Flags) {
	p.flags |= flags
}

// ClearFlags isclears given flags from flags of the Pack (&^)
func (p *Pack) ClearFlags(flags Flags) {
	p.flags &^= flags
}
