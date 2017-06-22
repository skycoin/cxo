package skyobject

import (
	"github.com/skycoin/cxo/data"
)

type User struct {
	Name   string
	Age    uint32
	Hidden []byte `enc:"-"`
}

type Group struct {
	Name    string
	Leader  Reference  `skyobject:"schema=cxo.User"`
	Members References `skyobject:"schema=cxo.User"`
	Curator Dynamic
}

type Developer struct {
	Name   string
	GitHub string
}

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
