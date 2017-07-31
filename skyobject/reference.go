package skyobject

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

//
// RgistryReference
//

// RegistryReference represents unique identifier of
// a Registry. The id is hash of encoded registry
type RegistryReference cipher.SHA256

// IsBlank returns true if the reference is blank
func (r RegistryReference) IsBlank() bool {
	return r == (RegistryReference{})
}

// String implements fmt.Stringer interface
func (r RegistryReference) String() string {
	return cipher.SHA256(r).Hex()
}

// Short returns first seven bytes of String
func (r RegistryReference) Short() string {
	return r.String()[:7]
}

//
// SchemaReference
//

// A SchemaReference represents reference to Schema
type SchemaReference cipher.SHA256

// IsBlank returns true if the reference is blank
func (s SchemaReference) IsBlank() bool {
	return s == (SchemaReference{})
}

// String implements fmt.Stringer interface
func (s SchemaReference) String() string {
	return cipher.SHA256(s).Hex()
}

// Short returns first seven bytes of String
func (r SchemaReference) Short() string {
	return r.String()[:7]
}

//
// Reference
//

// A Reference represents reference to object
type Reference struct {
	Hash cipher.SHA256 // hash of the reference

	// internals
	walkNode *walkNode `enc:"-"`
}

// IsBlank returns true if the Reference is blank
func (r *Reference) IsBlank() bool {
	return r.Hash == (cipher.SHA256{})
}

// Short returns first 7 bytes of Stirng
func (r *Reference) Short() string {
	return r.Hash.Hex()[:7]
}

// String implements fmt.Stringer interface
func (r *Reference) String() string {
	return r.Hash.Hex()
}

// Eq returns true if given reference is the same as this one
func (r *Reference) Eq(x *Reference) bool {
	return r.Hash == x.Hash
}

// Schema of the Referene. It returns nil
// if the Reference is not unpacked
func (r *Reference) Schema() Schema {
	if r.walkNode != nil {
		return r.walkNode.sch
	}
	return nil
}

// Value of the Reference. Result can be nil
// if the Reference is blank
func (r *Reference) Value() (obj interface{}, err error) {

	if r.IsBlank() {
		return // nil, nil
	}

	wn := r.walkNode

	if wn == nil {
		// TODO (kostyarin): make the error global
		err = errors.New(
			"can't get value of Reference detached from Pack")
		return
	}

	if wn.value != nil {
		// already have
		obj = wn.value
		return
	}

	var sch Schema
	var val []byte

	// schema

	if sch = r.Schema(); sch == nil {
		err = errors.New("can't get value of Reference:" +
			" nil schema, make sure that the Reference created by Pack")
		return
	}

	// obtain object

	if val, err = wn.pack.get(r.Hash); err != nil {
		return
	}

	if wn.pack.flags&GoTypes != 0 {

		if obj, err = wn.pack.unpackToGo(wn.sch.Name(), val); err != nil {
			return
		}

	} else {

		// TODO (kostyarin): other unpacking methods

		panic("not implemented yet")
	}

	wn.value = obj // keep

	return
}

