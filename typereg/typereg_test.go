package typereg

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

// example type
type X struct {
	Int    int64
	String string
}

func TestNewTypereg(t *testing.T) {
	var e Typereg
	e = NewTypereg()
	if e == nil {
		t.Fatal("NewTypereg returns nil")
	}
}

func handleUnexpectedPanic(t *testing.T) {
	if e := recover(); e != nil {
		t.Error("unexpected panic: ", e)
	}
}

func TestTypereg_Register(t *testing.T) {
	// (1) register and inspect underlying maps
	t.Run("register X", func(t *testing.T) {
		var (
			e Typereg = NewTypereg()
			u *enc
		)
		// handle possible panics
		defer handleUnexpectedPanic(t)
		e.Register(X{})
		u = e.(*enc)
		if len(u.types) != 1 {
			t.Fatal("wrong types registry length: want 1, got ", len(u.types))
		}
		if len(u.back) != 1 {
			t.Fatal("wrong back registry length: want 1, got ", len(u.types))
		}
		// u.types
		for tt, typ := range u.types {
			if tt == (Type{}) {
				t.Error("empty Type key in types registry")
			}
			if typ != reflect.TypeOf(X{}) {
				t.Error("wrong reflect.Type in types registry: ", typ)
			}
		}
		// u.back
		for typ, tt := range u.back {
			if tt == (Type{}) {
				t.Error("empty Type key in back registry")
			}
			if typ != reflect.TypeOf(X{}) {
				t.Error("wrong reflect.Type in back registry: ", typ)
			}
		}
	})
	// (2) panic if value is nil
	t.Run("nil-value panic", func(t *testing.T) {
		var e Typereg = NewTypereg()
		defer func() {
			if e := recover(); e == nil {
				t.Error("missing Register(nil) panic")
			} else if err, ok := e.(error); ok && err == ErrValueIsNil {
				return
			}
			t.Error("Register(nil) panics with wrong value")
		}()
		e.Register(nil)
	})
	// (3) already registered panic
	t.Run("alredy registered panic", func(t *testing.T) {
		var e Typereg = NewTypereg()
		defer func() {
			if e := recover(); e == nil {
				t.Error("missing panic on twice register")
			} else if err, ok := e.(error); ok && err == ErrAlredyRegistered {
				return
			}
			t.Error("Register(nil) panics with wrong value")
		}()
		e.Register(X{})
		e.Register(X{})
	})
	// (4) X{} and &X{}
	t.Run("pointer dereference", func(t *testing.T) {
		var e Typereg = NewTypereg()
		defer func() {
			if e := recover(); e == nil {
				t.Error("treat X{} and &X{} as different types")
			}
		}()
		e.Register(X{})
		e.Register(&X{})
	})
	// (5) synthetic collision panic
	t.Run("synthetic collision", func(t *testing.T) {
		var (
			e Typereg = NewTypereg()
			u *enc
		)
		defer func() {
			if e := recover(); e == nil {
				t.Error("missing collision panic")
			} else if err, ok := e.(error); ok && err == ErrHashCollision {
				return
			}
			t.Error("collision panics with wrong value")
		}()
		e.Register(X{})
		u = e.(*enc)
		// clear back mapping
		for k := range u.back {
			delete(u.back, k)
		}
		e.Register(X{})
	})
}

