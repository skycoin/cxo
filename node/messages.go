package node

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

// be sure that all messages implements Msg interface compiler time
var (
	// ping-, pongs

	_ Msg = &PingMsg{}
	_ Msg = &PongMsg{}

	// subscriptions

	_ Msg = &SubscribeMsg{}
	_ Msg = &UnsubscribeMsg{}

	// subscriptions reply

	_ Msg = &AcceptSubscriptionMsg{}
	_ Msg = &DenySubscriptionMsg{}

	// root, registry, data and requests

	_ Msg = &RootMsg{}
	_ Msg = &RequestDataMsg{}
	_ Msg = &DataMsg{}
	_ Msg = &RequestRegistryMsg{}
	_ Msg = &RegistryMsg{}
)

// a msgSource represents source of messages
type msgSource struct {
	lastMsgId uint32
}

// A Msg is common interface for CXO messages
type Msg interface {
	Id() uint32
	MsgType() MsgType // type of the message to encode
}

// MsgCore keeps msg id
type MsgCore struct {
	Identifier uint32
}

// Id returns id of a message
func (m *MsgCore) Id() uint32 { return m.Identifier }

// A PingMsg is service message and used
// by server to ping clients
type PingMsg struct {
	MsgCore
}

// MsgType implements Msg interface
func (*PingMsg) MsgType() MsgType { return PingMsgType }

// PongMsg is service message and used
// by client to reply for PingMsg
type PongMsg struct {
	MsgCore
}

// MsgType implements Msg interface
func (*PongMsg) MsgType() MsgType { return PongMsgType }

// A SubscribeMsg ...
type SubscribeMsg struct {
	MsgCore

	Feed cipher.PubKey
}

// MsgType implements Msg interface
func (*SubscribeMsg) MsgType() MsgType { return SubscribeMsgType }

// An UnsubscribeMsg ...
type UnsubscribeMsg struct {
	MsgCore

	Feed cipher.PubKey
}

// MsgType implements Msg interface
func (*UnsubscribeMsg) MsgType() MsgType { return UnsubscribeMsgType }

// An AcceptSubscriptionMsg ...
type AcceptSubscriptionMsg struct {
	MsgCore

	Feed cipher.PubKey
}

// MsgType implements Msg interface
func (*AcceptSubscriptionMsg) MsgType() MsgType {
	return AcceptSubscriptionMsgType
}

// A DenySubscriptionMsg ...
type DenySubscriptionMsg struct {
	MsgCore

	Feed cipher.PubKey
}

// MsgType implements Msg interface
func (*DenySubscriptionMsg) MsgType() MsgType { return DenySubscriptionMsgType }

// A RootMsg sent from one node to another one
// to update root object of feed described in
// Feed filed of the message
type RootMsg struct {
	MsgCore

	Feed     cipher.PubKey
	RootPack data.RootPack
}

// MsgType implements Msg interface
func (*RootMsg) MsgType() MsgType { return RootMsgType }

type RequestDataMsg struct {
	MsgCore

	Ref skyobject.Reference
}

// MsgType implements Msg interface
func (*RequestDataMsg) MsgType() MsgType { return RequestDataMsgType }

// A DataMsg reperesents a data
type DataMsg struct {
	MsgCore

	Data []byte
}

// MsgType implements Msg interface
func (*DataMsg) MsgType() MsgType { return DataMsgType }

type RequestRegistryMsg struct {
	MsgCore

	Ref skyobject.RegistryReference // registry reference
}

// MsgType implements Msg interface
func (*RequestRegistryMsg) MsgType() MsgType { return RequestRegistryMsgType }

type RegistryMsg struct {
	MsgCore

	Reg []byte
}

// MsgType implements Msg interface
func (*RegistryMsg) MsgType() MsgType { return RegistryMsgType }

// A MsgType represent msg prefix
type MsgType uint8

const (
	PingMsgType MsgType = 1 + iota // PingMsg 1
	PongMsgType                    // PongMsg 2

	SubscribeMsgType          // SubscribeMsg          3
	UnsubscribeMsgType        // UnsubscribeMsg        4
	AcceptSubscriptionMsgType // AcceptSubscriptionMsg 5
	DenySubscriptionMsgType   // DenySubscriptionMsg   6

	RootMsgType            // RootMsg             7
	RequestDataMsgType     // RequestDataMsg      8
	DataMsgType            // DataMsg             9
	RequestRegistryMsgType // RequestRegistryMsg 10
	RegistryMsgType        // RegistryMsg        11
)

// MsgType to string mapping
var msgTypeString = [...]string{
	PingMsgType: "Ping",
	PongMsgType: "Pong",

	SubscribeMsgType:          "Subscribe",
	UnsubscribeMsgType:        "Unsubscribe",
	AcceptSubscriptionMsgType: "AcceptSubscription",
	DenySubscriptionMsgType:   "DenySubscription",

	RootMsgType:            "Root",
	RequestDataMsgType:     "RequestData",
	DataMsgType:            "Data",
	RequestRegistryMsgType: "RequestRegistry",
	RegistryMsgType:        "Registry",
}

// String implements fmt.Stringer interface
func (m MsgType) String() string {
	if im := int(m); im > 0 && im < len(msgTypeString) {
		return msgTypeString[im]
	}
	return fmt.Sprintf("MsgType<%d>", m)
}

var forwardRegistry = [...]reflect.Type{
	PingMsgType: reflect.TypeOf(PingMsg{}),
	PongMsgType: reflect.TypeOf(PongMsg{}),

	SubscribeMsgType:          reflect.TypeOf(SubscribeMsg{}),
	UnsubscribeMsgType:        reflect.TypeOf(UnsubscribeMsg{}),
	AcceptSubscriptionMsgType: reflect.TypeOf(AcceptSubscriptionMsg{}),
	DenySubscriptionMsgType:   reflect.TypeOf(DenySubscriptionMsg{}),

	RootMsgType:            reflect.TypeOf(RootMsg{}),
	RequestDataMsgType:     reflect.TypeOf(RequestDataMsg{}),
	DataMsgType:            reflect.TypeOf(DataMsg{}),
	RequestRegistryMsgType: reflect.TypeOf(RequestRegistryMsg{}),
	RegistryMsgType:        reflect.TypeOf(RegistryMsg{}),
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
		[]byte{
			byte(msg.MsgType()),
		},
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
