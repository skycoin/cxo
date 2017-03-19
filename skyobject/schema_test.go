package skyobject

import (
	"reflect"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"

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
	t.Run("named", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		r.Register("User", User{})
		tn := typeName(reflect.TypeOf(User{}))
		if x, ok := r.nmr["User"]; !ok {
			t.Error("missing registered type")
		} else if x != tn {
			t.Error("registered with wrong type name")
		} else if ch, ok := r.reg[x]; !ok {
			t.Error("name registered, but type is not registered")
		} else if s, err := r.SchemaByReference(Reference(ch)); err != nil {
			t.Error("unexpected error: ", err)
		} else if s.Name() != tn {
			t.Error("registered type has wrong name: ", s.Name())
		}
	})
}

func TestRegistry_SaveSchema(t *testing.T) {
	t.Run("invalid type", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		var x interface{}
		defer shouldPanic(t)
		r.SaveSchema(x)
	})
	t.Run("valid", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		ur := r.SaveSchema(User{})
		if ur == (Reference{}) {
			t.Error("empty reference to saved type")
		}
		if _, ok := r.db.Get(cipher.SHA256(ur)); !ok {
			t.Error("saved schema missing in db")
		}
		typ := reflect.TypeOf(User{})
		if ch, ok := r.reg[typeName(typ)]; !ok {
			t.Error("saved schema missing in registery")
		} else if Reference(ch) != ur {
			t.Error("wrong reference for saved schema")
		}
	})
	t.Run("recursive", func(t *testing.T) {
		type Recur struct {
			Name   string
			Len    uint32
			Nested []Recur
		}
		r := NewRegistery(data.NewDB())
		ur := r.SaveSchema(Recur{})
		if ur == (Reference{}) {
			t.Error("empty reference to saved type")
		}
		if _, ok := r.db.Get(cipher.SHA256(ur)); !ok {
			t.Error("saved schema missing in db")
		}
		typ := reflect.TypeOf(Recur{})
		if ch, ok := r.reg[typeName(typ)]; !ok {
			t.Error("saved schema missing in registery")
		} else if Reference(ch) != ur {
			t.Error("wrong reference for saved schema")
		}
	})
}

func TestRegistry_SchemaByTypeName(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		_, err := r.SchemaByTypeName("missing")
		if err == nil {
			t.Error("missing error")
		}
	})
	t.Run("saved", func(t *testing.T) {
		r := NewRegistery(data.NewDB())
		r.SaveSchema(User{})
		tn := typeName(reflect.TypeOf(User{}))
		r.SchemaByTypeName(tn)
	})
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
	typ := reflect.Indirect(reflect.ValueOf(User{})).Type()
	got := typeName(typ)
	t.Log("typeName: ", got)
	if want := typ.PkgPath() + "." + typ.Name(); want != got {
		t.Errorf("wrong type name: want %q, got %q", want, got)
	}
}

func Test_isFlat(t *testing.T) {
	t.Run("flat", func(t *testing.T) {
		for _, k := range []reflect.Kind{
			reflect.Bool,
			reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64,
			reflect.String,
		} {
			if !isFlat(k) {
				t.Error("flat type treated as not flat: ", k)
			}
		}
	})
	t.Run("complex", func(t *testing.T) {
		for _, k := range []reflect.Kind{
			reflect.Slice,
			reflect.Array,
			reflect.Struct,
			reflect.Ptr,
		} {
			if isFlat(k) {
				t.Error("complex type treated as flat: ", k)
			}
		}
	})

}

func Test_mustGetSchemaOfTag(t *testing.T) {
	//
}
