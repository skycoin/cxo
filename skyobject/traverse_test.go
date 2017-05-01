package skyobject

import (
	"errors"
	"reflect"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

func getCntRoot() (*Container, *Root, cipher.SecKey) {
	c := NewContainer(data.NewDB())
	pk, sk := cipher.GenerateKeyPair()
	return c, c.NewRoot(pk, sk), sk
}

func TestValue_Kind(t *testing.T) {
	t.Run("any", func(t *testing.T) {
		slice := []byte{} // should be non-nil
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
			kind := (&Value{nil, &Schema{
				kind: uint32(k),
			}, slice}).Kind()
			if kind != k {
				t.Error("wrong kind: want %s, got %s", k, kind)
			}
		}

	})
	// TODO: nil value s and nil references
	t.Run("references", func(t *testing.T) {
		s := &Schema{
			kind: uint32(reflect.Ptr), // <- pointer
			name: []byte(ARRAY),       // <- reference
		}
		kind := (&Value{nil, s, nil}).Kind()
		if kind != reflect.Slice { // <- slice
			t.Error("wrong kind: want %s, got %s", reflect.Slice, kind)
		}
	})
	// t.Run("nils", func(t *testing.T) {
	// 	s := &Schema{
	// 		kind: uint32(reflect.Ptr), // <- pointer
	// 		name: []byte(arrayRef),    // <- reference
	// 	}
	// 	kind := (&Value{nil, s, nil}).Kind()
	// 	if kind != reflect.Slice { // <- slice
	// 		t.Error("wrong kind: want %s, got %s", reflect.Slice, kind)
	// 	}
	// })
}

func TestValue_Dereference(t *testing.T) {
	cnt, root, sk := getCntRoot()
	cnt.Register("User", User{})   // 1
	cnt.Register("Group", Group{}) // +1 -> 2
	cnt.Register("List", List{})   // +1 -> 3
	cnt.Register("Man", Man{})     // +1 -> 4
	root.Inject(Group{
		Name: "a group",
		Leader: cnt.Save(User{
			"Billy Kid", 16, 90,
		}),
		Members: cnt.SaveArray(
			User{"Bob Marley", 21, 0},
			User{"Alice Cooper", 19, 0},
			User{"Eva Brown", 30, 0},
		),
		Curator: cnt.Dynamic(Man{
			Name:    "Ned Kelly",
			Age:     28,
			Seecret: []byte("secret key"),
			Owner:   Group{},
			Friends: List{},
		}),
	}, sk)
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
	// Leader
	if fl, err := group.FieldByName("Leader"); err != nil {
		t.Errorf("get field 'Leader' error: %v", err)
	} else if fl.Kind() != reflect.Ptr {
		t.Error("expected reference, got ", fl.Kind())
	} else if d, err := fl.Dereference(); err != nil {
		t.Error(err)
	} else if d.Kind() != reflect.Struct {
		t.Error("wrong kind of dereferenced value ", d.Kind())
	} else if v, err := d.FieldByName("Name"); err != nil {
		t.Error(err)
	} else if un, err := v.String(); err != nil {
		t.Error(err)
	} else if un != "Billy Kid" {
		t.Error("wrong field value")
	}
	// Members
	if fl, err := group.FieldByName("Members"); err != nil {
		t.Errorf("get field 'Members' error: %v", err)
	} else {
		if fl.Kind() != reflect.Slice {
			t.Error("expected slice, got ", fl.Kind())
		} else {
			if l, err := fl.Len(); err != nil {
				t.Error(err)
			} else if l != 3 {
				t.Error("wrong count of references:", l)
			} else {
				names := []string{
					"Bob Marley",
					"Alice Cooper",
					"Eva Brown",
				}
				for i := 0; i < l; i++ {
					if idx, err := fl.Index(i); err != nil {
						t.Error(err)
					} else if idx.Kind() != reflect.Ptr {
						t.Error("expected reference, got", idx.Kind())
					} else if d, err := idx.Dereference(); err != nil {
						t.Error(err)
					} else if d.Kind() != reflect.Struct {
						t.Error("wrong kind of dereferenced value", d.Kind())
					} else if v, err := d.FieldByName("Name"); err != nil {
						t.Error(err)
					} else if un, err := v.String(); err != nil {
						t.Error(err)
					} else if un != names[i] {
						t.Error("wrong field value")
					}
				}
			}
		}
	}
	// Curator
	if fl, err := group.FieldByName("Curator"); err != nil {
		t.Errorf("get field 'Curator' error: %v", err)
	} else if fl.Kind() != reflect.Ptr {
		t.Error("expected reference, got ", fl.Kind())
	} else if d, err := fl.Dereference(); err != nil {
		t.Error(err)
	} else if d.Kind() != reflect.Struct {
		t.Error("wrong kind of dereferenced value ", d.Kind())
	} else if v, err := d.FieldByName("Name"); err != nil {
		t.Error(err)
	} else if un, err := v.String(); err != nil {
		t.Error(err)
	} else if un != "Ned Kelly" {
		t.Error("wrong field value")
	}
}

