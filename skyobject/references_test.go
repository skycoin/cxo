package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

func TestReference_IsBlank(t *testing.T) {
	var ref Reference
	if !ref.IsBlank() {
		t.Error("blank is not blank")
	}
	ref = Reference{1, 2, 3}
	if ref.IsBlank() {
		t.Error("non-blank is blank")
	}
}

func TestReference_String(t *testing.T) {
	for _, ref := range []Reference{
		Reference{},
		Reference{1, 2, 3},
	} {
		if ref.String() != cipher.SHA256(ref).Hex() {
			t.Error("wrong string")
		}
	}
}

func TestReferences_IsBlank(t *testing.T) {
	var refs References
	if !refs.IsBlank() {
		t.Error("blank is not balnk")
	}
	refs = References{Reference{}, Reference{}}
	if refs.IsBlank() {
		t.Error("non-blank is balnk")
	}
}

func TestDynamic_IsBlank(t *testing.T) {
	var dh Dynamic
	if !dh.IsBlank() {
		t.Error("blank is not blank")
	}
	dh.Schema = Reference{1, 2, 3}
	if dh.IsBlank() {
		t.Error("non-blank is blank")
	}
}

func TestDynamic_IsValid(t *testing.T) {
	var dh Dynamic
	if !dh.IsValid() {
		t.Error("blank dynamic reference treated as invalid")
	}
	dh.Schema = Reference{1, 2, 3}
	if dh.IsValid() {
		t.Error("invalid is valid")
	}
	dh.Schema = Reference{}
	dh.Object = Reference{1, 2, 3}
	if dh.IsValid() {
		t.Error("invalid is valid")
	}
	dh.Schema = Reference{1, 2, 3}
	if !dh.IsValid() {
		t.Error("valid is invalid")
	}
}
