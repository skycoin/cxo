package registry

import (
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

	r := testSchemaRegistry()

	ts, err := r.SchemaByName("test.TestStruct")
	if err != nil {
		t.Fatal(err)
	}

	if ts.IsReference() {
		t.Error("wrong IsReference resilt for non-reference struct")
	}

	for _, fl := range ts.Fields() {
		t.Log(fl.Name(), fl.Schema())
		if fs := fl.Schema(); fs.IsReference() {
			var el Schema
			var kindOf string
			switch fs.ReferenceType() {
			case ReferenceTypeSingle:
				el, kindOf = fs.Elem(), "Ref"
			case ReferenceTypeSlice:
				el, kindOf = fs.Elem(), "Refs"
			case ReferenceTypeDynamic:
				continue
			default:
				t.Error("malformed reference:", fs)
			}
			if el == nil {
				t.Error("nil Elem of", kindOf)
				continue
			}
			bn, err := r.SchemaByName(el.Name())
			if err != nil {
				t.Errorf("%s (%s) points to unregistered type", kindOf, fs)
			}
			if bn != el {
				t.Error("unnecessary memory overhead")
			}
		}
	}

}

func TestSchema_ReferenceType(t *testing.T) {
	// ReferenceType() ReferenceType

	r := testSchemaRegistry()

	ts, err := r.SchemaByName("test.TestStruct")
	if err != nil {
		t.Fatal(err)
	}

	for _, fl := range ts.Fields() {
		t.Log(fl.Name(), fl.Schema())
		fs := fl.Schema()
		if fs.IsReference() {
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

	r := testSchemaRegistry()

	for _, name := range []string{
		"test.TestSmallStruct",
		"test.TestStruct",
	} {
		t.Log(name)
		sc, err := r.SchemaByName(name)
		if err != nil {
			t.Fatal(err)
		}
		if !sc.HasReferences() {
			t.Error("schema with references, but HasReferences returns false")
		}
	}

	sc, err := r.SchemaByName("test.TestNumberStruct")
	if err != nil {
		t.Fatal(err)
	}
	if sc.HasReferences() {
		t.Error("schema without references, but HasReferences returns true")
	}

	ts, err := r.SchemaByName("test.TestStruct")
	if err != nil {
		t.Fatal(err)
	}

	var hasNotRefs = map[string]struct{}{
		"TestInt8":                   {},
		"TestInt16":                  {},
		"TestInt32":                  {},
		"TestInt64":                  {},
		"TestUint8":                  {},
		"TestUint16":                 {},
		"TestUint32":                 {},
		"TestUint64":                 {},
		"TestInt8Array":              {},
		"TestInt16Array":             {},
		"TestInt32Array":             {},
		"TestInt64Array":             {},
		"TestUint8Array":             {},
		"TestUint16Array":            {},
		"TestUint32Array":            {},
		"TestUint64Array":            {},
		"TestInt8Slice":              {},
		"TestInt16Slice":             {},
		"TestInt32Slice":             {},
		"TestInt64Slice":             {},
		"TestUint8Slice":             {},
		"TestUint16Slice":            {},
		"TestUint32Slice":            {},
		"TestUint64Slice":            {},
		"TestString":                 {},
		"TestNumberStructArray":      {},
		"TestNumberStructSlice":      {},
		"FieldTestInt8":              {},
		"FieldTestInt16":             {},
		"FieldTestInt32":             {},
		"FieldTestInt64":             {},
		"FieldTestUint8":             {},
		"FieldTestUint16":            {},
		"FieldTestUint32":            {},
		"FieldTestUint64":            {},
		"FieldTestInt8Array":         {},
		"FieldTestInt16Array":        {},
		"FieldTestInt32Array":        {},
		"FieldTestInt64Array":        {},
		"FieldTestUint8Array":        {},
		"FieldTestUint16Array":       {},
		"FieldTestUint32Array":       {},
		"FieldTestUint64Array":       {},
		"FieldTestInt8Slice":         {},
		"FieldTestInt16Slice":        {},
		"FieldTestInt32Slice":        {},
		"FieldTestInt64Slice":        {},
		"FieldTestUint8Slice":        {},
		"FieldTestUint16Slice":       {},
		"FieldTestUint32Slice":       {},
		"FieldTestUint64Slice":       {},
		"FieldTestString":            {},
		"FieldTestNumberStructArray": {},
		"FieldTestNumberStructSlice": {},
	}

	for _, fl := range ts.Fields() {
		t.Log(fl.Name(), fl.Schema())
		if _, ok := hasNotRefs[fl.Name()]; ok {
			if fl.Schema().HasReferences() {
				t.Error("HasReferences() returns true, " +
					"but schema has not references")
			}
			continue
		}
		if !fl.Schema().HasReferences() {
			t.Error("HasReferences() returns false, " +
				"but schema has (or is) references")
		}
	}

}

func TestSchema_Kind(t *testing.T) {
	// Kind() reflect.Kind

	// TODO (kostyarin): low priority

}

func TestSchema_Name(t *testing.T) {
	// Name() string

	r := testSchemaRegistry()

	for _, name := range []string{
		"test.TestNumberStruct",
		"test.TestSmallStruct",
		"test.TestStruct",
	} {
		t.Log(name)
		sc, err := r.SchemaByName(name)
		if err != nil {
			t.Fatal(err)
		}
		if sc.Name() != name {
			t.Errorf("wrong Name(): want %q, got %q", name, sc.Name())
		}
	}

}

func TestSchema_Len(t *testing.T) {
	// Len() int

	// TODO (kostyarin): low priority

}

func TestSchema_Fields(t *testing.T) {
	// Fields() []Field

	// TODO (kostyarin): low priority

}

func TestSchema_Elem(t *testing.T) {
	// Elem() (s Schema)

	// TODO (kostyarin): low priority

}

func TestSchema_RawName(t *testing.T) {
	// RawName() []byte

	r := testSchemaRegistry()

	for _, name := range []string{
		"test.TestNumberStruct",
		"test.TestSmallStruct",
		"test.TestStruct",
	} {
		t.Log(name)
		sc, err := r.SchemaByName(name)
		if err != nil {
			t.Fatal(err)
		}
		if rn := string(sc.RawName()); rn != name {
			t.Errorf("wrong Name(): want %q, got %q", name, rn)
		}
	}

}

func TestSchema_IsRegistered(t *testing.T) {
	// IsRegistered() bool

	r := testSchemaRegistry()

	for _, name := range []string{
		"test.TestNumberStruct",
		"test.TestSmallStruct",
		"test.TestStruct",
	} {
		t.Log(name)
		sc, err := r.SchemaByName(name)
		if err != nil {
			t.Fatal(err)
		}
		if !sc.IsRegistered() {
			t.Error("registered is not registered")
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
