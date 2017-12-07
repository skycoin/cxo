// Package msg represents node messages
package msg

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// protocol
//
// [1 byte] - type
// [ .... ] - encoded message
//

// Version is current protocol version
const Version uint16 = 3

// be sure that all messages implements Msg interface compiler time
var (

	// handshake

	_ Msg = &Syn{} // <- Syn (protocol version)
	_ Msg = &Ack{} // -> Ack ()

	// common replies

	_ Msg = &Ok{}  // -> Ok ()
	_ Msg = &Err{} // -> Err (error message)

	// subscriptions

	_ Msg = &Sub{}   // <- Sub (feed)
	_ Msg = &Unsub{} // <- Unsub (feed)

	// public server features

	_ Msg = &RqList{} // <- GetList ()
	_ Msg = &List{}   // -> Lsit  (feeds)

	// root (push and done)

	_ Msg = &Root{}     // <- Root (feed, nonce, seq, sig, val)
	_ Msg = &RootDone{} // -> RD   (feed, nonce, seq)

	// obejcts

	_ Msg = &RqObject{} // <- RqO (key, prefetch)
	_ Msg = &Object{}   // -> O   (val, vals)

	// preview

	_ Msg = &RqPreview{} // -> RqPreview (feed)
)

//
// Msg interface, MsgCore and messages
//

// A Msg is common interface for CXO messages
type Msg interface {
	Type() Type     // type of the message to encode
	Encode() []byte // encode the message to []byte prefixed with the Type
}

//
// handshake
//

// A Syn is handshake initiator message
type Syn struct {
	Protocol uint16
	NodeID   cipher.PubKey
}

// Type implements Msg interface
func (*Syn) Type() Type { return SynType }

// Ecnode the Syn
func (s *Syn) Encode() []byte { return encode(s) }

// An Ack is response for the Syn
// if handshake has been accepted
type Ack struct {
	NodeID cipher.PubKey
}

// Type implements Msg interface
func (*Ack) Type() Type { return AckType }

// Ecnode the Ack
func (a *Ack) Encode() []byte { return encode(a) }

//
// common
//

// An Ok is common success reply
type Ok struct{}

// Type implements Msg interface
func (*Ok) Type() Type { return OkType }

// Ecnode the Ok
func (*Ok) Encode() []byte {
	return []byte{
		byte(OkType),
	}
}

// A Err is common error reply
type Err struct {
	Err string // reason
}

// Type implements Msg interface
func (*Err) Type() Type { return ErrType }

// Ecnode the Err
func (e *Err) Encode() []byte { return encode(e) }

//
// subscriptions
//

// A Sub message is request for subscription
type Sub struct {
	Feed cipher.PubKey
}

// Type implements Msg interface
func (*Sub) Type() Type { return SubType }

// Ecnode the Sub
func (s *Sub) Encode() []byte {
	return append(
		[]byte{
			byte(SubType),
		},
		s.Feed[:]...,
	)
}

// An Unsub used to notify remote peer about
// unsubscribing from a feed
type Unsub struct {
	Feed cipher.PubKey
}

// Type implements Msg interface
func (*Unsub) Type() Type { return UnsubType }

// Ecnode the Unsub
func (u *Unsub) Encode() []byte {
	return append(
		[]byte{
			byte(UnsubType),
		},
		u.Feed[:]...,
	)
}

//
// list of feeds
//

// A RqList is request of list of feeds
type RqList struct{}

// Type implements Msg interface
func (*RqList) Type() Type { return RqListType }

// Ecnode the RqList
func (*RqList) Encode() []byte {
	return []byte{
		byte(RqListType),
	}
}

// A List is reply for RqList
type List struct {
	List []cipher.PubKey
}

// Type implements Msg interface
func (*List) Type() Type { return ListType }

// Ecnode the List
func (l *List) Encode() []byte { return encode(l) }

//
// root (push an done)
//

// A Root sent from one node to another one
// to update root object of feed described in
// Feed field of the message
type Root struct {
	Feed  cipher.PubKey // feed }
	Nonce uint64        // head } Root selector
	Seq   uint64        // seq  }

	Value []byte // encoded Root in person

	Sig cipher.Sig // signature
}

// Type implements Msg interface
func (*Root) Type() Type { return RootType }

// Ecnode the Root
func (r *Root) Encode() []byte { return encode(r) }

// A RootDone is response for the Root.
// A node that receives a Root object requires
// some time to fill or drop it. After that
// it sends the RootDone back to notify
// peer that this Root object no longer
// needed. E.g. a node holds a Root before
// sending, to prevent removing. And the
// node have to know when peer fills this
// root or drops it.
type RootDone struct {
	Feed  cipher.PubKey // feed
	Nonce uint64        // head
	Seq   uint64        // seq of the Root
}

// Type implements Msg interface
func (*RootDone) Type() Type { return RootDoneType }

