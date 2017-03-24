package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
)

// A Reference type represents reference to another object. If a Reference
// is a type of a field then the field must have schema=XXX tag
type Reference cipher.SHA256

// IsBlank returns true if the reference is blank
func (r *Reference) IsBlank() bool {
	return *r == (Reference{})
}

// String returns hexadecimal representation of the reference
func (r *Reference) String() string {
	return cipher.SHA256(*r).Hex()
}

// A References type represents references to array of another objects.
// If a References is a type of a field then the field must have schema=XXX tag
type References []Reference

// IsBlank returns true if the array is empty. It returns false if the array
// isnot empty, but all references it contains are empty
func (r References) IsBlank() bool {
	return len(r) == 0
}

// A Dynamic represents dynamic reference to any object and reference to its
// schema
type Dynamic struct {
	Schema Reference
	Object Reference
}

// IsBlank returns true if the Dynamic has no references
func (d *Dynamic) IsBlank() bool {
	return d.Schema.IsBlank() && d.Object.IsBlank()
}

// IsValid returns false if the Dynamic reference containes blank schema, but
// non-blank object, or blank object but non-blank schema. If both, schema and
// object, are blank, then the Dynamic reference is valid
func (d *Dynamic) IsValid() bool {
	if d.Schema.IsBlank() {
		return d.Object.IsBlank()
	}
	return !d.Object.IsBlank()
}
