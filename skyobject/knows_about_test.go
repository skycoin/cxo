package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

func Test_knowsAbout(t *testing.T) {

	c := getCont()

	pk, sk := cipher.GenerateKeyPair()

	pack, err := c.NewRoot(pk, sk, 0, c.CoreRegistry().Types())
	if err != nil {
		t.Fatal(err)
	}

}
