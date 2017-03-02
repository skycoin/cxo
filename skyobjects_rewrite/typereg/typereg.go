// Package enc introduced to encoding/decoding data
//
// It based on skycoin/skycoin/src/cipher/encoder
// Unfortunately, gnet (skycoin/skycoin/src/daemon/gnet)
// doesn't allow to use its internal encoder from
// outside. Big part of this package is copy-paste
// from gnet (dispatcher.go)
//
// We need encoded data to calcuate hash (SHA512)
// of the data and then use the hash as reference
// to the data
//
// The enc uses FNV-32 hash of (reflect.Type).Name() as
// type name that it stores in registry. Thus, we
// use only 4 byte as type-prefix
package typereg

import (
	"errors"
	"hash"
	"hash/fnv"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// A Typereg implements types registry that is used
// to encode and decode some values to []byte and back
// to go-native types. The skycoin/skycoin/src/cipher/encoder
// is used as underlying encoder
type Typereg interface {
	// Register registers new type to encoed/decode.
	// It panics if type with given name already registered or
	// there is fnv-hash-collision. It panics if given
	// argument is nil. There aren't any sync/mutex/locks,
	// thus we need to use Register before using Encode and
	// Decode. Otherwise correct work is not guaranted
	Register(interface{})
	// Encode encodes given value to []byte.
	// Result will be [type hash 4 byte][encoded value].
	// It never panics
	Encode(interface{}) ([]byte, error)
	// Decode decodes given slice to value.
	// Returned interface{} keeps _pointer_ to decoded struct
	// It never panics
	Decode([]byte) (interface{}, error)
}

// NewTypereg returns new Typereg instance
func NewTypereg() Typereg {
	return &enc{
		hash:  fnv.New32(),
		types: make(map[Type]reflect.Type),
		back:  make(map[reflect.Type]Type),
	}
}

// A Type represents LE-encoder FNV-32 hash of type name
type Type [4]byte

type enc struct {
	hash  hash.Hash32
	types map[Type]reflect.Type
	back  map[reflect.Type]Type
}

// typ returns reflect.Type of given interface
// pointer types converted to non-pointer
func (e *enc) typ(i interface{}) (typ reflect.Type, err error) {
	var val reflect.Value = reflect.ValueOf(i)
	if !val.IsValid() {
		err = ErrValueIsNil
		return
	}
	if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		val = val.Elem()
	}
	if !val.IsValid() {
		err = ErrValueIsNil
		return
	}
	typ = val.Type()
	return
}

// Type creates Type based on given argument
func (e *enc) Type(i interface{}) (t Type) {
	var (
		typ reflect.Type
		u   uint32

		err error
	)
	if typ, err = e.typ(i); err != nil {
		panic(err) // never happens
	}
	// get fvn-32 hash of type name
	e.hash.Reset()
	e.hash.Write([]byte(typ.Name()))
	u = e.hash.Sum32()
	// the hash is type
	t[0] = byte(u)
	t[1] = byte(u >> 8)
	t[2] = byte(u >> 16)
	t[3] = byte(u >> 24)
	return
}

var (
	ErrValueIsNil         = errors.New("value is nil")
	ErrAlredyRegistered   = errors.New("type already registered")
	ErrUnregistered       = errors.New("unregistered type")
	ErrShortBuffer        = errors.New("short buffer")
	ErrIncompliteDecoding = errors.New("data was not completely decoded")
	ErrHashCollision      = errors.New("hash collision")
)

func (e *enc) Register(i interface{}) {
	var (
		typ reflect.Type
		ok  bool
		t   Type

		err error
	)
	if typ, err = e.typ(i); err != nil {
		panic(err)
	}
	// if type already registered
	if _, ok = e.back[typ]; ok {
		panic(ErrAlredyRegistered)
	}
	t = e.Type(i)
	// type is not registered but its Type already
	// exists in e.types (collision)
	if _, ok = e.types[t]; ok {
		panic(ErrHashCollision)
	}
	// everything is ok
	e.types[t] = typ
	e.back[typ] = t
}

func (e *enc) Encode(i interface{}) (d []byte, err error) {
	var (
		typ reflect.Type
		t   Type
		ok  bool
	)
	if typ, err = e.typ(i); err != nil {
		return
	}
	if t, ok = e.back[typ]; !ok {
		err = ErrUnregistered
		return
	}
	// [type 4][msg ...] no length
	d = append(t[:], encoder.Serialize(i)...)
	return
}

func (e *enc) Decode(d []byte) (i interface{}, err error) {
	if len(d) < 4 {
		err = ErrShortBuffer
		return
	}
	var (
		t   Type
		typ reflect.Type
		val reflect.Value

		ok   bool
		used int
	)
	t[0], t[1], t[2], t[3] = d[0], d[1], d[2], d[3]
	if typ, ok = e.types[t]; !ok {
		err = ErrUnregistered
		return
	}
	val = reflect.New(typ)
	if used, err = deserializeMessage(d[4:], val); err != nil {
		return
	}
	if used != len(d)-4 {
		err = ErrIncompliteDecoding
		return
	}
	i = val.Interface()
	return
}

// copy-paste from gnet#dispatcher.go

// Wraps encoder.DeserializeRawToValue and traps panics as an error
func deserializeMessage(msg []byte, v reflect.Value) (n int, e error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case error:
				e = x
			case string:
				e = errors.New(x)
			default:
				e = errors.New("Message deserialization failed")
			}
		}
	}()
	n, e = encoder.DeserializeRawToValue(msg, v)
	return
}
