package skyobject

import (
	"fmt"

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

//
// Reference
//

// A Reference represents reference to object
type Reference struct {
	Hash cipher.SHA256 // hash of the reference

	value   interface{} `enc:"-"` // underlying value (set or unpacked)
	changed bool        `enc:"-"` // has been changed if true

	upper interface{} `enc:"-"` // reference to upper node (todo)

	sch Schema       `enc:"-"` // schema of the reference (can be nil)
	pu  PackUnpacker `enc:"-"` // pack/unpack
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

// Schema of the Reference. It can be nil
func (r *Reference) Schema() Schema {
	return r.sch
}

// IsChanged returns ture if the Reference has
// been changed. It also returns true if underlying
// (referenced) value has been changed
func (r *Reference) IsChanged() bool {
	return r.changed
}

// Set new value, changing the reference
func (r *Reference) Set(val interface{}) (err error) {

	// TODO

	switch vv := val.(type) {
	case *Reference:
		r.Hash = vv.Hash
		r.value = vv.value
		r.changed = true
	case cipher.SHA256:
		r.Hash = vv
		r.value = nil
		r.changed = true
	default:
		//
	}
	return

}

//
// Dynamic
//

// A Dynamic represents dynamic reference that contains
// reference to schema and reference to object
type Dynamic struct {
	Object Reference
	Schema SchemaReference
}

// IsBlank returns true if the reference is blank
func (d *Dynamic) IsBlank() bool {
	return d.Schema.IsBlank() && d.Object.IsBlank()
}

// IsValid returns true if the Dynamic is blank, full
// or hash reference to shcema only. A Dynamic is invalid if
// it has reference to object but deosn't have reference to
// shcema
func (d *Dynamic) IsValid() bool {
	if d.Schema.IsBlank() {
		return d.Object.IsBlank()
	}
	return true
}

// String implements fmt.Stringer interface
func (d *Dynamic) String() string {
	return "{" + d.Schema.String() + ", " + d.Object.String() + "}"
}

// Eq returns true if given reference is the same as this one
func (d *Dynamic) Eq(x *Dynamic) bool {
	return d.Schema == x.Schema && d.Object.Eq(&x.Object)
}

// TODO: Set, IsCahnged, Schema (?)
