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

	pings map[msg.ID]chan struct{}

	// filling

	wantq    chan skyobject.WCXO             // request wanted CX object
	requests map[cipher.SHA256][]chan []byte // wait reply

	// must drain
	full chan *skyobject.Root         // fill Root obejcts
	drop chan skyobject.DropRootError // root reference with error (reason)

	// filling Roots (hash of Root -> *Filler)
	fillers map[cipher.SHA256]*skyobject.Filler
}

func (s *Node) newConn(gc *gnet.Conn) (c *Conn) {

	c = new(Conn)

	c.s = s
	c.gc = gc

	// TODO (kostyarin): make all the 128s and 8s below configurable

	c.subs = make(map[cipher.PubKey]struct{})
	c.rr = make(map[msg.ID]*waitResponse)
	c.events = make(chan event, 128)
	c.timeout = make(chan msg.ID, 128)
	c.pings = make(map[msg.ID]chan struct{})

	c.wantq = make(chan skyobject.WCXO, 128)
	c.requests = make(map[cipher.SHA256][]chan []byte)
	c.full = make(chan *skyobject.Root, 8)         // saved
	c.drop = make(chan skyobject.DropRootError, 8) // need to drop
	c.fillers = make(map[cipher.SHA256]*skyobject.Filler)

	gc.SetValue(c) // for resubscriptions

	return
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
// It's blocking call. The Subscribe adds feed to related node
func (c *Conn) Subscribe(pk cipher.PubKey) (err error) {
	if err = pk.Verify(); err != nil {
		return
	}
	if err = c.s.AddFeed(pk); err != nil {
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

	var tc <-chan time.Time

	if pi := c.s.conf.PingInterval; pi > 0 {
		tm := time.NewTimer(pi)
		defer tm.Stop()

		tc = tm.C
	}

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

	// handshake timeout
	case <-tc:
		return ErrTimeout

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

	var tc <-chan time.Time

	if pi := c.s.conf.PingInterval; pi > 0 {
		tm := time.NewTimer(pi)
		defer tm.Stop()

		tc = tm.C
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

	//  handshake timeout
	case <-tc:
		return ErrTimeout

	case <-c.gc.Closed():
		return ErrConnClsoed
	}

}

func (c *Conn) onCreateCallback() {
	if occ := c.s.conf.OnCreateConnection; occ != nil {
		go occ(c) // don't block
	}
}

func (c *Conn) onCloseCallback() {
	if occ := c.s.conf.OnCloseConnection; occ != nil {
		go occ(c) // don't block
	}
}

func (c *Conn) terminateFillers() {

	c.s.Debugf(FillPin, "[%s] terminate fillers", c.Address())
	defer c.s.Debugf(FillPin, "[%s] fillers have been terminated", c.Address())

	go c.closeFillers()
	for {
		if len(c.fillers) == 0 {
			return // all done
		}
		select {
		case dre := <-c.drop:
			c.dropRoot(dre.Root, dre.Err, dre.Saved)
		case fr := <-c.full:
			c.rootFilled(fr)
		}
	}
}

func (c *Conn) handle() {
	c.s.Debugf(ConnPin, "[%s] start handling", c.gc.Address())
	defer c.s.Debugf(ConnPin, "[%s] stop handling", c.gc.Address())

	defer c.s.await.Done()

	var err error

	if err = c.handshake(); err != nil {
		c.s.Printf("[%s] handshake failed:", c.gc.Address(), err)
		return
	}

	defer c.s.delConnFromWantedObjects(c)

	c.s.addConn(c)
	defer c.s.delConn(c)

	c.onCreateCallback()
	defer c.onCloseCallback()

	var (
		closed  = c.gc.Closed()
		receive = c.gc.ReceiveQueue()
		events  = c.events
		timeout = c.timeout

		raw []byte
		m   msg.Msg
		evt event
		id  msg.ID
	)

	defer c.terminateFillers()

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

		// filling
		case dre := <-c.drop: // drop root
			c.dropRoot(dre.Root, dre.Err, dre.Saved)
		case fr := <-c.full:
			c.rootFilled(fr)
		case wcxo := <-c.wantq:
			c.addRequestedObjectToWaitingList(wcxo)
			c.Send(c.s.src.RequestObject(wcxo.Key))

		// closing
		case <-closed:
			return
		}
	}

}

func (c *Conn) dropRoot(r *skyobject.Root, err error, saved []cipher.SHA256) {

	c.s.Debugf(FillPin, "[%s] dropRoot %s: %v", c.gc.Address(), r.Short(), err)

	c.delFillingRoot(r)

	if ofb := c.s.conf.OnFillingBreaks; ofb != nil {
		ofb(c, r, err)
	}

	// TODO: remove the Root and decrement all saved objects

	//

}

func (c *Conn) rootFilled(r *skyobject.Root) {
	c.s.Debugf(FillPin, "[%s] rootFilled %s", c.Address(), r.Short())

	c.delFillingRoot(r)

	if orf := c.s.conf.OnRootFilled; orf != nil {
		orf(c, r)
	}
}

func (c *Conn) handlePong(pong *msg.Pong) {
	c.s.Debugf(ConnPin, "[%s] handlePong: %d", c.gc.Address(),
		pong.ResponseFor.Uint64())

	if ttm, ok := c.pings[pong.ResponseFor]; ok {
		close(ttm) // ok
		return
	}
	c.s.Debugf(ConnPin, "[%s] unexpected Pong received", c.Address())
}

func (c *Conn) handlePing(ping *msg.Ping) {
	c.s.Debugf(ConnPin, "[%s] handlePing: %d", c.gc.Address(),
		ping.ID.Uint64())

	c.Send(c.s.src.Pong(ping))
}

