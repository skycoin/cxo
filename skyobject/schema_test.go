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

func shouldNotPanic(t *testing.T) {
	if recover() != nil {
		t.Error("unexpected panicing")
	}
}

func cmpSchemes(a, b *Schema, t *testing.T) (equal bool) {
	equal = true
	if a.Kind() != b.Kind() {
		t.Errorf("missmatch kind: %s - %s", a.Kind(), b.Kind())
		equal = false
	}
	if a.Name() != b.Name() {
		t.Errorf("missmatch name: %q - %q", a.Name(), b.Name())
		equal = false
	}
	if len(a.elem) != len(b.elem) {
		t.Errorf("missmatch elements length: %d - %d",
			len(a.elem), len(b.elem))
		equal = false
	} else if len(a.elem) == 1 {
		equal = equal || cmpSchemes(&a.elem[0], &b.elem[0], t)
	}
	if a.length != b.length {
		t.Errorf("missmatch length: %d - %d",
			a.length, b.length)
		equal = false
	}
	if len(a.fields) != len(b.fields) {
		t.Errorf("missmatch fields length: %d - %d",
			len(a.fields), len(b.fields))
		equal = false
	} else {
		for i := range a.fields {
			af := a.fields[i]
			bf := b.fields[i]
			if af.Name() != bf.Name() {
				t.Errorf("missmatch field name: %q - %q", af.Name(), bf.Name())
				equal = false
			}
			if af.Tag() != bf.Tag() {
				t.Errorf("missmatch tags: `%s` - `%s`", af.Tag(), bf.Tag())
				equal = false
			}
			if string(af.ref) != string(bf.ref) {
				t.Errorf("missmatch reference: %q - %q",
					string(af.ref), string(bf.ref))
				equal = false
			}
			equal = equal || cmpSchemes(&af.schema, &bf.schema, t)
		}
	}
	return
}

func Test_newRegistery(t *testing.T) {
	t.Run("nil db", func(t *testing.T) {
		defer shouldPanic(t)
		newRegistery(nil)
	})
	t.Run("norm", func(t *testing.T) {
		db := data.NewDB()
		r := newRegistery(db)
		if r.db != db {
			t.Error("wrong db in registery")
		}
		if r.nnm == nil {
			t.Error("nil-map for registered types")
		}
		if r.reg == nil {
			t.Error("nil-map registery")
		}
	})
}

func TestRegistry_Register(t *testing.T) {
	t.Run("unnamed", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		defer shouldPanic(t)
		r.Register("Unnamed", []User{})
	})
	t.Run("named", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		r.Register("User", User{})
		if sck, ok := r.reg["User"]; !ok {
			t.Error("missing registered type")
		} else if sck == (Reference{}) {
			t.Error("empty reference")
		}
		typ := reflect.TypeOf(User{})
		if r.nnm[typ] != "User" {
			t.Error("something went wrong")
		}
	})
	t.Run("invalid type", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		var x interface{}
		defer shouldPanic(t)
		r.Register("User", x)
	})
	t.Run("empty name", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		defer shouldPanic(t)
		r.Register("", User{})
	})
	t.Run("special name prefix", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		defer shouldPanic(t)
		r.Register("-User", User{})
	})
	t.Run("unregistered nested", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		defer shouldPanic(t)
		type Y struct{}
		type X struct {
			Y Y
		}
		r.Register("X", X{})
	})
	t.Run("twice the same", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		defer shouldNotPanic(t)
		r.Register("X", User{})
		r.Register("X", User{})
	})
	t.Run("twice different", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		defer shouldPanic(t)
		type X struct{}
		r.Register("X", User{})
		r.Register("X", X{})
	})
	t.Run("unnamed type", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		defer shouldPanic(t)
		r.Register("User", int(0))
	})
	t.Run("valid", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		r.Register("User", User{})
		ur := r.SchemaReference(User{})
		if ur == (Reference{}) {
			t.Error("empty reference to saved type")
		}
		if _, ok := r.db.Get(cipher.SHA256(ur)); !ok {
			t.Error("saved schema missing in db")
		}
		typ := reflect.TypeOf(User{})
		if _, ok := r.reg["User"]; !ok {
			t.Error("saved schema missing in registery")
		} else if r.nnm[typ] != "User" {
			t.Error("wrong")
		}
	})
	t.Run("recursive", func(t *testing.T) {
		type Recur struct {
			Name   string
			Len    uint32
			Nested []Recur
		}
		r := newRegistery(data.NewDB())
		r.Register("Recur", Recur{})
		ur := r.SchemaReference(Recur{})
		if ur == (Reference{}) {
			t.Error("empty reference to saved type")
		}
		if _, ok := r.db.Get(cipher.SHA256(ur)); !ok {
			t.Error("saved schema missing in db")
		}
		typ := reflect.TypeOf(Recur{})
		if name, ok := r.nnm[typ]; !ok {
			t.Error("saved schema missing in registery")
		} else if r.reg[name] != ur {
			t.Error("wrong reference for saved schema")
		}
	})
	t.Run("self reference", func(t *testing.T) {
		type Recur struct {
			Name   string
			Len    uint32
			Nested Reference `skyobject:"schema=Recur"`
		}
		r := newRegistery(data.NewDB())
		r.Register("Recur", Recur{})
		ur := r.SchemaReference(Recur{})
		if ur == (Reference{}) {
			t.Error("empty reference to saved type")
		}
		if _, ok := r.db.Get(cipher.SHA256(ur)); !ok {
			t.Error("saved schema missing in db")
		}
		typ := reflect.TypeOf(Recur{})
		if name, ok := r.nnm[typ]; !ok {
			t.Error("saved schema missing in registery")
		} else if r.reg[name] != ur {
			t.Error("wrong reference for saved schema")
		}
	})
}

