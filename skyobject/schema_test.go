package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"reflect"
)

func Test_x(t *testing.T) {
	for _, i := range []interface{}{
		cipher.SHA256{},
		[]cipher.SHA256{},
		Dynamic{},
	} {
		typ := reflect.TypeOf(i)
		t.Log("Type: ", typ.String())
		t.Log("Name: ", typ.Name())
		t.Log("PkgPath: ", typ.PkgPath())
		t.Log("typeName: ", typeName(typ))
	}
}
