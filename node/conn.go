package node

import (
	"errors"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/msg"
	"github.com/skycoin/cxo/skyobject"
)

// onenctions related errors
var (
	ErrWriteQueueFull              = errors.New("write queue full")
	ErrWrongResponseID             = errors.New("wrong response ID")
	ErrWrongMsgType                = errors.New("wrong message type")
	ErrWrongResponseMsgType        = errors.New("wrong response messag type")
	ErrIncompatibleProtocolVersion = errors.New("incompatible protocol version")
	ErrWrongResponse               = errors.New("wrong response")
)

// A Conn represents conenction of Node
type Conn struct {
	s *Node // back reference for logs

	gc *gnet.Conn // underlying *gnet.Conn

	// not threads safe fields

	// subscriptions of the Conn to resusbcribe OnDial
	subs map[cipher.PubKey]struct{}
	rr   map[msg.ID]*waitResponse // request-response

	events  chan event  // events of the Conn
	timeout chan msg.ID // timeouts

	pings map[msg.ID]chan struct{} //
}

// Address returns remote address
func (c *Conn) Address() string {
	return c.gc.Address()
}

// Node returns related node
func (c *Conn) Node() (n *Node) {
	return c.s
}

// Gnet retursn underlying *gnet.Conn
func (c *Conn) Gnet() *gnet.Conn {
	return c.gc
}

// Close the Conn. It doesn't wait for
// closing returning immediately
func (c *Conn) Close() (err error) {

	c.s.Debugf(ConnPin, "[%s] Close", c.gc.Address())

	err = c.gc.Close()
	return
}

// Send given msg.Msg to peer
func (c *Conn) Send(m msg.Msg) error {
	c.s.Debugf(ConnPin, "[%s] Send %T ", c.gc.Address(), m)

	return c.SendRaw(msg.Encode(m))
}

// SendRaw encoded message to peer
func (c *Conn) SendRaw(rm []byte) (err error) {

	c.s.Debugf(ConnPin, "[%s] SendRaw %d bytes", c.gc.Address(), len(rm))

	select {
	case c.gc.SendQueue() <- rm:
	case <-c.gc.Closed():
		err = ErrConnClsoed
	default:
		c.gc.Close()
		err = ErrWriteQueueFull
	}
	return
}

// SendPing to peer. It's impossible if
// PingInterval of Config of related Node
// is zero
func (c *Conn) SendPing() {
	if c.s.conf.PingInterval <= 0 {
		return
	}
	c.events <- &sendPingEvent{make(chan struct{})}
}

type waitResponse struct {
	req  msg.Msg       // request
	rspn chan msg.Msg  // response (nil if no message required)
	err  chan error    // some err (must not be nil)
	ttm  chan struct{} // teminate timeout (nil if not timeout set)
}

// id id ID of the req Msg
func (c *Conn) sendRequest(id msg.ID, req msg.Msg,
	cerr chan error, rspn chan msg.Msg) (wr *waitResponse, err error) {

	wr = new(waitResponse)
	wr.req = req
	wr.rspn = rspn
	wr.err = cerr
	wr.ttm = make(chan struct{})

	c.rr[id] = wr

	// TODO (kostyarin): send queue can't be zero size-channel
	if err = c.Send(req); err != nil {
		delete(c.rr, id)
		return
	}

	if rt := c.s.conf.ResponseTimeout; rt > 0 {
		wr.ttm = make(chan struct{})
		go c.waitTimeout(id, wr, rt)
	}
	return
}

func (c *Conn) waitTimeout(id msg.ID, wr *waitResponse, rt time.Duration) {

	tm := time.NewTimer(rt)
	defer tm.Stop()

	select {
	case <-wr.ttm:
		return
	case <-tm.C:
		select {
		case c.timeout <- id:
		case <-c.gc.Closed():
		}
	}

}

// Subscribe starts exchanging given feed with peer.
// It's blocking call
func (c *Conn) Subscribe(pk cipher.PubKey) (err error) {
	if err = pk.Verify(); err != nil {
		return
	}
	errc := make(chan error)
	c.subscribeEvent(pk, errc)
	return <-errc
}

