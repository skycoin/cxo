package registry

import (
	"reflect"
	"testing"

	"github.com/skycoin/skycoin/src/cipher/encoder"
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
	e := ErrFiltered{true, NewPrefix("asdf")}
	if !e.Sending() {
		t.Error("Sending returns false for sending error")
	}
	e = ErrFiltered{false, NewPrefix("asdf")}
	if e.Sending() {
		t.Error("Sending returns true for receiving error")
	}
}

func TestErrFiltered_Error(t *testing.T) {
	e := ErrFiltered{false, NewPrefix("asdf")}
	if e.Error() == "" {
		t.Error("empty error")
	}
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry(-1)
	if r.max != MaxSize {
		t.Error("wrong limit set")
	}
	r = NewRegistry(0)
	if r.max != 0 {
		t.Error("wrong limit set")
	}
	r = NewRegistry(1)
	if r.max != 1 {
		t.Error("wrong limit set")
	}
}

type Any struct{ String string }

func TestRegistry_Register(t *testing.T) {
	t.Run("validate", func(t *testing.T) {
		var p Prefix
		copy(p[:], "    ")
		r := NewRegistry(0)
		defer shouldPanic(t)
		r.Register(p, Any{})
	})
	type Some struct{ Int int64 }
	t.Run("prefix twice", func(t *testing.T) {
		r := NewRegistry(0)
		p := NewPrefix("asdf")
		r.Register(p, Any{})
		defer shouldPanic(t)
		r.Register(p, Some{})
	})
	t.Run("type twice", func(t *testing.T) {
		r := NewRegistry(0)
		r.Register(NewPrefix("anym"), Any{})
		defer shouldPanic(t)
		r.Register(NewPrefix("some"), Any{})
	})
	t.Run("invalid type", func(t *testing.T) {
		r := NewRegistry(0)
		type X interface{}
		defer shouldPanic(t)
		r.Register(NewPrefix("some"), X(nil))
	})
}

func TestRegistry_AddSendFilter(t *testing.T) {
	t.Run("nil filter", func(t *testing.T) {
		r := NewRegistry(0)
		defer shouldPanic(t)
		r.AddSendFilter(nil)
	})
	// TODO: low priority
}

func TestRegistry_AddReceiveFilter(t *testing.T) {
	t.Run("nil filter", func(t *testing.T) {
		r := NewRegistry(0)
		defer shouldPanic(t)
		r.AddReceiveFilter(nil)
	})
	// TODO: low priority
}

func TestRegistry_AllowSend(t *testing.T) {
	t.Run("no filters", func(t *testing.T) {
		r := NewRegistry(0)
		if !r.AllowSend(NewPrefix("anym")) {
			t.Error("not allowed")
		}
		if !r.AllowSend(NewPrefix("some")) {
			t.Error("not allowed")
		}
	})
	t.Run("filters", func(t *testing.T) {
		r := NewRegistry(0)
		p := NewPrefix("asdf")
		r.AddSendFilter(func(prefix Prefix) bool { return prefix == p })
		if r.AllowSend(NewPrefix("some")) {
			t.Error("allow")
		}
		if !r.AllowSend(p) {
			t.Error("not allowed")
		}
	})
}

func TestRegistry_AllowReceive(t *testing.T) {
	t.Run("no filters", func(t *testing.T) {
		r := NewRegistry(0)
		if !r.AllowReceive(NewPrefix("anym")) {
			t.Error("not allowed")
		}
		if !r.AllowReceive(NewPrefix("some")) {
			t.Error("not allowed")
		}
	})
	t.Run("filters", func(t *testing.T) {
		r := NewRegistry(0)
		p := NewPrefix("asdf")
		r.AddReceiveFilter(func(prefix Prefix) bool { return prefix == p })
		if r.AllowReceive(NewPrefix("some")) {
			t.Error("allow")
		}
		if !r.AllowReceive(p) {
			t.Error("not allowed")
		}
	})
}

func TestRegistry_Type(t *testing.T) {
	r := NewRegistry(0)
	p := NewPrefix("anym")
	r.Register(p, Any{})
	if _, ok := r.Type(NewPrefix("asdf")); ok {
		t.Error("got unregistered type")
	}
	if typ, ok := r.Type(p); !ok {
		t.Error("missing registered type")
	} else if typ != reflect.Indirect(reflect.ValueOf(Any{})).Type() {
		t.Error("wrong type")
	}
}

