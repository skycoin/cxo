package registry

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// Flags of unpacking
type Flags int

// common Flags
const (
	flagIsSet Flags = 1 << iota // service

	HashTableIndex // hash-table index for Refs
	EntireRefs     // unpack entire Refs
)

// A Pack represents ...
type Pack interface {
	Registry() *Registry // related registry

	Get(key cipher.SHA256) (val []byte, err error) // get value by key
	Set(key cipher.SHA256, val []byte) (err error) // set k-v pair
	Add(val []byte) (key cipher.SHA256, err error) // set+calculate hash

	Flags() Flags     // flags of the Pack
	SetFlags(Flags)   // set Flags
	UnsetFlags(Flags) // unset Flags
}

func get(pack Pack, hash cipher.SHA256, obj interface{}) (err error) {

	var val []byte

	if val, err = pack.Get(hash); err != nil {
		return
	}

	err = encoder.DeserializeRaw(val, obj)
	return
}
