package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"strings"
)

// registry reference

// RegistryReference represents unique identifier of
// a Registry. The id is hash of encoded registry
type RegistryReference cipher.SHA256

func (r *RegistryReference) IsBlank() bool {
	return *r == (RegistryReference{})
}

func (r *RegistryReference) String() string {
	return strings.Replace(cipher.SHA256(*r).Hex(), " ", "", -1)
}

// schema reference

type SchemaReference cipher.SHA256

func (s *SchemaReference) IsBlank() bool {
	return *s == (SchemaReference{})
}

func (s *SchemaReference) String() string {
	return strings.Replace(cipher.SHA256(*s).Hex(), " ", "", -1)
}

// reference

type Reference cipher.SHA256

func (r *Reference) IsBlank() bool {
	return *r == (Reference{})
}

func (r *Reference) String() string {
	return strings.Replace(cipher.SHA256(*r).Hex(), " ", "", -1)
}

// references

type References []Reference

func (r References) IsBlank() bool {
	return len(r) == 0
}

// dynamic

type Dynamic struct {
	Object Reference
	Schema SchemaReference
}

func (d *Dynamic) IsBlank() bool {
	return d.Schema.IsBlank() && d.Object.IsBlank()
}

func (d *Dynamic) IsValid() bool {
	if d.Schema.IsBlank() {
		return d.Object.IsBlank()
	}
	return true
}
