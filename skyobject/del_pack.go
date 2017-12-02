package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject/registry"
)

type delPack struct {
	*Pack

	last cipher.SHA256
	val  []byte
}

func (c *Container) getDelPack(
	r *registry.Root, //   :
) (
	pack registry.Pack, // :
	err error, //          :
) {

	var reg *registry.Registry
	if reg, err = c.Registry(r.Reg); err != nil {
		return
	}

	var originPack = c.getPack(reg)

	return &delPack{Pack: originPack}
}

func (d *delPack) Get(key cipher.SHA256) (val []byte, err error) {
	if key == d.last {
		return d.val, nil
	}
	return d.Pack.Get(key)
}
