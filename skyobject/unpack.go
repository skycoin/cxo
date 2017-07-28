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
	EntireMerkleTrees Flag = 1 << iota // unpack entire Merkle trees
	EntireTree                         // unpack all possible
	HashTableIndex                     // use hash-table index for Merkle-trees
	ViewOnly                           // don't allow modifications
	GoTypes                            // pack/unpack from/to golang-values

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
// for new objects saving them in the end.
// The Pack also used to unpack a Root,
// modify it and walk through. The Pack is
// not thread safe. All objects of the
// Pack are not thread safe
type Pack struct {
	c *Container

	r   *Root
	reg *Registry

	flags Flag // packing flags

	types *Types // types mapping

	cache   map[cipher.SHA256][]byte
	unsaved map[cipher.SHA256][]byte
}

// internal get/set/del/add/save methods that works with cipher.SHA256
// instead of Reference

// get by hash from cache or from database
func (p *Pack) get(key cipher.SHA256) (val []byte) {
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
func (p *Pack) Save() (err error) {

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

	// Do we need to unpack entire tree?

	if p.flags&EntireTree == 0 {
		return // do nothing carefully
	}

	// unpack all possible

	//

	return
}

// Root of the Pack
func (p *Pack) Root() *Root { return p.r }

// Registry of the Pack
func (p *Pack) Registry() *Registry { return p.reg }

// Flags of the Pack
func (p *Pack) Flags() Flag { return p.flags }

// Refs returns unpacked references of underlying
// Root. It's not equal to pack.Root().Refs, becaue
// the method returns unpacked references
func (p *Pack) Refs() []Dynamic {
	//
}

// RefByIndex returns one of Root.Refs. If ok is false
// then it's about index out of range error. It's easy to
// use (*Pack).Refs() method to get all Dynamic references
// of underlying Root. But if the tree is not unpacked
// entirely then you can unpack it partially (depending
// your needs) using this method
func (p *Pack) RefByIndex(i int) (dr Dynamic, ok bool) {
	//
}

// SetRefByIndex replaces one of Root.Refs with given obejct.
// If it returns false, then it's about index out of range.
// You can use any object of registered if the Pack created
// with Types. Otherwise the obj must to be Dynamic
func (p *Pack) SetRefByIndex(i int, obj interface{}) (ok bool) {
	//
}

func (p *Pack) testSchemaReference(schr SchemaReference) {
	if dr.Schema != (SchemaReference{}) {
		if _, err := p.reg.SchemaByReference(dr.Schema); err != nil {
			// tobe safe (don't corrupt DB)
			p.c.DB().Close()
			p.c.Panic("unexpected schema referene in given Dynamic [%s]:",
				dr.Schema.Short(),
				err)
		}
	}
}

// Append another Dynamic referenc to Refs of
// underlying Root. If the Pack created with
// Types then you can use any object of
// registered, otherwise it must be
// instance of Dynamic
func (p *Pack) Append(obj interface{}) {
	var dr Dynamic
	var ok bool

	if dr, ok = obj.(Dynamic); !ok {

		//

	} else {
		// check possible end-user mistakes to
		// make his life easer
		p.testSchemaReference(dr.Schema)
	}

	p.r.Refs = append(p.r.Refs, dr) // append
}

type WalkNode struct {
	// TODO (kostyarin): reference, upper node

	value interface{} // golang value or Value

	upper WalkNode // upper node
	pack  *Pack    // back reference to related Pack
}

// Value of the node. It returns nil if the node
// represents nil (empty reference for example).
// It returns golang value if related Pack created
// with appropriate flags. In other cases it returns
// Value (skyobject.Value)
func (w *WalkNode) Value() (obj interface{}) {
	return w.value
}

// SetValue replaces underlying value with given one.
// ViewOnly flag of related Pack doesn't affect this
// method. New value will be set. But it never be
// saved. Any skyobject.Value can't be passed to
// the method
func (w *WalkNode) SetValue(obj interface{}) {
	key, val := w.Pack().save(obj)
	//
	typ := typeOf(obj)

}

// Pack returns related Pack
func (w *WalkNode) Pack() *Pack {
	return w.pack
}

// Upper returns upper (closer to root) node.
// It returns nil if this node is root
func (w *WalkNode) Upper() (upper WalkNode) {
	return w.upper
}
