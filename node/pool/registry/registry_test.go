package registry

import (
	"testing"
)

func shouldPanic(t *testing.T) {
	if recover() == nil {
		t.Error("missing panic")
	}

}

func TestNewPrefix(t *testing.T) {
	t.Run("short length", func(t *testing.T) {
		defer shouldPanic(t)
		NewPrefix("s") // short
	})
	t.Run("long length", func(t *testing.T) {
		defer shouldPanic(t)
		NewPrefix("very_long_string") // long
	})
	t.Run("validate", func(t *testing.T) {
		defer shouldPanic(t)
		NewPrefix("    ") // spaces are illegal
	})
	t.Run("content", func(t *testing.T) {
		s := "prfx"
		p := NewPrefix(s)
		for i := 0; i < PrefixLength; i++ {
			if s[i] != p[i] {
				t.Error("nvalid byte in prefix created by NewPrefix")
			}
		}
	})
}

func TestPrefix_Validate(t *testing.T) {
	t.Run("special", func(t *testing.T) {
		var p Prefix
		copy(p[:], "----")
		if err := p.Validate(); err == nil {
			t.Error("misisng error")
		} else if err != ErrSpecialCharacterInPrefix {
			t.Error("unexpected error:", err)
		}
	})
	t.Run("space", func(t *testing.T) {
		var p Prefix
		copy(p[:], "    ")
		if err := p.Validate(); err == nil {
			t.Error("misisng error")
		} else if err != ErrSpaceInPrefix {
			t.Error("unexpected error:", err)
		}
	})
	t.Run("unprintable", func(t *testing.T) {
		var p Prefix
		copy(p[:], "\b5Ὂg̀")
		if err := p.Validate(); err == nil {
			t.Error("misisng error")
		} else if err != ErrUnprintableCharacterInPrefix {
			t.Error("unexpected error:", err)
		}
	})
	t.Run("valid", func(t *testing.T) {
		var p Prefix
		copy(p[:], "asdf")
		if err := p.Validate(); err != nil {
			t.Error("unexpected error:", err)
		}
	})
}

func TestPrefix_String(t *testing.T) {
	p := NewPrefix("asdf")
	if p.String() != "asdf" {
		t.Error("unexpected string:", p.String())
	}
}

func TestErrFiltered_Sending(t *testing.T) {
	//
}

func TestErrFiltered_Error(t *testing.T) {
	//
}

func TestNewRegistry(t *testing.T) {
	//
}

func TestRegistry_Register(t *testing.T) {
	//
}

func TestRegistry_AddSendFilter(t *testing.T) {
	//
}

func TestRegistry_AddReceiveFilter(t *testing.T) {
	//
}

func TestRegistry_AllowSend(t *testing.T) {
	//
}

func TestRegistry_AllowReceive(t *testing.T) {
	//
}

func TestRegistry_Type(t *testing.T) {
	//
}

func TestRegistry_Enocde(t *testing.T) {
	//
}

func TestRegistry_DecodeHead(t *testing.T) {
	//
}

func TestRegistry_Decode(t *testing.T) {
	//
}

func TestRegistry_DecodeValue(t *testing.T) {
	//
}
