package concept

import (
	"bytes"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func shouldNotPanic(t *testing.T) {
	if err := recover(); err != nil {
		t.Error("unexpected panic:", err)
	}
}

type Ref cipher.SHA256

func (r *Ref) S1() string {
	return cipher.SHA256(*r).Hex()
}

func (r Ref) S2() string {
	return cipher.SHA256(r).Hex()
}

var gString string

func BenchmarkRef_String(b *testing.B) {
	val := []byte("something")
	ref := Ref(cipher.SumSHA256(val))

	b.Run("deref", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			gString = ref.S1()
		}
		b.ReportAllocs()
	})

	b.Run("stack", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			gString = ref.S2()
		}
		b.ReportAllocs()
	})
}

type ref cipher.SHA256

type RefCont struct {
	Ref   ref
	Value string
}

func TestRef_lowercaseType(t *testing.T) {
	val := []byte("somthing")
	rc := RefCont{
		Ref:   ref(cipher.SumSHA256(val)),
		Value: "yo-ho-ho",
	}
	data := encoder.Serialize(rc)
	var dec RefCont
	if err := encoder.DeserializeRaw(data, &dec); err != nil {
		t.Error(err)
	}
	if dec.Ref != rc.Ref {
		t.Error("can't encode/decode")
	} else if dec.Ref == (ref{}) {
		t.Error("so, it's very confusing me")
	}
}

type embedRef struct {
	Ref
	val interface{} `enc:"-"`
}

func TestRef_lowercaseEmbeddedField(t *testing.T) {
	val := []byte("somthing")
	er := embedRef{
		Ref: Ref(cipher.SumSHA256(val)),
		val: "yo-ho-ho",
	}
	data := encoder.Serialize(er)
	var dec embedRef
	if err := encoder.DeserializeRaw(data, &dec); err != nil {
		t.Error(err)
	}
	if dec.Ref != er.Ref {
		t.Error("can't encode/decode")
	} else if dec.Ref == (Ref{}) {
		t.Error("so, it's very confusing me")
	}
	if bytes.Compare(data, encoder.Serialize(er.Ref)) != 0 {
		t.Error("can't embed")
	}
}

var gMap = map[cipher.SHA256][]byte{}

func Becnchmark_celarMap(b *testing.B) {

	var bm map[cipher.SHA256][]byte

	for _, s := range []string{
		"hey",
		"ho",
		"lalaley",
		"x",
		"ho-ho-ho",
		"and bottle of rum",
		"seven",
		"degree",
	} {
		val := []byte(s)
		bm[cipher.SumSHA256(val)] = val
	}

	var copyMap = func(s, d map[cipher.SHA256][]byte, b *testing.B) {
		b.StopTimer()
		defer b.StartTimer()

		for k, v := range s {
			d[k] = v
		}
	}

	b.Run("make", func(b *testing.B) {

		for i := 0; i < b.N; i++ {

			copyMap(bm, gMap, b)
			gMap = make(map[cipher.SHA256][]byte) // celar

		}

		b.ReportAllocs()

	})

	b.Run("delete", func(b *testing.B) {

		for i := 0; i < b.N; i++ {

			copyMap(bm, gMap, b)
			for k := range gMap {
				delete(gMap, k)
			}

		}

		b.ReportAllocs()

	})

}

type Reference struct {
	Hash cipher.SHA256 // hash of the reference

	value   interface{} `enc:"-"` // underlying value (set or unpacked)
	changed bool        `enc:"-"` // has been changed if true

	upper interface{} `enc:"-"` // reference to upper node (todo)

	sch interface{} `enc:"-"` // schema of the reference (can be nil)
	pu  interface{} `enc:"-"` // pack/unpack
}

type Dynamic struct {
	Object Reference
	Schema cipher.SHA256
}

func Test_sizes(t *testing.T) {
	ref := Reference{}
	dr := Dynamic{}

	rd := encoder.Serialize(ref)
	dd := encoder.Serialize(dr)

	if len(rd) != len(cipher.SHA256{}) {
		t.Error("wrong ref length")
	}

	if len(dd) != 2*len(cipher.SHA256{}) {
		t.Error("wrong dyn. length")
	}
}
