package skyobject

import (
	"reflect"
	"testing"

	"github.com/skycoin/skycoin/src/cipher/encoder"

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
				kind: uint32(k),
			}, nil}).Kind()
			if kind != k {
				t.Error("wrong kind: want %s, got %s", k, kind)
			}
		}

	})
	t.Run("references", func(t *testing.T) {
		s := &Schema{
			kind: uint32(reflect.Ptr), // <- pointer
			name: []byte(arrayRef),    // <- reference
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
	vs, err := root.Values()
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
	if len(group.Fields()) != 4 {
		t.Error("wrong number of fields: ", len(group.Fields()))
	}
	for _, fn := range group.Fields() {
		fl, err := group.FieldByName(fn)
		if err != nil {
			t.Errorf("get field %q error: %v", fn, err)
			continue
		}
		if fl.Kind() == reflect.Ptr { // if reference
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
	type Bool bool
	root := getRoot()
	root.Inject(Bool(true))
	root.Inject(Bool(false))
	vs, err := root.Values()
	if err != nil {
		t.Error(err)
		return
	}
	if len(vs) != 2 {
		t.Error("unexpected values length: ", len(vs))
		return
	}
	for i, want := range []bool{true, false} {
		val := vs[i]
		if val.Kind() != reflect.Bool {
			t.Error("unexpected kind of bool: ", val.Kind())
			return
		}
		if ln := len(val.(*value).od); ln != 1 {
			t.Error("wrong length of boolend data: ", ln)
		}
		if b, err := val.Bool(); err != nil {
			t.Error(err)
		} else if b != want {
			t.Errorf("wrong value: want %t, got %t", want, b)
		}
	}
}

func TestValue_Int(t *testing.T) {
	t.Run("another", func(t *testing.T) {
		root := getRoot()
		root.Inject(String("hello"))
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		if _, err := vs[0].Int(); err == nil {
			t.Error("missing error")
		}
	})
	t.Run("all", func(t *testing.T) {
		root := getRoot()
		root.Inject(Int8(0))
		root.Inject(Int16(1))
		root.Inject(Int32(2))
		root.Inject(Int64(3))
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 4 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		kinds := []reflect.Kind{
			reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Int64,
		}
		sizes := []int{1, 2, 4, 8}
		for i, want := range []int64{0, 1, 2, 3} {
			val := vs[i]
			if val.Kind() != kinds[i] {
				t.Error("unexpected kind of int: ", val.Kind())
			}
			if ln := len(val.(*value).od); ln != sizes[i] {
				t.Error("wrong length of boolend data: ", ln)
			}
			if x, err := val.Int(); err != nil {
				t.Error(err)
			} else if x != want {
				t.Errorf("wrong value: want %t, got %t", want, x)
			}
		}
	})
}

func TestValue_Uint(t *testing.T) {
	t.Run("another", func(t *testing.T) {
		root := getRoot()
		root.Inject(String("hello"))
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		if _, err := vs[0].Uint(); err == nil {
			t.Error("missing error")
		}
	})
	t.Run("all", func(t *testing.T) {
		root := getRoot()
		root.Inject(Uint8(0))
		root.Inject(Uint16(1))
		root.Inject(Uint32(2))
		root.Inject(Uint64(3))
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 4 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		kinds := []reflect.Kind{
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64,
		}
		sizes := []int{1, 2, 4, 8}
		for i, want := range []uint64{0, 1, 2, 3} {
			val := vs[i]
			if val.Kind() != kinds[i] {
				t.Error("unexpected kind of int: ", val.Kind())
			}
			if ln := len(val.(*value).od); ln != sizes[i] {
				t.Error("wrong length of boolend data: ", ln)
			}
			if x, err := val.Uint(); err != nil {
				t.Error(err)
			} else if x != want {
				t.Errorf("wrong value: want %t, got %t", want, x)
			}
		}
	})
}

func TestValue_String(t *testing.T) {
	t.Run("another", func(t *testing.T) {
		root := getRoot()
		root.Inject(Int16(0))
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		if _, err := vs[0].String(); err == nil {
			t.Error("missing error")
		}
	})
	t.Run("all", func(t *testing.T) {
		root := getRoot()
		hello := "hello"
		root.Inject(String(hello))
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		val := vs[0]
		if val.Kind() != reflect.String {
			t.Error("unexpected kind of int: ", val.Kind())
		}
		if ln := len(val.(*value).od); ln != len(hello)+4 {
			t.Error("wrong length of boolend data: ", ln)
		}
		if x, err := val.String(); err != nil {
			t.Error(err)
		} else if x != hello {
			t.Errorf("wrong value: want %t, got %t", hello, x)
		}
	})
}

func TestValue_Bytes(t *testing.T) {
	t.Run("another", func(t *testing.T) {
		root := getRoot()
		root.Inject(Int16(0))
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		if _, err := vs[0].Bytes(); err == nil {
			t.Error("missing error")
		}
	})
	t.Run("all", func(t *testing.T) {
		root := getRoot()
		hello, cya := "hello", "cya"
		root.Inject(String(hello))
		root.Inject(Bytes(cya))
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 2 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		kinds := []reflect.Kind{
			reflect.String,
			reflect.Slice,
		}
		sizes := []int{len(hello) + 4, len(cya) + 4}
		for i, want := range []string{hello, cya} {
			val := vs[i]
			if val.Kind() != kinds[i] {
				t.Error("unexpected kind: ", val.Kind())
			}
			if ln := len(val.(*value).od); ln != sizes[i] {
				t.Error("wrong length of data: ", ln)
			}
			if x, err := val.Bytes(); err != nil {
				t.Error(err)
			} else if string(x) != want {
				t.Errorf("wrong value: want %t, got %t", want, string(x))
			}
		}
	})
}

func TestValue_Float(t *testing.T) {
	t.Run("another", func(t *testing.T) {
		root := getRoot()
		root.Inject(Int16(0))
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		if _, err := vs[0].Bytes(); err == nil {
			t.Error("missing error")
		}
	})
	t.Run("all", func(t *testing.T) {
		root := getRoot()
		root.Inject(Float32(5.5))
		root.Inject(Float64(7.7))
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 2 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		kinds := []reflect.Kind{
			reflect.Float32,
			reflect.Float64,
		}
		sizes := []int{4, 8}
		for i, want := range []float64{5.5, 7.7} {
			val := vs[i]
			if val.Kind() != kinds[i] {
				t.Error("unexpected kind: ", val.Kind())
			}
			if ln := len(val.(*value).od); ln != sizes[i] {
				t.Error("wrong length of data: ", ln)
			}
			if x, err := val.Float(); err != nil {
				t.Error(err)
			} else if x != want {
				t.Errorf("wrong value: want %t, got %t", want, x)
			}
		}
	})
}

func TestValue_Fields(t *testing.T) {
	t.Run("another", func(t *testing.T) {
		root := getRoot()
		root.Inject(Int16(0))
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		if len(vs[0].Fields()) != 0 {
			t.Error("got fields on non-struct type")
		}
	})
	t.Run("all", func(t *testing.T) {
		root := getRoot()
		root.RegisterSchema("User", User{})
		strucures := []interface{}{
			Group{},
			List{},
			User{},
			Man{},
		}
		for _, s := range strucures {
			root.Inject(s)
		}
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != len(strucures) {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		sizes := []int{
			// Name    string
			// Leader  Reference  `skyobject:"schema=User"`
			// Members References `skyobject:"schema=User"`
			// Curator Dynamic
			4 + len(Reference{}) + 4 + 2*len(Reference{}),
			// Name   string ``
			// Age    Age    `json:"age"`
			// Hidden int    `enc:"-"`
			4 + 4 + 4,
			// Name     string
			// Members  References `skyobject:"schema=User"`
			// MemberOf []Group
			4 + 4,
			// Name    string
			// Age     Age
			// Seecret []byte
			// Owner   Group
			// Friends List
			4 + 4 + 4 +
				(4 + len(Reference{}) + 4 + 2*len(Reference{})) + // Group
				(4 + 4 + 4), // List
		}
		for i, want := range strucures {
			val := vs[i]
			if val.Kind() != reflect.Struct {
				t.Error("unexpected kind: ", val.Kind())
			}
			if ln := len(val.(*value).od); ln != len(encoder.Serialize(want)) {
				t.Error("wrong length of data: ", ln)
			} else {
				if ln != sizes[i] {
					t.Error("unexpected size: ", ln, sizes[i])
				}
			}
			fields := val.Fields()
			typ := reflect.TypeOf(want)
			if g, w := len(fields), typ.NumField(); g != w {
				if typ.Name() == "User" && g != 2 {
					t.Errorf("wrong number of fields: %d - %d", w, g)
					continue
				}
			}
			for i, n := range fields {
				if n != typ.Field(i).Name {
					t.Error("wrong field name")
				}
			}
		}
	})
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
