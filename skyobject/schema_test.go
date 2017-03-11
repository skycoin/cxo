package skyobject

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
)

func Test_getSchema(t *testing.T) {
	// type User struct {
	// 	Name   string
	// 	Age    int64
	// 	Hidden string `enc:"-"`
	// }
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
		tag  string
	}{
		{"Name", uint32(reflect.String), "string", ""},
		{"Age", uint32(reflect.Int64), "int64", ""},
	} {
		x := s.Fields[i]
		if x.Name != f.name {
			t.Error("wrong field name")
		}
		if x.Kind != f.kind {
			t.Error("wrong field kind")
		}
		if x.Type != f.typ {
			t.Error("wrong field type")
		}
		if x.Tag != f.tag {
			t.Error("wrong field tag")
		}
	}

}

func Test_getField(t *testing.T) {
	// TODO
}

func ExampleSchema_String() {
	db := data.NewDB()
	c := NewContainer(db)
	r := c.NewRoot()
	r.Register("User", User{})
	schk, ok := r.SchemaKey("User")
	if !ok {
		// fatal error
		return
	}
	data, ok := db.Get(schk)
	if !ok {
		// fatal error
		return
	}
	var s Schema
	if err := encoder.DeserializeRaw(data, &s); err != nil {
		// fatal error
		return
	}
	fmt.Println(s.String())

	// Output:
	// User {Name string `` <string>; Age int64 `` <int64>}
}
