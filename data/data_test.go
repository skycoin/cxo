package data

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func Test_encodingRootPack(t *testing.T) {
	var rp RootPack
	encoder.Serialize(&rp)
}
