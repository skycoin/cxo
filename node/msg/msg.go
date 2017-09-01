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
	MsgType() MsgType // type of the message to encode
}

// A Ping is service message and used
// by server to ping clients
type Ping struct {
	ID ID
}

// MsgType implements Msg interface
func (*Ping) MsgType() MsgType { return PingType }

// Pong is service message and used
// by client to reply for PingMsg
type Pong struct {
	ResponseFor ID
}

// MsgType implements Msg interface
func (*Pong) MsgType() MsgType { return PongType }

// A Hello is handhake initiator message
type Hello struct {
	ID       ID
	Protocol uint16
	Address  cipher.Address // reserved for future
}

// MsgType implements Msg interface
func (*Hello) MsgType() MsgType { return HelloType }

// An Accept is response for the Hello
// if handhake has been accepted
type Accept struct {
	ResponseFor ID
	Address     cipher.Address // reserved for future
}

// MsgType implements Msg interface
func (*Accept) MsgType() MsgType { return AcceptType }

// A Reject is response for the Hello
// if subsctioption has not been accepted
type Reject struct {
	ResponseFor ID
	Address     cipher.Address // reserved for future
	Err         string         // reason
}

// MsgType implements Msg interface
func (*Reject) MsgType() MsgType { return RejectType }

// A Subscribe ...
type Subscribe struct {
	ID   ID
	Feed cipher.PubKey
}

// MsgType implements Msg interface
func (*Subscribe) MsgType() MsgType { return SubscribeType }

// An Unsubscribe ...
type Unsubscribe struct {
	ID   ID
	Feed cipher.PubKey
}

// MsgType implements Msg interface
func (*Unsubscribe) MsgType() MsgType { return UnsubscribeType }

// An AcceptSubscription ...
type AcceptSubscription struct {
	ResponseFor ID
	Feed        cipher.PubKey
}

// MsgType implements Msg interface
func (*AcceptSubscription) MsgType() MsgType {
	return AcceptSubscriptionType
}

// A RejectSubscription ...
type RejectSubscription struct {
	ResponseFor ID
	Feed        cipher.PubKey
}

// MsgType implements Msg interface
func (*RejectSubscription) MsgType() MsgType {
	return RejectSubscriptionType
}

// A RequestListOfFeeds is transport message to obtain list of
// feeds from a public server
type RequestListOfFeeds struct {
	ID ID
}

// MsgType implements Msg interface
func (*RequestListOfFeeds) MsgType() MsgType {
	return RequestListOfFeedsType
}

// A ListOfFeeds is reply for ListOfFeed
type ListOfFeeds struct {
	ResponseFor ID
	List        []cipher.PubKey
}

// MsgType implements Msg interface
func (*ListOfFeeds) MsgType() MsgType { return ListOfFeedsType }

// A NonPublicServer is reply for ListOfFeed while requested
// server is not public
type NonPublicServer struct {
	ResponseFor ID
}

// MsgType implements Msg interface
func (*NonPublicServer) MsgType() MsgType { return NonPublicServerType }

// A Root sent from one node to another one
// to update root object of feed described in
// Feed filed of the message
type Root struct {
	Feed  cipher.PubKey
	Value []byte     // encoded Root in person
	Sig   cipher.Sig // signature
}

// MsgType implements Msg interface
func (*Root) MsgType() MsgType { return RootType }

// A RootDone is response for the Root.
// A node that receivs a Root obejct requires
// some tiem to fill or drop it. After that
// it sends the RootDone back to notify
// peer that this Root obejct no longer
// needed. E.g. a node holds a Root before
// sending, to prevent removing. And the
// node have to know when peer fills this
// root or drops it.
type RootDone struct {
	Feed cipher.PubKey // feed
	Seq  uint64        // seq of the Root
}

// MsgType implements Msg interface
func (*RootDone) MsgType() MsgType { return RootDoneType }

// A RequestObject represents a Msg that request a data by hash
type RequestObject struct {
	Key cipher.SHA256
}

// MsgType implements Msg interface
func (*RequestObject) MsgType() MsgType { return RequestObjectType }

// An Object reperesents encoded obejct
type Object struct {
	Value []byte // encoded object in person
}

// MsgType implements Msg interface
func (*Object) MsgType() MsgType { return ObjectType }

//
// MsgType / Encode / Deocode / String()
//

// A MsgType represent msg prefix
type MsgType uint8

// MsgTypes
const (
	PingType MsgType = 1 + iota // Ping 1
	PongType                    // Pong 2

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

// MsgType to string mapping
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
func (m MsgType) String() string {
	if im := int(m); im > 0 && im < len(msgTypeString) {
		return msgTypeString[im]
	}
	return fmt.Sprintf("MsgType<%d>", m)
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

// An ErrInvalidMsgType represents decoding error when
// incoming message is malformed and its type invalid
type ErrInvalidMsgType struct {
	msgType MsgType
}

// MsgType return MsgType which cuse the error
func (e ErrInvalidMsgType) MsgType() MsgType {
	return e.msgType
}

// Error implements builtin error interface
func (e ErrInvalidMsgType) Error() string {
	return fmt.Sprint("invalid message type: ", e.msgType.String())
}

// Encode given message to []byte prefixed by MsgType
func Encode(msg Msg) (p []byte) {
	p = append(
		[]byte{byte(msg.MsgType())},
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
	ErrIncomplieDecoding = errors.New("incomplite decoding")
)

// Decode encoded MsgType-prefixed data to message.
// It can returns encoding errors or ErrInvalidMsgType
func Decode(p []byte) (msg Msg, err error) {
	if len(p) < 1 {
		err = ErrEmptyMessage
		return
	}
	mt := MsgType(p[0])
	if mt <= 0 || int(mt) >= len(forwardRegistry) {
		err = ErrInvalidMsgType{mt}
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
