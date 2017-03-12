package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

func TestContainer_SaveDynamicHref(t *testing.T) {
	c := NewContainer(data.NewDB())
	dhk := c.SaveDynamicHref(User{"Alice Cooper", 23, ""})
	if data, ok := c.db.Get(dhk); !ok {
		t.Error("missing dynamic href in db")
		return
	} else {
		var dh DynamicHref
		if err := encoder.DeserializeRaw(data, &dh); err != nil {
			t.Error("unexpected error: ", err)
			return
		}
		if _, ok := c.db.Get(dh.Schema); !ok {
			t.Error("missing saved schema")
		}
		if _, ok := c.db.Get(dh.ObjKey); !ok {
			t.Error("missing saved object")
		}
	}
}