// Unsubscribe stops exchangin given feed with peer.
// It's non-blocking call
func (c *Conn) Unsubscrube(pk cipher.PubKey) {
	c.unsubscribeEvent(pk)
}

// ListOfFeeds of peer if it's public. It's blocking call
func (c *Conn) ListOfFeeds() (list []cipher.PubKey, err error) {
	errc := make(chan error)
	rspn := make(chan msg.Msg)
	c.listOfFeedsEvent(rspn, errc)
	select {
	case err = <-errc:
		return
	case m := <-rspn:
		lof, ok := m.(*msg.ListOfFeeds)
		if !ok {
			err = ErrWrongResponseMsgType
			return
		}
		list = lof.List
	}
	return
}

func (c *Conn) handshake() (err error) {

	c.s.Debugf(ConnPin, "[%s] acceptHandshake", c.gc.Address())

	if c.gc.IsIncoming() {
		return c.acceptHandshake()
	}
	return c.performHandshake()
}

func (c *Conn) acceptHandshake() (err error) {

	c.s.Debugf(ConnPin, "[%s] acceptHandshake", c.gc.Address())

	select {
	case raw := <-c.gc.ReceiveQueue():
		var m msg.Msg
		if m, err = msg.Decode(raw); err != nil {
			return
		}
		if hello, ok := m.(*msg.Hello); ok {
			if hello.Protocol == msg.Version {
				return c.Send(c.s.src.Accept(hello))
			}
			return c.Send(c.s.src.Reject(hello,
				ErrIncompatibleProtocolVersion.Error()))
		}
		return ErrWrongResponseMsgType

		// TODO (kostyarin): handshake timeout

	case <-c.gc.Closed():
		return ErrConnClsoed
	}
}

func (c *Conn) performHandshake() (err error) {

	c.s.Debugf(ConnPin, "[%s] performHandshake", c.gc.Address())

	hello := c.s.src.Hello()

	if err = c.Send(hello); err != nil {
		return
	}

	select {
	case raw := <-c.gc.ReceiveQueue():
		var m msg.Msg
		if m, err = msg.Decode(raw); err != nil {
			return
		}
		switch tt := m.(type) {
		case *msg.Accept:
			if tt.ResponseFor == hello.ID {
				return
			}
			return ErrWrongResponseID
		case *msg.Reject:
			return errors.New(tt.Err)
		default:
			return ErrWrongResponseMsgType
		}

	// TODO (kostyarin): handshake timeout

	case <-c.gc.Closed():
		return ErrConnClsoed
	}

}

func (c *Conn) handle() {
	c.s.Debug(ConnPin, "[%s] start handling", c.gc.Address())
	defer c.s.Debug(ConnPin, "[%s] stop handling", c.gc.Address())

	defer c.s.await.Done()

	var err error

	if err = c.handshake(); err != nil {
		c.s.Printf("[%s] handshake failed:", c.gc.Address(), err)
		return
	}

	c.s.addConn(c)
	defer c.s.delConn(c)

	if occ := c.s.conf.OnCreateConnection; occ != nil {
		go occ(c) // don't block
	}

	defer func() {
		if occ := c.s.conf.OnCloseConnection; occ != nil {
			go occ(c) // don't block
		}
	}()

	var (
		closed  = c.gc.Closed()
		receive = c.gc.ReceiveQueue()
		fill    = c.s.newFiller() // TODO
		events  = c.events
		timeout = c.timeout

		raw []byte
		m   msg.Msg
		evt event
		id  msg.ID
	)

	// TODO
	defer func() {
		fill.clsoe()
		done := make(chan struct{})
		go func() {
			defer close(done)
			fill.wait()
		}()
		for {
			select {
			case dre := <-fill.drop:
				c.dropRoot(dre.Root, dre.Err)
				fill.del(dre.Root)
			case fr := <-fill.full:
				c.rootFilled(fr)
				fill.del(fr)
			case <-done:
				for _, fr := range fill.fillers {
					c.dropRoot(fr.Root(), ErrConnClsoed) // drop
				}
				return
			}
		}
	}()

	for {
		select {

		// events
		case evt = <-events:
			evt.Handle(c)

		case id = <-timeout:
			if wr, ok := c.rr[id]; ok {
				delete(c.rr, id)
				wr.err <- ErrTimeout // terminated by timeout
			}
			// already

		// receive
		case raw = <-receive:
			if m, err = msg.Decode(raw); err != nil {
				c.s.Printf("[ERR] [%s] error decoding message: %v",
					c.gc.Address(), err)
				return
			}
			if err = c.handleMsg(m); err != nil {
				c.s.Printf("[ERR] [%s] error handling message: %v",
					c.gc.Address(), err)
				return
			}

		// filling (TODO)
		case dre := <-fill.drop:
			c.dropRoot(dre.Root, dre.Err)
			fill.del(dre.Root)
		case fr := <-fill.full:
			c.rootFilled(fr)
			fill.del(fr)
		case wcxo := <-fill.wantq:
			fill.waiting(wcxo)
			c.Send(c.s.src.RequestObject(wcxo.Hash))

		// closing
		case <-closed:
			return
		}
	}

}

