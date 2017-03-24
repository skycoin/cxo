package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

func TestRoot_Touch(t *testing.T) {
	//
}

func TestRoot_Inject(t *testing.T) {
	//
}

func Test_signature(t *testing.T) {
	pub, sec := cipher.GenerateKeyPair()
	b := []byte("hello")
	hash := cipher.SumSHA256(b)
	sig := cipher.SignHash(hash, sec)
	if err := cipher.VerifySignature(pub, sig, hash); err != nil {
		t.Error(err)
	}
}

func Test_encodeDecode(t *testing.T) {
	c := getCont()
	r := c.NewRoot(pubKey())
	r.Register("User", User{})
	p := r.Encode()
	re, err := decodeRoot(p)
	if err != nil {
		t.Error(err)
		return
	}
	if re.Time != r.Time {
		t.Error("wrong time")
	}
	if re.Seq != r.Seq {
		t.Error("wrong seq")
	}
	if len(re.Refs) != len(r.Refs) {
		t.Error("wrong Refs length")
	}
	for i, ref := range re.Refs {
		if r.Refs[i] != ref {
			t.Error("wrong reference", i)
		}
	}
	if len(r.reg.reg) != len(re.Reg) {
		t.Error("wrong Reg length")
	}
	for _, ent := range re.Reg {
		if r.reg.reg[ent.K] != ent.V {
			t.Error("wrong entity ", ent.K)
		}
	}
}

func TestRoot_Encode(t *testing.T) {
	pub, sec := cipher.GenerateKeyPair()
	// encode
	c1 := getCont()
	r1 := c1.NewRoot(pub)
	r1.Register("User", User{})
	r1.SaveSchema(Group{})
	r1.Sign(sec)
	sig := r1.Sig
	p := r1.Encode()
	if r1.Pub != pub {
		t.Error("pub key was changed during encoding")
	}
	if r1.Sig != sig {
		t.Error("signature was changed during encoding")
	}
	// decode
	c2 := getCont()
	if ok, err := c2.SetEncodedRoot(p, r1.Pub, r1.Sig); err != nil {
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
