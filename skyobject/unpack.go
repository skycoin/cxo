package skyobject

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

// common packing and unpacking errors
var (
	ErrViewOnlyTree    = errors.New("view only tree")
	ErrIndexOutOfRange = errors.New("index out of range")
)

// A Flag represents unpacking flags
type Flag int

const (
	EntireMerkleTrees Flag = 1 << iota // unpack entire Merkle trees
	EntireTree                         // unpack all possible
	HashTableIndex                     // use hash-table index for Merkle-trees
	ViewOnly                           // don't allow modifications
	GoTypes                            // pack/unpack from/to golang-values

	Default Flag = GoTypes
)

// A Types represents mapping from registered names
// of a Regsitry to reflect.Type and inversed way
type Types struct {
	Direct  map[string]reflect.Type // registered name -> refelect.Type
	Inverse map[reflect.Type]string // refelct.Type -> registered name
}

// A Pack represents database cache for
// new objects. It uses in-memory cache
// for new objects saving them in the end.
// The Pack also used to unpack a Root,
// modify it and walk through. The Pack is
// not thread safe. All objects of the
// Pack are not thread safe
type Pack struct {
	c *Container

	r   *Root
	reg *Registry

	root *WalkNode

	flags Flag // packing flags

	types *Types // types mapping

	cache   map[cipher.SHA256][]byte
	unsaved map[cipher.SHA256][]byte
}

// internal get/set/del/add/save methods that works with cipher.SHA256
// instead of Reference

// get by hash from cache or from database
// the method returns error if object not
// found
func (p *Pack) get(key cipher.SHA256) (val []byte, err error) {
	var ok bool
	if val, ok = p.unsaved[key]; ok {
		return
	}
	if val, ok = p.cache[key]; ok {
		return
	}
	// ignore DB error
	err = p.c.DB().View(func(tx data.Tv) (_ error) {
		val = tx.Objects().Get(key)
		return
	})

	if err == nil && val == nil {
		err = fmt.Errorf("object [%s] not found", key.Hex()[:7])
	}

	return
}

// calculate hash and perform 'set'
func (p *Pack) add(val []byte) (key cipher.SHA256) {
	key = cipher.SumSHA256(val)
	p.Set(key, val)
	return
}

// save encoded CX object (key (hash), value []byte)
func (p *Pack) set(key cipher.SHA256, val []byte) {
	if _, ok := p.cache[key]; ok {
		return
	}
	p.unsaved[key] = val
}

// save interface and get its key and encoded value
func (p *Pack) save(obj interface{}) (key cipher.SHA256, val []byte) {
	data = encoder.Serialize(obj)
	key = cipher.SumSHA256(data)
	p.Set(key, val)
	return
}

//delete from cach and unsaved objects
func (p *Pack) del(key cipher.SHA256) {
	delete(p.cache, key)
	delete(p.unsaved, key)
}

// TORM (kostyarin): useless because of (*Container).CleanUp
//
// // Sync internal cache with DB. The
// // method allways returns ErrViewOnly if
// // the Pack created with ViewOnly flag.
// // In other cases, the error will be DB
// // error or nil.
// func (p *Pack) Sync() (err error) {
//
// 	if p.flags&ViewOnly != 0 {
// 		return ErrViewOnlyTree // can't sync view only
// 	}
//
// 	err = p.c.DB().Update(func(tx data.Tu) (err error) {
// 		return tx.Objects().SetMap(p.unsaved)
// 	})
// 	if err == nil {
// 		for key, val := range p.unsaved {
// 			p.cache[key] = val
// 			delete(p.unsaved, key)
// 		}
// 	}
// 	return
// }

// Save all cahnges in DB
func (p *Pack) Save() (root data.RootPack, err error) {

	// setup timestamp and seq number

	// single transaction required (to perform rollback on error)

	err = p.c.DB().Update(func(tx data.Tu) (err error) {

		// save Root

		// TODO (kostyarin): save Root

		// save objects
		if len(p.unsaved) == 0 {
			return
		}

		err = tx.Objects().SetMap(p.unsaved)
		return
	})

	return

}