// SetValue replacing the Reference with new, that points
// to given value. It return error if type of given value
// is not type of the Referece. Use nil to clear the
// Reference, making it blank. Feel free to pass another
// Reference or cipher.SHA256 (where the Reference and the
// SHA256 checksum must be of saved object)
func (r *Reference) SetValue(obj interface{}) (err error) {

	if obj == nil {
		r.clear()
		return
	}

	wn := r.walkNode

	if wn == nil {
		// TODO (kostyarin): make the error global
		err = errors.New(
			"can't set value to Reference detached from Pack")
		return
	}

	var ok bool
	if ok, err = r.setHash(obj); ok || err != nil {
		return
	}

	if wn.pack.flags&GoTypes != 0 {

		typ := typeOf(obj)

		var sch Schema

		if name, ok := p.types.Inverse[typ]; !ok {
			// detailed error
			err = fmt.Errorf(`can't set value to Reference:
    given object not found in Types map
    reflect.Type of the object: %s`,
				typ.String())
			return
		} else if sch, err = p.reg.SchemaByName(name); err != nil {
			// dtailed error
			err = fmt.Errorf(`can't set value to Reference:
    wrong Types of Pack; schema name found in Types, but schema by the name not
    found in Registry
    error:                      %s
    registry reference:         %s
    schema name:                %s
    reflect.Type of the obejct: %s`,
				err,
				p.reg.Reference().Short(),
				name,
				typ.String())
			return
		} else if sch != wn.sch {
			// detailed error
			err = fmt.Errorf(`can't set value to Reference:
    type of given object '%T' is not type of the Reference '%s'`,
				obj, wn.sch.String())
			return
		}

		// else everything is ok

		// setup references of given obj

		if err = wn.pack.setupReferencesOfGo(obj); err != nil {
			return
		}

		// save the object getting hash back
		key, _ := p.save(obj)

		// setup

		wn.value = obj
		if key != r.Hash {
			wn.unsave()
		}

		r.Hash = key

	} else {
		// TODO (kostyarin): other packing methods
		panic("not implemented yet")
	}

	wn.set(r) // set to place

	return

}

func (r *Reference) setHash(obj interface{}) (ok bool, err error) {
	var hash cipher.SHA256

	switch x := obj.(type) {
	case Reference:
		if err = r.setReference(&x); err == nil {
			ok = true
		}
		return
	case *Reference:
		if err = r.setReference(x); err == nil {
			ok = true
		}
		return
	case cipher.SHA256:
		hash = x
	case *cipher.SHA256:
		hash = *x
	default:
		return
	}

	if hash == r.Hash {
		return // the same
	}

	if hash == (cipher.SHA256{}) {
		r.clear()
		return
	}

	wn := r.walkNode // is not nil (checked by caller)

	// get value from DB

	var value interface{}
	if value, err = wn.pack.unpack(wn.sch, x.Hash); err != nil {
		return
	}

	// set up

	wn.value = value
	wn.unsave()

	r.Hash = x.Hash

	wn.set(r) // set to place

	ok = true

	return
}

func (r *Reference) setReference(x *Reference) (err error) {

	if r.Hash == x.Hash {
		return
	}

	if x.IsBlank() {
		r.clear()
		return
	}

	wn := r.walkNode // r.walkNode is not nil
	xn := x.walkNode

	var value interface{}

	if xn != nil {
		if xn.sch != nil && xn.sch != wn.sch {
			err = fmt.Errorf(`can't set value to Reference:
    wrong schema of given Refrence (argument)
    want:       %s
    got (arg):  %s`,
				wn.sch.String(),
				xn.sch.String())
			return
		}
		value = xn.value
	}

	if value == nil {
		if value, err = wn.pack.unpack(wn.sch, x.Hash); err != nil {
			return
		}
	}

	// set up

	wn.value = value
	wn.unsave()

	r.Hash = x.Hash

	wn.set(r) // set to place

	return
}

func (r *Reference) clear() {
	if r.Hash == (cipher.SHA256{}) {
		return
	}
	r.Hash = cipher.SHA256{}
	if wn := r.walkNode; wn != nil {
		wn.value = nil
		wn.unsave() // changed
		wn.set(r)
	}
}

// Copy returs copy of this reference. The copy detached
// from its place. Use this mehtod to put the reference to
// any other place
func (r *Reference) Copy() (cp Reference) {
	cp.Hash = r.Hash
	if wn := r.walkNode; wn != nil {
		cp.walkNode = &walkNode{
			sch:  wn.sch,
			pack: wn.pack,
		}
	}
	return
}

// Detach this reference from its place.
// After this call the Reference detached from
// its place and SetValue no longer affects
// keeper of the Reference (the place), but
// the Reference still belongs to related Pack.
// The Copy method returns detached References.
// Detaching not clears Schema of the Reference
func (r *Reference) Detach() {
	if wn := r.walkNode; wn != nil {
		wn.place = nil
	}
}