func TestRegistry_Enocde(t *testing.T) {
	t.Run("unregistered", func(t *testing.T) {
		r := NewRegistry(0)
		defer shouldPanic(t)
		r.Enocde(Any{})
	})
	t.Run("filtered", func(t *testing.T) {
		r := NewRegistry(0)
		p := NewPrefix("ANYM")
		r.Register(p, Any{})
		// don't allow
		r.AddSendFilter(func(prefix Prefix) bool { return prefix != p })
		if _, err := r.Enocde(Any{}); err == nil {
			t.Error("missing error")
		} else if fe, ok := err.(*ErrFiltered); !ok {
			t.Error("wrong error type")
		} else if !fe.Sending() {
			t.Error("wrong error 'sending'")
		}
	})
	t.Run("size limit", func(t *testing.T) {
		r := NewRegistry(1)
		p := NewPrefix("ANYM")
		r.Register(p, Any{})
		defer shouldPanic(t)
		r.Enocde(Any{"thing"})
	})
	t.Run("head", func(t *testing.T) {
		r := NewRegistry(0)
		p := NewPrefix("ANYM")
		r.Register(p, Any{})
		b, err := r.Enocde(Any{"thing"})
		if err != nil {
			t.Fatal(err)
		}
		if len(b) < PrefixLength+4 {
			t.Fatal("invalid length")
		}
		if p.String() != string(b[:PrefixLength]) {
			t.Error("wrong data prefix")
		}
		var ln uint32
		err = encoder.DeserializeRaw(b[PrefixLength:PrefixLength+4], &ln)
		if err != nil {
			t.Fatal(err)
		}
		if int(ln) != len(b)-PrefixLength-4 {
			t.Error("wrong message body length")
		}
	})
}

func TestRegistry_DecodeHead(t *testing.T) {
	t.Run("invalid length", func(t *testing.T) {
		r := NewRegistry(0)
		_, _, err := r.DecodeHead(make([]byte, PrefixLength+4+1))
		if err == nil {
			t.Error("missing error")
		} else if err != ErrInvalidHeadLength {
			t.Error("unexpected error:", err)
		}
	})
	// // for 32-bit CPUs only
	// t.Run("negative length", func(t *testing.T) {
	// 	r := NewRegistry(0)
	// 	var l int32 = -2
	// 	ln := uint32(l)
	// 	p := NewPrefix("asdf")
	// 	head := append(p[:], encoder.SerializeAtomic(ln)...)
	// 	_, _, err := r.DecodeHead(head)
	// 	if err == nil {
	// 		t.Error("missing error")
	// 	} else if err != ErrNegativeLength {
	// 		t.Error("unexpected error:", err)
	// 	}
	// })
	t.Run("size limit", func(t *testing.T) {
		r := NewRegistry(1)
		p := NewPrefix("asdf")
		head := append(p[:], encoder.SerializeAtomic(uint32(5))...)
		_, _, err := r.DecodeHead(head)
		if err == nil {
			t.Error("missing error")
		} else if err != ErrSizeLimit {
			t.Error("unexpected error:", err)
		}
	})
	t.Run("valid", func(t *testing.T) {
		r := NewRegistry(0)
		p := NewPrefix("asdf")
		head := append(p[:], encoder.SerializeAtomic(uint32(5))...)
		px, lx, err := r.DecodeHead(head)
		if err != nil {
			t.Error(err)
		} else if lx != 5 {
			t.Error("unexpected length:", lx)
		} else if px != p {
			t.Error("unexpected prefix:", px.String())
		}
	})
}

func TestRegistry_Decode(t *testing.T) {
	// TODO: mid. priority
}

func TestRegistry_DecodeValue(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		data := encoder.Serialize(Any{"asdf"})
		r := NewRegistry(0)
		typ := reflect.Indirect(reflect.ValueOf(Any{})).Type()
		val := reflect.New(typ)
		if err := r.DecodeValue(data, val); err != nil {
			t.Fatal(err)
		}
		a := val.Interface().(*Any)
		if a.String != "asdf" {
			t.Error("wrong value")
		}
	})
	t.Run("incomplite", func(t *testing.T) {
		data := encoder.Serialize(Any{"asdf"})
		data = append(data, data...)
		r := NewRegistry(0)
		typ := reflect.Indirect(reflect.ValueOf(Any{})).Type()
		val := reflect.New(typ)
		if err := r.DecodeValue(data, val); err == nil {
			t.Error("missing error")
		} else if err != ErrIncompliteDecoding {
			t.Error("unexpected error:", err)
		}
	})
}
