package node

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/skyobject"
)

// be sure that all messages implements Msg interface
var (
	_ Msg = &AddFeedMsg{}
	_ Msg = &DelFeedMsg{}
	_ Msg = &RootMsg{}
	_ Msg = &RequestDataMsg{}
	_ Msg = &DataMsg{}
	_ Msg = &RequestRegistryMsg{}
	_ Msg = &RegistryMsg{}
)

// A Msg is ommon interface for CXO messages
type Msg interface {
	MsgType() MsgType
}

// A AddFeedMsg sent from one node to another one to
// notify the remote node about new feed the sender
// interesting in
type AddFeedMsg struct {
	Feed cipher.PubKey
}

// MsgType implements Msg interface
func (*AddFeedMsg) MsgType() MsgType { return AddFeedMsgType }

// A DelFeedMsg sent from one node to another one to
// notify the remote node about feed the sender
// doesn't interesting in anymore
type DelFeedMsg struct {
	Feed cipher.PubKey
}

// MsgType implements Msg interface
func (*DelFeedMsg) MsgType() MsgType { return DelFeedMsgType }

// A RootMsg sent from one node to another one
// to update root object of feed described in
// Feed filed of the message
type RootMsg struct {
	Feed     cipher.PubKey
	RootPack skyobject.RootPack
}

// MsgType implements Msg interface
func (*RootMsg) MsgType() MsgType { return RootMsgType }

type RequestDataMsg struct {
	Ref skyobject.Reference
}

// MsgType implements Msg interface
func (*RequestDataMsg) MsgType() MsgType { return RequestDataMsgType }

// A DataMsg reperesents a data
type DataMsg struct {
	Data []byte
}

// MsgType implements Msg interface
func (*DataMsg) MsgType() MsgType { return DataMsgType }

type RequestRegistryMsg struct {
	Ref skyobject.RegistryReference // registry reference
}

// MsgType implements Msg interface
func (*RequestRegistryMsg) MsgType() MsgType { return RequestRegistryMsgType }

type RegistryMsg struct {
	Reg []byte
}

// MsgType implements Msg interface
func (*RegistryMsg) MsgType() MsgType { return RegistryMsgType }

// A MsgType represent msg prefix
type MsgType uint8

const (
	AddFeedMsgType         MsgType = 1 + iota // AddFeedMsg 1
	DelFeedMsgType                            // DelFeedMsg 2
	RootMsgType                               // RootMsg 3
	RequestDataMsgType                        // RequestDataMsg 4
	DataMsgType                               // DataMsg 5
	RequestRegistryMsgType                    // RequestRegistryMsg 6
	RegistryMsgType                           // RegistryMsg 7
)

// MsgType to string mapping
var msgTypeString = [...]string{
	AddFeedMsgType:         "ADD",
	DelFeedMsgType:         "DEL",
	RootMsgType:            "ROOT",
	RequestDataMsgType:     "RQDT",
	DataMsgType:            "DATA",
	RequestRegistryMsgType: "RQREG",
	RegistryMsgType:        "REG",
}

// String implements fmt.Stringer interface
func (m MsgType) String() string {
	if im := int(m); im > 0 && im < len(msgTypeString) {
		return msgTypeString[im]
	}
	return fmt.Sprintf("MsgType<%d>", m)
}

var forwardRegistry = [...]reflect.Type{
	AddFeedMsgType:         reflect.TypeOf(AddFeedMsg{}),
	DelFeedMsgType:         reflect.TypeOf(DelFeedMsg{}),
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