// Ecnode the RootDone
func (r *RootDone) Encode() []byte { return encode(r) }

//
// objects
//

// A RqObject represents a Msg that request a data by hash
type RqObject struct {
	Key      cipher.SHA256   // request
	Prefetch []cipher.SHA256 // prefetch objects
}

// Type implements Msg interface
func (*RqObject) Type() Type { return RqObjectType }

// Ecnode the RqObject
func (r *RqObject) Encode() []byte { return encode(r) }

// An Object reperesents encoded object
type Object struct {
	Value    []byte   // encoded object in person
	Prefetch [][]byte // prefetch obejcts or nil
}

// Type implements Msg interface
func (*Object) Type() Type { return ObjectType }

// Ecnode the Object
func (o *Object) Encode() []byte { return encode(o) }

//
// preview
//

// RqReview is request for feeds preview
type RqPreview struct {
	Feed cipher.PubKey
}

// Type implements Msg interface
func (*RqPreview) Type() Type { return RqPreviewType }

// Ecnode the RqPreview
func (r *RqPreview) Encode() []byte { return encode(r) }

//
// Type / Encode / Deocode / String()
//

// A Type represent msg prefix
type Type uint8

// Types
const (
	SynType = 1 + iota // 1
	AckType            // 2

	OkType  // 3
	ErrType // 4

	SubType   // 5
	UnsubType // 6

	RqListType // 7
	ListType   // 8

	RootType     // 9
	RootDoneType // 10

	RqObjectType // 11
	ObjectType   // 12

	RqPreviewType // 13 (sic!)
)

// Type to string mapping
var msgTypeString = [...]string{
	SynType: "Syn",
	AckType: "Ack",

	OkType:  "Ok",
	ErrType: "Err",

	SubType:   "Sub",
	UnsubType: "Unsub",

	RqListType: "RqList",
	ListType:   "List",

	RootType:     "Root",
	RootDoneType: "RootDone",

	RqObjectType: "RqObject",
	ObjectType:   "Object",

	RqPreviewType: "RqPreview",
}

// String implements fmt.Stringer interface
func (m Type) String() string {
	if im := int(m); im > 0 && im < len(msgTypeString) {
		return msgTypeString[im]
	}
	return fmt.Sprintf("Type<%d>", m)
}

var forwardRegistry = [...]reflect.Type{
	SynType: reflect.TypeOf(Syn{}),
	AckType: reflect.TypeOf(Ack{}),

	OkType:  reflect.TypeOf(Ok{}),
	ErrType: reflect.TypeOf(Err{}),

	SubType:   reflect.TypeOf(Sub{}),
	UnsubType: reflect.TypeOf(Unsub{}),

	RqListType: reflect.TypeOf(RqList{}),
	ListType:   reflect.TypeOf(List{}),

	RootType:     reflect.TypeOf(Root{}),
	RootDoneType: reflect.TypeOf(RootDone{}),

	RqObjectType: reflect.TypeOf(RqObject{}),
	ObjectType:   reflect.TypeOf(Object{}),

	RqPreviewType: reflect.TypeOf(RqPreview{}),
}

// An InvalidTypeError represents decoding error when
// incoming message is malformed and its type invalid
type InvalidTypeError struct {
	typ Type
}

// Type return Type which cause the error
func (e InvalidTypeError) Type() Type {
	return e.typ
}

// Error implements builting error interface
func (e InvalidTypeError) Error() string {
	return fmt.Sprint("invalid message type: ", e.typ.String())
}

var (
	// ErrEmptyMessage occurs when you
	// try to Decode an empty slice
	ErrEmptyMessage = errors.New("empty message")
	// ErrIncomplieDecoding occurs when incoming message
	// decoded correctly but the decoding doesn't use
	// entire encoded message
	ErrIncomplieDecoding = errors.New("incomplete decoding")
)

// encode given message to []byte prefixed by Type
func encode(msg Msg) (p []byte) {
	p = append(
		[]byte{
			byte(msg.Type()),
		},
		encoder.Serialize(msg)...,
	)
	return
}

// Decode encoded Type-prefixed data to message.
// It can returns encoding errors or InvalidTypeError
func Decode(p []byte) (msg Msg, err error) {

	if len(p) < 1 {
		err = ErrEmptyMessage
		return
	}

	var mt = Type(p[0])

	if mt <= 0 || int(mt) >= len(forwardRegistry) {
		err = InvalidTypeError{mt}
		return
	}

	var (
		typ = forwardRegistry[mt]
		val = reflect.New(typ)

		n int
	)

	if n, err = encoder.DeserializeRawToValue(p[1:], val); err != nil {
		return
	}

	if n+1 != len(p) {
		err = ErrIncomplieDecoding
		return
	}

	msg = val.Interface().(Msg)
	return
}
