// Package msg represents node messages
package msg

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// Version is current protocol version
const Version uint16 = 2

// be sure that all messages implements Msg interface compiler time
var (
	// ping-, pongs

	_ Msg = &Ping{} // <- Ping
	_ Msg = &Pong{} // -> Pong

	// handshake

	_ Msg = &Hello{}  // <- Hello  (address, protocol version)
	_ Msg = &Accept{} // -> Accept (address)
	_ Msg = &Reject{} // -> Reject (address, error message)

	// subscriptions

	_ Msg = &Subscribe{}   // <- S
	_ Msg = &Unsubscribe{} // <- U

	// subscriptions reply

	_ Msg = &AcceptSubscription{} // -> AS
	_ Msg = &RejectSubscription{} // -> RS (error)

	// public server features

	_ Msg = &RequestListOfFeeds{} // <- RLoF
	_ Msg = &ListOfFeeds{}        // -> LoF
	_ Msg = &NonPublicServer{}    // -> NPS

	// root, registry, data and requests

	_ Msg = &Root{}     // <- Root
	_ Msg = &RootDone{} // ->

	_ Msg = &RequestObject{} // <- RqO
	_ Msg = &Object{}        // -> O
)

//
// Msg interface, MsgCore and messages
//

// A Msg is common interface for CXO messages
type Msg interface {
	Type() Type // type of the message to encode
}

// A Ping is service message and used
// by server to ping clients
type Ping struct {
	ID ID
}

// Type implements Msg interface
func (*Ping) Type() Type { return PingType }

// Pong is service message and used
// by client to reply for PingMsg
type Pong struct {
	ResponseFor ID
}

// Type implements Msg interface
func (*Pong) Type() Type { return PongType }

// A Hello is handshake initiator message
type Hello struct {
	ID       ID
	Protocol uint16
	Address  cipher.Address // reserved for future
}

// Type implements Msg interface
func (*Hello) Type() Type { return HelloType }

// An Accept is response for the Hello
// if handshake has been accepted
type Accept struct {
	ResponseFor ID
	Address     cipher.Address // reserved for future
}

// Type implements Msg interface
func (*Accept) Type() Type { return AcceptType }

// A Reject is response for the Hello
// if subscription has not been accepted
type Reject struct {
	ResponseFor ID
	Address     cipher.Address // reserved for future
	Err         string         // reason
}

// Type implements Msg interface
func (*Reject) Type() Type { return RejectType }

// A Subscribe ...
type Subscribe struct {
	ID   ID
	Feed cipher.PubKey
}

// Type implements Msg interface
func (*Subscribe) Type() Type { return SubscribeType }

// An Unsubscribe ...
type Unsubscribe struct {
	ID   ID
	Feed cipher.PubKey
}

// Type implements Msg interface
func (*Unsubscribe) Type() Type { return UnsubscribeType }

// An AcceptSubscription ...
type AcceptSubscription struct {
	ResponseFor ID
	Feed        cipher.PubKey
}

// Type implements Msg interface
func (*AcceptSubscription) Type() Type {
	return AcceptSubscriptionType
}

// A RejectSubscription ...
type RejectSubscription struct {
	ResponseFor ID
	Feed        cipher.PubKey
}

// Type implements Msg interface
func (*RejectSubscription) Type() Type {
	return RejectSubscriptionType
}

// A RequestListOfFeeds is transport message to obtain list of
// feeds from a public server
type RequestListOfFeeds struct {
	ID ID
}

// Type implements Msg interface
func (*RequestListOfFeeds) Type() Type {
	return RequestListOfFeedsType
}

// A ListOfFeeds is reply for ListOfFeed
type ListOfFeeds struct {
	ResponseFor ID
	List        []cipher.PubKey
}

// Type implements Msg interface
func (*ListOfFeeds) Type() Type { return ListOfFeedsType }

// A NonPublicServer is reply for ListOfFeed while requested
// server is not public
type NonPublicServer struct {
	ResponseFor ID
}

// Type implements Msg interface
func (*NonPublicServer) Type() Type { return NonPublicServerType }

// A Root sent from one node to another one
// to update root object of feed described in
// Feed field of the message
type Root struct {
	Feed  cipher.PubKey
	Seq   uint64     // for node internals
	Value []byte     // encoded Root in person
	Sig   cipher.Sig // signature
}

// Type implements Msg interface
func (*Root) Type() Type { return RootType }

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
	Feed cipher.PubKey // feed
	Seq  uint64        // seq of the Root
}

