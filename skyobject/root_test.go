package skyobject

import (
	"testing"
)

func TestRoot_Touch(t *testing.T) {
	//
}

func TestRoot_Inject(t *testing.T) {
	//
}

func TestRoot_Encode(t *testing.T) {
	pk := pubKey()
	// encode
	c1 := getCont()
	r1 := c1.NewRoot(pk)
	r1.Register("User", User{})
	r1.SaveSchema(Group{})
	p := r1.Encode()
	// decode
	c2 := getCont()
	if ok, err := c2.SetEncodedRoot(p); err != nil {
		t.Error(err)
	} else if !ok {
		t.Error("can't set encoded root")
	} else if len(c2.reg.reg) != len(c1.reg.reg) {
		t.Error("wrong registry")
	}
}

func TestRoot_SchemaByReference(t *testing.T) {
	//
}

func TestRoot_Save(t *testing.T) {
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

func TestRoot_Register(t *testing.T) {
	//
}

func TestRoot_Values(t *testing.T) {
	//
}
