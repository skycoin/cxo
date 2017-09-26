package registry

import (
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// A Dynamic represents reference to object
// of some type. The Dynamic keeps reference
// to object and reference to the type
// It is like an interface
type Dynamic struct {
	Hash   cipher.SHA256 // Hash of CX object
	Schema SchemaRef     // Schema is reference to schema
}

// IsValid returns true if the Dynamic is blank, full
// or has reference to shcema only. A Dynamic is invalid if
// it has reference to object but deosn't have reference to
// shcema
func (d *Dynamic) IsValid() bool {
	if d.Schema.IsBlank() {
		return d.Object == (cipher.SHA256{})
	}
	return true
}

// Short string
func (d *Dynamic) Short() string {
	return fmt.Sprintf("{%s:%s}", d.Schema.Short(), d.Object.Hex()[:7])
}

// String implements fmt.Stringer interface
func (d *Dynamic) String() string {
	return "{" + d.Schema.String() + ", " + d.Object.Hex() + "}"
}

// Value of the Dynamic
func (d *Dynamic) Value(pack Pack, obj interface{}) (err error) {
	if true == d.IsBlank() {
		return ErrReferenceRepresentsNil
	}
	// obtain encoded object
	var val []byte
	if val, err = pack.Get(d.Hash); err != nil {
		return
	}
	return encoder.DeserializeRaw(val, obj)
}

// SetValue replacing the Dynamic.Hash with new.
// Use nil to make it blank. Be careful, the SetValue
// never checks and sets Schema hash
func (d *Dynamic) SetValue(pack Pack, obj interface{}) (err error) {
	if true == isNil(obj) {
		d.Clear()
		return
	}
	d.Hash = pack.Add(encoder.Serialize(obj))
	return
}

// Clear the Dynamic making it blank.
// It clears both object reference and schema
// reference
func (d *Dynamic) Clear() {
	d.Hash = cipher.SHA256{}
	d.Schema = SchemaRef{}
}
