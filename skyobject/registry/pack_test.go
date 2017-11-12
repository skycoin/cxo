package registry

import (
	//"testing"

	"github.com/skycoin/skycoin/src/cipher"
	//"github.com/skycoin/skycoin/src/cipher/encoder"
)

// dummy pack

type dummyPack struct {
	reg   *Registry
	flags Flags
	vals  map[cipher.SHA256][]byte
}

func (d *dummyPack) Registry() *Registry {
	return d.reg
}

func (d *dummyPack) Get(key cipher.SHA256) (val []byte, err error) {
	var ok bool
	if val, ok = d.vals[key]; !ok {
		err = ErrNotFound
	}
	return
}

func (d *dummyPack) Set(key cipher.SHA256, val []byte) (_ error) {
	d.vals[key] = val
	return
}

func (d *dummyPack) Add(val []byte) (key cipher.SHA256, err error) {
	key = cipher.SumSHA256(val)
	err = d.Set(key, val)
	return
}

func (d *dummyPack) Flags() Flags {
	return d.flags
}

func (d *dummyPack) AddFlags(flags Flags) {
	d.flags |= flags
}

func (d *dummyPack) ClearFlags(flags Flags) {
	d.flags &^= flags
}

func (d *dummyPack) Has(key cipher.SHA256) (ok bool) {
	_, ok = d.vals[key]
	return
}
