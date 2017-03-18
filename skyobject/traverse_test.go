package skyobject

import (
	"reflect"
	"testing"

	"github.com/skycoin/cxo/data"
)

func getRoot() *Root {
	c := NewContainer(data.NewDB())
	return c.NewRoot(pubKey())
}

func TestMissingSchema_Error(t *testing.T) {
	// TODO
}

func TestMissingSchema_Key(t *testing.T) {
	// TODO
}

func TestMissingObject_Error(t *testing.T) {
	// TODO
}

func TestMissingObject_Key(t *testing.T) {
	// TODO
}

func TestValue_Kind(t *testing.T) {
	t.Run("any", func(t *testing.T) {
		for _, k := range []reflect.Kind{
			reflect.Bool,
			reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64,
			reflect.String,
			reflect.Slice,
			reflect.Array,
			reflect.Struct,
			reflect.Ptr,
		} {
			kind := (&value{nil, &Schema{
				schemaHead: schemaHead{kind: uint32(k)},
			}, nil}).Kind()
			if kind != k {
				t.Error("wrong kind: want %s, got %s", k, kind)
			}
		}

	})
	t.Run("references", func(t *testing.T) {
		s := &Schema{
			schemaHead: schemaHead{
				kind:     uint32(reflect.Ptr), // <- pointer
				typeName: []byte(arrayRef),    // <- reference
			},
		}
		kind := (&value{nil, s, nil}).Kind()
		if kind != reflect.Slice { // <- slice
			t.Error("wrong kind: want %s, got %s", reflect.Slice, kind)
		}
	})
}

func TestValue_Dereference(t *testing.T) {
	root := getRoot()
	root.RegisterSchema("User", User{})
	root.Inject(Group{
		Name: "a group",
		Leader: root.Save(User{
			"Billy Kid", 16, 90,
		}),
		Members: root.SaveArray(
			User{"Bob Marley", 21, 0},
			User{"Alice Cooper", 19, 0},
			User{"Eva Brown", 30, 0},
		),
		Curator: root.Dynamic(Man{
			Name:    "Ned Kelly",
			Age:     28,
			Seecret: []byte("secret key"),
			Owner:   Group{},
			Friends: List{},
		}),
	})
	vs, err := root.Values() // vs containe Group
	if err != nil {
		t.Error(err)
		return
	}
	if len(vs) != 1 {
		t.Error("unexpected values length: ", len(vs))
		return
	}
	group := vs[0]
	if group.Kind() != reflect.Struct {
		t.Error("unexpected kind of group: ", group.Kind())
		return
	}
	for _, fn := range group.Fields() {
		t.Log("FIELD: ", fn)
		fl, err := group.FieldByName(fn)
		if err != nil {
			t.Errorf("get field %q error: %v", fn, err)
			continue
		}
		t.Log("VALUE ", fl.Schema().Name())
		if fl.Kind() == reflect.Ptr { // if reference
			t.Log("REFERENCE ", fl.Schema().Name())
			var d Value
			if d, err = fl.Dereference(); err != nil {
				t.Error(err)
				continue
			}
			_ = d
		}
	}
}

func TestValue_Bool(t *testing.T) {
	//
}

func TestValue_Int(t *testing.T) {
	//
}

func TestValue_Uint(t *testing.T) {
	//
}

func TestValue_String(t *testing.T) {
	//
}

func TestValue_Bytes(t *testing.T) {
	//
}

func TestValue_Float(t *testing.T) {
	//
}

func TestValue_Fields(t *testing.T) {
	//
}

func TestValue_FieldByName(t *testing.T) {
	//
}

func TestValue_Len(t *testing.T) {
	//
}

func TestValue_Index(t *testing.T) {
	//
}

func TestValue_Schema(t *testing.T) {
	//
}

func TestRoot_Values(t *testing.T) {
	//
}

func TestSchema_Size(t *testing.T) {
	//
}

func Test_getLength(t *testing.T) {
	//
}

func Test_fixedSize(t *testing.T) {
	//
}
