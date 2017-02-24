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

func (m *msgContext) Broadcast(msg interface{}) error {
	return m.n.Incoming().Broadcast(msg)
}

// IncomingContext is feed-side context
type IncomingContext interface {
	MsgContext
	// Breadcast sends given msg to all subscribers of the feed.
	// It can return encoding error
	Broadcast(msg interface{}) (err error)
}

// OutgoingContext is subscriber side context
type OutgoingContext interface {
	MsgContext
}

// IncomingHandler represents interface that
// messages that received by incoming connections (by feed)
// must implement
type IncomingHandler interface {
	// HandleIncoming is called when message received. If the method
	// returns non-nil error then connection will terminated with
	// the error
	HandleIncoming(IncomingContext) (terminate error)
}

// IncomingHandler represents interface that
// messages that received by outgoing connections (by subscriber)
// must implement
type OutgoingHndler interface {
	// HandleOutgoing is called when message received. If the method
	// returns non-nil error then connection will terminated with
	// the error
	HandleOutgoing(OutgoingContext) (terminate error)
}

// A Msg represents any piece of data
type Msg struct {
	Body []byte
}

func (m *Msg) Handle(ctx *gnet.MessageContext, ni interface{}) (err error) {
	var (
		n  Node             = ni.(Node)
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
		if nd := n.(*node); nd.testHook != nil {
			nd.testHook(nd, testReceivedIncoming)
		}
		return inh.HandleIncoming(&msgContext{
			r:  rpk,
			gc: gc,
			n:  n,
		})
	} // else
	var ogh OutgoingHndler
	if ogh, ok = msg.(OutgoingHndler); !ok {
		err = ErrOutgoingMessageInterface
		return
	}
	// for tests
	if nd := n.(*node); nd.testHook != nil {
		nd.testHook(nd, testReceivedOutgoign)
	}
	return ogh.HandleOutgoing(&msgContext{
		r:  rpk,
		gc: gc,
		n:  n,
	})
}
