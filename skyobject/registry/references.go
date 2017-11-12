package registry

import (
	"github.com/skycoin/skycoin/src/cipher"
)

// A RegistryRef represetns reference to Registry
type RegistryRef cipher.SHA256

// Short returns short string
func (r RegistryRef) Short() string {
	return cipher.SHA256(r).Hex()[:7]
}

// String implements fmt.Stringer interface
func (r RegistryRef) String() string {
	return cipher.SHA256(r).Hex()
}

// IsBlank returns true if the RegistryRef is blank
func (r RegistryRef) IsBlank() bool {
	return r == (RegistryRef{})
}

// A SchemaRef represetns reference to Schema
type SchemaRef cipher.SHA256

// Short returns short string
func (s SchemaRef) Short() string {
	return cipher.SHA256(s).Hex()[:7]
}

// String implements fmt.Stringer interface
func (s SchemaRef) String() string {
	return cipher.SHA256(s).Hex()
}

// IsBlank returns true if the SchemaRef is blank
func (s SchemaRef) IsBlank() bool {
	return s == (SchemaRef{})
}
