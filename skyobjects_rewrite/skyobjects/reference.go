package skyobjects

import "github.com/skycoin/skycoin/src/cipher"

type href struct {
	Type cipher.SHA256
	Data []byte
}

// IReference interface.
type IReference interface {
	save(c *Container)
	GetKey() cipher.SHA256
	GetType() cipher.SHA256
	GetSchema(c *Container) *Schema
	GetField(c *Container, fieldName string) IReference
	GetChildren(c *Container) []IReference
	GetDescendants(*Container, map[cipher.SHA256]IReference)
	Deserialize(dataOut interface{}) error
}

// Reference struct.
type Reference struct {
	key       cipher.SHA256
	schemaKey cipher.SHA256 `enc:"-"`
	data      []byte        `enc:"-"`
	value     interface{}   `enc:"-"`
	// c         *Container    `enc:"-"`
}

// Key is a single key.
type Key cipher.SHA256

// ToReference converts key to reference.
func (k Key) ToReference(c *Container) IReference {
	return c.Get(cipher.SHA256(k))
}

// SHA256 converts Key to SHA256.
func (k Key) SHA256() cipher.SHA256 {
	return cipher.SHA256(k)
}

// KeyArray is an array of keys.
type KeyArray []cipher.SHA256

// ToReferences converts key array to array of References.
func (a KeyArray) ToReferences(c *Container) (objRefArray []IReference) {
	for _, key := range a {
		objRefArray = append(objRefArray, c.Get(key))
	}
	return
}
