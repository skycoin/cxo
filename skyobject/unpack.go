package skyobject

import (
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// An Unpacker represetns interface
// that can unpack CX objects
type Unpacker interface {
	Get(cipher.SHA256) []byte // encoded object by ref or nil
	//
}

// A Packer represents interface
// that can pack CX objectss
type Packer interface {
	Set(val interface{}) (key cipher.SHA256, obj []byte) // save object
	//
}

// A PackUnpacker represents interface
// that can pack and unpack CX objects
type PackUnpacker interface {
	Packer
	Unpacker
}
