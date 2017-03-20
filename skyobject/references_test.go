package skyobject

import (
	"testing"
)

func TestRoot_Save(t *testing.T) {
	root := getRoot()
	ref := root.Save(String("hello"))
	od, ok := root.cnt.get(ref)
	if !ok {
		t.Error("no saved")
	}
	if len(od) != len("hello")+4 {
		t.Error("wrong len: ", len(od))
	}
}

func TestReference_String(t *testing.T) {
	//
}

func TestRoot_SaveArray(t *testing.T) {
	//
}

func TestRoot_SaveSchema(t *testing.T) {
	//
}

func TestRoot_Dynamic(t *testing.T) {
	//
}

func TestRoot_RegisterSchema(t *testing.T) {
	//
}