func TestTypereg_Encode(t *testing.T) {
	t.Run("value is nil err", func(t *testing.T) {
		if _, err := NewTypereg().Encode(nil); err == nil {
			t.Error("missing nil-value encoding error")
		} else if err != ErrValueIsNil {
			t.Errorf("wrong error: wanr ErrValueIsNil, got %q", err)
		}
	})
	t.Run("null-pointer", func(t *testing.T) {
		var x *X
		if _, err := NewTypereg().Encode(x); err == nil {
			t.Error("missing nil-value encoding error")
		} else if err != ErrValueIsNil {
			t.Errorf("wrong error: wanr ErrValueIsNil, got %q", err)
		}
	})
	t.Run("unregistered err", func(t *testing.T) {
		if _, err := NewTypereg().Encode(X{150, "some"}); err == nil {
			t.Error("missing nil-value encoding error")
		} else if err != ErrUnregistered {
			t.Errorf("wrong error: wanr ErrUnregistered, got %q", err.Error())
		}
	})
	t.Run("encode", func(t *testing.T) {
		e := NewTypereg()
		e.Register(X{})
		data, err := e.Encode(X{150, "some"})
		if err != nil {
			t.Errorf("unexpected error: %q", err.Error())
			return
		}
		if data == nil {
			t.Error("result is nil")
		}
		if len(data) <= 4 {
			t.Error("short write")
		}
		tt := e.(*enc).back[reflect.TypeOf(X{})]
		if bytes.Compare(tt[:], data[:4]) != 0 {
			t.Error("wrong type prefix")
		}
	})
	// X{} vs &X{}
	t.Run("encode pointer", func(t *testing.T) {
		var (
			e Typereg = NewTypereg()

			xs X  = X{150, "some"}
			xp *X = &xs
		)
		e.Register(X{})
		ds, err := e.Encode(xs)
		if err != nil {
			t.Errorf("unexpected error: %q", err.Error())
			return
		}
		dp, err := e.Encode(xp)
		if err != nil {
			t.Errorf("unexpected error: %q", err.Error())
			return
		}
		if bytes.Compare(ds, dp) != 0 {
			t.Error("encoding by pointer returns different result")
		}
	})
}

func TestTypereg_Decode(t *testing.T) {
	t.Run("short buffer", func(t *testing.T) {
		if _, err := NewTypereg().Decode(nil); err == nil {
			t.Error("missign ErrShortBuffer")
		} else if err != ErrShortBuffer {
			t.Errorf("wrong err: want ErrShortBuffer, got %q", err.Error())
		}
		if _, err := NewTypereg().Decode([]byte{0, 0, 0}); err == nil {
			t.Error("missign ErrShortBuffer")
		} else if err != ErrShortBuffer {
			t.Errorf("wrong err: want ErrShortBuffer, got %q", err.Error())
		}
	})
	t.Run("unregistered", func(t *testing.T) {
		e := NewTypereg()
		e.Register(X{})
		data, err := e.Encode(X{150, "some"})
		if err != nil {
			t.Error("unexpected error: ", err)
			return
		}
		if _, err = NewTypereg().Decode(data); err == nil {
			t.Error("missing ErrUnregistered")
		} else if err != ErrUnregistered {
			t.Errorf("wrong error: wnat ErrUnregistered, got %q", err.Error())
		}
	})
	t.Run("incomplite decoding", func(t *testing.T) {
		e := NewTypereg()
		e.Register(X{})
		data, err := e.Encode(X{150, "some"})
		if err != nil {
			t.Error("unexpected error: ", err)
			return
		}
		data = append(data, "tail"...)
		if _, err = e.Decode(data); err == nil {
			t.Error("missing ErrIncompliteDecoding")
		} else if err != ErrIncompliteDecoding {
			t.Errorf("wrong error: want ErrIncompliteDecoding, got %q",
				err.Error())
		}
	})
}

func Test_encode_decode(t *testing.T) {
	var (
		e Typereg = NewTypereg()
		x X       = X{150, "some"}

		data []byte
		err  error

		val interface{}
		ok  bool
		z   *X
	)
	e.Register(x)
	if data, err = e.Encode(x); err != nil {
		t.Error("unexpected error: ", err)
		return
	}
	if val, err = e.Decode(data); err != nil {
		t.Error("unexpected error: ", err)
	}
	if z, ok = val.(*X); !ok {
		t.Errorf("unexpected type of decoded value: want %T, got %T", x, val)
		return
	}
	if z.Int != x.Int || z.String != x.String {
		t.Error("encoding->decoding changes data")
	}
}

func Example_simple() {

	type User struct {
		Name string
		Age  int32
	}

	var (
		e Typereg = NewTypereg()
		x User    = User{"Alice", 21}

		data []byte
		err  error

		val interface{}
		ok  bool
		d   *User // decoded value (pointer)
	)

	// register

	e.Register(User{})

	// encode

	if data, err = e.Encode(x); err != nil {
		// handle error
	}

	//
	// send -> receive data
	//

	// decode

	if val, err = e.Decode(data); err != nil {
		// handle error
	}

	if d, ok = val.(*User); !ok {
		// this branch never executes in the example
	}

	fmt.Printf("%s %d", d.Name, d.Age)

	// Output:
	//
	// Alice 21

}
