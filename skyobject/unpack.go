package skyobject

import (
	"errors"
	"fmt"
	"reflect"
	"time"

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
	EntireTree     Flag = 1 << iota // unpack all possible
	HashTableIndex                  // use hash-table index for Merkle-trees
	ViewOnly                        // don't allow modifications
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

	flags Flag   // packing flags
	types *Types // types mapping

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
	p.set(key, val)
	return
}

// save encoded CX object (key (hash), value []byte)
func (p *Pack) set(key cipher.SHA256, val []byte) {
	p.unsaved[key] = val
}

// save interface and get its key and encoded value
func (p *Pack) save(obj interface{}) (key cipher.SHA256, val []byte) {
	val = encoder.Serialize(obj)
	key = p.add(val)
	return
}

// delete from unsaved objects (TORM (kostyarin): never used)
func (p *Pack) del(key cipher.SHA256) {
	delete(p.unsaved, key)
}

// Save all cahnges in DB returning packed updated Root.
// Use the result to publish upates (node package related)
func (p *Pack) Save() (root data.RootPack, err error) {

	// TODO (kostyarin): track saving time and put it into
	//                   statistic of Container

	if p.flags&ViewOnly != 0 {
		err = ErrViewOnlyTree // can't save view only tree
		return
	}

	// setup timestamp and seq number
	p.r.Time = time.Now().UnixNano() //
	//

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

	if err == nil {
		p.unsaved = make(map[cipher.SHA256][]byte) // clear
	}

	return

}

// Initialize the Pack. It creates Root WalkNode and
// unpack entire tree if appropriate flag is set
func (p *Pack) init() (err error) {
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
		if p.r.Refs[i].walkNode == nil {
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
func (p *Pack) Append(objs ...interface{}) (err error) {
	if len(objs) == 0 {
		return
	}

	drs := make([]Dynamic, 0, len(objs))

	for _, obj := range objs {
		var dr Dynamic

		wn := new(walkNode)
		wn.pack = p

		dr.walkNode = wn

		if err = dr.SetValue(obj); err != nil {
			return
		}
		drs = append(drs, dr)
	}

	p.r.Refs = append(p.r.Refs, drs...) // append

	// reattach

	for i, dr := range p.r.Refs {
		if dr.walkNode != nil {
			dr.Attach(p.r.Refs, i)
		}
	}
	return
}

func (p *Pack) schemaOf(obj interface{}) (sch Schema, err error) {
	typ := typeOf(obj)

	if name, ok := p.types.Inverse[typ]; !ok {
		// detailed error
		err = fmt.Errorf(
			"can't get Schema of %T: given object not found in Types map",
			obj)
		return
	} else if sch, err = p.reg.SchemaByName(name); err != nil {
		// dtailed error
		err = fmt.Errorf(`wrong Types of Pack:
    schema name found in Types, but schema by the name not found in Registry
    error:                      %s
    registry reference:         %s
    schema name:                %s
    reflect.Type of the obejct: %s`,
			err,
			p.reg.Reference().Short(),
			name,
			typ.String())
	}
	return
}

// Dynamic creates Dynamic by given object. The obj must to be
// a goalgn value of registered type. The method panics on first
// error. Passing nil returns blank Dynamic
func (p *Pack) Dynamic(obj interface{}) (dr Dynamic) {
	wn := new(walkNode)
	wn.pack = p

	dr.walkNode = wn

	if err := dr.SetValue(obj); err != nil {
		panic(err)
	}
	return
}

func (p *Pack) unpack(sch Schema, hash cipher.SHA256) (value interface{},
	err error) {

	var val []byte
	if val, err = p.get(hash); err != nil {
		return
	}

	value, err = p.unpackToGo(sch.Name(), val)
	return
}

// if val is unaddressable then, in some cases, we need to make
// it addressable to be able to call (*Pack).setupToGo
func makeAddressable(typ reflect.Type, val reflect.Value) (reflect.Value,
	interface{}) {

	nv := reflect.Indirect(reflect.New(typ))
	nv.Set(val)
	return nv, nv.Interface()
}
