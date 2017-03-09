package skyobject

import (
	"testing"

	"github.com/skycoin/cxo/data"
)

func TestContainer_NewDynamicHref(t *testing.T) {
	c := NewContainer(data.NewDB())
	dh := c.NewDynamicHref(User{"Alice Cooper", 23, ""})
	if _, ok := c.db.Get(dh.Schema); !ok {
		t.Error("missing saved schema")
	}
	if _, ok := c.db.Get(dh.ObjKey); !ok {
		t.Error("missing saved object")
	}
}
