package skyobject

import (
	"errors"
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"
)

//
// Ref
//

// A Ref represents reference to object
// Use SetHash method to update Hash
// field of the Ref
type Ref struct {
	// Hash of CX object
	Hash cipher.SHA256
	// internals
	rn *refNode `enc:"-"` // schema, value, pack

	// true if the Hash field has been
	// changed or fresh and non-blank
	ch bool `enc:"-"`
}

func (r *Ref) isInitialized() bool {
	return r.rn != nil
}

type refNode struct {
	sch   Schema      // scehma of related Ref
	value interface{} // golang-value of related Ref
	pack  *Pack       // related Pack
}

// IsBlank returns true if the Ref is blank
func (r *Ref) IsBlank() bool {
	return r.Hash == (cipher.SHA256{})
}

// Short returns first 7 bytes of Stirng
func (r *Ref) Short() string {
	return r.Hash.Hex()[:7]
}

// String implements fmt.Stringer interface
func (r *Ref) String() string {
	return r.Hash.Hex()
}

// Eq returns true if given reference is the same as this one
func (r *Ref) Eq(x *Ref) bool {
	return r.Hash == x.Hash
}

// Schema of the Referene. It returns nil
// if the Ref is not unpacked or has not a
// Schema (the Ref is not a part of a struct)
func (r *Ref) Schema() Schema {
	if r.rn != nil {
		return r.rn.sch
	}
	return nil
}

// Value of the Ref. The obj will be pointer of some type.
// It can be pointer to nil if the Ref is blank but has
// a Schema. For example
//
//     valueInterface, err := ref.Value()
//     if err != nil {
//         // handle the err
//     }
//     usr := valueInterface.(*User)
//
//     // the usr can be nil if the ref is blank,
//     // but if the Value call doesn't return an error
//     // the the valueInterface will be type of the *User
//     // (will be type of the Schema of the Ref)
//
// Type of the obj is pointer, even if you pass non-pointer
// to the SetValue method
func (r *Ref) Value() (obj interface{}, err error) {

	if r.rn == nil {
		err = errors.New("can't get value: the Ref is not attached to Pack")
		return
	}

	if r.rn.value != nil {
		obj = r.rn.value // already have
		return
	}

	if r.Schema() == nil {
		err = errors.New("can't get value: Schema of the Ref is nil")
		return
	}

	if r.IsBlank() {
		if obj, err = r.rn.pack.nilPtrOf(r.Schema().Name()); err != nil {
			return
		}
		r.rn.value = obj // keep
		return
	}

	// obtain encoded object
	var val []byte
	if val, err = r.rn.pack.get(r.Hash); err != nil {
		return
	}

	// unpack and setup
	if obj, err = r.rn.pack.unpackToGo(r.rn.sch.Name(), val); err != nil {
		return
	}
	r.rn.value = obj // keep

	return
}

// SetHash of the Ref to given one
func (r *Ref) SetHash(hash cipher.SHA256) {
	if r.Hash == hash {
		return
	}
	r.Hash = hash
	r.ch = true
	if r.rn != nil {
		r.rn.value = nil // clear related value
	}
	return
}

// SetValue replacing the Ref with new, that points
// to given value. It return error if type of given value
// is not type of the Referece. Use nil to clear the
// Ref, making it blank. The nil does not clear Schema
// For examples:
//
//     // this ref represents reference to *User
//     if err := ref.SetValue(User{"Alice"}); err != nil {
//         // handle the err
//     }
//
// Feel free to pass pointer or non-pointer objects
//
//     if err := ref.SetValue(&User{"Eva"}); err != nil {
//         // handle the err
//     }
//
// A Ref has it's schema described by struct tag. Thus you
// can't pass object of another type
//
//     if err := ref.SetValue(Dog{"Bobick"}); err != nil {
//          // we are here
//      }
//
// You can't set a value if related Pack created with
// ViewOnly flag (or with blank secret key)
func (r *Ref) SetValue(obj interface{}) (err error) {

	if obj == nil {
		println("(1)")
		r.Clear()
		return
	}

	if r.rn == nil {
		return errors.New("can't set value: the Ref is not attached to Pack")
	}

	if r.rn.pack.flags&ViewOnly != 0 {
		return ErrViewOnlyTree
	}

	var sch Schema
	if sch, obj, err = r.rn.pack.initialize(obj); err != nil {
		return fmt.Errorf("can't set value to Ref: %v", err)
	}

	if r.rn.sch != nil && r.rn.sch != sch {
		return fmt.Errorf(`can't set value: type of given object "%T"`+
			" is not type of the Ref %q", obj, r.rn.sch.String())
	}

	r.rn.sch = sch // keep schema (if it was nil or the same)

	if obj == nil {
		r.Clear() // the obj was a nil pointer of some type
		return
	}

	r.rn.value = obj // keep
	if key, val := r.rn.pack.dsave(obj); key != r.Hash {
		r.Hash = key
		r.ch = true
		r.rn.pack.set(key, val) // save
	}

	return
}

// Clear the Ref making it blank. The Clear
// not clears Schema
func (r *Ref) Clear() (err error) {
	if r.Hash == (cipher.SHA256{}) {
		return // already clear
	}
	r.Hash = cipher.SHA256{}
	r.ch = true
	if r.rn != nil {
		r.rn.value = nil
	}
	return
}

// Copy returs copy of this reference.
// The Copy will have the same schema and hash.
// But underlying value (golagn value) will be nil
// and the value can be extracted from DB. This will
// be a new instance. Anyway, you can just ref.Value()
// to get it. If the Ref is part of Refs, the Copy
// will not be a part of the Refs
func (r *Ref) Copy() (cp Ref) {
	cp.Hash = r.Hash
	cp.ch = true // fresh object
	if r.rn != nil {
		cp.rn = &refNode{
			sch:  r.rn.sch,
			pack: r.rn.pack,
		}
	}
	return
}

func (r *Ref) Save() {
	if r.rn == nil {
		return
	}
	if r.rn.value == nil {
		return
	}

	key, val := r.rn.pack.dsave(r.rn.value)
	if r.Hash == key {
		return // exactly the same
	}
	r.rn.pack.set(key, val)
	r.ch = true
}
