package skyobject

import (
	"testing"
	//"time"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node/log"
)

type User struct {
	Name   string
	Age    uint32
	Hidden []byte `enc:"-"`
}

type Group struct {
	Name    string
	Leader  Ref  `skyobject:"schema=cxo.User"`
	Members Refs `skyobject:"schema=cxo.User"`
	Curator Dynamic
}

type Developer struct {
	Name   string
	GitHub string
}

func getRegisty() *Registry {
	return NewRegistry(func(r *Reg) {
		r.Register("cxo.User", User{})
		r.Register("cxo.Group", Group{})
		r.Register("cxo.Developer", Developer{})
	})
}

func getCont() *Container {
	conf := NewConfig()
	conf.Registry = getRegisty()
	if testing.Verbose() {
		conf.Log.Debug = true
		conf.Log.Pins = log.All
	}
	return NewContainer(data.NewMemoryDB(), conf)
}
