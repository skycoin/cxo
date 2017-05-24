package skyobject

import (
	"github.com/skycoin/cxo/data"
)

func getRegisty() (reg *Registry) {
	reg = NewRegistry()
	reg.Register("cxo.User", User{})
	reg.Register("cxo.Group", Group{})
	reg.Register("cxo.Developer", Developer{})
	return
}

func getCont() *Container {
	return NewContainer(data.NewMemoryDB(), getRegisty())
}
