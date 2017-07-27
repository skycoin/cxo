package skyobject

import (
	"errors"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// common packing and unpacking errors
var (
	ErrViewOnlyTree = errors.New("view only tree")
)

// A Flag represents unpacking flags
type Flag int

const (
	MerkleTree       Flag = 1 << iota // use merkle tree
	EntireMerkleTree                  // unpack entire Merkle tree
	EntireTree                        // unpack all possible
	HashTable                         // use hash-table index for Merkle-trees
	ViewOnly                          // don't allow modifications
	GoTypes                           // pack/unpack from/to golang-values

	Default Flag = MerkleTree | GoTypes
)

// A Types represents mapping from registered names
// of a Regsitry to reflect.Type and inversed way
type Types struct {
	Direct  map[string]reflect.Type // registered name -> refelect.Type
	Inverse map[reflect.Type]string // refelct.Type -> registered name
}

// A Pack represents database cache for
// new objects. It uses in-memory cache
// of new objects saving them in the end.
// The Pack also used to unpack a Root,
// modify it and walk through
type Pack struct {
	c *Container

	r   *Root
	reg *Registry

	root WalkNode // root for walking

	flags Flag // packing flags

	types Types // types mapping

	cache   map[cipher.SHA256][]byte
	unsaved map[cipher.SHA256][]byte
}

//
// TODO: get interface{}, set interface{}
//

// Get encoded CX object by key
func (p *Pack) Get(key cipher.SHA256) (val []byte) {
	var ok bool
	if val, ok = p.unsaved[key]; ok {
		return
	}
	if val, ok = p.cache[key]; ok {
		return
	}
	// ignore DB error
	p.c.DB().View(func(tx data.Tv) (_ error) {
		val = tx.Objects().Get(key)
		return
	})
	return
}

// Add new encode object returning its hash
func (p *Pack) Add(val []byte) (key cipher.SHA256) {
	key = cipher.SumSHA256(val)
	p.Set(key, val)
	return
}

// Set is Save for already encoded object with its hash
func (p *Pack) Set(key cipher.SHA256, val []byte) {
	if _, ok := p.cache[key]; ok {
		return
	}
	p.unsaved[key] = val
}

// Save and object in the Pack returning its hash and encoded value
func (p *Pack) Save(obj interface{}) (key cipher.SHA256, val []byte) {
	data = encoder.Serialize(obj)
	key = cipher.SumSHA256(data)
	p.Set(key, val)
	return
}

// Del an CX object by key from cache
func (p *Pack) Del(key cipher.SHA256) {
	delete(p.cache, key)
	delete(p.unsaved, key)
}

// Sync internal cache with DB
func (p *Pack) Sync() (err error) {
	if p.flags&ViewOnly != 0 {
		return ErrViewOnlyTree // can't sync view only
	}
	err = p.c.DB().Update(func(tx data.Tu) (err error) {
		return tx.Objects().SetMap(p.unsaved)
	})
	if err == nil {
		for key, val := range p.unsaved {
			p.cache[key] = val
			delete(p.unsaved, key)
		}
	}
	return
}