func (c *Conn) dropRoot(r *skyobject.Root, err error) {
	c.s.Debugf(FillPin, "[%s] dropRoot %s: %v", c.gc.Address(), r.Short(), err)

	if ofb := c.s.conf.OnFillingBreaks; ofb != nil {
		ofb(c, r, err)
	}

	if c.s.conf.DropNonFullRotos {
		c.s.Debug(FillPin,
			"can't drop non-full Root: feature is not implemented")
		// TODO (kostyarin): remove root using Container
		//                   (add appropriate method to the
		//                   Container)
	}
}

func (c *Conn) rootFilled(r *skyobject.Root) {
	c.s.Debugf(FillPin, "[%s] rootFilled %s", c.Address(), r.Short())

	if orf := c.s.conf.OnRootFilled; orf != nil {
		orf(c, r)
	}
	c.s.sendToAllOfFeed(r.Pub, c.s.src.Root(r))
}

func (c *Conn) handlePong(pong *msg.Pong) {
	if ttm, ok := c.pings[pong.ResponseFor]; ok {
		close(ttm) // ok
	}
	c.s.Debugf(ConnPin, "[%s] unexpected Pong received", c.Address())
}

func (c *Conn) handlePing(ping *msg.Ping) {
	c.Send(c.s.src.Pong(ping))
}

func (c *Conn) handleSubscribe(subs *msg.Subscribe) {
	if c.s.addConnToFeed(c, subs.Feed) == false {
		c.Send(c.s.src.RejectSubscription(subs))
		return
	}
	c.subs[subs.Feed] = struct{}{} // add to internal list
	c.Send(c.s.src.AcceptSubscription(subs))
}

func (c *Conn) handleUnsubscribe(unsub *msg.Unsubscribe) {
	if deleted := c.s.delConnFromFeed(c, unsub.Feed); deleted {
		delete(c.subs, unsub.Feed)
	}
}

func (c *Conn) handleAcceptSubscription(as *msg.AcceptSubscription) {
	if wr, ok := c.rr[as.ResponseFor]; ok {
		if wr.ttm != nil {
			close(wr.ttm) // terminate timeout goroutine
		}
		if sub, ok := wr.req.(*msg.Subscribe); !ok {
			wr.err <- ErrWrongResponseMsgType
		} else if sub.Feed != as.Feed {
			wr.err <- ErrWrongResponse
		} else {
			wr.err <- nil // ok
		}
	} else {
		c.s.Printf("[ERR] [%s] unexpected AcceptSubscription msg")
	}
	delete(c.rr, as.ResponseFor)
}

