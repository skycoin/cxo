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

	// TODO (kostyarin): automatic track changes
	// TrackChanges                    // automatic trach changes (experimental)
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

	sk cipher.SecKey

	unsaved map[cipher.SHA256][]byte

	// TOOD (kostyarin): track changes
}

// don't save, just get k-v
func (p *Pack) dsave(obj interface{}) (key cipher.SHA256, val []byte) {
	val = encoder.Serialize(obj)
	key = cipher.SumSHA256(val)
	return
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

// Save all changes in DB returning packed updated Root.
// Use the result to publish upates (node package related)
func (p *Pack) Save() (rp data.RootPack, err error) {
	p.c.Debugln(VerbosePin, "(*Pack).Save", p.r.Pub.Hex()[:7], p.r.Seq)

	tp := time.Now() // time point

	if p.sk == (cipher.SecKey{}) {
		err = fmt.Errorf("can't save Root of %s: empty secret key",
			p.r.Pub.Hex()[:7])
	}

	if p.flags&ViewOnly != 0 {
		err = ErrViewOnlyTree // can't save view only tree
		return
	}

	// TODO (kostyarin): track changes

	// single transaction required (to perform rollback on error)
	err = p.c.DB().Update(func(tx data.Tu) (err error) {

		// save Root

		roots := tx.Feeds().Roots(p.r.Pub)
		if roots == nil {
			return ErrNoSuchFeed
		}

		// seq number and prev hash
		var seq uint64
		var prev cipher.SHA256
		if last := roots.Last(); last != nil {
			seq, prev = last.Seq+1, last.Hash
		}

		// setup
		p.r.Seq = seq
		p.r.Time = time.Now().UnixNano()
		p.r.Prev = prev

		val := p.r.Encode()

		p.r.Hash = cipher.SumSHA256(val)
		p.r.Sig = cipher.SignHash(p.r.Hash, p.sk)

		rp.Hash = p.r.Hash
		rp.IsFull = true
		rp.Prev = p.r.Prev
		rp.Root = val
		rp.Seq = p.r.Seq
		rp.Sig = p.r.Sig

		if err = roots.Add(&rp); err != nil {
			return
		}
		// save objects
		return tx.Objects().SetMap(p.unsaved)
	})

	if err == nil {
		p.unsaved = make(map[cipher.SHA256][]byte) // clear
	}

	st := time.Now().Sub(tp)

	p.c.Debugf(PackSavePin, "%s saved after %v", p.r.Short(), st)

	p.c.stat.addPackSave(st)
	return
}

// Initialize the Pack. It creates Root WalkNode and
// unpack entire tree if appropriate flag is set
func (p *Pack) init() (err error) {
	// Do we need to unpack entire tree?
	if p.flags&EntireTree != 0 {
		// unpack all possible
		_, err = p.RootRefs()
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

// RootRefs unpack all main (see Refs field of Root) closest
// to Root references and returns them. It breaks on first error,
// returning objects unpacked before the error. If any of the
// references is blank then appropriate interface{} in the objs
// reply will be nil
func (p *Pack) RootRefs() (objs []interface{}, err error) {
	p.c.Debugln(VerbosePin, "(*Pack).RootRefs", p.r.Short())

	if len(p.r.Refs) == 0 {
		return // (empty) nil, nil
	}

	objs = make([]interface{}, 0, len(p.r.Refs))

	for i := range p.r.Refs {
		var obj interface{}
		if obj, err = p.RefByIndex(i); err != nil {
			return
		}
		objs = append(objs, obj)
	}
	return
}

func (p *Pack) validateRootRefsIndex(i int) (err error) {
	if i < 0 || i >= len(p.r.Refs) {
		err = ErrIndexOutOfRange
	}
	return
}

// RefByIndex unpacks and returns one of Root.Refs
func (p *Pack) RefByIndex(i int) (obj interface{}, err error) {
	p.c.Debugln(VerbosePin, "(*Pack).RefByIndex", p.r.Short(), i)

	if err = p.validateRootRefsIndex(i); err != nil {
		return
	}

	if p.r.Refs[i].walkNode == nil {
		p.r.Refs[i].walkNode = &walkNode{pack: p}
	}
	obj, err = p.r.Refs[i].Value()
	return
}

// SetByIndex repaces Root.Refs by index with given object.
// Use nil to make the object blank. Type of the object must
// be registered in related Regsitry. It panics, if type of
// the object is not registered
func (p *Pack) SetRefByIndex(i int, obj interface{}) (err error) {
	p.c.Debugf(VerbosePin, "(*Pack).SetRefByIndex %s %d %T", p.r.Short(), i,
		obj)

	if err = p.validateRootRefsIndex(i); err != nil {
		return
	}
	p.r.Refs[i] = p.Dynamic(obj)
	return
}

// Append given obejcts to Root.Refs. Types of the objects must
// be registered in related Regsitry. It panics, if type of one
// of the object is not registered
func (p *Pack) Append(objs ...interface{}) {

	var first interface{} // for debug logs
	if len(objs) != 0 {
		first = objs[1]
	}

	p.c.Debugf(VerbosePin, "(*Pack).Append %s %d %T", p.r.Short(), len(objs),
		first)

	for _, obj := range objs {
		p.r.Refs = append(p.r.Refs, p.Dynamic(obj))
	}
	return
}

// Clear Root.Refs making the slice empty
func (p *Pack) Clear() {
	p.r.Refs = nil
}

// perform
//
// - get schema of the obj
// - if type of the object is not pointer, then the method makes it pointer
// - initialize (setupTo) the obj
//
// reply
//
// sch - schema of nil if obj is nil (or err is not)
// ptr - (1) pointer to value of the obj, (2) the obj if the obj is pointer
//       to non-nil, (3) nil if obj is nil (4) nil if obj represents
//       nil-pointer of some type (for example: var usr *User, the usr
//       is nil pointer to User), (5) nil if err is not
// err - first error
func (p *Pack) initialize(obj interface{}) (sch Schema, ptr interface{},
	err error) {

	if obj == nil {
		return // nil, nil, nil
	}

	// val -> non-pointer value for setupToGo method
	// typ -> non-pointer type for schemaOf method
	// ptr -> pointer

	val := reflect.ValueOf(obj)
	typ := val.Type()

	// we need the obj to be pointer
	if typ.Kind() != reflect.Ptr {
		valp := reflect.New(typ)
		valp.Elem().Set(val)
		ptr = valp.Interface()
	} else {
		typ = typ.Elem() // we need non-pointer type for schemaOf method
		if !val.IsNil() {
			val = val.Elem() // non-pointer value for setupToGo method
			ptr = obj        // it is already a pointer
		}
	}

	if sch, err = p.schemaOf(typ); err != nil {
		return
	}

	if ptr != nil {
		err = p.setupToGo(val)
	}
	return
}

// get Schema by given reflect.Type, the type must not be a pointer
func (p *Pack) schemaOf(typ reflect.Type) (sch Schema, err error) {
	if name, ok := p.types.Inverse[typ]; !ok {
		// detailed error
		err = fmt.Errorf("can't get Schema of %s:"+
			" given object not found in Types.Inverse map",
			typ.String())
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

// blank new Dynamic
func (p *Pack) dynamic() (dr Dynamic) {
	dr.wn = &walkNode{pack: p}
	return
}

// Dynamic creates Dynamic by given object. The obj must to be
// a goalgn value of registered type. The method panics on first
// error. Passing nil returns blank Dynamic
func (p *Pack) Dynamic(obj interface{}) (dr Dynamic) {
	p.c.Debugf(VerbosePin, "(*Pack).Dynamic %s %T", p.r.Short(), obj)

	dr = p.dynamic()

	if err := dr.SetValue(obj); err != nil {
		panic(err)
	}
	return
}

// blank new Ref
func (p *Pack) ref() (r Ref) {
	r.wn = &walkNode{pack: p}
	return
}

// Ref of given object. Type of the object must be registered in
// related Registry. The method panics on first error. Passing nil
// returns blank Ref
func (p *Pack) Ref(obj interface{}) (ref Ref) {
	p.c.Debugf(VerbosePin, "(*Pack).Ref %s %T", p.r.Short(), obj)

	ref = p.ref()

	if err := ref.SetValue(obj); err != nil {
		panic(err)
	}
	return
}

func (p *Pack) refs() (r Refs) {
	r.wn = &walkNode{pack: p}
	r.degree = p.c.conf.MerkleDegree
	if p.flags&HashTableIndex != 0 {
		r.index = make(map[cipher.SHA256]*Ref)
	}
	return
}

// Refs creates Refs by given objects. Types of the objects must be
// registered in related registry. The method panics on first error.
// The method skips all nils. Passing nil, only nils, or nothing,
// returns blank Refs
func (p *Pack) Refs(objs ...interface{}) (r Refs) {

	var first interface{} // for debug logs
	if len(objs) != 0 {
		first = objs[1]
	}

	p.c.Debugf(VerbosePin, "(*Pack).Refs %s %d %T", p.r.Short(), len(objs),
		first)

	r = p.refs()

	if err := r.Append(objs...); err != nil {
		panic(err)
	}
	return
}
