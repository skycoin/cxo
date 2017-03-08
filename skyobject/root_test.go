package skyobject

import (
	"time"

	"testing"
)

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

func TestRoot_Touch(t *testing.T) {
	// TODO:
}
