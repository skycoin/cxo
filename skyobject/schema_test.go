package skyobject

import (
	"testing"

	"github.com/skycoin/cxo/data"
)

func shouldPanic(t *testing.T) {
	if recover() == nil {
		t.Error("missing panicing")
	}
}

func TestNewRegistery(t *testing.T) {
	t.Run("nil db", func(t *testing.T) {
		defer shouldPanic(t)
		NewRegistery(nil)
	})
	t.Run("norm", func(t *testing.T) {
		db := data.NewDB()
		r := NewRegistery(db)
		if r.db != db {
			t.Error("wrong db in registery")
		}
		if r.nmr == nil {
			t.Error("nil-map for registered types")
		}
		if r.reg == nil {
			t.Error("nil-map registery")
		}
	})
}

func TestRegistry_Register(t *testing.T) {
	t.Run("unnamed", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		defer shouldPanic(t)
		r.Register("Unnamed", []User{})
	})
}

func TestRegistry_SaveSchema(t *testing.T) {
	//
}

func TestRegistry_SchemaByTypeName(t *testing.T) {
	//
}

func TestRegistry_SchemaByReference(t *testing.T) {
	//
}

func TestRegistry_getSchema(t *testing.T) {
	//
}

func TestRegistry_getField(t *testing.T) {
	//
}

func TestSchema_Name(t *testing.T) {
	//
}

func TestSchema_Kind(t *testing.T) {
	//
}

func TestSchema_Elem(t *testing.T) {
	//
}

func TestSchema_Len(t *testing.T) {
	//
}

func TestSchema_Fields(t *testing.T) {
	//
}

func TestSchema_setElem(t *testing.T) {
	//
}

func TestSchema_isNamed(t *testing.T) {
	//
}

func TestSchema_isSaved(t *testing.T) {
	//
}

func TestSchema_load(t *testing.T) {
	//
}

func TestSchema_String(t *testing.T) {
	//
}

func TestField_Kind(t *testing.T) {
	//
}

func TestField_TypeName(t *testing.T) {
	//
}

func TestField_Name(t *testing.T) {
	//
}

func TestField_Schema(t *testing.T) {
	//
}

func TestField_Tag(t *testing.T) {
	//
}

func TestField_tagSchemaName(t *testing.T) {
	//
}

func TestField_isReference(t *testing.T) {
	//
}

func TestField_String(t *testing.T) {
	//
}

func TestSchema_reset(t *testing.T) {
	//
}

func TestSchema_Encode(t *testing.T) {
	//
}

func TestSchema_Decode(t *testing.T) {
	//
}

func Test_typeName(t *testing.T) {
	//
}

func Test_isFlat(t *testing.T) {
	//
}

func Test_mustGetSchemaOfTag(t *testing.T) {
	//
}