func TestRegistry_SchemaByName(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		_, err := r.SchemaByName("missing")
		if err == nil {
			t.Error("missing error")
		}
	})
	t.Run("saved", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		r.Register("User", User{})
		s, err := r.SchemaByName("User")
		if err != nil {
			t.Error(err)
			return
		}
		if s.Name() != "User" {
			t.Error("something wrong")
		}
		t.Log("Schema: ", s)
	})
}

func TestRegistry_SchemaByReference(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		_, err := r.SchemaByReference(Reference{1, 2, 3})
		if err == nil {
			t.Error("missing error")
		}
	})
	t.Run("saved", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		r.Register("User", User{})
		ur := r.SchemaReference(User{})
		s, err := r.SchemaByReference(ur)
		if err != nil {
			t.Error(err)
			return
		}
		if s.Kind() != reflect.Struct {
			t.Error("wrong kind")
		}
		if s.Name() != "User" {
			t.Error("wrong type name")
		}
		if len(s.fields) != 2 {
			t.Error("wrong fields coount")
			return
		}
		// Name string
		name := s.fields[0]
		if name.Kind() != reflect.String {
			t.Error("#1 wrong kind")
		}
		if name.Name() != "Name" {
			t.Error("#1 wrong name")
		}
		if name.TypeName() != "" {
			t.Error("#1 wrong type name")
		}
		// Age Age
		age := s.fields[1]
		if age.Kind() != reflect.Uint32 {
			t.Error("#2 wrong kind")
		}
		if age.Name() != "Age" {
			t.Error("#1 wrong name")
		}
		if age.TypeName() != "Age" {
			t.Error("#2 wrong type name")
		}
	})
	t.Run("complex", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		r.Register("User", User{})
		// totododododododo
		r.Register("Group", Group{})
		gr := r.SchemaReference(Group{})
		s, err := r.SchemaByReference(gr)
		if err != nil {
			t.Error(err)
			return
		}
		cmpSchemes(s, &Schema{
			name: []byte("Group"),
			kind: uint32(reflect.Struct),
			fields: []Field{
				Field{
					name: []byte("Name"),
				},
				Field{
					name: []byte("Leader"),
					tag:  []byte(`skyobject:"schema=User"`),
					ref:  []byte("User"),
				},
				Field{
					name: []byte("Members"),
					tag:  []byte(`skyobject:"schema=User"`),
					ref:  []byte("User"),
				},
				Field{
					name: []byte("Curator"),
				},
			},
		}, t)
		// schema of User
		us := r.getSchema(reflect.TypeOf(User{}))
		if err := us.load(); err != nil {
			t.Error("error loading User: ", err)
		}
		// Leader
		if leader, err := s.fields[1].Schema(); err != nil {
			t.Error(err)
		} else {
			if el, err := leader.Elem(); err != nil {
				t.Error(err)
			} else {
				if !cmpSchemes(el, us, t) {
					t.Error("leader is not equal")
				}
			}
		}
		// Members
		if members, err := s.fields[1].Schema(); err != nil {
			t.Error(err)
		} else {
			if el, err := members.Elem(); err != nil {
				t.Error(err)
			} else {
				if !cmpSchemes(el, us, t) {
					t.Error("leader is not equal")
				}
			}
		}
	})
}

