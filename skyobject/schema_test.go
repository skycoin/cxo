package skyobject

import (
	"reflect"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
)

func shouldPanic(t *testing.T) {
	if recover() == nil {
		t.Error("missing panic")
	}
}

// and Decode
func TestSchema_Encode(t *testing.T) {
	var reg schemaReg
	sv := Schema{
		schemaHead: schemaHead{
			kind:     uint32(reflect.Array),
			typeName: []byte("Zorro"),
			sr:       &reg,
		},
		elem: shortSchema{
			schemaHead: schemaHead{
				kind: uint32(reflect.Array),
			},
			schema: []Schema{
				Schema{
					schemaHead: schemaHead{
						kind:     uint32(reflect.Int8),
						typeName: []byte("Age"),
						sr:       &reg,
					},
				},
			},
		},
		length: 12,
	}
	var sr Schema
	if err := sr.Decode(sv.Encode()); err != nil {
		t.Error("encoding error: ", err)
		return
	}
	if sr.kind != uint32(reflect.Array) {
		t.Error("wrong kind decoded:", sr.Kind())
	}
}

func Test_newSchemaReg(t *testing.T) {
	t.Run("nil-db", func(t *testing.T) {
		defer shouldPanic(t)
		newSchemaReg(nil)
	})
	t.Run("norm", func(t *testing.T) {
		reg := newSchemaReg(data.NewDB())
		if reg.nmr == nil {
			t.Error("missing name registry")
		}
		if reg.reg == nil {
			t.Error("missing type name registry")
		}
	})
}

func Test_schemaReg_Register(t *testing.T) {
	reg := newSchemaReg(data.NewDB())
	reg.Register("Man", Man{})
	if name, ok := reg.nmr["Man"]; !ok {
		t.Error("missing registered name")
	} else if name != typeName(reflect.TypeOf(Man{})) {
		t.Error("wrong registered type name: ", name)
	} else if sk, ok := reg.reg[name]; !ok {
		t.Error("missing registered type")
	} else if sk == (cipher.SHA256{}) {
		t.Error("empty key of registered type")
	}
}

func Test_schemaReg_schemaByRegisteredName(t *testing.T) {
	reg := newSchemaReg(data.NewDB())
	reg.Register("Man", Man{})
	if sv, err := reg.schemaByRegisteredName("Man"); err != nil {
		t.Error("unexpected error: ", err)
	} else {
		if !sv.IsNamed() {
			t.Error("named type is unnamed")
		}
		if sv.Name() != typeName(reflect.TypeOf(Man{})) {
			t.Error("wrong type name: ", sv.Name())
		}
		if sv.Len() != 0 {
			t.Error("non-zero length for struct: ", sv.Len())
		}
		if sv.Kind() != reflect.Struct {
			t.Error("wrong kind of struct: ", sv.Kind())
		}
		if _, err := sv.Elem(); err == nil {
			t.Error("a struct has element type")
		}
		if len(sv.Fields()) != 5 {
			t.Error("wrong fields count: ", len(sv.Fields()))
		}
		//
		// TODO: fields
		//
	}
}

func Test_schemaReg_schemaByName(t *testing.T) {
	//
}

func Test_schemaReg_schemaByKey(t *testing.T) {
	//
}

func Test_schemaHead_Kind(t *testing.T) {
	//
}

func Test_schemaHead_IsNamed(t *testing.T) {
	//
}

func Test_schemaHead_Name(t *testing.T) {
	//
}

func Test_shortSchema_Schema(t *testing.T) {
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

func TestSchema_toShort(t *testing.T) {
	//
}

func Test_Field_Name(t *testing.T) {
	//
}

func Test_Field_Tag(t *testing.T) {
	//
}

func Test_Field_Schema(t *testing.T) {
	//
}

func Test_Field_IsReference(t *testing.T) {
	//
}

func Test_Field_SchemaOfReference(t *testing.T) {
	//
}

func Test_Field_TypeName(t *testing.T) {
	//
}

func Test_schemaReg_getSchema(t *testing.T) {
	//
}

func Test_schemaReg_getSchemaOfType(t *testing.T) {
	//
}

func Test_schemaReg_getField(t *testing.T) {
	//
}

func TestSchema_String(t *testing.T) {
	reg := newSchemaReg(data.NewDB())
	man := reg.getSchema(Man{})
	t.Log(man.String())
	mans := reg.getSchema([]Man{})
	t.Log(mans.String())
	t.Log(schemaRegString(reg))
}

func Test_typeName(t *testing.T) {
	t.Run("unnamed", func(t *testing.T) {
		if name := typeName(reflect.TypeOf([]int{})); name != "" {
			t.Error("got name for unnamed type")
		}
		if name := typeName(reflect.TypeOf(int(0))); name != "" {
			t.Error("got name for builtin type")
		}
	})
	t.Run("named", func(t *testing.T) {
		type X struct{}
		typ := reflect.TypeOf(X{})
		if name := typeName(typ); name == "" {
			t.Error("empty name for named type")
		} else if name != typ.PkgPath()+"."+typ.Name() {
			t.Error("wrong type name: ", name)
		}
	})
}

func Test_isBasic(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		for _, kind := range []reflect.Kind{
			reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64,
		} {
			if !isBasic(kind) {
				t.Error("basic type treated as non-basic: ", kind.String())
			}
		}
	})
	t.Run("non-basic", func(t *testing.T) {
		for _, kind := range []reflect.Kind{
			reflect.String,
			reflect.Array,
			reflect.Struct,
			reflect.Slice,
		} {
			if isBasic(kind) {
				t.Error("non-basic type treated as basic: ", kind.String())
			}
		}
	})
}

func Test_isFlat(t *testing.T) {
	t.Run("flat", func(t *testing.T) {
		for _, kind := range []reflect.Kind{
			reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64,
			reflect.String,
		} {
			if !isFlat(kind) {
				t.Error("flat type treated as non-flat: ", kind.String())
			}
		}
	})
	t.Run("non-flat", func(t *testing.T) {
		for _, kind := range []reflect.Kind{
			reflect.Array,
			reflect.Struct,
			reflect.Slice,
		} {
			if isFlat(kind) {
				t.Error("non-flat type treated as flat: ", kind.String())
			}
		}
	})
}

func Test_schemaNameFromTag(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		_, err := schemaNameFromTag("")
		if err == nil {
			t.Error("misisng error")
		}
	})
	t.Run("invalid", func(t *testing.T) {
		_, err := schemaNameFromTag("href,schema=,no-ref")
		if err == nil {
			t.Error("missing error")
		}
	})
	t.Run("valid", func(t *testing.T) {
		name, err := schemaNameFromTag("href,schema=User,no-ref")
		if err != nil {
			t.Error("unexpected error")
		}
		if name != "User" {
			t.Error("wrong schema name: want User, got ", name)
		}
	})
}

func inspectSchema(s *Schema, t *testing.T) {
	var elem = func(e *Schema, err error) string {
		if e == nil {
			return "<nil>"
		}
		return e.Name()
	}
	t.Logf(`Schema{
	Kind:   %s
	Name:   %s,
	sr:     %p,
	Elem:   %s,
	Len:    %d,
	Fields: %v,
}`,
		s.Kind(),
		s.Name(),
		s.sr,
		elem(s.Elem()),
		s.Len(),
		s.fields,
	)
}
