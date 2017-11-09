package skyobject

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

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
	// EntireTree - load all possible. Stop on first error
	EntireTree Flag = 1 << iota
	// EntireMerkleTree - load Refs with all brances (not leafs). Otherwise,
	// the Refs will be loaded partially depending needs. The EntireTree and
	// HashTableIndex flags set this one
	EntireMerkleTree
	// HashTableIndex - use hash-table index for Refs to speeds
	// up RefByHash method
	HashTableIndex
	// ViewOnly don't allows modifications
	ViewOnly

	// intrnal flags
	freshRoot // root is not based on some other Root we should not remove
)

// A Pack represents database cache for
// new objects. It uses in-memory cache
// for new objects saving them in the end.
// The Pack also used to unpack a Root,
// modify it and walk through. The Pack is
// not thread safe. All objects of the
// Pack are not thread safe. Underlying Root
// object is changing. Fields of the Root
// contains actual values after Save
// method (if it's successful) and before
// one other method of the Pack called
type Pack struct {
	c *Container // back reference

	r   *Root     // wrapped Root
	reg *Registry // related registry (not nil)

	flags Flag   // packing flags
	types *Types // types mapping

	unsaved map[cipher.SHA256][]byte

	sk     cipher.SecKey // secret key
	unhold sync.Once     // unhold once (for Close method)
}

