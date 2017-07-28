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

	// create WalkNode of Root

	p.root = &WalkNode{
		root: true,
		pack: p,
	}

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

func (p *Pack) dynamicFromInterface(obj interface{}) (dr Dynamic) {
	var ok bool
	if dr, ok = obj.(Dynamic); !ok {
		// create Dynamic from the obj
		dr = p.dynamicFromObj(obj)
	} else {
		// check possible end-user mistakes to
		// make his life easer
		p.testSchemaReference(dr.Schema)
	}
	return
}

func (p *Pack) dynamicFromObj(obj interface{}) (dr Dynamic) {

	typ := typeOf(obj)

	if name, ok := p.types.Inverse[typ]; !ok {
		//
	} else if sch, err := p.reg.SchemaByName(name); err != nil {
		p.c.DB().Close()
		p.c.Panicf(`wrong Types of Pack:
    schema name found in Types, but schema by the name not found in Registry
    error:                  %s
    registry reference:     %s
    schema name:            %s
    reflect.Type of obejct: %s`,
			err,
			p.reg.Reference().Short(),
			name,
			typ.String())
	}

	key, val := p.save(obj)
	var ref Reference

	//

	dr.Object = ref
	return
}

// Append another Dynamic referenc to Refs of
// underlying Root. If the Pack created with
// Types then you can use any object of
// registered, otherwise it must be
// instance of Dynamic
func (p *Pack) Append(obj interface{}) {
	dr := p.dynamicFromInterface(obj)
	dr.attach(p.root)
	p.r.Refs = append(p.r.Refs, dr) // append
}

// Pop removes last Dynamic reference of
// underlying Root.Refs returning it.
// The returned Dynamic will be detached
// and you can use it anywhere else until
// the Pack is alive. For example you can
// append it to the underlying Root later.
// The detching is necessary for golang GC
// to collect the result (Dynamic) if it is
// no longer needed. The Pop method is
// opposite to Append. The method returns
// blank Dynamic reference that (can't be used)
// if the Root.Refs is empty
func (p *Pack) Pop() (dr Dynamic, err error) {

	if len(p.r.Refs) == 0 {
		return
	}

	dr = p.r.Refs[len(p.r.Refs)-1] // get last

	if dr.walkNode == nil {
		// packed, we need to unpack it
		if err = p.unpackDynamic(&dr, p.root); err != nil {
			dr = Dynamic{} // blank Dynamic
			return
		}
	}

	// remove the dr from Root.Refs

	p.r.Refs[len(p.r.Refs)-1] = Dynamic{} // clear for GC
	p.r.Refs = p.r.Refs[:len(p.r.Refs)-1] // reduce length

	//

	dr.attach(p.root)
	return

}

func (p *Pack) unpackToGo(sch Schema, val []byte) (obj interface{}, err error) {
	var typ reflect.Type
	var ok bool

	if typ, ok = p.types.Direct[dr.walkNode.sch.Name()]; !ok {
		err = fmt.Errorf("missing reflect.Type of %q schema in Types.Direct",
			sch.Name())
		return
	}

	ptr := reflect.New(typ)

	if _, err = encoder.DeserializeRawToValue(val, ptr); err != nil {
		return
	}

	obj = reflect.Indirect(ptr).Interface()

	return
}

func (p *Pack) unpackDynamic(dr *Dynamic, upper *WalkNode) (err error) {

	if dr.walkNode == nil {
		dr.walkNode = new(WalkNode)
	}

	// is the dr valid?

	if !dr.IsValid() {
		return ErrInvalidDynamicReference
	}

	// does the dr contain any?

	if dr.IsBlank() {
		// no schema, no object
		dr.walkNode.attach(upper) // last step
		return
	}

	// not blank, but object can be nil anyway

	// schema

	dr.walkNode.sch, err = p.reg.SchemaByReference(dr.Schema)
	if err != nil {
		return
	}

	// is there an object
	if dr.Object == (cipher.SHA256{}) {
		// no object (nil)
		dr.walkNode.attach(upper) // last step
		return
	}

	// get object from database

	var val []byte
	if val, err = p.get(dr.Object); err != nil {
		return
	}

	// unpack to golang value or to Value

	if p.flags&GoTypes != 0 {
		// golang value
		dr.walkNode.value, err = p.unpackToGo(dr.walkNode.sch, val)
	} else {
		// Value (skyobject.Value)
		dr.walkNode.value, err = p.unpackToValue(dr.walkNode.sch, val)
	}

	if err != nil {
		return
	}

	dr.walkNode.attach(upper) // last step
	return

}

// Save and object getting Reference to it
func (p *Pack) Reference(obj interface{}) (ref Reference) {
	// TODO (kostyarin): implement
	return
}

// Dynamic creates Dynamic by given object. The call of
// the method only allowed if the Pack created with Types
func (p *Pack) Dynamic(obj interface{}) (dr Dynamic) {
	// TODO (kostyarin): implement
	return
}

func (p *Pack) References(objs ...interface{}) (refs References) {
	// TODO (kostyarin): implement
	return
}

// TODO (kostyarin): lowercase it
//
// A WalkNode is internl type and likely to
// be removed to lowercased walkNode
type WalkNode struct {
	value interface{} // golang value or Value (skyobject.Value)

	root bool // true if the node represents Root

	sch     Schema    // schema of the value
	upper   *WalkNode // upper node
	unsaved bool      // true if the node new or changed
	pack    *Pack     // back reference to related Pack
}

// Value of the node. It returns nil if the node
// represents nil (empty reference for example).
// It returns golang value if related Pack created
// with appropriate flags. In other cases it returns
// Value (skyobject.Value). The method returns nil
// if the node represents Root obejct. Use methods
// of Pack (such as Append, Pop) to access  and
// modify Root
func (w *WalkNode) Value() (obj interface{}) {
	return w.value
}

// TODO (kostyarin): remove the method
//
// // SetValue replaces underlying value with given one.
// // ViewOnly flag of related Pack doesn't affect this
// // method. New value will be set. But it never be
// // saved. It's safe to pass a Value (skyobject.Value)
// // It's impossible to SetValue it the node represents
// // Root obejct, in this case the method does nothing
// func (w *WalkNode) SetValue(obj interface{}) {
//
// 	if w.root == true {
// 		return
// 	}
//
// 	// TODO (kostyarin): implement the fucking method
//
// }

// Schema of the node. It can be nil if underlyng
// value is nil (for example blank Dynamic referecne)
func (w *WalkNode) Schema() Schema {
	return w.sch
}

// Pack returns related Pack
func (w *WalkNode) Pack() *Pack {
	return w.pack
}

// Upper returns upper (closer to root) node.
// It returns nil if this node is root. And it
// returns nil if the node has been detached
// from its upper node
func (w *WalkNode) Upper() (upper WalkNode) {
	return w.upper
}

func (w *WalkNode) attach(upper *WalkNode) {
	if upper == nil {
		w.upper = nil
	} else {
		w.upper = upper
		w.pack = upper.pack
		w.unsave() // mark as changed
	}
}

// mark the node and all
// nodes above as changed
func (w *WalkNode) unsave() {
	// using non-recursive algorithm
	for up, i := w, 0; up != nil; up, i = up.upper, i+1 {
		if i > 0 && up.unsaved == true {
			break
		}
		up.unsaved = true
	}
}