func TestValue_Bool(t *testing.T) {
	type Bool bool
	cnt, root, sk := getCntRoot()
	type X struct {
		Bools [2]Bool
	}
	cnt.Register("X", X{})
	root.Inject(X{Bools: [2]Bool{true, false}}, sk)
	vs, err := root.Values()
	if err != nil {
		t.Error(err)
		return
	}
	if len(vs) != 1 {
		t.Error("unexpected values length: ", len(vs))
		return
	}
	bools, err := vs[0].FieldByName("Bools")
	if err != nil {
		t.Error(err)
		return
	}
	ln, err := bools.Len()
	if err != nil {
		t.Error(err)
		return
	}
	if ln != 2 {
		t.Error("unexpected values length: ", len(vs))
		return
	}
	for i, want := range []bool{true, false} {
		val, err := bools.Index(i)
		if err != nil {
			t.Error(err)
			return
		}
		if val.Kind() != reflect.Bool {
			t.Error("unexpected kind of bool: ", val.Kind())
			return
		}
		if ln := len(val.od); ln != 1 {
			t.Error("wrong length of boolean data: ", ln)
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
		cnt, root, sk := getCntRoot()
		type X struct {
			String String
		}
		cnt.Register("X", X{})
		root.Inject(X{String("hello")}, sk)
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		fl, err := vs[0].FieldByName("String")
		if err != nil {
			t.Error(err)
			return
		}
		if _, err = fl.Int(); err == nil { // Int instead of String
			t.Error("missing error")
		}
	})
	t.Run("all", func(t *testing.T) {
		cnt, root, sk := getCntRoot()
		type X struct {
			Int8  Int8
			Int16 Int16
			Int32 Int32
			Int64 Int64
		}
		cnt.Register("X", X{})
		root.Inject(X{0, 1, 2, 3}, sk)
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		x := vs[0]
		kinds := []reflect.Kind{
			reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Int64,
		}
		sizes := []int{1, 2, 4, 8}
		wants := []int64{0, 1, 2, 3}
		var i int // index
		err = x.RangeFields(func(_ string, val *Value) error {
			want := wants[i]
			if val.Kind() != kinds[i] {
				t.Error("unexpected kind of int: ", val.Kind())
			}
			if ln := len(val.od); ln != sizes[i] {
				t.Error("wrong length of boolend data: ", ln)
			}
			if x, err := val.Int(); err != nil {
				t.Error(err)
			} else if x != want {
				t.Errorf("wrong value: want %t, got %t", want, x)
			}
			i++
			return nil
		})
		if err != nil {
			t.Error(err)
		}
	})
}

