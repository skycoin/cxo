package node

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"
)

func init() {
	var _ gnet.Message = &Msg{}
	gnet.RegisterMessage(gnet.MessagePrefixFromString("DATA"), Msg{})
	gnet.VerifyMessages()
}

// A ConnectionSide represents connection interface
type ConnectionSide interface {
	PubKey() cipher.PubKey
	Address() string
}

type remoteConnectionSide struct {
	*msgContext
}

func (r remoteConnectionSide) PubKey() cipher.PubKey {
	return r.r
}

func (r remoteConnectionSide) Address() string {
	return r.gc.Addr()
}

type localConnectionSide struct {
	*msgContext
}

func (r localConnectionSide) PubKey() cipher.PubKey {
	return r.n.PubKey()
}

func (r localConnectionSide) Address() string {
	return r.gc.Conn.LocalAddr().String()
}

// MsgContext represents context that message can
// use when it handled
type MsgContext interface {
	Remote() ConnectionSide            // get information about remote node
	Local() ConnectionSide             // get information about local node
	Reply(msg interface{}) (err error) // send response to this message
}

type msgContext struct {
	r  cipher.PubKey // remote public key
	gc *gnet.Connection
	n  Node
}

func (m *msgContext) Remote() ConnectionSide {
	return remoteConnectionSide{m}
}

func (m *msgContext) Local() ConnectionSide {
	return localConnectionSide{m}
}

func (m *msgContext) Reply(msg interface{}) (err error) {
	var (
		d *Msg
	)
	if d, err = m.n.encode(msg); err != nil {
		m.n.Printf("[ERR] error encoding message: %q, %T, %v,",
			err.Error(),
			msg,
			msg)
		return
	}
	m.gc.ConnectionPool.SendMessage(m.gc, d)
	return
}

// IncomingHandler represents interface that
// messages that received by incoming connections (by feed)
// must implement
type IncomingHandler interface {
	// HandleIncoming is called when message received. If the method
	// returns non-nil error then connection will terminated with
	// the error. The user argument is user-data provided to
	// node constructor (NewNode)
	HandleIncoming(cxt MsgContext, user interface{}) (terminate error)
}

// IncomingHandler represents interface that
// messages that received by outgoing connections (by subscriber)
// must implement
type OutgoingHandler interface {
	// HandleOutgoing is called when message received. If the method
	// returns non-nil error then connection will terminated with
	// the error. The user argument is user-data provided to
	// node constructor (NewNode)
	HandleOutgoing(cxt MsgContext, user interface{}) (terminate error)
}

// A Msg represents any piece of data. It used for intrnals.
// The goal of the Msg is wrapper of messages
type Msg struct {
	Body []byte
}

func (m *Msg) Handle(ctx *gnet.MessageContext, ni interface{}) (err error) {
	var (
		n  Node             = ni.(Node)
		nd *node            = n.(*node)
		gc *gnet.Connection = ctx.Conn

		rpk      cipher.PubKey // remote public key
		incoming bool          // connection

		msg interface{}

		ok bool
	)
	if rpk, incoming, err = n.handleMessage(gc); err != nil {
		n.Print("[ERR] got data message, but connection is not established yet")
		return
	}
	// decode body
	if msg, err = n.decode(m.Body); err != nil {
		n.Print("[ERR] can't decode given message: ", err)
		return
	}
	if incoming {
		var inh IncomingHandler
		if inh, ok = msg.(IncomingHandler); !ok {
			err = ErrIncomingMessageInterface
			return
		}
		// for tests
		if nd.testHook != nil {
			nd.testHook(nd, testReceivedIncoming)
		}
		return inh.HandleIncoming(&msgContext{
			r:  rpk,
			gc: gc,
			n:  n,
		}, nd.user)
	} // else
	var ogh OutgoingHandler
	if ogh, ok = msg.(OutgoingHandler); !ok {
		err = ErrOutgoingMessageInterface
		return
	}
	// for tests
	if nd.testHook != nil {
		nd.testHook(nd, testReceivedOutgoign)
	}
	return ogh.HandleOutgoing(&msgContext{
		r:  rpk,
		gc: gc,
		n:  n,
	}, nd.user)
}
