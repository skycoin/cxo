package skyobject

/*
import (
	"reflect"
	"strings"
	"testing"
)

type User struct {
	Name   string
	Age    int64
	Hidden string `enc:"-"`
}

func Test_typeName(t *testing.T) {
	x := reflect.TypeOf(User{})
	if typeName(x) != strings.ToLower(x.Name()) {
		t.Error("wrong type name")
	}
}

func Test_getField(t *testing.T) {
	ft := reflect.StructField{
		Name: "name",
		Type: reflect.TypeOf(User{}),
		Tag:  reflect.StructTag(`enc:"asdf",slyobject:"zxcv"`),
	}
	sf := getField(ft)
	if sf.Name != "name" {
		t.Error("erong field name")
	}
	if sf.Tag != string(ft.Tag) {
		t.Error("erong tag")
	}
	if sf.Type != typeName(reflect.TypeOf(User{})) {
		t.Error("wrong type name")
	}
	if sf.Kind != uint32(reflect.Struct) {
		t.Error("wrong kind")
	}
}

func Test_getSchema(t *testing.T) {
	s := getSchema(User{})
	if s == nil {
		t.Error("getSchema returns nil")
	}
	if s.Name != reflect.TypeOf(User{}).Name() {
		t.Error("wrong schema name")
	}
	if len(s.Fields) != 2 {
		t.Error("wrong number of fields")
	}
}

func TestSchema_String(t *testing.T) {
	s := getSchema(User{})
	ss := "User\n" +
		"  Name string `` <String>\n" +
		"  Age int64 `` <Int64>\n"
	if s.String() != ss {
		t.Error("wrong string: ", s.String())
	}
}
*/
