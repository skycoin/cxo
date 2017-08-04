package node

import (
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node/gnet"
)

//
// TODO: DRY (by refactoring may be, or moving outside to separate package)
//

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
	_ Msg = &RejectSubscriptionMsg{}

	// public server features

	_ Msg = &RequestListOfFeedsMsg{}
	_ Msg = &ListOfFeedsMsg{}
	_ Msg = &NonPublicServerMsg{}

	// root, registry, data and requests

	_ Msg = &RootMsg{}
	_ Msg = &RequestDataMsg{}
	_ Msg = &DataMsg{}
)

//
// msgSource
//

// a msgSource represents source of messages,
// it creates messages with unique id per session
type msgSource struct {
	lastMsgID uint32
}

func (m *msgSource) getID() uint32 {
	return atomic.AddUint32(&m.lastMsgID, 1)
}

func (m *msgSource) NewPingMsg() (msg *PingMsg) {
	msg = new(PingMsg)
	return
}

func (m *msgSource) NewPongMsg() (msg *PongMsg) {
	msg = new(PongMsg)
	return
}

func (m *msgSource) NewSubscribeMsg(feed cipher.PubKey) (msg *SubscribeMsg) {
	msg = &SubscribeMsg{Feed: feed}
	msg.Identifier = m.getID()
	return
}

func (m *msgSource) NewUnsubscribeMsg(
	feed cipher.PubKey) (msg *UnsubscribeMsg) {

	msg = &UnsubscribeMsg{Feed: feed}
	return
}

func (m *msgSource) NewAcceptSubscriptionMsg(responseID uint32,
	feed cipher.PubKey) (msg *AcceptSubscriptionMsg) {

	msg = &AcceptSubscriptionMsg{Feed: feed}
	msg.ResponseForID = responseID
	return
}

func (m *msgSource) NewRejectSubscriptionMsg(responseID uint32,
	feed cipher.PubKey) (msg *RejectSubscriptionMsg) {

	msg = &RejectSubscriptionMsg{Feed: feed}
	msg.ResponseForID = responseID
	return
}

func (m *msgSource) NewRootMsg(feed cipher.PubKey,
	rp data.RootPack) (msg *RootMsg) {

	msg = &RootMsg{Feed: feed, RootPack: rp}
	return
}

func (m *msgSource) NewRequestDataMsg(
	ref cipher.SHA256) (msg *RequestDataMsg) {

	msg = &RequestDataMsg{Ref: ref}
	return
}

func (m *msgSource) NewDataMsg(data []byte) (msg *DataMsg) {
	msg = &DataMsg{Data: data}
	return
}

func (m *msgSource) NewRequestListOfFeedsMsg() (msg *RequestListOfFeedsMsg) {
	msg = new(RequestListOfFeedsMsg)
	msg.Identifier = m.getID()
	return
}

func (m *msgSource) NewListOfFeedsMsg(responseID uint32,
	list []cipher.PubKey) (msg *ListOfFeedsMsg) {

	msg = new(ListOfFeedsMsg)
	msg.ResponseForID = responseID
	msg.List = list
	return
}

func (m *msgSource) NewNonPublicServerMsg(
	responseID uint32) (msg *NonPublicServerMsg) {

	msg = new(NonPublicServerMsg)
	msg.ResponseForID = responseID
	return
}

//
// Node developer usability and code readability methods
//

func (s *Node) sendPingMsg(c *gnet.Conn) bool {
	return s.sendMessage(c, s.src.NewPingMsg())
}

func (s *Node) sendPongMsg(c *gnet.Conn) bool {
	return s.sendMessage(c, s.src.NewPongMsg())
}

func (s *Node) sendSubscribeMsg(c *gnet.Conn, feed cipher.PubKey) bool {
	return s.sendMessage(c, s.src.NewSubscribeMsg(feed))
}

func (s *Node) sendUnsubscribeMsg(c *gnet.Conn, feed cipher.PubKey) bool {
	return s.sendMessage(c, s.src.NewUnsubscribeMsg(feed))
}

