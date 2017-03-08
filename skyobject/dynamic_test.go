package skyobject

/*
import (
	"reflect"
	"testing"

	"github.com/skycoin/cxo/data"
)

func TestContainer_NewDynamicHref(t *testing.T) {
	c := NewContainer(data.NewDB())
	dh := c.NewDynamicHref(User{})
	if dh == nil {
		t.Error("NewDynamicHref returns nil")
		return
	}
	if dh.Schema.Name != reflect.TypeOf(User{}).Name() {
		t.Error("invalid schema")
	}
	if data, ok := c.db.Get(dh.ObjKey); !ok {
		t.Error("missing object in db")
	} else if len(data) == 0 {
		t.Error("empty object stored")
	}
}
*/