func (c *Conn) sendLastFull(pk cipher.PubKey) {
	if r, err := c.s.so.LastFull(pk); err == nil {
		c.Send(c.s.src.Root(r))
	}
}

func (c *Conn) handleSubscribe(subs *msg.Subscribe) {
	c.s.Debugf(SubscrPin, "[%s] handleSubscribe: %s", c.gc.Address(),
		subs.Feed.Hex()[:7])

	if osr := c.s.conf.OnSubscribeRemote; osr != nil {
		if reject := osr(c, subs.Feed); reject != nil {
			c.s.Debugf(SubscrPin, "[%s] remote subscription rejected: %v",
				c.Address(), reject)
			c.Send(c.s.src.RejectSubscription(subs))
			return
		}
	}

	if c.s.addConnToFeed(c, subs.Feed) == false {
		c.Send(c.s.src.RejectSubscription(subs))
		return
	}
	c.subs[subs.Feed] = struct{}{} // add to internal list
	c.Send(c.s.src.AcceptSubscription(subs))
	c.sendLastFull(subs.Feed)
}

func (c *Conn) unsubscribe(pk cipher.PubKey) {
	c.s.delConnFromFeed(c, pk)  // delete from Node feed->conns mapping
	delete(c.subs, pk)          // delete from resubscriptions
	c.delFillingRootsOfFeed(pk) // stop filling
}

func (c *Conn) handleUnsubscribe(unsub *msg.Unsubscribe) {
	c.s.Debugf(SubscrPin, "[%s] handleUnsubscribe: %s", c.gc.Address(),
		unsub.Feed.Hex()[:7])

	if our := c.s.conf.OnUnsubscribeRemote; our != nil {
		our(c, unsub.Feed)
	}

	c.unsubscribe(unsub.Feed)
}

func (c *Conn) handleAcceptSubscription(as *msg.AcceptSubscription) {
	c.s.Debugf(SubscrPin, "[%s] handleAcceptSubscription: %s",
		c.gc.Address(), as.Feed.Hex()[:7])

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

		if c.s.addConnToFeed(c, as.Feed) == false {
			return // don't share the feed anymore
		}
		c.subs[as.Feed] = struct{}{} // add to internal list
		c.sendLastFull(as.Feed)

	} else {
		c.s.Printf("[ERR] [%s] unexpected AcceptSubscription msg")
	}
	delete(c.rr, as.ResponseFor)
}

func (c *Conn) handleRejectSubscription(rs *msg.RejectSubscription) {
	c.s.Debugf(SubscrPin, "[%s] handleRejectSubscription: %s",
		c.gc.Address(), rs.Feed.Hex()[:7])

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
		c.s.Printf("[ERR] [%s] unexpected RejectSubscription msg")
	}
}

func (c *Conn) handleRequestListOfFeeds(rlof *msg.RequestListOfFeeds) {
	c.s.Debugf(ConnPin, "[%s] handleRequestListOfFeeds", c.gc.Address())

	if c.s.conf.PublicServer == false {
		c.Send(c.s.src.NonPublicServer(rlof))
		return
	}
	c.Send(c.s.src.ListOfFeeds(rlof, c.s.Feeds()))
}

func (c *Conn) handleListOfFeeds(lof *msg.ListOfFeeds) {
	c.s.Debugf(ConnPin, "[%s] handleListOfFeeds", c.gc.Address())

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
	c.s.Debugf(ConnPin, "[%s] handleNonPublicServer", c.gc.Address())

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

func (c *Conn) handleRoot(rm *msg.Root) {
	c.s.Debugf(FillPin, "[%s] handleRoot of %s", c.gc.Address(),
		rm.Feed.Hex()[:7])

	r, err := c.s.so.AddEncodedRoot(rm.Sig, rm.Value)
	if err != nil {
		c.s.Printf("[ERR] [%s] error adding received Root: %v", c.Address(),
			err)
		return
	}

	// send the Root forward for all
	c.s.sendToAllOfFeedExcept(r.Pub, rm, c)

	if r.IsFull { // we already have the Root and it's full
		return // don't call OnRootRecived and OnRootFilled callbacks
	}

	if orr := c.s.conf.OnRootReceived; orr != nil {
		go orr(c, r)
	}

	c.fillRoot(r)
}

func (c *Conn) handleRequestObject(ro *msg.RequestObject) {
	c.s.Debugf(FillPin, "[%s] handleRequestObject: %s", c.gc.Address(),
		ro.Key.Hex()[:7])

	if val, _, _ := c.s.db.CXDS().Get(ro.Key); val != nil {
		c.Send(c.s.src.Object(val))
		return
	}

	// add to list of requested obejcts, waiting for incoming objects
	// to send it later
	c.s.wantObejct(ro.Key, c)
}

func (c *Conn) handleObject(o *msg.Object) {

	key := cipher.SumSHA256(o.Value)

	c.s.Debugf(FillPin, "[%s] handleObject: %s", c.gc.Address(),
		key.Hex()[:7])

	if rs, ok := c.requests[key]; ok {

		// store in CXDS
		if _, err := c.s.db.CXDS().Set(key, o.Value); err != nil {
			c.s.Printf(
				"[CRIT] [%s] can't set received obejct: %v, terminating...",
				c.Address(), err)
			go c.s.Close() // terminate all
			return
		}

		// awake fillers
		for _, gotq := range rs {
			// the gotq has 1 length and this sending is not blocking
			gotq <- o.Value
		}
		delete(c.requests, key)

		// notify another conections that has been requested
		// for this obejct
		c.s.gotObject(key, o)

	} else {
		c.s.Debugf(FillPin, "[%s] got obejct the node doesn't want: %s",
			c.Address(), key.Hex()[:7])
		return
	}

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

	default:
		err = ErrWrongMsgType
	}

	return
}