// Close the Pack, unhold underlying Root
// (if it's holded), release some resources
func (p *Pack) Close() {
	if p.flags&freshRoot == 0 {
		p.unhold.Do(func() {
			p.c.Unhold(p.r.Pub, p.r.Seq)
		})
	}
	p.c = nil     // GC
	p.r = nil     // GC
	p.reg = nil   // GC
	p.types = nil // GC
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
	var rc uint32
	val, rc, err = p.c.DB().CXDS().Get(key)
	if err == nil && rc == 0 {
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

// Save all changes in DB returning error if any.
// If the Save returns an error then this Pack instance
// turn invalid and can't be used anymore
func (p *Pack) Save() (err error) {
	p.c.Debugln(VerbosePin, "(*Pack).Save", p.r.Pub.Hex()[:7], p.r.Seq)

	var tp = time.Now() // time point

	if p.sk == (cipher.SecKey{}) {
		return fmt.Errorf("can't save Root of %s: empty secret key",
			p.r.Pub.Hex()[:7])
	}

	if p.flags&ViewOnly != 0 {
		err = ErrViewOnlyTree // can't save view only tree
		return
	}

	// 1) keep the seq to decrement it next time
	// 2) range over tree getting Volume/Amount of related objects
	//   2.1) get the V/A of already existeing objects
	//   2.2) save new objects indexing them
	// 3) keep all saved objects to decrement them on failure
	var (
		seqDec = p.r.Seq                     // base
		saved  = make(map[cipher.SHA256]int) // decr. on fail
		holded bool                          // new root
	)

	err = p.c.DB().IdxDB().Tx(func(feeds data.Feeds) (err error) {

		var rs data.Roots
		if rs, err = feeds.Roots(p.r.Pub); err != nil {
			return
		}

		// get seq + 1 of last saved
		var called bool
		err = rs.Descend(func(ir *data.Root) (err error) {
			p.r.Seq = ir.Seq + 1 // next seq
			p.r.Prev = ir.Hash   // previous
			called = true
			return data.ErrStopIteration
		})
		if err != nil {
			return
		}

		if called == false {
			p.r.Seq = 0                // first Root
			p.r.Prev = cipher.SHA256{} // first Root
		}

		// save recursive
		sr := new(saveRecursive)
		sr.p = p
		sr.saved = saved

		for i := range p.r.Refs {
			err = sr.saveRecursiveDynamic(reflect.ValueOf(&p.r.Refs[i]).Elem())
			if err != nil {
				return
			}
		}

		// setup
		p.r.Time = time.Now().UnixNano()

		val := p.r.Encode()

		p.r.Hash = cipher.SumSHA256(val)
		p.r.Sig = cipher.SignHash(p.r.Hash, p.sk)

		var ir = new(data.Root)

		// set up the ir
		ir.Seq = p.r.Seq
		ir.Prev = p.r.Prev
		ir.Hash = p.r.Hash
		ir.Sig = p.r.Sig

		// save Root in CXDS
		saved[p.r.Hash]++
		if _, err = p.c.db.CXDS().Set(p.r.Hash, val); err != nil {
			return
		}

		// save Root in index
		if err = rs.Set(ir); err != nil {
			return
		}

		// save registry in CXDS
		saved[cipher.SHA256(p.r.Reg)]++
		_, err = p.c.db.CXDS().Set(cipher.SHA256(p.r.Reg), p.reg.Encode())
		if err != nil {
			return
		}

		// we are inside transaction and we should hold new Root from here
		// to prevent some situations

		p.c.Hold(p.r.Pub, p.r.Seq)
		holded = true
		return
	})

	if err != nil {

		// failure

		// decrement all saved objects
		for hash, c := range saved {
			for i := 0; i < c; i++ {
				if _, dece := p.c.DB().CXDS().Dec(hash); dece != nil {
					p.c.Printf("[ERR] can't decrement %s: %v", hash.Hex()[:7],
						dece)
				}
			}
		}

		// unhold the new Root
		if holded {
			p.c.Unhold(p.r.Pub, p.r.Seq)
		}

		// if the holded is true then p.r.Seq is seq of new Root
		// and seqDec contains seq of previous Root. Since, the
		// Close method Unholds Root by p.r.Seq, then we have to
		// prevent this and unhold the Root manually

		p.unhold.Do(func() {}) // prevent unholding on close

		// we have to unhold previous Root only
		// it the freshRoot flag is not set
		if p.flags&freshRoot == 0 {
			p.c.Unhold(p.r.Pub, seqDec)
		}

	} else {

		// success

		// we have to unhold previous Root if
		// the new one based on another Root

		if p.flags&freshRoot != 0 {

			// this Root is not based on anything, it's fresh, created
			// from scratch, but next root will be based on this new
			// Root, and this new Root is holded, the Close method unholds
			// it, but we need to unset the freshRoot flag

			p.unsetFlag(freshRoot)

		} else {

			// this Root is based on another one, we need to unhold previous
			// Root object; this new Root alredy holded inside transaction

			p.c.Unhold(p.r.Pub, seqDec)

		}

		// all objects has been saved and we can clear
		// cache to reduce memory pressure

		p.unsaved = make(map[cipher.SHA256][]byte)

	}

	st := time.Now().Sub(tp)

	if err == nil {
		p.c.packSave.Add(st) // add to statistic
		p.c.Debugf(PackSavePin, "%s saved after %v, %d objects saved",
			p.r.Short(), st, len(saved))
	} else {
		p.c.Debugf(PackSavePin, "%s error saving after %v", p.r.Short(), st)
	}

	return
}

// Initialize the Pack. It unpacks entire tree
// if appropriate flag is set
func (p *Pack) init() (err error) {
	// Do we need to unpack entire tree?
	if p.flags&EntireTree != 0 {
		// unpack all possible
		if _, err = p.RootRefs(); err != nil {
			return
		}
	}
	return
}

// Root of the Pack. It returns underlying Root object
// that changes with changes in Pack. Fields of the Root
// are actual after every successful Save (or Unpack) and
// before first chagnes
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

func validateIndex(i, ln int) (err error) {
	if i < 0 {
		err = fmt.Errorf("negative index %d", i)
	} else if i >= ln {
		err = fmt.Errorf("index out of range %d (len %d)", i, ln)
	}
	return
}

func (p *Pack) validateRootRefsIndex(i int) error {
	return validateIndex(i, len(p.r.Refs))
}

// RefByIndex unpacks and returns one of Root.Refs
func (p *Pack) RefByIndex(i int) (obj interface{}, err error) {
	p.c.Debugln(VerbosePin, "(*Pack).RefByIndex", p.r.Short(), i)

	if err = p.validateRootRefsIndex(i); err != nil {
		return
	}

	if p.r.Refs[i].isInitialized() == false {
		p.initializeDynamic(&p.r.Refs[i])
	}
	obj, err = p.r.Refs[i].Value()
	return
}

// SetRefByIndex replaces Root.Refs by index with given object.
// Use nil to make the object blank. Type of the object must
// be registered in related Registry. It panics, if type of
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

// Append given objects to Root.Refs. Types of the objects must
// be registered in related Registry. It panics, if type of one
// of the object is not registered
func (p *Pack) Append(objs ...interface{}) {

	var first interface{} // for debug logs
	if len(objs) != 0 {
		first = objs[0]
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
	p.unsaved = make(map[cipher.SHA256][]byte)
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
		val = valp.Elem() // addressable non-pinter value for setupToGo method
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

// make nil pointer of given type
func makeNilOf(typ reflect.Type) (nilPtr interface{}) {
	if typ.Kind() != reflect.Ptr {
		typ = reflect.PtrTo(typ) // pointer to
	}
	return reflect.New(typ).Elem().Interface() // elem is nil of some type
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
    reflect.Type of the object: %s`,
			err,
			p.reg.Reference().Short(),
			name,
			typ.String())
	}
	return
}

func (p *Pack) initializeDynamic(dr *Dynamic) {
	dr.dn = &drNode{pack: p}
}

// Dynamic creates Dynamic by given object. The obj must to be
// a golang value of registered type. The method panics on first
// error. Passing nil returns blank Dynamic
func (p *Pack) Dynamic(obj interface{}) (dr Dynamic) {
	p.c.Debugf(VerbosePin, "(*Pack).Dynamic %s %T", p.r.Short(), obj)

	p.initializeDynamic(&dr)

	if err := dr.SetValue(obj); err != nil {
		panic(err)
	}
	return
}

func (p *Pack) initializeRef(r *Ref) {
	r.rn = &refNode{pack: p}
}

// Ref of given object. Type of the object must be registered in
// related Registry. The method panics on first error. Passing nil
// returns blank Ref
func (p *Pack) Ref(obj interface{}) (ref Ref) {
	p.c.Debugf(VerbosePin, "(*Pack).Ref %s %T", p.r.Short(), obj)

	p.initializeRef(&ref)

	if err := ref.SetValue(obj); err != nil {
		panic(err)
	}
	return
}

func (p *Pack) initializeRefs(r *Refs) {
	r.rn = &refsNode{pack: p}
	r.degree = p.c.conf.MerkleDegree
	if p.flags&HashTableIndex != 0 {
		r.rn.index = make(map[cipher.SHA256]*RefsElem)
	}
}

// Refs creates Refs by given objects. Types of the objects must be
// registered in related registry. The method panics on first error.
// The method skips all nils. Passing nil, only nils, or nothing,
// returns blank Refs
func (p *Pack) Refs(objs ...interface{}) (r Refs) {

	var first interface{} // for debug logs
	if len(objs) != 0 {
		first = objs[0]
	}

	p.c.Debugf(VerbosePin, "(*Pack).Refs %s %d %T", p.r.Short(), len(objs),
		first)

	p.initializeRefs(&r)

	if err := r.Append(objs...); err != nil {
		panic(err)
	}
	return
}

// SetFlag is experimental. Handle with care
func (p *Pack) SetFlag(flag Flag) {
	p.setFlag(flag)
}

// for tests, no checks
func (p *Pack) setFlag(flag Flag) {
	p.flags = p.flags | flag
}

// UnsetFlag is experimental. It's impossible
// to clear ViewOnly flag (it panics). Handle with
// care
func (p *Pack) UnsetFlag(flag Flag) {
	if flag&ViewOnly != 0 {
		panic("can't unset ViewOnly flag: recrete the Pack")
	}
	p.unsetFlag(flag)
}

// for tests, no checks
func (p *Pack) unsetFlag(flag Flag) {
	p.flags = p.flags &^ flag
}
