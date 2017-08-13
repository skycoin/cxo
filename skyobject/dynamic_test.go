package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

func TestDynamic_IsBlank(t *testing.T) {
	// IsBlank() bool

	var dr Dynamic
	if !dr.IsBlank() {
		t.Error("balnk is not blank")
	}
	dr.SchemaRef = SchemaRef{1, 2, 3}
	if dr.IsBlank() {
		t.Error("non-balnk is blank")
	}
	dr.SchemaRef, dr.Object = SchemaRef{}, cipher.SHA256{1, 2, 3}
	if dr.IsBlank() {
		t.Error("non-balnk is blank")
	}
}

func TestDynamic_IsValid(t *testing.T) {
	// IsValid() bool

	// TODO (kostyarin): low priority
}

func TestDynamic_Short(t *testing.T) {
	// Short() string

	// TODO (kostyarin): low priority
}

func TestDynamic_String(t *testing.T) {
	// String() string

	// TODO (kostyarin): low priority
}

func TestDynamic_Schema(t *testing.T) {
	// Schema() (sch Schema, err error)

	//
}

func TestDynamic_Value(t *testing.T) {
	// Value() (obj interface{}, err error)

	//
}

func TestDynamic_Clear(t *testing.T) {
	// Clear() (err error)

	//
}

func TestDynamic_Copy(t *testing.T) {
	// Copy() (cp Dynamic)

	//
}