func TestRegistry_getSchema(t *testing.T) {
	t.Run("flat", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		for _, i := range []interface{}{
			false,
			int8(0), int16(0), int32(0), int64(0),
			uint8(0), uint16(0), uint32(0), uint64(0),
			float32(0), float64(0),
			"empty",
		} {
			typ := reflect.TypeOf(i)
			sch := r.getSchema(typ)
			if sch == nil {
				t.Error("got nil-schema")
				continue
			}
			if sch.Kind() != typ.Kind() {
				t.Errorf("wrong kind: want %s, got %s",
					typ.Kind().String(),
					sch.Kind().String())
			}
			if sch.Name() != "" {
				t.Error("non-empty name for unnamed type: ", sch.Name())
			}
		}
	})
	t.Run("invalid", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		for _, i := range []interface{}{
			nil,
			make(chan struct{}),
			func() {},
			int(0),
			uint(0),
		} {
			typ := reflect.TypeOf(i)
			func() {
				defer shouldPanic(t)
				r.getSchema(typ)
			}()
		}
	})
	t.Run("flat named", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		for _, i := range []interface{}{
			Bool(false),
			Int8(0), Int16(0), Int32(0), Int64(0),
			Uint8(0), Uint16(0), Uint32(0), Uint64(0),
			Float32(0), Float64(0),
			String("empty"),
		} {
			typ := reflect.TypeOf(i)
			sch := r.getSchema(typ)
			if sch == nil {
				t.Error("got nil-schema")
				continue
			}
			if sch.Kind() != typ.Kind() {
				t.Errorf("wrong kind: want %s, got %s",
					typ.Kind().String(),
					sch.Kind().String())
			}
			if sch.Name() != typ.Name() {
				t.Error("wrong type name: ", sch.Name())
			}
		}
	})
	t.Run("complex unnamed", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		for _, i := range []interface{}{
			[]bool{},
			[1]int8{}, [2]int16{}, [3]int32{}, [4]int64{},
			[]uint8{}, [16]uint16{}, [32]uint32{}, []uint64{},
			[]float32{}, []float64{},
			struct{ Name string }{"empty"},
		} {
			typ := reflect.TypeOf(i)
			sch := r.getSchema(typ)
			if sch == nil {
				t.Error("got nil-schema")
				continue
			}
			if sch.Kind() != typ.Kind() {
				t.Errorf("wrong kind: want %s, got %s",
					typ.Kind().String(),
					sch.Kind().String())
			}
			if sch.Name() != "" {
				t.Error("non-empty name for unnamed type: ", sch.Name())
			}
			switch typ.Kind() {
			case reflect.Array:
				if sch.Len() != typ.Len() {
					t.Errorf("wrong length of array: want %d, got %d",
						typ.Len(), sch.Len())
				}
				fallthrough
			case reflect.Slice:
				el, err := sch.Elem()
				if err != nil {
					t.Error(err)
					continue
				}
				if el.Kind() != typ.Elem().Kind() {
					t.Errorf("invalid kind of element: want %s, got %s",
						typ.Elem().Kind().String(),
						el.Kind().String())
				}
			case reflect.Struct:
				if len(sch.Fields()) != typ.NumField() {
					t.Errorf("wrong fields count: want %d, got %d",
						typ.NumField(), len(sch.Fields()))
				}
			FieldsRange:
				for i := 0; i < typ.NumField(); i++ {
					fl := typ.Field(i)
					for _, sf := range sch.Fields() {
						if sf.Name() == fl.Name {
							if sf.Kind() != fl.Type.Kind() {
								t.Error("wrong kind of field: ", sf.Kind())
							}
							continue FieldsRange
						}
						t.Error("can't find field: ", fl.Name)
					}
				}
			}

		}
	})
	t.Run("references", func(t *testing.T) {
		r := newRegistery(data.NewDB())
		for _, i := range []interface{}{
			Reference{},
			References{},
			Dynamic{},
		} {
			typ := reflect.TypeOf(i)
			sch := r.getSchema(typ)
			// must not be saved
			if _, ok := r.reg[sch.Name()]; ok {
				t.Error("saved reference scehma: ", sch.Name())
			}
		}
	})
	// TODO: complex named recursive
	// TODO: skip fields
	// TODO: elem of references
}

