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

	walkNode *WalkNode `enc:"-"` // walkNode of this Reference
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

func (r *Reference) attach(upper *WalkNode) {
	r.walkNode.attach(upper)
}

//
// Dynamic
//

// A Dynamic represents dynamic reference that contains
// reference to schema and reference to object
type Dynamic struct {
	Object cipher.SHA256   // reference to object
	Schema SchemaReference // reference to Schema

	// internals

	walkNode *WalkNode // WalkNode of this Dynamic
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

func (d *Dynamic) attach(upper *WalkNode) {
	d.walkNode.attach(upper)
}