func TestValue_Uint(t *testing.T) {
	t.Run("another", func(t *testing.T) {
		type X struct {
			String String
		}
		cnt, root, sk := getCntRoot()
		cnt.Register("X", X{})
		root.Inject(X{String("hello")}, sk)
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		fld, err := vs[0].FieldByName("String")
		if err != nil {
			t.Error(err)
			return
		}
		if _, err := fld.Uint(); err == nil {
			t.Error("missing error")
		}
	})
	t.Run("all", func(t *testing.T) {
		type X struct {
			Uint8  Uint8
			Uint16 Uint16
			Uint32 Uint32
			Uint64 Uint64
		}
		cnt, root, sk := getCntRoot()
		cnt.Register("X", X{})
		root.Inject(X{0, 1, 2, 3}, sk)
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
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
		wants := []uint64{0, 1, 2, 3}
		var i int // index
		err = vs[0].RangeFields(func(_ string, val *Value) error {
			want := wants[i]
			if val.Kind() != kinds[i] {
				t.Error("unexpected kind of int: ", val.Kind())
			}
			if ln := len(val.od); ln != sizes[i] {
				t.Error("wrong length of boolend data: ", ln)
			}
			if x, err := val.Uint(); err != nil {
				t.Error(err)
			} else if x != want {
				t.Errorf("wrong value: want %t, got %t", want, x)
			}
			i++
			return nil
		})
		if err != nil {
			t.Error(err)
		}
	})
}

