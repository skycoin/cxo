package node

import (
	"errors"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/msg"
	"github.com/skycoin/cxo/skyobject"
)

// connections related errors
var (
	ErrWriteQueueFull              = errors.New("write queue full")
	ErrWrongResponseID             = errors.New("wrong response ID")
	ErrWrongMsgType                = errors.New("wrong message type")
	ErrWrongResponseMsgType        = errors.New("wrong response messag type")
	ErrIncompatibleProtocolVersion = errors.New("incompatible protocol version")
	ErrWrongResponse               = errors.New("wrong response")
)

// A Conn represents connection of Node
type Conn struct {
	s *Node // back reference for logs

	gc *gnet.Conn // underlying *gnet.Conn

	// not threads safe fields

	// subscriptions of the Conn to resubscribe OnDial
	subs map[cipher.PubKey]*sentRoot
	rr   map[msg.ID]*waitResponse // request-response

	events  chan event  // events of the Conn
	timeout chan msg.ID // timeouts

	pings map[msg.ID]chan struct{}

	// send a held Root to peer, the Root
	// should be unheld after use
	sendRoot chan *skyobject.Root

	// filling

	wantq    chan skyobject.WCXO             // request wanted CX object
	requests map[cipher.SHA256][]chan []byte // wait reply

	// must drain
	full chan skyobject.FullRoot      // fill Root objects
	drop chan skyobject.DropRootError // root reference with error (reason)

	// filling Roots (hash of Root -> *Filler)
	fillers map[cipher.SHA256]*skyobject.Filler

	// synchronisation internals

	// done means that all finialization performed
	// and handling goroutine exists
	done chan struct{}
}

type sentRoot struct {
	// current Root object that
	// remote node is filling
	current *skyobject.Root
	// next Root object that will be sent
	// after the current one
	next *skyobject.Root
}

// unhold current and next
func (s *sentRoot) unhold(so *skyobject.Container) {
	if sc := s.current; sc != nil {
		so.Unhold(sc.Pub, sc.Seq)
	}
	if sn := s.next; sn != nil {
		so.Unhold(sn.Pub, sn.Seq)
	}
}

func (s *Node) newConn(gc *gnet.Conn) (c *Conn) {

	c = new(Conn)

	c.s = s
	c.gc = gc

	// TODO (kostyarin): make all the 128s and 8s below configurable

	c.subs = make(map[cipher.PubKey]*sentRoot)
	c.rr = make(map[msg.ID]*waitResponse)
	c.events = make(chan event, 128)
	c.timeout = make(chan msg.ID, 128)
	c.pings = make(map[msg.ID]chan struct{})
	c.sendRoot = make(chan *skyobject.Root, 8)

	c.wantq = make(chan skyobject.WCXO, 128)
	c.requests = make(map[cipher.SHA256][]chan []byte)
	c.full = make(chan skyobject.FullRoot, 8)      // saved
	c.drop = make(chan skyobject.DropRootError, 8) // need to drop
	c.fillers = make(map[cipher.SHA256]*skyobject.Filler)

	c.done = make(chan struct{})

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

// Gnet returns underlying *gnet.Conn
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

// SendRoot object to peer. This can be ignored if
// peer is now filling newer Root
func (c *Conn) SendRoot(r *skyobject.Root) (_ error) {
	c.s.Debugf(FillPin, "[%s] SendRoot %s", c.Address(), r.Short())

	c.s.so.Hold(r.Pub, r.Seq)
	select {
	case c.sendRoot <- r:
	case <-c.gc.Closed():
		c.s.so.Unhold(r.Pub, r.Seq)
		return ErrConnClosed
	}
	return
}

// SendRaw encoded message to peer
func (c *Conn) SendRaw(rm []byte) (err error) {

	c.s.Debugf(ConnPin, "[%s] SendRaw %d bytes", c.gc.Address(), len(rm))

	select {
	case c.gc.SendQueue() <- rm:
	case <-c.gc.Closed():
		err = ErrConnClosed
	default:
		c.gc.Close()
		err = ErrWriteQueueFull
	}
	return
}

func (c *Conn) enqueueEvent(evt event) {
	select {
	case c.events <- evt:
	case <-c.gc.Closed():
	}
}

// SendPing to peer. It's impossible if
// PingInterval of Config of related Node
// is zero
func (c *Conn) SendPing() {
	if c.s.conf.PingInterval <= 0 {
		return
	}
	c.enqueueEvent(&sendPingEvent{make(chan struct{})})
}

type waitResponse struct {
	req  msg.Msg       // request
	rspn chan msg.Msg  // response (nil if no message required)
	err  chan error    // some err (must not be nil)
	ttm  chan struct{} // teminate timeout (nil if not timeout set)
}

// ID of the req Msg
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
// It's blocking call. The Subscribe adds feed to
// related node, even if it returns error (and only
// if given public key is valid)
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

// Unsubscribe stops exchanging given feed with peer.
// It's non-blocking call
func (c *Conn) Unsubscribe(pk cipher.PubKey) {
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
		return ErrConnClosed
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
		return ErrConnClosed
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
			c.dropRoot(dre)
		case fr := <-c.full:
			c.rootFilled(fr)
		}
	}
}