func (c *Conn) handleRejectSubscription(rs *msg.RejectSubscription) {
	if wr, ok := c.rr[rs.ResponseFor]; ok {
		if wr.ttm != nil {
			close(wr.ttm) // terminate timeout goroutine
		}
		if sub, ok := wr.req.(*msg.Subscribe); !ok {
			wr.err <- ErrWrongResponseMsgType
		} else if sub.Feed != rs.Feed {
			wr.err <- ErrWrongResponse
		} else {
			wr.err <- ErrSubscriptionRejected
		}
		delete(c.rr, rs.ResponseFor)
	} else {
		c.s.Printf("[ERR] [%s] unexpected AcceptSubscription msg")
	}
}

func (c *Conn) handleRequestListOfFeeds(rlof *msg.RequestListOfFeeds) {
	if c.s.conf.PublicServer == false {
		c.Send(c.s.src.NonPublicServer(rlof))
		return
	}
	c.Send(c.s.src.ListOfFeeds(rlof, c.s.Feeds()))
}

func (c *Conn) handleListOfFeeds(lof *msg.ListOfFeeds) {
	if wr, ok := c.rr[lof.ResponseFor]; ok {
		if wr.ttm != nil {
			close(wr.ttm) // terminate timeout goroutine
		}
		if _, ok := wr.req.(*msg.RequestListOfFeeds); !ok {
			wr.err <- ErrWrongResponseMsgType
		} else {
			wr.rspn <- lof
		}
		delete(c.rr, lof.ResponseFor)
	} else {
		c.s.Printf("[ERR] [%s] unexpected ListOfFeeds msg")
	}
}

func (c *Conn) handleNonPublicServer(nps *msg.NonPublicServer) {
	if wr, ok := c.rr[nps.ResponseFor]; ok {
		if wr.ttm != nil {
			close(wr.ttm) // terminate timeout goroutine
		}
		if _, ok := wr.req.(*msg.RequestListOfFeeds); !ok {
			wr.err <- ErrWrongResponseMsgType
		} else {
			wr.err <- ErrNonPublicPeer
		}
		delete(c.rr, nps.ResponseFor)
	} else {
		c.s.Printf("[ERR] [%s] unexpected ListOfFeeds msg")
	}
}

func (c *Conn) handleRoot(r *msg.Root) {
	// TODO: filling
}

func (c *Conn) handleRequestObject(ro *msg.RequestObject) {
	if val, _, _ := c.s.db.CXDS().Get(ro.Key); val != nil {
		c.Send(c.s.src.Object(ro.Key, val))
		return
	}
	c.Send(c.s.src.NotFound(ro.Key))
}

func (c *Conn) handleObject(o *msg.Object) {
	// TODO: filling
}

func (c *Conn) handleNotFound(nf *msg.NotFound) {
	// TODO: filling (fatal for the connection)
}

func (c *Conn) handleMsg(m msg.Msg) (err error) {

	c.s.Debugf(MsgPin, "[%s] handleMsg %T", c.gc.Address(), m)

	switch tt := m.(type) {

	case *msg.Pong:
		c.handlePong(tt)
	case *msg.Ping:
		c.handlePing(tt)

	// subscriptions

	case *msg.Subscribe:
		c.handleSubscribe(tt)
	case *msg.Unsubscribe:
		c.handleUnsubscribe(tt)

	// subscriptions reply

	case *msg.AcceptSubscription:
		c.handleAcceptSubscription(tt)
	case *msg.RejectSubscription:
		c.handleRejectSubscription(tt)

	// public server features

	case *msg.RequestListOfFeeds:
		c.handleRequestListOfFeeds(tt)
	case *msg.ListOfFeeds:
		c.handleListOfFeeds(tt)
	case *msg.NonPublicServer:
		c.handleNonPublicServer(tt)

	// root, registry, data and requests

	case *msg.Root:
		c.handleRoot(tt)

	case *msg.RequestObject:
		c.handleRequestObject(tt)
	case *msg.Object:
		c.handleObject(tt)
	case *msg.NotFound:
		c.handleNotFound(tt)

	default:
		err = ErrWrongMsgType
	}

	return
}
