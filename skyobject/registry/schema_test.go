package registry

import (
	"reflect"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func testCalculateSchemaRef(s Schema) SchemaRef {
	return SchemaRef(cipher.SumSHA256(s.Encode()))
}

func TestSchema_Reference(t *testing.T) {
	// Reference() SchemaRef

	var (
		reg = testRegistry()

		sc, sg Schema
		sr     SchemaRef

		err error
	)

	for _, tt := range testTypes() {

		t.Log(tt.Name)

		if sc, err = reg.SchemaByName(tt.Name); err != nil {
			t.Error(err)
			continue
		}

		if sr = sc.Reference(); sr == (SchemaRef{}) {
			t.Error("blank SchemaRef")
			continue
		}

		if cc := testCalculateSchemaRef(sc); cc != sr {
			t.Errorf("wrong ScehmaRef: want %s, got %s", cc.Short(), sr.Short())
			continue
		}

		if sg, err = reg.SchemaByReference(sr); err != nil {
			t.Error("can't get Schema by SchemaRef:", err)
			continue
		}

		if sg != sc {
			t.Error("different ponters to the same schema (memory pressure)")
		}

	}

}

func TestSchema_IsReference(t *testing.T) {
	// IsReference() bool

	var (
		reg = testRegistry()

		ts  Schema
		err error
	)

	for _, tt := range testTypes() {

		t.Log(tt.Name)

		if ts, err = reg.SchemaByName(tt.Name); err != nil {
			t.Error("can't find Shema by name:", err)
			continue
		}

		if true == ts.IsReference() {
			t.Error("IsReference() returns true for non-reference type")
			// keep going on
		}

		for _, fl := range ts.Fields() {
			t.Log(tt.Name, ">", fl.Name())

			if fs := fl.Schema(); true == fs.IsReference() {

				if tt.Name != "test.TestGroup" {
					t.Error("unexpected reference")
					continue
				}

				switch typ := fs.ReferenceType(); typ {
				case ReferenceTypeNone:
					t.Error("IsReference() returns true but ReferenceType is " +
						"ReferenceTypeNone")
				case ReferenceTypeSingle:
					if fl.Name() != "Curator" {
						t.Error("unexpected Ref")
						continue
					}
				case ReferenceTypeSlice:
					if fl.Name() != "Members" {
						t.Error("unexpected Refs")
						continue
					}
				case ReferenceTypeDynamic:
					if fl.Name() != "Developer" {
						t.Error("unexpected Dynamic reference")
					}
				default:
					t.Error("IsReference() returns true but ReferenceType is "+
						"undefined %d", typ)
				}

				if el := fs.Elem(); el.Name() != "test.User" {
					t.Error("unknown Schema of reference")
				} else if gs, err := reg.SchemaByName("test.User"); err != nil {
					t.Error(err)
				} else if gs != el {
					t.Error("unnecessary memory overhead")
				}

			}

		}

	}

}

func TestSchema_ReferenceType(t *testing.T) {
	// ReferenceType() ReferenceType

	var (
		reg = testRegistry()

		gr  Schema
		err error
	)

	if gr, err = reg.SchemaByName("test.TestGroup"); err != nil {
		t.Fatal(err)
	}

	for _, fl := range gr.Fields() {
		t.Log(fl.Name(), fl.Schema())

		var fs = fl.Schema()

		if true == fs.IsReference() {
			switch fs.ReferenceType() {
			case ReferenceTypeSingle, ReferenceTypeSlice, ReferenceTypeDynamic:
			default:
				t.Error("malformed reference (wrong ref. type):", fs)
			}
			continue
		}

		if rt := fs.ReferenceType(); rt != ReferenceTypeNone {
			t.Error("non-reference has ReferenceType", rt)
		}

	}

}

func TestSchema_HasReferences(t *testing.T) {
	// HasReferences() bool

	var (
		reg = testRegistry()

		s   Schema
		err error
	)

	// todo inclding deep nested schemas with references

	for _, tt := range testTypes() {

		t.Log(tt.Name)

		if s, err = reg.SchemaByName(tt.Name); err != nil {
			continue
		}

		if true == s.HasReferences() {
			if s.Name() == "test.Group" {
				continue // ok
			} else {
				t.Error(tt.Name, "has references")
			}
		}

	}

	// deep nested reference

	type A struct {
		Name string
	}

	type B struct {
		A Ref `skyobject:"schema=test.A"`
	}

	type C struct {
		B B
	}

	type D struct {
		S []B
	}

	type E struct {
		A [3]D
	}

	reg = NewRegistry(func(r *Reg) {
		r.Register("test.A", A{})
		r.Register("test.B", B{})
		r.Register("test.C", C{})
		r.Register("test.D", D{})
		r.Register("test.E", E{})
	})

	if s, err = reg.SchemaByName("test.E"); err != nil {
		t.Fatal(err)
	}

	if false == s.HasReferences() {
		t.Error("has not references (deep)")
	}

}

func TestSchema_Kind(t *testing.T) {
	// Kind() reflect.Kind

	// TODO (kostyarin): low priority

}

func TestSchema_Name(t *testing.T) {
	// Name() string

	var reg = testRegistry()

	for _, tt := range testTypes() {

		t.Log(tt.Name)

		if s, err := reg.SchemaByName(tt.Name); err != nil {
			t.Error(err)
		} else if name := s.Name(); name != tt.Name {
			t.Errorf("wrong Schema name: want %q, got %q", tt.Name, name)
		}

	}

}

func TestSchema_Len(t *testing.T) {
	// Len() int

	var (
		reg = testRegistry()

		s   Schema
		err error
	)

	defer shouldNotPanic(t) // reflect methods can panic

	for _, tt := range testTypes() {

		t.Log(tt.Name)

		if s, err = reg.schemaByName(tt.Name); err != nil {
			t.Error(err)
			continue
		}

		if s.Len() != 0 {
			t.Error("got non-zero Len of struct")
		}

		var rt = reflect.Indirect(reflect.ValueOf(tt.Val)).Type()

		for i, fl := range s.Fields() {

			t.Log(tt.Name, ">", fl.Name())

			var rf = rt.FieldByIndex(i).Type // refelct.Type of the field

			if kind := fl.Kind(); kind == reflect.Array {

				if rf.Type.Kind() != reflect.Array {
					t.Error("unexpected array")
					continue
				}

				if sl, rl := fl.Schema().Len(), rf.Len(); sl != rl {
					t.Errorf("invalid length of array: want %d, got %d", sl, rl)
				}

			} else if s.Len() != 0 {

				t.Error("got non-zero Len of", kind.String())

			}

		}

	}

}

func TestSchema_Fields(t *testing.T) {
	// Fields() []Field

	var (
		reg = testRegistry()

		s   Schema
		err error
	)

	defer shouldNotPanic(t) // reflect methods can panic

	for _, tt := range testTypes() {

		t.Log(tt.Name)

		if s, err = reg.schemaByName(tt.Name); err != nil {
			t.Error(err)
			continue
		}

		var rt = reflect.Indirect(reflect.ValueOf(tt.Val)).Type()

		for i, fl := range s.Fields() {

			t.Log(tt.Name, ">", fl.Name())

			var rf = rt.FieldByIndex(i) // refelct.StructField

			// ccompare names

			if name := fl.Name(); name != rf.Name {
				t.Error("wrong field name: want %q, got %q", rf.Name, name)
			}

			if kind := fl.Kind(); kind == reflect.Ptr {
				continue // Ref, Refs, Dynaimc (skip them)
			} else if reflectKind := rf.Type.Kind(); kind != reflectKind {
				t.Error("invalid kind of struct field: want %s, got %s",
					reflectKind.String(), kind.String())
			}

		}

	}

}

func TestSchema_Elem(t *testing.T) {
	// Elem() (s Schema)

	// The Elem is element of a reference (except
	// Dynaimc), of an array, or of a slice

	var (
		reg = testRegistry()

		s   Schema
		err error
	)

	defer shouldNotPanic(t) // reflect methods can panic

	for _, tt := range testTypes() {

		t.Log(tt.Name)

		if s, err = reg.schemaByName(tt.Name); err != nil {
			t.Error(err)
			continue
		}

		if s.Elem() != nil {
			t.Error("unexpected Elem of struct")
		}

		var rt = reflect.Indirect(reflect.ValueOf(tt.Val)).Type()

		for i, fl := range s.Fields() {

			t.Log(tt.Name, ">", fl.Name())

			var rf = rt.FieldByIndex(i) // refelct.StructField

			// Ref, Refs, array of slice

			var fs = fl.Schema()
			var el = fs.Elem()

			switch kind := fs.Kind(); kind {
			case reflect.Ptr:

				switch fs.ReferenceType() {
				case ReferenceTypeSingle, ReferenceTypeSlice:

					if el == nil {
						t.Error("misisng Elem")
						continue
					}

					var tsn string
					if tsn, err = TagSchemaName(rf.Tag); err != nil {
						t.Error("missing tagged schema")
					}

					if el.Name() != tsn {
						t.Errorf("wrong schema iof reference: want %q, got %q",
							tsn, el.Name())
					}

				default:

					if el != nil {
						t.Error("unexpected Elem")
					}

				}

			case reflect.Array, reflect.Slice:

				if el == nil {
					t.Error("missing Elem")
				} else if ek, rk := el.Kind(), rf.Type.Kind(); ek != rk {
					t.Errorf("invalid kind of Elem: want %s, got %s",
						rk.String(), ek.String())
				}

			default:

				if el != nil {
					t.Error("unexpected Elem")
				}

			}

		}

	}

}

func TestSchema_RawName(t *testing.T) {
	// RawName() []byte

	var reg = testRegistry()

	for _, tt := range testTypes() {

		t.Log(tt.Name)

		if s, err := reg.SchemaByName(tt.Name); err != nil {
			t.Error(err)
		} else if name := string(s.RawName()); name != tt.Name {
			t.Errorf("wrong Schema RawName: want %q, got %q", tt.Name, name)
		}

	}

}

func TestSchema_IsRegistered(t *testing.T) {
	// IsRegistered() bool

	var (
		reg = testRegistry()

		s   Schema
		err error
	)

	for _, tt := range testTypes() {

		t.Log(tt.Name)

		if s, err = reg.SchemaByName(tt.Name); err != nil {
			t.Error(err)
			continue
		}

		if false == s.IsRegistered() {
			t.Error("IsRegistered returns false for registered schema")
			// keep going on
		}

		// only named structures can be registered

		for _, fl := range s.Fields() {

			var fs = fl.Schema()

			if fs.Kind() == reflect.Struct {
				// TODO (kostyarin): for future to increase coverage easy way
				continue
			}

			if true == fs.IsRegistered() {
				t.Error("IsRegistered of non-struct returns true")
			}

		}

	}

}

func TestSchema_Encode(t *testing.T) {
	// Encode() (b []byte)

	// TODO: low priority

}

func TestSchema_Size(t *testing.T) {
	// Size(p []byte) (n int, err error)

	r := testSchemaRegistry()

	sc, err := r.schemaByName("test.TestStruct")
	if err != nil {
		t.Fatal(err)
	}

	ts := TestStruct{
		TestInt8:  0,
		TestInt16: 1,
		TestInt32: 2,
		TestInt64: 3,

		TestUint8:  4,
		TestUint16: 5,
		TestUint32: 6,
		TestUint64: 7,

		TestInt8Array:  [3]TestInt8{8, 9, 10},
		TestInt16Array: [3]TestInt16{11, 12, 13},
		TestInt32Array: [3]TestInt32{14, 15, 16},
		TestInt64Array: [3]TestInt64{17, 18, 19},

		TestUint8Array:  [3]TestUint8{20, 21, 22},
		TestUint16Array: [3]TestUint16{23, 24, 25},
		TestUint32Array: [3]TestUint32{26, 27, 28},
		TestUint64Array: [3]TestUint64{29, 30, 31},

		TestInt8Slice:  []TestInt8{32, 33, 34, 35},
		TestInt16Slice: []TestInt16{36, 37, 38, 39},
		TestInt32Slice: []TestInt32{40, 41, 42, 43},
		TestInt64Slice: []TestInt64{44, 45, 46, 47},

		TestUint8Slice:  []TestUint8{48, 49, 50, 51},
		TestUint16Slice: []TestUint16{52, 53, 54, 55},
		TestUint32Slice: []TestUint32{56, 57, 58, 59},
		TestUint64Slice: []TestUint64{60, 61, 62, 63},

		TestString:      "64",
		TestSmallStruct: TestSmallStruct{},

		TestNumberStructArray: [3]TestNumberStruct{{65}, {66}, {67}},
		TestNumberStructSlice: []TestNumberStruct{{68}, {69}, {70}, {71}},

		Ref:     Ref{},
		Refs:    Refs{},
		Dynamic: Dynamic{},

		// not embedded fields

		FieldTestInt8:  72,
		FieldTestInt16: 73,
		FieldTestInt32: 74,
		FieldTestInt64: 75,

		FieldTestUint8:  76,
		FieldTestUint16: 77,
		FieldTestUint32: 78,
		FieldTestUint64: 79,

		FieldTestInt8Array:  [3]TestInt8{80, 81, 82},
		FieldTestInt16Array: [3]TestInt16{83, 84, 85},
		FieldTestInt32Array: [3]TestInt32{86, 87, 88},
		FieldTestInt64Array: [3]TestInt64{89, 90, 91},

		FieldTestUint8Array:  [3]TestUint8{92, 93, 94},
		FieldTestUint16Array: [3]TestUint16{95, 96, 97},
		FieldTestUint32Array: [3]TestUint32{98, 99, 100},
		FieldTestUint64Array: [3]TestUint64{101, 102, 103},

		FieldTestInt8Slice:  []TestInt8{104, 105, 106, 107},
		FieldTestInt16Slice: []TestInt16{108, 109, 110, 111},
		FieldTestInt32Slice: []TestInt32{112, 113, 114, 115},
		FieldTestInt64Slice: []TestInt64{116, 117, 118, 119},

		FieldTestUint8Slice:  []TestUint8{120, 121, 122, 123},
		FieldTestUint16Slice: []TestUint16{124, 125, 126, 127},
		FieldTestUint32Slice: []TestUint32{128, 129, 130, 131},
		FieldTestUint64Slice: []TestUint64{132, 133, 134, 135},

		FieldTestString:      "136",
		FieldTestSmallStruct: TestSmallStruct{},

		FieldTestNumberStructArray: TestNumberStructArray{
			{137},
			{138},
			{139},
		},
		FieldTestNumberStructSlice: TestNumberStructSlice{},

		FieldRef:     Ref{},
		FieldRefs:    Refs{},
		FieldDynamic: Dynamic{},
	}

	var p = encoder.Serialize(ts)
	//var i int

	if n, err := sc.Size(p); err != nil {
		t.Error(err)
	} else if n != len(p) {
		t.Error("wriong size")
	}

	// empty
	var tse TestStruct
	ep := encoder.Serialize(tse)
	if n, err := sc.Size(ep); err != nil {
		t.Error(err)
	} else if n != len(ep) {
		t.Error("wrong size")
	}

	// truncate
	for i := len(p) - 1; i >= 0; i-- {
		if _, err := sc.Size(p[:i]); err == nil {
			t.Error("missing error")
		}
	}

}

func TestSchema_String(t *testing.T) {
	// String() string

	// TODO: low priority

}
