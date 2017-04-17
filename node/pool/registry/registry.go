// Package registry implemnets prefix->refelct.Type registry.
// The redistry used to register types. Filter them by prefix.
// Encode and decode the types
package registry

import (
	"errors"
	"fmt"
	"reflect"
	"unicode"

	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// prefix related constants
const (
	// PrefixLength is length of a Prefix
	PrefixLength int = 4
	// PrefixSpecial is reserved character, not allowed for
	// using in Prefixes. Beside the rune any space character or
	// an unprintabel characters are not allowed too
	PrefixSpecial rune = '-'

	// MaxSize is max message size allwowed by default
	MaxSize int = 8192
)

// Prefix represents message prefix, that
// is unique identifier of type of a message
type Prefix [PrefixLength]byte

// prefix related errors
var (
	ErrSpecialCharacterInPrefix     = errors.New("special character in prefix")
	ErrSpaceInPrefix                = errors.New("space in prefix")
	ErrUnprintableCharacterInPrefix = errors.New(
		"unprintable character in preix")
)

// Validate returns error if the Prefix is invalid
func (p Prefix) Validate() error {
	var r rune
	for _, r = range p.String() {
		if r == PrefixSpecial {
			return ErrSpecialCharacterInPrefix
		}
		if r == ' ' { // ASCII space
			return ErrSpaceInPrefix
		}
		if !unicode.IsPrint(r) {
			return ErrUnprintableCharacterInPrefix
		}
	}
	return
}

// String implements fmt.Stringer interface
func (p Prefix) String() string {
	return string(p)
}

// PrefixFilterFunc is an allow-filter
type PrefixFilterFunc func(Prefix) bool

// ErrFiltered represents errors that occurs when some
// prefix rejected by a `PrefixFilterFunc`s
type ErrFiltered struct {
	send   bool
	prefix Prefix
}

// Sending returns true if the error occurs during sending a message.
// Otherwise this error emitted by receive-filters
func (e *ErrFiltered) Sending() bool {
	return e.send
}

// Error implements error interface
func (e *ErrFiltered) Error() string {
	if e.send {
		return fmt.Sprintf("[%s] rejected by send-filter", e.prefix.String())
	}
	return fmt.Sprintf("[%s] rejected by receive-filter", e.prefix.String())
}

// set of prefix filters
type filter []PrefixFilterFunc

func (f filter) allow(p Prefix) (allow bool) {
	if len(f) == 0 {
		return true // allow
	}
	for _, filter := range f {
		if filter(p) {
			return true // allow
		}
	}
	return // deny
}

// A Message repreents any type that can be registered, encoded and
// decoded
type Message interface{}

// A Registry represents registry of types
// to send/receive. The Registry is not thread safe.
// I.e. you need to call all Register/AddSendFilter/AddReceiveFilter
// before using the Registry, before share reference to it to other
// goroutines
type Registry struct {
	max int // max size of message allowed

	reg map[Prefix]reflect.Type
	inv map[reflect.Type]Prefix

	sendf filter
	recvf filter
}

func (*Registry) typeOf(i Message) reflect.Type {
	return reflect.Indirect(reflect.ValueOf(i)).Type()
}

// Register type of given value with given prefix. The method
// panics if prefix is malformed or already registered, and
// if the type already registered. The method also panics if
// the type can't be encoded
func (r *Registry) Register(prefix Prefix, i Message) {
	if err := prefix.Validate(); err != nil {
		panic(err)
	}
	if _, ok := r.reg[prefix]; ok {
		panic("register prefix twice")
	}
	typ := r.typeOf(i)
	if _, ok := r.inv[typ]; ok {
		panic("register type twice")
	}
	encoder.Serialize(i) // can panic if the type is invalid for the encoder
	r.reg[prefix] = typ
	r.reg[typ] = prefix
}

// AddSendFilter to filter sending. By default all registered types allowed
// to sending. The filter should returns true for prefixes allowed to send
func (r *Registry) AddSendFilter(filter PrefixFilterFunc) {
	if filter == nil {
		panic("nil filter")
	}
	r.sendf = append(r.sendf, filter)
}

// AddReceiveFilter to filter receiving. By default all registered types allowed
// to receive. The filter should returns true for prefixes allowed to receive
func (r *Registry) AddReceiveFilter(filter PrefixFilterFunc) {
	if filter == nil {
		panic("nil filter")
	}
	r.recvf = append(r.recvf, filter)
}

// AllowSend returns true if all send-filters allows messages with
// given prefix to send
func (r *Registry) AllowSend(p Prefix) (allow bool) {
	allow = r.sendf.allow(p)
	return
}

// AllowReceive returns true if all send-filters allows messages with
// given prefix to be received. The method can be used to break connection
// with unwanted messages skipping reading the whole message
func (r *Registry) AllowReceive(p Prefix) (allow bool) {
	allow = r.recvf.allow(p)
	return
}

// Type returns registered reflect.Type by given prefix and true or
// nil and false if the Prefix was not registered
func (r *Registry) Type(p Prefix) (typ reflect.Type, ok bool) {
	typ, ok = r.reg[p]
	return
}

// Encode given message. The method panics if type of given value
// is not registered. It can return filtering error (see ErrFiltered).
// The method also panics if size of encoded message greater then
// limit provided to NewRegistery. The result contains prefix,
// length, and encoded message. The size limit is size limit of
// the message itself. I.e. max size of the p can be the limit
// + 4 (encoded length) + PrefixLength
func (r *Registry) Enocde(m Message) (p []byte, err error) {
	typ := r.typeOf(m)
	prefix, ok := r.reg[typ]
	if !ok {
		panic("encoding unregistered type: " + typ.String())
	}
	if !r.sendf.allow(prefix) {
		err = &ErrFiltered{true, prefix} // sending
		return
	}
	em := encoder.Serialize(m)
	if r.max > 0 && len(em) > r.max {
		panic(fmt.Errorf(
			"try to send message greater than size limit %T, %d (%d)",
			m, len(em), p.conf.MaxMessageSize,
		))
	}
	p = make([]byte, 0, 4+PrefixLength+len(em))
	p = append(p, prefix[:]...)
	p = append(p, encoder.SerializeAtomic(uint32(len(em)))...)
	p = append(p, em...)

	return
}

// Registry related errors
var (
	ErrNotRegistered      = errors.New("not registered prefix")
	ErrIncompliteDecoding = errors.New("incomlite decoding")
	ErrInvalidHeadLength  = errors.New("invalid head length")
	ErrNegativeLength     = errors.New("negative length")
	ErrSizeLimit          = errors.New("size limit reached")
)

// DecodeHead decodes message head: Prefix + encoded length. Length
// of given head should be PrefixLength + 4 (length of message). The
// method returns error if decoded length of message exceeds size
// limit provided to NewRegistery
func (r *Registry) DecodeHead(head []byte) (p Prefix, l int, err error) {
	if len(head) != PrefixLength+4 {
		err = ErrInvalidHeadLength
		return
	}
	copy(p[:], head)
	var ln uint32
	if err = encoder.DeserializeRaw(head[PrefixLength:], &ln); err != nil {
		return
	}
	l = int(ln)
	if l < 0 {
		err = ErrNegativeLength
	} else if l > r.max {
		err = ErrSizeLimit
	}
	return
}

// Decode body of a message type of which described by given prefix. The
// method can returns decoding or filtering error (see ErrFiltered). The
// method can also returns ErrNotRegistered
func (r *Registry) Decode(prefix Prefix, p []byte) (v Message, err error) {
	typ, ok := r.Type(prefix)
	if !ok {
		err = ErrNotRegistered
		return
	}
	if !r.recvf.allow(prefix) {
		err = &ErrFiltered{false, prefix} // receiving
		return
	}
	val := reflect.New(typ)
	if err = r.DecodeValue(p, val); err != nil {
		return
	}
	v = val.Interface()
	return
}

// DecodeValue skips all receive-filters and uses reflect.Value instead of
// Prefix. The method is short-hand for encoder.DeserializeRawToValue
func (*Registry) DecodeValue(p []byte, val reflect.Value) (err error) {
	var n int
	n, err = encoder.DeserializeRawToValue(p, val)
	if err == nil && n < len(p) {
		err = ErrIncompliteDecoding
	}
	return
}