func (s *Node) sendAcceptSubscriptionMsg(c *gnet.Conn, responseID uint32,
	feed cipher.PubKey) bool {

	return s.sendMessage(c, s.src.NewAcceptSubscriptionMsg(responseID, feed))
}

func (s *Node) sendRejectSubscriptionMsg(c *gnet.Conn, responseID uint32,
	feed cipher.PubKey) bool {

	return s.sendMessage(c, s.src.NewRejectSubscriptionMsg(responseID, feed))
}

func (s *Node) sendRootMsg(c *gnet.Conn, feed cipher.PubKey,
	rp data.RootPack) bool {

	return s.sendMessage(c, s.src.NewRootMsg(feed, rp))
}

func (s *Node) sendRequestDataMsg(c *gnet.Conn, ref cipher.SHA256) bool {
	return s.sendMessage(c, s.src.NewRequestDataMsg(ref))
}

func (s *Node) sendDataMsg(c *gnet.Conn, data []byte) bool {
	return s.sendMessage(c, s.src.NewDataMsg(data))
}

func (s *Node) sendRequestListOfFeedsMsg(c *gnet.Conn) bool {
	return s.sendMessage(c, s.src.NewRequestListOfFeedsMsg())
}

func (s *Node) sendListOfFeedsMsg(c *gnet.Conn, responseID uint32,
	list []cipher.PubKey) bool {

	return s.sendMessage(c, s.src.NewListOfFeedsMsg(responseID, list))
}

func (s *Node) sendNonPublicServerMsg(c *gnet.Conn, responseID uint32) bool {
	return s.sendMessage(c, s.src.NewNonPublicServerMsg(responseID))
}

//
// Msg interface, MsgCore and messages
//

// A Msg is common interface for CXO messages
type Msg interface {
	ID() uint32          // id of the message
	ResponseFor() uint32 // the message is response for
	MsgType() MsgType    // type of the message to encode
}

// IdentifiedMsg is embeddable type. This typ should be a
// part of a *Msg that has ID
type IdentifiedMsg struct {
	Identifier uint32 // identifier in person
}

// ID implements ID method of Msg interface
func (i *IdentifiedMsg) ID() uint32 { return i.Identifier }

// ResponseFor implements ResponseFor method of Msg interface
func (*IdentifiedMsg) ResponseFor() uint32 { return 0 }

// A ResponsedMsg represents an embeddabel struct that
// should be embedded to messages that has ResponseFor
type ResponsedMsg struct {
	ResponseForID uint32 // ID of message the message response for
}

// ID implements ID method of Msg interface
func (r *ResponsedMsg) ID() uint32 { return 0 }

// ResponseFor implements ResponseFor method of Msg interface
func (r *ResponsedMsg) ResponseFor() uint32 { return r.ResponseForID }

type msgCoreStub struct{}

func (msgCoreStub) ID() uint32          { return 0 }
func (msgCoreStub) ResponseFor() uint32 { return 0 }

// A PingMsg is service message and used
// by server to ping clients
type PingMsg struct {
	msgCoreStub
}

// MsgType implements Msg interface
func (*PingMsg) MsgType() MsgType { return PingMsgType }

// PongMsg is service message and used
// by client to reply for PingMsg
type PongMsg struct {
	msgCoreStub
}

// MsgType implements Msg interface
func (*PongMsg) MsgType() MsgType { return PongMsgType }

// A SubscribeMsg ...
type SubscribeMsg struct {
	IdentifiedMsg

	Feed cipher.PubKey
}

// MsgType implements Msg interface
func (*SubscribeMsg) MsgType() MsgType { return SubscribeMsgType }

// An UnsubscribeMsg ...
type UnsubscribeMsg struct {
	msgCoreStub

	Feed cipher.PubKey
}

// MsgType implements Msg interface
func (*UnsubscribeMsg) MsgType() MsgType { return UnsubscribeMsgType }

// An AcceptSubscriptionMsg ...
type AcceptSubscriptionMsg struct {
	ResponsedMsg

	Feed cipher.PubKey
}

// MsgType implements Msg interface
func (*AcceptSubscriptionMsg) MsgType() MsgType {
	return AcceptSubscriptionMsgType
}

// A RejectSubscriptionMsg ...
type RejectSubscriptionMsg struct {
	ResponsedMsg

	Feed cipher.PubKey
}

