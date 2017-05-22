package skyobject

import (
	"github.com/skycoin/cxo/data"
)

func getCont() *Container {
	reg := NewRegistry()
	reg.Register("cxo.User", User{})
	reg.Register("cxo.Group", Group{})
	reg.Register("cxo.Developer", Developer{})
	return NewContainer(data.NewMemoryDB(), reg)
}
