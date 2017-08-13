package skyobject

import (
	"errors"
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"
)

//
// RegistryRef
//

// RegistryRef represents unique identifier of
// a Registry. The id is hash of encoded registry
type RegistryRef cipher.SHA256

// IsBlank returns true if the reference is blank
func (r RegistryRef) IsBlank() bool {
	return r == (RegistryRef{})
}

// String implements fmt.Stringer interface
func (r RegistryRef) String() string {
	return cipher.SHA256(r).Hex()
}

// Short returns first seven bytes of String
func (r RegistryRef) Short() string {
	return r.String()[:7]
}

//
// SchemaRef
//

// A SchemaRef represents reference to Schema
type SchemaRef cipher.SHA256

// IsBlank returns true if the reference is blank
func (s SchemaRef) IsBlank() bool {
	return s == (SchemaRef{})
}

// String implements fmt.Stringer interface
func (s SchemaRef) String() string {
	return cipher.SHA256(s).Hex()
}

// Short returns first seven bytes of String
func (r SchemaRef) Short() string {
	return r.String()[:7]
}

//
// Ref
//

// A Ref represents reference to object.
// You should not change fields of the Ref
// manually
type Ref struct {
	// Hash of CX object
	Hash cipher.SHA256
	// internals
	wn *walkNode `enc:"-"`
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
	if r.wn != nil {
		return r.wn.sch
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

	if r.wn == nil {
		err = errors.New("can't get value: the Ref is not attached to Pack")
		return
	}

	if r.wn.value != nil {
		obj = r.wn.value // already have (already tracked)
		return
	}

	if r.Schema() == nil {
		err = errors.New("can't get value: Schema of the Ref is nil")
		return
	}

	if r.IsBlank() {
		if obj, err = r.wn.pack.nilPtrOf(r.Schema().Name()); err != nil {
			return
		}
		r.wn.value = obj // keep
		r.trackChanges() // if AutoTrackChanges enabled
		return
	}

	// obtain encoded object
	var val []byte
	if val, err = r.wn.pack.get(r.Hash); err != nil {
		return
	}

	// unpack and setup
	if obj, err = r.wn.pack.unpackToGo(r.wn.sch.Name(), val); err != nil {
		return
	}
	r.wn.value = obj // keep

	r.trackChanges() // if AutoTrackChanges enabled
	return
}

func (r *Ref) trackChanges() {
	if f := r.wn.pack.flags; f&AutoTrackChanges != 0 && f&ViewOnly == 0 {
		r.wn.pack.Push(r) // track
	}
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
		r.Clear()
		return
	}

	if r.wn == nil {
		return errors.New("can't set value: the Ref is not attached to Pack")
	}

	if r.wn.pack.flags&ViewOnly != 0 {
		return ErrViewOnlyTree
	}

	var sch Schema
	if sch, obj, err = r.wn.pack.initialize(obj); err != nil {
		return fmt.Errorf("can't set value to Ref: %v", err)
	}

	if r.wn.sch != nil && r.wn.sch != sch {
		return fmt.Errorf(`can't set value: type of given object "%T"`+
			" is not type of the Ref %q", obj, r.wn.sch.String())
	}

	r.wn.sch = sch // keep schema (if it was nil or the same)

	if obj == nil {
		r.Clear() // the obj was a nil pointer of some type
		return
	}

	r.wn.value = obj // keep

	if key, val := r.wn.pack.dsave(obj); key != r.Hash {
		r.Hash = key
		r.wn.pack.set(key, val) // save
		if err = r.wn.unsave(); err != nil {
			return
		}
	}

	r.trackChanges()
	return
}

// if the Ref has wn.value
func (r *Ref) commit() (err error) {

	if r.wn == nil {
		panic("commit not initialized Ref")
	}

	if r.Hash == (cipher.SHA256{}) && r.wn.value == nil {
		return // everything is ok
	}

	if r.wn.value == nil {
		return // never happens
	}

	key, val := r.wn.pack.dsave(r.wn.value)
	if key == r.Hash {
		return
	}

	r.Hash = key
	r.wn.pack.set(key, val) // save
	err = r.wn.unsave()     // bubble up
	return
}

// Clear the Ref making it blank. The Clear
// not clears Schema
func (r *Ref) Clear() (err error) {
	if r.Hash == (cipher.SHA256{}) {
		return // already clear
	}
	r.Hash = cipher.SHA256{}
	if r.wn != nil {
		r.wn.value = nil
		if err = r.wn.unsave(); err != nil { // bubble changes up
			return
		}
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
	if r.wn != nil {
		cp.wn = &walkNode{
			sch:  r.wn.sch,
			pack: r.wn.pack,
		}
	}
	return
}