// Initialize the Pack. It creates Root WalkNode and
// unpack entire tree if appropriate flag is set
func (p *Pack) init() (err error) {

	// create WalkNode of Root

	p.root = &WalkNode{
		root: true,
		pack: p,
	}

	// Do we need to unpack entire tree?

	if p.flags&EntireTree != 0 {
		// unpack all possible
		_, err = p.Refs()
	}

	return
}

// Root of the Pack
func (p *Pack) Root() *Root { return p.r }

// Registry of the Pack
func (p *Pack) Registry() *Registry { return p.reg }

// Flags of the Pack
func (p *Pack) Flags() Flag { return p.flags }

//
// unpack Root.Refs
//

// Refs returns unpacked references of underlying
// Root. It's not equal to pack.Root().Refs, becaue
// the method returns unpacked references. Actually
// the method makes the same referenes "unpacked"
func (p *Pack) Refs() (drs []Dynamic, err error) {

	for i := range p.r.Refs {
		if dr.walkNode == nil {
			if _, err = p.r.Refs[i].Value(); err != nil {
				return
			}
		}
	}

	return
}

// RefByIndex returns one of Root.Refs. The error can
// be ErrIndexOutOfRange. It's easy to use (*Pack).Refs()
// method to get all Dynamic references of underlying
// Root. But if the tree is not unpacked entirely then
// you can unpack it partially (depending  your needs)
// using this method
func (p *Pack) RefByIndex(i int) (dr Dynamic, err error) {

	if i < 0 || i >= len(p.r.Refs) {
		err = ErrIndexOutOfRange
		return
	}

	if p.r.Refs[i].walkNode == nil {
		_, err = p.r.Refs[i].Value() // unpack
	}

	dr = p.r.Refs[i]
	return
}

// Append another Dynamic reference to Refs of
// underlying Root. If the Pack created with
// Types then you can use any object of
// registered, otherwise it must be
// instance of Dynamic. If Root.Refs is unpacked
// then this method reattaches them to new
// slice (created by append). Thus, a developer
// doesn't need to care about it
func (p *Pack) Append(obj interface{}) (err error) {
	var dr Dynamic

	wn := new(walkNode)
	wn.pack = p

	dr.walkNode = wn

	if err = dr.SetValue(obj); err != nil {
		return
	}

	p.r.Refs = append(p.r.Refs, dr) // append

	// reattach

	for i, dr := range p.r.Refs {
		if dr.walkNode != nil {
			dr.Attach(p.r.Refs, i)
		}
	}
}

// Pop removes last Dynamic reference of
// underlying Root.Refs returning it.
// The returned Dynamic will be detached
// and you can use it anywhere else until
// the Pack is alive. For example you can
// append it to the underlying Root later.
// The detaching is necessary for golang GC
// to collect the result (Dynamic) if it is
// no longer needed. The Pop method is
// opposite to Append. The method returns
// blank Dynamic reference that (can't be used)
// if the Root.Refs is empty. The result wil be
// unpacked
func (p *Pack) Pop() (dr Dynamic, err error) {

	if len(p.r.Refs) == 0 {
		return
	}

	dr = p.r.Refs[len(p.r.Refs)-1] // get last

	if dr.walkNode == nil {
		wn := new(walkNode)
		wn.pack = p
		dr.walkNode = wn
		_, err = dr.Value() // unpack
	} else {
		dr.Detach()
	}

	// remove the dr from Root.Refs

	p.r.Refs[len(p.r.Refs)-1] = Dynamic{} // clear for GC
	p.r.Refs = p.r.Refs[:len(p.r.Refs)-1] // reduce length

	return

}

// Save and object getting Reference to it
func (p *Pack) Reference(obj interface{}) (ref Reference) {
	// TODO (kostyarin): implement
	return
}

// Dynamic creates Dynamic by given object. The obj can
// be another Dynamic reference or goalgn value (if the Pack
// created with Types). The method panics on first error
// (for example: type of obj is not registered). Passing
// nil returns blank Dynamic
func (p *Pack) Dynamic(obj interface{}) (dr Dynamic) {
	wn := new(walkNode)
	wn.pack = p

	dr.walkNode = wn

	_, err := dr.SetValue(obj)
	return
}

func (p *Pack) References(objs ...interface{}) (refs References) {
	// TODO (kostyarin): implement
	return
}
