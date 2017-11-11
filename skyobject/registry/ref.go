package registry

import (
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

//
// Ref
//

// A Ref represents reference to object.
// It is like a pointer
type Ref struct {
	Hash cipher.SHA256 // Hash of CX object
}

// IsBlank returns true if the Ref is blank
func (r *Ref) IsBlank() bool {
	return r.Hash == (cipher.SHA256{})
}

// Short returns first 7 bytes of Stirng
func (r *Ref) Short() string {
	return r.Hash.Hex()[:7]
}

// String implements fmt.Stringer interface
func (r *Ref) String() string {
	return r.Hash.Hex()
}

// Value of the Ref
func (r *Ref) Value(pack Pack, obj interface{}) (err error) {

	if r.IsBlank() {
		return ErrReferenceRepresentsNil
	}

	// obtain encoded object
	var val []byte
	if val, err = pack.Get(r.Hash); err != nil {
		return
	}

	return encoder.DeserializeRaw(val, obj)
}

// SetValue replacing the Ref with new. Use nil-interface{} to clear
func (r *Ref) SetValue(pack Pack, obj interface{}) (err error) {

	if true == isNil(obj) {
		r.Clear()
		return
	}

	r.Hash = pack.Add(encoder.Serialize(obj))

	return

}

// Clear the Ref making it blank
func (r *Ref) Clear() {
	r.Hash = cipher.SHA256{}
}

// isNil interface or nil pointer of a type
func isNil(obj interface{}) bool {

	if obj == nil {
		return true
	}

	val := reflect.ValueOf(obj)

	return val.Kind() == reflect.Ptr && val.IsNil()
}
