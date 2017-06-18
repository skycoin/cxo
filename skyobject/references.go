package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
)

//
// RgistryReference
//

// RegistryReference represents unique identifier of
// a Registry. The id is hash of encoded registry
type RegistryReference cipher.SHA256

// IsBlank returns true if the reference is blank
func (r *RegistryReference) IsBlank() bool {
	return *r == (RegistryReference{})
}

// String implements fmt.Stringer interface
func (r *RegistryReference) String() string {
	return cipher.SHA256(*r).Hex()
}

//
// SchemaReference
//

// A SchemaReference represents reference to Schema
type SchemaReference cipher.SHA256

// IsBlank returns true if the reference is blank
func (s *SchemaReference) IsBlank() bool {
	return *s == (SchemaReference{})
}

// String implements fmt.Stringer interface
func (s *SchemaReference) String() string {
	return cipher.SHA256(*s).Hex()
}

//
// Reference
//

// A Reference represents reference to object
type Reference cipher.SHA256

// IsBlank returns true if the reference is blank
func (r *Reference) IsBlank() bool {
	return *r == (Reference{})
}

// String implements fmt.Stringer interface
func (r *Reference) String() string {
	return cipher.SHA256(*r).Hex()
}

//
// References
//

// A References represents list of references to objects
type References []Reference

// IsBlank returns true if the reference is blank
func (r References) IsBlank() bool {
	return len(r) == 0
}

// String implements fmt.Stringer interface
func (r References) String() (s string) {
	if len(r) == 0 {
		s = "[]"
		return
	}
	s = "["
	for i, x := range r {
		s += x.String()
		if i < len(r)-1 {
			s += ", "
		}
	}
	s += "]"
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
// or hash reference to shcema only. A Dynamic isinvalid if
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

//
// Root
//

// A RootReference represents reference to Root oobject
type RootReference cipher.SHA256

// IsBlank returns true if the reference is blank
func (r *RootReference) IsBlank() bool {
	return *r == (RootReference{})
}

// String implements fmt.Stringer interface
func (r *RootReference) String() string {
	return cipher.SHA256(*r).Hex()
}