func TestRegistry_getField(t *testing.T) {
	r := newRegistery(data.NewDB())
	r.Register("User", User{})
	r.Register("Group", Group{})
	typ := reflect.TypeOf(Group{})
	s := r.getSchema(typ)
	if err := s.load(); err != nil {
		t.Error(err)
	}
	if len(s.Fields()) != 4 {
		t.Error("wrong fields count: want 4, got: ", len(s.Fields()))
		return
	}
	for i, sf := range s.Fields() {
		fl := typ.Field(i)
		if sf.Name() != fl.Name {
			t.Error("wrong name of field #", i)
		}
		if sf.Tag() != fl.Tag {
			t.Errorf("wrong tag of field %q", sf.Name())
		}
		if sf.Kind() != fl.Type.Kind() {
			if i == 1 || i == 2 || i == 3 {
				if sf.Kind() != reflect.Ptr {
					t.Error("wrong kind of reference")
				} else if !sf.isReference() {
					t.Error("isReference wrong")
				}
			}
			switch i {
			case 1:
				if sf.TypeName() != SINGLE {
					t.Error("wrong field type: ", sf.TypeName())
				}
				if fs, err := sf.Schema(); err != nil {
					t.Error(err)
				} else {
					if el, err := fs.Elem(); err != nil {
						t.Error(err)
					} else if el.Name() != "User" {
						t.Error("wrong schema of element of reference")
					}
				}
			case 2:
				if sf.TypeName() != ARRAY {
					t.Error("wrong field type: ", sf.TypeName())
				}
				if fs, err := sf.Schema(); err != nil {
					t.Error(err)
				} else {
					if el, err := fs.Elem(); err != nil {
						t.Error(err)
					} else if el.Name() != "User" {
						t.Error("wrong schema of element of reference")
					}
				}
			case 3:
				if sf.TypeName() != DYNAMIC {
					t.Error("wrong field type: ", sf.TypeName())
				}
				if fs, err := sf.Schema(); err != nil {
					t.Error(err)
				} else {
					if _, err := fs.Elem(); err == nil {
						t.Error("missing error")
					}
				}
			default:
				t.Errorf("wrong kind of field %q: want %s, got %s",
					sf.Name(),
					fl.Type.Kind().String(),
					sf.Kind().String())
			}
		}
	}
}

func TestSchema_Name(t *testing.T) {
	// TODO
}

func TestSchema_Kind(t *testing.T) {
	// TODO
}

func TestSchema_Elem(t *testing.T) {
	// TODO
}

func TestSchema_Len(t *testing.T) {
	// TODO
}

func TestSchema_Fields(t *testing.T) {
	// TODO
}

func TestSchema_setElem(t *testing.T) {
	// TODO
}

func TestSchema_isNamed(t *testing.T) {
	// TODO
}

func TestSchema_isSaved(t *testing.T) {
	// TODO
}

func TestSchema_load(t *testing.T) {
	r := newRegistery(data.NewDB())
	r.Register("User", User{})
	s := r.getSchema(reflect.TypeOf(User{}))
	// s already registered
	if len(s.fields) != 0 {
		t.Error("wrong fields count: ", len(s.fields))
	}
	s.load()
	if len(s.fields) != 2 {
		t.Error("wrong fields count: ", len(s.fields))
	}
}

func TestSchema_String(t *testing.T) {
	// TODO
}

func TestField_Kind(t *testing.T) {
	// TODO
}

func TestField_TypeName(t *testing.T) {
	// TODO
}

func TestField_Name(t *testing.T) {
	// TODO
}

func TestField_Schema(t *testing.T) {
	// TODO --------------------------------------------------------------------
}

func TestField_Tag(t *testing.T) {
	// TODO
}

func TestField_tagSchemaName(t *testing.T) {
	// TODO
}

func TestField_isReference(t *testing.T) {
	// TODO
}

func TestField_String(t *testing.T) {
	// TODO
}

func Test_encodingSchema_Schema(t *testing.T) {
	// to do or not to do
}

func Test_newEncodingSchema(t *testing.T) {
	// to do or not to do
}

func Test_newEncodingField(t *testing.T) {
	// to do or not to do
}

func Test_encodingField_Field(t *testing.T) {
	// to do or not to do
}

func TestSchema_Encode(t *testing.T) {
	r := newRegistery(data.NewDB())
	r.Register("User", User{})
	s, err := r.SchemaByName("User")
	if err != nil {
		t.Error(err)
		return
	}
	sd := s.Encode()
	t.Log("(*Schema).Encode() length: ", len(sd))
	if len(sd) == 0 {
		t.Error("encode to zero length value")
	}
}

func TestSchema_Decode(t *testing.T) {
	// encode
	r := newRegistery(data.NewDB())
	r.Register("User", User{})
	s, err := r.SchemaByName("User")
	if err != nil {
		t.Error(err)
		return
	}
	sd := s.Encode()
	if len(sd) == 0 {
		t.Error("encode to zero length value")
	}
	// decode
	var ns *Schema = r.newSchema()
	if err := ns.Decode(sd); err != nil {
		t.Error("decoding error: ", err)
	}
	if !cmpSchemes(s, ns, t) {
		//
	}
}

func Test_mustGetSchemaOfTag(t *testing.T) {
	//
}