func (c *Conn) unholdSent() {
	for _, sent := range c.subs {
		sent.unhold(c.s.so)
	}
}

func (c *Conn) drainSendRoot() {
	for {
		select {
		case r := <-c.sendRoot:
			c.s.so.Unhold(r.Pub, r.Seq)
		default:
			return
		}

	}
}

func (c *Conn) handle(hs chan<- error) {
	c.s.Debugf(ConnPin, "[%s] start handling", c.gc.Address())
	defer c.s.Debugf(ConnPin, "[%s] stop handling", c.gc.Address())

	defer c.s.await.Done()
	defer close(c.done)

	var err error

	if err = c.handshake(); err != nil {
		if false == c.gc.IsIncoming() {
			// send to (*Node).Connect() or to (*Node).ConnectOrGet()
			hs <- err
			return
		}
		c.s.Printf("[%s] handshake failed: %v", c.gc.Address(), err)
		return
	}

	if false == c.gc.IsIncoming() {
		close(hs) // release the hs
	}

	defer c.s.delConnFromWantedObjects(c)
	defer c.drainSendRoot() // unhold roots in the sendRoot channel

	c.s.addConn(c)
	defer c.s.delConn(c)

	c.onCreateCallback()
	defer c.onCloseCallback()

	var (
		closed   = c.gc.Closed()
		receive  = c.gc.ReceiveQueue()
		events   = c.events
		timeout  = c.timeout
		sendRoot = c.sendRoot

		raw []byte
		m   msg.Msg
		evt event
		id  msg.ID
		r   *skyobject.Root
	)

	defer c.terminateFillers()

	defer c.unholdSent() // release sent Root objects

	for {
		select {

		// root objects
		case r = <-sendRoot:
			if sent, ok := c.subs[r.Pub]; ok {
				if sc := sent.current; sc == nil {
					sent.current = r
					c.Send(c.s.src.Root(r))
				} else if r.Seq <= sc.Seq {
					c.s.so.Unhold(r.Pub, r.Seq)
					continue // drop older or the same
				} else if sn := sent.next; sn == nil {
					sent.next = r // keep next
				} else if r.Seq <= sn.Seq {
					c.s.so.Unhold(r.Pub, r.Seq)
					continue // drop older or the same
				} else {
					c.s.so.Unhold(sn.Pub, sn.Seq)
					sent.next = r // replace next with newer
				}
			} else {
				c.s.so.Unhold(r.Pub, r.Seq) // drop it otherwise
			}

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
			c.dropRoot(dre)
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

func (c *Conn) dropRootRelated(r *skyobject.Root, incrs []cipher.SHA256,
	reason error) {

	if ofb := c.s.conf.OnFillingBreaks; ofb != nil {
		ofb(c, r, reason)
	}

	// the Root is not saved in database but some related objects
	// have been saved and we have to decrement them all

	// actually the Root is saved, but it's not saved in index

	if err := c.s.db.CXDS().MultiDec(incrs); err != nil {
		c.s.Printf("[ERR] [%s] can't drop Root %s related objects: %v",
			c.Address(), r.Short(), err)
	}

	c.Send(c.s.src.RootDone(r.Pub, r.Seq)) // notify peer
}

func (c *Conn) dropRoot(dre skyobject.DropRootError) {

	c.s.Debugf(FillPin, "[%s] dropRoot %s: %v", c.gc.Address(),
		dre.Root.Short(), dre.Err)

	c.s.dropavg.AddStartTime(dre.Tp) // stat

	c.delFillingRoot(dre.Root)
	c.dropRootRelated(dre.Root, dre.Incrs, dre.Err)

}

func (c *Conn) rootFilled(fr skyobject.FullRoot) {
	c.s.Debugf(FillPin, "[%s] rootFilled %s", c.Address(), fr.Root.Short())

	c.delFillingRoot(fr.Root)

	// we have to be sure that this connection
	// is subscribed to feed of this Root
	if _, ok := c.subs[fr.Root.Pub]; !ok {

		// drop the Root and remove all related saved objects,
		// because we are not subscribed to the feed anymore

		c.s.dropavg.AddStartTime(fr.Tp) // stat
		c.dropRootRelated(fr.Root, fr.Incrs, ErrUnsubscribed)
		return
	}

	// (1) hold Root
	// (2) defered unhold
	// (3) add it to index DB
	// (4) perform callback

	c.s.so.Hold(fr.Root.Pub, fr.Root.Seq)
	defer c.s.so.Unhold(fr.Root.Pub, fr.Root.Seq)

	err := c.s.db.IdxDB().Tx(func(feeds data.Feeds) (err error) {
		var rs data.Roots
		if rs, err = feeds.Roots(fr.Root.Pub); err != nil {
			return
		}
		return rs.Set(&data.Root{
			Seq:  fr.Root.Seq,
			Prev: fr.Root.Prev,
			Hash: fr.Root.Hash,
			Sig:  fr.Root.Sig,
		})
	})

	if err != nil {
		// can't add this Root to DB, so, let's drop it
		c.s.Debugf(FillPin, "[ERR] [%s] can't save Root in IdxDB: %v",
			c.Address(), err)
		c.s.dropavg.AddStartTime(fr.Tp) // stat
		c.dropRootRelated(fr.Root, fr.Incrs, err)
		return
	}

	c.s.fillavg.AddStartTime(fr.Tp) // stat

	if orf := c.s.conf.OnRootFilled; orf != nil {
		c.s.so.Hold(fr.Root.Pub, fr.Root.Seq) // hold for the callback
		go func() {
			defer c.s.so.Unhold(fr.Root.Pub, fr.Root.Seq) // unhold
			fr.Root.IsFull = true
			orf(c, fr.Root)
		}()
	}

	c.Send(c.s.src.RootDone(fr.Root.Pub, fr.Root.Seq)) // notify peer
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
	if r, err := c.s.so.LastRoot(pk); err == nil {
		c.SendRoot(r)
		c.s.so.Unhold(pk, r.Seq) // LastRoot holds the Root
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
	if _, ok := c.subs[subs.Feed]; !ok {
		c.subs[subs.Feed] = &sentRoot{} // add to internal list
	}
	c.Send(c.s.src.AcceptSubscription(subs))
	c.sendLastFull(subs.Feed)
}

func (c *Conn) unsubscribe(pk cipher.PubKey) {
	c.s.delConnFromFeed(c, pk) // delete from Node feed->conns mapping
	if sent, ok := c.subs[pk]; ok {
		sent.unhold(c.s.so) // unhold sent and prepared to sending
		delete(c.subs, pk)  // delete from resubscriptions
	}
	c.delFillingRootsOfFeed(pk) // stop filling
}

func (c *Conn) handleUnsubscribe(unsub *msg.Unsubscribe) {
	c.s.Debugf(HandlePin, "[%s] handleUnsubscribe: %s", c.gc.Address(),
		unsub.Feed.Hex()[:7])

	if our := c.s.conf.OnUnsubscribeRemote; our != nil {
		our(c, unsub.Feed)
	}

	c.unsubscribe(unsub.Feed)
}

func (c *Conn) handleAcceptSubscription(as *msg.AcceptSubscription) {
	c.s.Debugf(HandlePin, "[%s] handleAcceptSubscription: %s",
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
		if _, ok := c.subs[as.Feed]; !ok {
			c.subs[as.Feed] = &sentRoot{} // add to internal list
		}
		c.sendLastFull(as.Feed)

	} else {
		c.s.Printf("[ERR] [%s] unexpected AcceptSubscription msg", c.Address())
	}
	delete(c.rr, as.ResponseFor)
}

func (c *Conn) handleRejectSubscription(rs *msg.RejectSubscription) {
	c.s.Debugf(HandlePin, "[%s] handleRejectSubscription: %s",
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
		c.s.Printf("[ERR] [%s] unexpected RejectSubscription msg", c.Address())
	}
}

func (c *Conn) handleRequestListOfFeeds(rlof *msg.RequestListOfFeeds) {
	c.s.Debugf(HandlePin, "[%s] handleRequestListOfFeeds", c.gc.Address())

	if c.s.conf.PublicServer == false {
		c.Send(c.s.src.NonPublicServer(rlof))
		return
	}
	c.Send(c.s.src.ListOfFeeds(rlof, c.s.Feeds()))
}

func (c *Conn) handleListOfFeeds(lof *msg.ListOfFeeds) {
	c.s.Debugf(HandlePin, "[%s] handleListOfFeeds", c.gc.Address())

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
		c.s.Printf("[ERR] [%s] unexpected ListOfFeeds msg", c.Address())
	}
}

func (c *Conn) handleNonPublicServer(nps *msg.NonPublicServer) {
	c.s.Debugf(HandlePin, "[%s] handleNonPublicServer", c.gc.Address())

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
		c.s.Printf("[ERR] [%s] unexpected ListOfFeeds msg", c.Address())
	}
}

func (c *Conn) handleRoot(rm *msg.Root) {
	c.s.Debugf(HandlePin, "[%s] handleRoot {%s:%d}", c.gc.Address(),
		rm.Feed.Hex()[:7], rm.Seq)

	r, err := c.s.so.AddEncodedRoot(rm.Sig, rm.Value)
	if err != nil {
		c.Send(c.s.src.RootDone(rm.Feed, rm.Seq)) // done
		c.s.Printf("[ERR] [%s] error adding received Root: %v", c.Address(),
			err)
		return
	}

	// send the Root forward for all
	c.s.broadcastRoot(r, c)

	if r.IsFull { // we already have the Root and it's full
		c.Send(c.s.src.RootDone(rm.Feed, rm.Seq)) // done

		// don't call OnRootRecived and OnRootFilled callbacks
		return
	}

	if orr := c.s.conf.OnRootReceived; orr != nil {
		go orr(c, r)
	}

	c.fillRoot(r)
}

func (c *Conn) handleRootDone(rd *msg.RootDone) {
	c.s.Debugf(HandlePin, "[%s] handleRootDone {%s:%d}", c.gc.Address(),
		rd.Feed.Hex()[:7], rd.Seq)

	sent, ok := c.subs[rd.Feed]
	if !ok {
		c.s.Debugf(FillPin,
			"[%s] wrong RootDone message received (not subscribed)",
			c.Address())
		return
	}

	if sc := sent.current; sc.Seq == rd.Seq {
		c.s.so.Unhold(sc.Pub, sc.Seq) // unhold
		sent.current = nil            // clear
		// if there is next
		if sn := sent.next; sn != nil {
			sent.current = sn                  // next turns current
			sent.next = nil                    // next turns vacant
			c.Send(c.s.src.Root(sent.current)) // send it
		}
		return
	}

	c.s.Debugf(FillPin, "[%s] wrong RootDone message received (unknown seq)",
		c.Address())
	return

}

func (c *Conn) handleRequestObject(ro *msg.RequestObject) {
	c.s.Debugf(HandlePin, "[%s] handleRequestObject: %s", c.gc.Address(),
		ro.Key.Hex()[:7])

	if val, _, _ := c.s.db.CXDS().Get(ro.Key); val != nil {
		c.Send(c.s.src.Object(val))
		return
	}

	// add to list of requested objects, waiting for incoming objects
	// to send it later
	c.s.wantObject(ro.Key, c)
}

func (c *Conn) handleObject(o *msg.Object) {

	key := cipher.SumSHA256(o.Value)

	c.s.Debugf(HandlePin, "[%s] handleObject: %s", c.gc.Address(),
		key.Hex()[:7])

	if rs, ok := c.requests[key]; ok {

		// store in CXDS
		if _, err := c.s.db.CXDS().Set(key, o.Value); err != nil {
			c.s.Printf(
				"[CRIT] [%s] can't set received object: %v, terminating...",
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
		// for this object
		c.s.gotObject(key, o)

	} else {
		c.s.Debugf(FillPin, "[%s] got object the node doesn't want: %s",
			c.Address(), key.Hex()[:7])
		return
	}

}

func (c *Conn) handleMsg(m msg.Msg) (err error) {

	c.s.Debugf(HandlePin, "[%s] handleMsg %T", c.gc.Address(), m)

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
	case *msg.RootDone:
		c.handleRootDone(tt)

	case *msg.RequestObject:
		c.handleRequestObject(tt)
	case *msg.Object:
		c.handleObject(tt)

	default:
		err = ErrWrongMsgType
	}

	return
}
