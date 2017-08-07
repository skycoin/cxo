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
		err = fmt.Errorf("can't save Root of %s empty secret key",
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

// RootRefs returns unpacked references of underlying
// Root. It's not equal to pack.Root().Refs, becaue
// the method returns unpacked references. Actually
// the method makes the same referenes "unpacked"
func (p *Pack) RootRefs() (drs []Dynamic, err error) {
	p.c.Debugln(VerbosePin, "(*Pack).Refs", p.r.Short())

	for i := range p.r.Refs {
		if p.r.Refs[i].walkNode == nil {
			p.r.Refs[i].walkNode = &walkNode{pack: p}
			if p.flags&EntireTree != 0 {
				if _, err = p.r.Refs[i].Value(); err != nil {
					return
				}
			}
		}
	}

	return
}

// RefByIndex returns one of Root.Refs
func (p *Pack) RefByIndex(i int) (dr Dynamic, err error) {
	p.c.Debugln(VerbosePin, "(*Pack).RefByIndex", p.r.Short(), i)

	if i < 0 || i >= len(p.r.Refs) {
		err = ErrIndexOutOfRange
		return
	}
	if p.r.Refs[i].walkNode == nil {
		p.r.Refs[i].walkNode = &walkNode{pack: p}
		if p.flags&EntireTree != 0 {
			_, err = p.r.Refs[i].Value() // unpack
		}

	}
	dr = p.r.Refs[i]
	return
}

func (p *Pack) SetRefByIndex(i int, obj interface{}) (err error) {
	p.c.Debugf(VerbosePin, "(*Pack).SetRefByIndex %s %d %T", p.r.Short(), i,
		obj)

	if i < 0 || i >= len(p.r.Refs) {
		return ErrIndexOutOfRange
	}
	p.r.Refs[i] = p.Dynamic(obj)
	return
}

// Append given obejct to Refs of
// underlying Root
func (p *Pack) Append(objs ...interface{}) {
	p.c.Debugln(VerbosePin, "(*Pack).Append", p.r.Short(), len(objs))

	for _, obj := range objs {
		p.r.Refs = append(p.r.Refs, p.Dynamic(obj))
	}
	return
}

// Clear referenfes of Root
func (p *Pack) Clear() {
	p.r.Refs = nil
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
	p.c.Debugf(VerbosePin, "(*Pack).Dynamic %s %T", p.r.Short(), obj)

	wn := new(walkNode)
	wn.pack = p

	dr.walkNode = wn

	if err := dr.SetValue(obj); err != nil {
		panic(err)
	}
	return
}

// Ref of given object
func (p *Pack) Ref(obj interface{}) (ref Ref) {
	if obj == nil {
		ref.walkNode = &walkNode{pack: p}
		return
	}

	sch, err := p.schemaOf(obj)
	if err != nil {
		panic(err)
	}
	ref.walkNode = &walkNode{
		sch:  sch,
		pack: p,
	}
	if err := ref.SetValue(obj); err != nil {
		panic(err)
	}
	return
}

// Refs creates Refs by given objects
func (p *Pack) Refs(objs ...interface{}) (r Refs) {
	if len(objs) == 0 {
		r.wn = &walkNode{pack: p}
		return
	}
	sch, err := p.schemaOf(objs[0])
	if err != nil {
		panic(err)
	}
	r.degree = p.c.conf.MerkleDegree
	r.wn = &walkNode{
		sch:  sch,
		pack: p,
	}
	if p.flags&HashTableIndex != 0 {
		r.index = make(map[cipher.SHA256]*Ref)
	}
	if err := r.Append(objs...); err != nil {
		panic(err)
	}
	return
}
