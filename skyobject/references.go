package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

//
// use special named types for references
//

// A Reference type represents reference to another object. If a Reference
// is a type of a field then the field must have schema=XXX tag
type Reference cipher.SHA256

// Save an object to db and get reference-key to it
func (r *Root) Save(i interface{}) Reference {
	return Reference(r.cnt.db.AddAutoKey(encoder.Serialize(i)))
}

func (r *Reference) String() string {
	return cipher.SHA256(*r).Hex()
}

// A References type represents references to array of another objects.
// If a References is a type of a field then the field must have schema=XXX tag
type References []Reference

// A Dynamic represents dynamic reference to any object and reference to its
// schema
type Dynamic struct {
	Schema Reference
	Object Reference
}

func (d *Dynamic) IsValid() bool {
	if d.Schema == (Reference{}) {
		return d.Object == (Reference{})
	}
	return d.Object != (Reference{})
}
