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

// Value of the Dynamic. The obj argument
// must be a non-nil pointer
func (d *Dynamic) Value(
	pack Pack, //       : pack to get
	obj interface{}, // : pointer to object to decode to
) (
	err error, //       : get or decode error
) {

	if true == d.IsBlank() {
		return ErrReferenceRepresentsNil
	}

	return get(pack, d.Hash, obj)
}

// SetValue replacing the Dynamic.Hash with new.
// Use nil to make it blank. Be careful, the SetValue
// never checks and sets Schema hash. E.g. the
// SchemaRef field will not be changed
func (d *Dynamic) SetValue(
	pack Pack, //       : pack to save
	obj interface{}, // : an encodable object
) (
	err error, //       : saving error
) {

	if true == isNil(obj) {
		d.Clear()
		return
	}

	d.Hash = pack.Add(encoder.Serialize(obj))

	return
}

// Clear the Dynamic making it blank.
// It clears both object reference and
// schema reference
func (d *Dynamic) Clear() {
	d.Hash = cipher.SHA256{}
	d.Schema = SchemaRef{}
}