// MsgType implements Msg interface
func (*RejectSubscriptionMsg) MsgType() MsgType {
	return RejectSubscriptionMsgType
}

// A RequestListOfFeedsMsg is transport message to obtain list of
// feeds from a public server
type RequestListOfFeedsMsg struct {
	IdentifiedMsg
}

// MsgType implements Msg interface
func (*RequestListOfFeedsMsg) MsgType() MsgType {
	return RequestListOfFeedsMsgType
}

// A ListOfFeedsMsg is reply for ListOfFeedMsg
type ListOfFeedsMsg struct {
	ResponsedMsg

	List []cipher.PubKey
}

// MsgType implements Msg interface
func (*ListOfFeedsMsg) MsgType() MsgType { return ListOfFeedsMsgType }

// A NonPublicServerMsg is reply for ListOfFeedMsg while requested
// server is not public
type NonPublicServerMsg struct {
	ResponsedMsg
}

// MsgType implements Msg interface
func (*NonPublicServerMsg) MsgType() MsgType { return NonPublicServerMsgType }

// A RootMsg sent from one node to another one
// to update root object of feed described in
// Feed filed of the message
type RootMsg struct {
	msgCoreStub

	Feed     cipher.PubKey
	RootPack data.RootPack
}

// MsgType implements Msg interface
func (*RootMsg) MsgType() MsgType { return RootMsgType }

// A RequestDataMsg represents a Msg that request a data by hash
type RequestDataMsg struct {
	msgCoreStub

	Ref cipher.SHA256
}

// MsgType implements Msg interface
func (*RequestDataMsg) MsgType() MsgType { return RequestDataMsgType }

// A DataMsg reperesents a data
type DataMsg struct {
	msgCoreStub

	Data []byte
}

// MsgType implements Msg interface
func (*DataMsg) MsgType() MsgType { return DataMsgType }

//
// MsgType / Encode / Deocode / String()
//

// A MsgType represent msg prefix
type MsgType uint8

// MsgTypes
const (
	PingMsgType MsgType = 1 + iota // PingMsg 1
	PongMsgType                    // PongMsg 2

	SubscribeMsgType          // SubscribeMsg           3
	UnsubscribeMsgType        // UnsubscribeMsg         4
	AcceptSubscriptionMsgType // AcceptSubscriptionMsg  5
	RejectSubscriptionMsgType // RejectSubscriptionMsg    6

	RequestListOfFeedsMsgType // RequestListOfFeedsMsg  7
	ListOfFeedsMsgType        // ListOfFeedsMsg         8
	NonPublicServerMsgType    // NonPublicServerMsg     9

	RootMsgType        // RootMsg            10
	RequestDataMsgType // RequestDataMsg     11
	DataMsgType        // DataMsg            12
)

// MsgType to string mapping
var msgTypeString = [...]string{
	PingMsgType: "Ping",
	PongMsgType: "Pong",

	SubscribeMsgType:          "Subscribe",
	UnsubscribeMsgType:        "Unsubscribe",
	AcceptSubscriptionMsgType: "AcceptSubscription",
	RejectSubscriptionMsgType: "RejectSubscription",

	RequestListOfFeedsMsgType: "RequestListOfFeeds",
	ListOfFeedsMsgType:        "ListOfFeeds",
	NonPublicServerMsgType:    "NonPublicServer",

	RootMsgType:        "Root",
	RequestDataMsgType: "RequestData",
	DataMsgType:        "Data",
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
	RejectSubscriptionMsgType: reflect.TypeOf(RejectSubscriptionMsg{}),

	RequestListOfFeedsMsgType: reflect.TypeOf(RequestListOfFeedsMsg{}),
	ListOfFeedsMsgType:        reflect.TypeOf(ListOfFeedsMsg{}),
	NonPublicServerMsgType:    reflect.TypeOf(NonPublicServerMsg{}),

	RootMsgType:        reflect.TypeOf(RootMsg{}),
	RequestDataMsgType: reflect.TypeOf(RequestDataMsg{}),
	DataMsgType:        reflect.TypeOf(DataMsg{}),
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
