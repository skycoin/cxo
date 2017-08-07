package skyobject

import (
	"errors"
	"fmt"
	"reflect"

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

// A Ref represents reference to object
type Ref struct {
	// Hash of CX object
	Hash cipher.SHA256
	// internals
	walkNode *walkNode `enc:"-"`
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
// Schema (the Ref is not a part of
// a struct)
func (r *Ref) Schema() Schema {
	if r.walkNode != nil {
		return r.walkNode.sch
	}
	return nil
}

// Value of the Ref. Result can be nil
// if the Ref is blank. The result is
// pointer to golang value
func (r *Ref) Value() (obj interface{}, err error) {
	if r.IsBlank() {
		return // nil, nil
	}
	wn := r.walkNode
	if wn == nil {
		err = errors.New("can't get value of detached Ref")
		return
	}
	if wn.value != nil { // already have
		obj = wn.value
		return
	}
	var sch Schema
	var val []byte
	// schema
	if sch = r.Schema(); sch == nil {
		err = errors.New("can't get value of Ref: nil schema")
		return
	}
	// obtain object
	if val, err = wn.pack.get(r.Hash); err != nil {
		return
	}
	if obj, err = wn.pack.unpackToGo(wn.sch.Name(), val); err != nil {
		return
	}
	wn.value = obj // keep
	return
}

// SetValue replacing the Ref with new, that points
// to given value. It return error if type of given value
// is not type of the Referece. Use nil to clear the
// Ref, making it blank
func (r *Ref) SetValue(obj interface{}) (err error) {
	if obj == nil {
		r.clear()
		return
	}
	wn := r.walkNode
	if wn == nil {
		err = errors.New("can't set value to detached Ref")
		return
	}
	p := wn.pack
	typ := typeOf(obj)
	var sch Schema
	if name, ok := p.types.Inverse[typ]; !ok {
		// detailed error
		err = fmt.Errorf(`can't set value to Ref:
    given object not found in Types map
    reflect.Type of the object: %s`,
			typ.String())
		return
	} else if sch, err = p.reg.SchemaByName(name); err != nil {
		// dtailed error
		err = fmt.Errorf(`can't set value to Ref:
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
		err = fmt.Errorf(`can't set value to Ref:
    type of given object '%T' is not type of the Ref '%s'`,
			obj, wn.sch.String())
		return
	}
	// else everything is ok
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		// obj represents nil of typ
		if val.IsNil() {
			r.clear()
			return
		}
	} else { // not a pointer
		err = errors.New("can't accept non-pointer value")
		return
	}
	// setup references of given obj
	if err = p.setupToGo(val); err != nil {
		return
	}
	key, _ := p.save(obj) // save the object getting hash back
	// setup
	prev := r.Hash
	wn.value = obj
	r.Hash = key
	if key != prev {
		wn.unsave()
	}
	return
}

func (r *Ref) clear() {
	if r.Hash == (cipher.SHA256{}) {
		return
	}
	r.Hash = cipher.SHA256{}
	if wn := r.walkNode; wn != nil {
		wn.value = nil
		wn.unsave() // changed
	}
}

// Copy returs copy of this reference
func (r *Ref) Copy() (cp Ref) {
	cp.Hash = r.Hash
	if wn := r.walkNode; wn != nil {
		cp.walkNode = &walkNode{
			sch:  wn.sch,
			pack: wn.pack,
		}
	}
	return
}
