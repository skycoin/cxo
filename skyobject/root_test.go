package skyobject

// TODO: redo

import (
//"testing"
//"time"
)

/*
func TestNewRoot(t *testing.T) {
	tp := time.Now().UnixNano()
	root := NewRoot(User{})
	if root.Seq != 0 {
		t.Error("invalid Seq for fresh root: ", root.Seq)
	}
	if root.Time < tp || root.Time > time.Now().UnixNano() {
		t.Error("invalid time for fresh root")
	}
	if root.Schema.Name != getSchema(User{}).Name {
		t.Error("invalid schema")
	}
	if len(root.Root) == 0 {
		t.Error("invalid root object")
	}
}

func TestRoot_Register(t *testing.T) {
	r := NewRoot(User{})
	r.Register("user", User{})
	if len(r.registry) != 1 {
		t.Error("don't register")
		return
	}
	for k, v := range r.registry {
		if k != "user" {
			t.Error("registered with wrong name")
		}
		if v.Name != reflect.TypeOf(User{}).Name() {
			t.Error("wrong schema registered")
		}
	}
}

func TestRoot_Schema(t *testing.T) {
	r := NewRoot(User{})
	if s, ok := r.Schema("user"); ok {
		t.Error("returns schema that should not have")
	} else if s != nil {
		t.Error("it's crazy")
	}
	r.Register("user", User{})
	if s, ok := r.Schema("user"); !ok {
		t.Error("can't return schema that should have")
	} else if s == nil {
		t.Error("misisng schema")
	} else if s.Name != reflect.TypeOf(User{}).Name() {
		t.Error("wrong schema")
	}
}

func TestRoot_Touch(t *testing.T) {
	// TODO:
}
*/