// Type implements Msg interface
func (*RootDone) Type() Type { return RootDoneType }

// A RequestObject represents a Msg that request a data by hash
type RequestObject struct {
	Key cipher.SHA256
}

// Type implements Msg interface
func (*RequestObject) Type() Type { return RequestObjectType }

// An Object reperesents encoded object
type Object struct {
	Value []byte // encoded object in person
}

// Type implements Msg interface
func (*Object) Type() Type { return ObjectType }

//
// Type / Encode / Deocode / String()
//

// A Type represent msg prefix
type Type uint8

// Types
const (
	PingType Type = 1 + iota // Ping 1
	PongType                 // Pong 2

	HelloType  // Hello  3
	AcceptType // Accept 4
	RejectType // Reject 5

	SubscribeType          // Subscribe           6
	UnsubscribeType        // Unsubscribe         7
	AcceptSubscriptionType // AcceptSubscription  8
	RejectSubscriptionType // RejectSubscription  9

	RequestListOfFeedsType // RequestListOfFeeds  10
	ListOfFeedsType        // ListOfFeeds         11
	NonPublicServerType    // NonPublicServer     12

	RootType          // Root           10
	RootDoneType      // RootDone       11
	RequestObjectType // RequestObject  12
	ObjectType        // Object         13
)

// Type to string mapping
var msgTypeString = [...]string{
	PingType: "Ping",
	PongType: "Pong",

	HelloType:  "Hello",
	AcceptType: "Accept",
	RejectType: "Reject",

	SubscribeType:          "Subscribe",
	UnsubscribeType:        "Unsubscribe",
	AcceptSubscriptionType: "AcceptSubscription",
	RejectSubscriptionType: "RejectSubscription",

	RequestListOfFeedsType: "RequestListOfFeeds",
	ListOfFeedsType:        "ListOfFeeds",
	NonPublicServerType:    "NonPublicServer",

	RootType:          "Root",
	RootDoneType:      "RootDone",
	RequestObjectType: "RequestObject",
	ObjectType:        "Object",
}

// String implements fmt.Stringer interface
func (m Type) String() string {
	if im := int(m); im > 0 && im < len(msgTypeString) {
		return msgTypeString[im]
	}
	return fmt.Sprintf("Type<%d>", m)
}

var forwardRegistry = [...]reflect.Type{
	PingType: reflect.TypeOf(Ping{}),
	PongType: reflect.TypeOf(Pong{}),

	HelloType:  reflect.TypeOf(Hello{}),
	AcceptType: reflect.TypeOf(Accept{}),
	RejectType: reflect.TypeOf(Reject{}),

	SubscribeType:          reflect.TypeOf(Subscribe{}),
	UnsubscribeType:        reflect.TypeOf(Unsubscribe{}),
	AcceptSubscriptionType: reflect.TypeOf(AcceptSubscription{}),
	RejectSubscriptionType: reflect.TypeOf(RejectSubscription{}),

	RequestListOfFeedsType: reflect.TypeOf(RequestListOfFeeds{}),
	ListOfFeedsType:        reflect.TypeOf(ListOfFeeds{}),
	NonPublicServerType:    reflect.TypeOf(NonPublicServer{}),

	RootType:          reflect.TypeOf(Root{}),
	RootDoneType:      reflect.TypeOf(RootDone{}),
	RequestObjectType: reflect.TypeOf(RequestObject{}),
	ObjectType:        reflect.TypeOf(Object{}),
}

// An ErrInvalidType represents decoding error when
// incoming message is malformed and its type invalid
type ErrInvalidType struct {
	msgType Type
}

// Type return Type which cause the error
func (e ErrInvalidType) Type() Type {
	return e.msgType
}

// Error implements builting error interface
func (e ErrInvalidType) Error() string {
	return fmt.Sprint("invalid message type: ", e.msgType.String())
}

// Encode given message to []byte prefixed by Type
func Encode(msg Msg) (p []byte) {
	p = append(
		[]byte{byte(msg.Type())},
		encoder.Serialize(msg)...)
	return
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

// Decode encoded Type-prefixed data to message.
// It can returns encoding errors or ErrInvalidType
func Decode(p []byte) (msg Msg, err error) {
	if len(p) < 1 {
		err = ErrEmptyMessage
		return
	}
	mt := Type(p[0])
	if mt <= 0 || int(mt) >= len(forwardRegistry) {
		err = ErrInvalidType{mt}
		return
	}
	typ := forwardRegistry[mt]
	val := reflect.New(typ)
	var n int
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
