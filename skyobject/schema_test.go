package skyobject

import (
	"reflect"
	"testing"
)

func Test_typeNames(t *testing.T) {
	t.Log("refTypeName:     ", refTypeName)
	t.Log("refsTypeName:    ", refsTypeName)
	t.Log("dynamicTypeName: ", dynamicTypeName)
}

func Test_getSchema(t *testing.T) {

	type User struct {
		Name   string
		Age    int64
		Hidden string `enc:"-"`
	}

	s := getSchema(User{})
	if s.Name != reflect.TypeOf(User{}).Name() {
		t.Error("invalid schema name: ", s.Name)
	}
	if len(s.Fields) != 2 {
		t.Error("invalid fields count: ", len(s.Fields), s)
		return
	}
	for i, f := range []struct {
		name string
		kind uint32
		typ  string
		tag  reflect.StructTag
	}{
		{"Name", uint32(reflect.String), "string", ""},
		{"Age", uint32(reflect.Int64), "int64", ""},
	} {
		x := s.Fields[i]
		if x.Name != f.name {
			t.Error("wrong field name")
		}
		if x.Schema.Kind != f.kind {
			t.Error("wrong field kind")
		}
		if x.Schema.Name != f.typ {
			t.Error("wrong field type")
		}
		if x.Tag != f.tag {
			t.Error("wrong field tag")
		}
	}

}
