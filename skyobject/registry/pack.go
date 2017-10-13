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

	// HashTableIndex is flag that enables hash-table index inside
	// the Refs. The index speeds up accessing by hash. The index
	// is not designed to be used with many elements with the same
	// hash. Because, the big O of some internal operations that
	// uses the index is O(m), where m is number of elements with
	// the same index. Be careful, your Refs can have many elements
	// with blank hash, that will be indexed as allways. Thus, the
	// index can be used with one or few elements per hash for the
	// Refs. The flag will be stored inised the Refs after first
	// access. For example
	//
	//     pack.SetFlag(registry.HashTableIndex)
	//     if _, err := someRefs.Len(pack); err != nil {
	//         // something wrong
	//     }
	//
	//     pack.UnsetFlag(registry.HashTableIndex)
	//     if _, err := someOtherRefs.Len(pack); err != nil {
	//         // something wrong
	//     }
	//
	// After the Len() (or any other call), the someRefs stores
	// flags of the pack inside. And you can unset the flag if
	// you want. Thus, the someRefs uses has-table index, but
	// the someOtherRefs does not
	HashTableIndex
	// EntireRefs is flag that forces the Refs to be unpacked
	// entirely. By default (without the flag), the Refs uses
	// lazy loading, where all branches loads only if it's
	// necessary. The Refs stores this flag inside like the
	// HashTableIndex flag (see above)
	EntireRefs
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