func TestValue_String(t *testing.T) {
	t.Run("another", func(t *testing.T) {
		type X struct {
			Int16 Int16
		}
		cnt, root, sk := getCntRoot()
		cnt.Register("X", X{})
		root.Inject(X{0}, sk)
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		fld, err := vs[0].FieldByName("Int16")
		if err != nil {
			t.Error(err)
			return
		}
		if _, err := fld.String(); err == nil {
			t.Error("missing error")
		}
	})
	t.Run("all", func(t *testing.T) {
		type X struct {
			String String
		}
		cnt, root, sk := getCntRoot()
		cnt.Register("X", X{})
		hello := "hello"
		root.Inject(X{String(hello)}, sk)
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		val, err := vs[0].FieldByName("String")
		if err != nil {
			t.Error(err)
			return
		}
		if val.Kind() != reflect.String {
			t.Error("unexpected kind of int: ", val.Kind())
		}
		if ln := len(val.od); ln != len(hello)+4 {
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
		type X struct {
			Int16 Int16
		}
		cnt, root, sk := getCntRoot()
		cnt.Register("X", X{})
		root.Inject(X{0}, sk)
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		fld, err := vs[0].FieldByName("Int16")
		if err != nil {
			t.Error(err)
			return
		}
		if _, err := fld.Bytes(); err == nil {
			t.Error("missing error")
		}
	})
	t.Run("all", func(t *testing.T) {
		type X struct {
			String string
			Bytes  []byte
		}
		cnt, root, sk := getCntRoot()
		cnt.Register("X", X{})
		hello, cya := "hello", "cya"
		root.Inject(X{hello, []byte(cya)}, sk)
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		kinds := []reflect.Kind{
			reflect.String,
			reflect.Slice,
		}
		sizes := []int{len(hello) + 4, len(cya) + 4}
		wants := []string{hello, cya}
		var i int
		err = vs[0].RangeFields(func(_ string, val *Value) error {
			want := wants[i]
			if val.Kind() != kinds[i] {
				t.Error("unexpected kind: ", val.Kind())
			}
			if ln := len(val.od); ln != sizes[i] {
				t.Error("wrong length of data: ", ln)
			}
			if x, err := val.Bytes(); err != nil {
				t.Error(err)
			} else if string(x) != want {
				t.Errorf("wrong value: want %t, got %t", want, string(x))
			}
			i++
			return nil
		})
		if err != nil {
			t.Error(err)
		}
	})
}

func TestValue_Float(t *testing.T) {
	t.Run("another", func(t *testing.T) {
		type X struct {
			Int16 Int16
		}
		cnt, root, sk := getCntRoot()
		cnt.Register("X", X{})
		root.Inject(X{0}, sk)
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		fld, err := vs[0].FieldByName("Int16")
		if err != nil {
			t.Error(err)
			return
		}
		if _, err := fld.Bytes(); err == nil {
			t.Error("missing error")
		}
	})
	t.Run("all", func(t *testing.T) {
		type X struct {
			Float32 float32
			Float64 float64
		}
		cnt, root, sk := getCntRoot()
		cnt.Register("X", X{})
		root.Inject(X{5.5, 7.7}, sk)
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		kinds := []reflect.Kind{
			reflect.Float32,
			reflect.Float64,
		}
		sizes := []int{4, 8}
		wants := []float64{5.5, 7.7}
		var i int
		err = vs[0].RangeFields(func(_ string, val *Value) error {
			want := wants[i]
			if val.Kind() != kinds[i] {
				t.Error("unexpected kind: ", val.Kind())
			}
			if ln := len(val.od); ln != sizes[i] {
				t.Error("wrong length of data: ", ln)
			}
			if x, err := val.Float(); err != nil {
				t.Error(err)
			} else if x != want {
				t.Errorf("wrong value: want %t, got %t", want, x)
			}
			i++
			return nil
		})
		if err != nil {
			t.Error(err)
		}
	})
}

func TestValue_Fields(t *testing.T) {
	t.Run("another", func(t *testing.T) {
		type X struct {
			Int16 int16
		}
		cnt, root, sk := getCntRoot()
		cnt.Register("X", X{})
		root.Inject(X{0}, sk)
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		if fld, err := vs[0].FieldByName("Int16"); err != nil {
			t.Error(err)
		} else if len(fld.Fields()) != 0 {
			t.Error("got fields on non-struct type")
		}
	})
	t.Run("all", func(t *testing.T) {
		cnt, root, sk := getCntRoot()
		cnt.Register("User", User{})
		cnt.Register("Group", Group{})
		cnt.Register("List", List{})
		cnt.Register("Man", Man{})
		strucures := []interface{}{
			Group{},
			List{},
			User{},
			Man{},
		}
		for _, s := range strucures {
			root.Inject(s, sk)
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
			// # Group
			// Name    string
			// Leader  Reference  `skyobject:"schema=User"`
			// Members References `skyobject:"schema=User"`
			// Curator Dynamic
			4 + len(Reference{}) + 4 + 2*len(Reference{}),
			// # List
			// Name     string
			// Members  References `skyobject:"schema=User"`
			// MemberOf []Group
			4 + 4 + 4,
			// # User
			// Name   string ``
			// Age    Age    `json:"age"`
			// Hidden int    `enc:"-"`
			4 + 4,
			// # Man
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
			if ln := len(val.od); ln != len(encoder.Serialize(want)) {
				t.Error("wrong length of data: ", ln)
			} else {
				if ln != sizes[i] {
					t.Error("unexpected size: ", ln, sizes[i])
				}
			}
			fields := val.Fields()
			typ := reflect.TypeOf(want)
			if g, w := len(fields), typ.NumField(); g != w {
				if typ.Name() == "User" { // Hidden
					if g != 2 {
						t.Errorf("wrong number of fields: %d - %d", w, g)
					}
				} else {
					t.Errorf("wrong number of fields: %d - %d", w, g)
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
	t.Run("simple", func(t *testing.T) {
		cnt, root, sk := getCntRoot()
		cnt.Register("User", User{})
		name := "Alice"
		age := Age(21)
		root.Inject(User{name, age, 0}, sk)
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		// Name   string ``
		// Age    Age    `json:"age"`
		// Hidden int    `enc:"-"`
		// 4 + 4
		val := vs[0]
		if val.Kind() != reflect.Struct {
			t.Error("unexpected kind: ", val.Kind())
		}
		if ln := len(val.od); ln != len(name)+4+4 {
			t.Error("wrong length of data: ", ln)
		}
		// Name
		if fname, err := val.FieldByName("Name"); err != nil {
			t.Error(err)
		} else if ln := len(fname.od); ln != 4+len(name) {
			t.Error("wrong length of encoded field: ", ln)
		} else if s, err := fname.String(); err != nil {
			t.Error(err)
		} else if s != name {
			t.Error("wrong name: ", s)
		}
		// Age
		if fage, err := val.FieldByName("Age"); err != nil {
			t.Error(err)
		} else if ln := len(fage.od); ln != 4 {
			t.Error("wrong length of encoded field: ", ln)
		} else if i, err := fage.Uint(); err != nil {
		} else if i != uint64(age) {
			t.Error("wrong age: ", i)
		}
	})
	t.Run("references", func(t *testing.T) {
		cnt, root, sk := getCntRoot()
		cnt.Register("User", User{})
		cnt.Register("Group", Group{})
		cnt.Register("List", List{})
		cnt.Register("Man", Man{})
		root.Inject(Group{
			Name:   "The Group",
			Leader: cnt.Save(User{"Alice", 21, 0}),
			Members: cnt.SaveArray(
				User{"Bob", 32, 0},
				User{"Eva", 33, 0},
				User{"Tom", 34, 0},
				User{"Amy", 35, 0},
			),
			Curator: cnt.Dynamic(Man{
				Name: "Tony Hawk",
			}),
		}, sk)
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
		// Name
		if fname, err := val.FieldByName("Name"); err != nil {
			t.Error(err)
		} else if ln := len(fname.od); ln != 4+len("The Group") {
			t.Error("wrong length of encoded field: ", ln)
		} else if s, err := fname.String(); err != nil {
			t.Error(err)
		} else if s != "The Group" {
			t.Error("wrong name: ", s)
		}
		// Leader
		if fleader, err := val.FieldByName("Leader"); err != nil {
			t.Error(err)
		} else if ln := len(fleader.od); ln != len(Reference{}) {
			t.Error("wrong length of encoded field: ", ln)
		} else if fleader.Kind() != reflect.Ptr {
			t.Error("invalid kind of reference: ", fleader.Kind())
		}
		// TODO: check reference
		// Members
		if fmembers, err := val.FieldByName("Members"); err != nil {
			t.Error(err)
		} else if ln := len(fmembers.od); ln != 4+4*len(Reference{}) {
			t.Error("wrong length of encoded field: ", ln)
		} else if fmembers.Kind() != reflect.Slice {
			t.Error("invalid kind of references: ", fmembers.Kind())
		}
		// TODO: check reference
		// Curator
		if fcurator, err := val.FieldByName("Curator"); err != nil {
			t.Error(err)
		} else if ln := len(fcurator.od); ln != 2*len(Reference{}) {
			t.Error("wrong length of encoded field: ", ln)
		} else if fcurator.Kind() != reflect.Ptr {
			t.Error("invalid kind of reference: ", fcurator.Kind())
		}
		// TODO: check reference
	})
}

func TestValue_RangeFields(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		cnt, root, sk := getCntRoot()
		name := "Alice"
		age := Age(21)
		cnt.Register("User", User{})
		root.Inject(User{name, age, 0}, sk)
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if len(vs) != 1 {
			t.Error("unexpected values length: ", len(vs))
			return
		}
		// Name   string ``
		// Age    Age    `json:"age"`
		// Hidden int    `enc:"-"`
		// 4 + 4
		val := vs[0]
		if val.Kind() != reflect.Struct {
			t.Error("unexpected kind: ", val.Kind())
		}
		if ln := len(val.od); ln != len(name)+4+4 {
			t.Error("wrong length of data: ", ln)
		}
		err = val.RangeFields(func(fname string, val *Value) error {
			switch fname {
			case "Name":
				if ln := len(val.od); ln != 4+len(name) {
					t.Error("wrong length of encoded field: ", ln)
				} else if s, err := val.String(); err != nil {
					t.Error(err)
				} else if s != name {
					t.Error("wrong name: ", s)
				}
			case "Age":
				if ln := len(val.od); ln != 4 {
					t.Error("wrong length of encoded field: ", ln)
				} else if i, err := val.Uint(); err != nil {
				} else if i != uint64(age) {
					t.Error("wrong age: ", i)
				}
			}
			return nil
		})
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("references", func(t *testing.T) {
		cnt, root, sk := getCntRoot()
		cnt.Register("User", User{})
		cnt.Register("Group", Group{})
		cnt.Register("List", List{})
		cnt.Register("Man", Man{})
		root.Inject(Group{
			Name:   "The Group",
			Leader: cnt.Save(User{"Alice", 21, 0}),
			Members: cnt.SaveArray(
				User{"Bob", 32, 0},
				User{"Eva", 33, 0},
				User{"Tom", 34, 0},
				User{"Amy", 35, 0},
			),
			Curator: cnt.Dynamic(Man{
				Name: "Tony Hawk",
			}),
		}, sk)
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
		err = val.RangeFields(func(fname string, val *Value) error {
			switch fname {
			case "Name":
				if ln := len(val.od); ln != 4+len("The Group") {
					t.Error("wrong length of encoded field: ", ln)
				} else if s, err := val.String(); err != nil {
					t.Error(err)
				} else if s != "The Group" {
					t.Error("wrong name: ", s)
				}
			case "Leader":
				if ln := len(val.od); ln != len(Reference{}) {
					t.Error("wrong length of encoded field: ", ln)
				} else if val.Kind() != reflect.Ptr {
					t.Error("invalid kind of reference: ", val.Kind())
				}
			case "Members":
				if ln := len(val.od); ln != 4+4*len(Reference{}) {
					t.Error("wrong length of encoded field: ", ln)
				} else if val.Kind() != reflect.Slice {
					t.Error("invalid kind of references: ", val.Kind())
				}
			case "Curator":
				if ln := len(val.od); ln != 2*len(Reference{}) {
					t.Error("wrong length of encoded field: ", ln)
				} else if val.Kind() != reflect.Ptr {
					t.Error("invalid kind of reference: ", val.Kind())
				}
			}
			return nil
		})
	})
	t.Run("pass error", func(t *testing.T) {
		cnt, root, sk := getCntRoot()
		cnt.Register("User", User{})
		root.Inject(User{"Alice", 21, 0}, sk)
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
		var ErrExample = errors.New("an example error")
		err = val.RangeFields(func(fname string, val *Value) error {
			return ErrExample // for example
		})
		if err != ErrExample {
			t.Error("error was replaced")
		}
	})
	t.Run("stop", func(t *testing.T) {
		cnt, root, sk := getCntRoot()
		cnt.Register("User", User{})
		root.Inject(User{"Alice", 21, 0}, sk)
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
		err = val.RangeFields(func(fname string, val *Value) error {
			return ErrStopRange
		})
		if err != nil {
			t.Error("unexpected error: ", err)
		}
	})
}

func TestValue_Len(t *testing.T) {
	t.Run("another", func(t *testing.T) {
		type X struct {
			User    User
			Bool    bool
			Int8    int8
			Int16   int16
			Int32   int32
			Int64   int64
			Uint8   int8
			Uint16  int16
			Uint32  int32
			Uint64  int64
			Float32 bool
			Float64 bool
		}
		cnt, root, sk := getCntRoot()
		cnt.Register("User", User{})
		cnt.Register("X", X{})
		root.Inject(X{}, sk)
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if ln := len(vs); ln != 1 {
			t.Error("wrong values length: ", ln)
			return
		}
		var i int
		err = vs[0].RangeFields(func(_ string, val *Value) error {
			if _, err := val.Len(); err == nil {
				t.Error("missing error")
			}
			i++
			return nil
		})
		if err != nil {
			t.Error(err)
		}
		if i != reflect.TypeOf(X{}).NumField() {
			t.Error("unexpected fields length: ", i)
		}
	})
	t.Run("all", func(t *testing.T) {
		type Users []User
		type Bools []Bool
		type Ary [10]User
		type Y [3]int32
		type X struct {
			Users  Users
			Bools  Bools
			Ary    Ary
			Y      Y
			String String
		}
		cnt, root, sk := getCntRoot()
		cnt.Register("User", User{})
		cnt.Register("X", X{})
		root.Inject(X{
			Users{User{}, User{}, User{}, User{}},
			Bools{Bool(false), Bool(true)},
			Ary{},
			Y{},
			String("hi"),
		}, sk)
		vs, err := root.Values()
		if err != nil {
			t.Error(err)
			return
		}
		if ln := len(vs); ln != 1 {
			t.Error("unexpected values length: ", ln)
			return
		}
		lengths := []int{4, 2, 10, 3, 2}
		var i int
		err = vs[0].RangeFields(func(_ string, val *Value) error {
			if l, err := val.Len(); err != nil {
				t.Error(err)
			} else if l != lengths[i] {
				t.Errorf("unexpected length: want %d, got %d", lengths[i], l)
			}
			i++
			return nil
		})
		if err != nil {
			t.Error(err)
		}
	})
}

func TestValue_Index(t *testing.T) {
	type Users []User
	type Bools []Bool
	type Ary [10]uint32
	type Y [3]int32
	type X struct {
		Users Users
		Bools Bools
		Ary   Ary
		Y     Y
	}
	cnt, root, sk := getCntRoot()
	cnt.Register("User", User{})
	cnt.Register("X", X{})
	root.Inject(X{
		Users{
			User{"Bob", 32, 0},
			User{"Eva", 33, 0},
			User{"Tom", 34, 0},
			User{"Amy", 35, 0},
		},
		Bools{Bool(false), Bool(true)},
		Ary{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		Y{3, 2, 1},
	}, sk)
	values := []interface{}{
		Users{
			User{"Bob", 32, 0},
			User{"Eva", 33, 0},
			User{"Tom", 34, 0},
			User{"Amy", 35, 0},
		},
		Bools{Bool(false), Bool(true)},
		Ary{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		Y{3, 2, 1},
	}
	vs, err := root.Values()
	if err != nil {
		t.Error(err)
		return
	}
	if len(vs) != 1 {
		t.Error("unexpected values length: ", len(vs))
		return
	}
	lengths := []int{4, 2, 10, 3}
	var i int
	err = vs[0].RangeFields(func(_ string, val *Value) error {
		if l, err := val.Len(); err != nil {
			t.Error(err)
		} else if l != lengths[i] {
			t.Errorf("unexpected length: want %d, got %d", lengths[i], l)
		} else {
			// negative
			if _, err := val.Index(-1); err == nil {
				t.Error("missing error")
			}
			// legal
			for j := 0; j < l; j++ {
				if d, err := val.Index(j); err != nil {
					t.Error(err)
				} else {
					cmpValue(d, byIndexes(values, i, j), t)
				}
			}
			// greater
			if _, err := val.Index(l + 1); err == nil {
				t.Error("missing error")
			}
		}
		i++
		return nil
	})
}

func TestValue_Schema(t *testing.T) {
	//
}

func TestSchema_Size(t *testing.T) {
	//
}

// ========================================================================== //
//                                                                            //
//                                helpers                                     //
//                                                                            //
// ========================================================================== //

func byIndexes(a []interface{}, i, j int) interface{} {
	return reflect.ValueOf(a[i]).Index(j).Interface()
}

func cmpValue(val *Value, i interface{}, t *testing.T) bool {
	typ := reflect.TypeOf(i)
	if val.Kind() != typ.Kind() {
		t.Errorf("wrong kind: expected %s, got %s", val.Kind(), typ.Kind())
		return false
	}
	if val.Schema().Name() != "" {
		if sn, tn := val.Schema().Name(), typ.Name(); sn != tn {
			t.Errorf("wrong type name: expected %q, got %q", tn, sn)
			return false
		}
	}
	return true
}
