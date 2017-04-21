package node

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// be sure that all messages implements Msg interface
var (
	_ Msg = &SyncMsg{}
	_ Msg = &RootMsg{}
	_ Msg = &RequestMsg{}
	_ Msg = &DataMsg{}
)

// A Msg is ommon interface for CXO messages
type Msg interface {
	MsgType() MsgType
}

// A SyncMsg sent from one node to another one to
// notify the remote node about feeds the sender
// interesting in
type SyncMsg struct {
	Feeds []cipher.PubKey
}

// MsgType implements Msg interface
func (*SyncMsg) MsgType() MsgType { return SyncMsgType }

// A RootMsg sent from one node to another one
// to update root object of feed described in
// Feed filed of the message
type RootMsg struct {
	Feed cipher.PubKey
	Sig  cipher.Sig
	Root []byte
}

// MsgType implements Msg interface
func (*RootMsg) MsgType() MsgType { return RootMsgType }

// A RequestMsg represents requesting data
// the node want to get
type RequestMsg struct {
	Hash cipher.SHA256
}

// MsgType implements Msg interface
func (*RequestMsg) MsgType() MsgType { return RequestMsgType }

// A DataMsg reperesents data the node
// request for
type DataMsg struct {
	Data []byte
}

// MsgType implements Msg interface
func (*DataMsg) MsgType() MsgType { return DataMsgType }

// A MsgType represent msg prefix
type MsgType uint8

const (
	SyncMsgType    MsgType = 1 + iota // SyncMsg 1
	RootMsgType                       // RootMsg 2
	RequestMsgType                    // RequestMsg 3
	DataMsgType                       // DataMsg 4
)

// MsgType to string mapping
var msgTypeString = [...]string{
	SyncMsgType:    "SYNC",
	RootMsgType:    "ROOT",
	RequestMsgType: "REQUEST",
	DataMsgType:    "DATA",
}

// String implements fmt.Stringer interface
func (m MsgType) String() string {
	if im := int(m); im > 0 && im < len(msgTypeString) {
		return msgTypeString[im]
	}
	return fmt.Sprintf("MsgType<%d>", m)
}

var forwardRegistry = [...]reflect.Type{
	SyncMsgType:    reflect.TypeOf(SyncMsg{}),
	RootMsgType:    reflect.TypeOf(RootMsg{}),
	RequestMsgType: reflect.TypeOf(RequestMsg{}),
	DataMsgType:    reflect.TypeOf(DataMsg{}),
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
