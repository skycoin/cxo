package node

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/net/factory"

	"github.com/skycoin/cxo/node/msg"
	"github.com/skycoin/cxo/skyobject/registry"
)

// feed of a connection
type connFeed struct {

	// lock
	sync.Mutex

	// currently filling Root object;
	// the Root is held by this node
	// and by remote peer too
	fill *registry.Root

	// a Root that waits to be filled;
	// only one pending Root allowed,
	// if remote peer send new Root to
	// fill, then this pending Root
	// will be repalced with new one,
	// because the node worries about
	// latest Root only, and the node
	// fills one Root per connection-
	// -feed at the same time; the
	// Root is held by this node and
	// by remote peer too
	fillPending *registry.Root

	// a Root that sent to peer and held
	// by this node waiting for response
	sent *registry.Root

	// if new Root obejct published and
	// the 'sent' Root is not filled by
	// peer yet, then the new Root waits;
	// only one waiting Root per connection-
	// -feed allowed; a Root newer then the
	// sentPending replaces the sentPending
	sentPending *registry.Root
}

// A Conn represent connection of the Node
type Conn struct {
	*factory.Connection

	// lock
	mx sync.Mutex

	// is incoming or not
	incoming bool

	// is tcp or udp
	tcp bool

	// back reference
	n *Node

	// peer id
	peerID NodeID

	// feeds this connection share with peer
	feeds map[cipher.PubKey]*connFeed

	// amount of Root obejcts currently
	// filling by the Conn
	fillingRoots int

	// messege seq number (for request-response)
	seq uint32

	// requests
	reqs map[uint32]chan<- msg.Msg

	// channels (from factory.Connection)
	sendq  chan<- []byte
	closeq <-chan struct{}

	// wait for receiving loop
	await sync.WaitGroup

	// close once
	closeo sync.Once
}

func (n *Node) newConnection(
	fc *factory.Connection,
	isIncoming bool,
	isTCP bool,
) (
	c *Conn,
) {

	c = new(Conn)

	c.Connection = fc
	c.incoming = isIncoming
	c.tcp = isTCP

	c.n = n

	c.feeds = make(map[cipher.PubKey]*connFeed)
	c.reqs = make(map[uint32]chan<- msg.Msg)

	c.sendq = fc.GetChanOut()
	c.closeq = make(chan struct{})

	n.addPendingConn(c)

	//
	// the next step is c.handshake() and c.run()
	//

	return
}

// start handling
func (c *Conn) run() {
	c.await.Add(1)
	go c.receiving()
}

func (c *Conn) decodeRaw(raw []byte) (seq, rseq uint32, m msg.Msg, err error) {

	if len(raw) < 9 {
		err = errors.New("invlaid messege received: too short")
		return
	}

	seq = binary.LittleEndian.Uint32(raw)
	raw = raw[4:]

	rseq = binary.LittleEndian.Uint32(raw)
	raw = raw[4:]

	m, err = msg.Decode(raw)
	return
}

//
// info
//

// IsTCP returns true if this conenctions
// is tcp connection
func (c *Conn) IsTCP() (tcp bool) {
	return c.tcp
}

// IsUDP retursn true if this conenctions
// si udp conenction
func (c *Conn) IsUDP() (udp bool) {
	return c.tcp == false
}

// PeerID is ID of remote peer that used
// for internals and unique
func (c *Conn) PeerID() (id NodeID) {
	return c.peerID
}

// IsIncoming returns true if this Conn is
// incoming and accepted by listener
func (c *Conn) IsIncomig() (ok bool) {
	return c.incoming
}

// IsOutgoing is inverse of the IsIncoming
func (c *Conn) IsOutgoing() (ok bool) {
	return c.incoming == false
}

// Node returns related Node
func (c *Conn) Node() (node *Node) {
	return c.n
}

// Address returns remote address
// represetned as string
func (c *Conn) Address() (address string) {
	return c.GetRemoteAddr().String()
}

// Feeds returns list of feeds this connection
// share with peer
func (c *Conn) Feeds() (feeds []cipher.PubKey) {
	c.mx.Lock()
	defer c.mx.Unlock()

	feeds = make([]cipher.PubKey, 0, len(c.feeds))

	for pk := range c.feeds {
		feeds = append(feeds, pk)
	}

	return

}

func connString(isIncoming, isTCP bool, addr string) (s string) {

	if isIncoming == true {
		s = "-> "
	} else {
		s = "<- "
	}

	if isTCP == true {
		s += "tcp://"
	} else {
		s += "udp://"
	}

	return s + addr
}

// String returns string "-> network://remote_address"
// for example: "-> tcp://127.0.0.1:8887". Where the
// arrow is "->" for incoming connections and is "<-"
// for outgoing
func (c *Conn) String() (s string) {
	return connString(c.incoming, c.tcp, c.Address())
}

//
// requests
//

// RemoteFeeds requests list of feeds that remote peer share.
// It's possible if the remote peer is public server, otherwise
// it returns "not a public server" error. The request has
// timeout configured by Config
func (c *Conn) RemoteFeeds() (feeds []cipher.PubKey, err error) {

	// TODO

	return
}

// Subscribe to gievn feed of remote peer
func (c *Conn) Subscribe(feed cipher.PubKey) (err error) {

	// TODO

	return

}

// Unsubscribe from given feed of remote peer
func (c *Conn) Unsubscribe(feed cipher.PubKey) (err error) {

	// TODO

	return

}

//
// terminate
//

// Close the Conn
func (c *Conn) Close() (err error) {

	c.closeo.Do(func() {

		// TODO

		c.Connection.Close()
	})

	return
}

func (c *Conn) nextSeq() uint32 {
	return atomic.AddUint32(&c.seq, 1)
}

func (c *Conn) encodeMsg(seq, rseq uint32, m msg.Msg) (raw []byte) {

	var em = m.Encode()

	raw = make([]byte, 8, 8+len(em))

	binary.LittleEndian.PutUint32(raw, seq)
	binary.LittleEndian.PutUint32(raw[:4], rseq)

	raw = append(raw, em...)

	return

}

func (c *Conn) sendMsg(seq, rseq uint32, m msg.Msg) {
	c.sendRaw(c.encodeMsg(seq, rseq, m))
}

func (c *Conn) sendRaw(raw []byte) {

	select {
	case c.sendq <- raw:
	case <-c.closeq:
	}

}

func (c *Conn) closeWithError(err error) {

	// TODO

}

func (c *Conn) fatality(args ...interface{}) {

	var err = errors.New(fmt.Sprint(args...))

	c.n.Print("[ERR] ", err)
	c.closeWithError(err)
}

func (c *Conn) receiving() {

	defer c.await.Done()

	var (
		receiveq = c.GetChanIn()
		closeq   = c.closeq

		seq, rseq uint32
		m         msg.Msg
		err       error

		raw []byte
		ok  bool
	)

	for {

		select {

		case raw, ok = <-receiveq:

			if ok == false {
				return
			}

			// [ 4 seq ][ 4 rseq ][ 1 msg type ]

			if len(raw) < 9 {
				c.fatality("invalid messege received: samll size")
				return
			}

			// seq of the Msg
			seq = binary.LittleEndian.Uint32(raw)
			raw = raw[4:]

			// response for a seq or zero
			rseq = binary.LittleEndian.Uint32(raw)
			raw = raw[4:]

			if m, err = msg.Decode(raw); err != nil {
				c.fatality("can't decode received messege: ", err)
				return
			}

			// the messege can be a response for a request
			if rq, ok := c.isResponse(rseq); ok == true {
				rq <- m
				continue
			}

			if err = c.handle(seq, m); err != nil {
				c.fatality("error handling messege: ", err)
				return
			}

		case <-closeq:
			return

		}

	}

}

func (c *Conn) isResponse(rseq uint32) (rq chan<- msg.Msg, ok bool) {
	c.mx.Lock()
	defer c.mx.Unlock()

	rq, ok = c.reqs[rseq]
	return
}

func (c *Conn) addRequest(seq uint32, rq chan<- msg.Msg) {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.reqs[seq] = rq
}

func (c *Conn) delRequest(seq uint32) {
	c.mx.Lock()
	defer c.mx.Unlock()

	delete(c.reqs, seq)
}

func (c *Conn) sendRequest(m msg.Msg) (reply msg.Msg, err error) {

	var (
		tr *time.Timer
		tc <-chan time.Time
	)

	if rt := c.n.config.ResponseTimeout; rt > 0 {
		tr = time.NewTimer(rt)
		tc = tr.C

		defer tr.Stop()
	}

	var (
		rq  = make(chan msg.Msg)
		seq = c.nextSeq()
	)

	c.addRequest(seq, rq)
	defer c.delRequest(seq)

	c.sendMsg(seq, 0, m)

	select {
	case rq <- reply:

		return

	case <-tc:

		return nil, ErrTimeout

	case <-c.closeq:

		return nil, ErrClosed
	}

}

func (c *Conn) sendErr(rseq uint32, err error) {
	c.sendMsg(c.nextSeq(), rseq, &msg.Err{err.Error()})
}

func (c *Conn) sendOk(rseq uint32) {
	c.sendMsg(c.nextSeq(), rseq, &msg.Ok{})
}

// handle messeges except responses and handshakes
func (c *Conn) handle(seq uint32, m msg.Msg) (err error) {

	switch x := m.(type) {

	// subscriptions

	case *msg.Sub: // <- Sub (feed)

		return c.handleSub(seq, x)

	case *msg.Unsub: // <- Unsub (feed)

		return c.handleUnsub(seq, x)

	// public server features

	case *msg.RqList: // <- RqList ()

		return c.handleRqList(seq, x)

	// the *List is response and handled outside the handle()

	// root (push and done)

	case *msg.Root: // <- Root (feed, nonce, seq, sig, val)

		return c.handleRoot(seq, x)

	case *msg.RootDone: // -> RD   (feed, nonce, seq)

		return c.handleRootDone(seq, x)

	// obejcts

	case *msg.RqObject: // <- RqO (key, prefetch)

		return c.handleRqObject(seq, x)

	case *msg.Object: // -> O   (val, vals)

		return c.handleObject(seq, x)

	// preview

	case *msg.RqPreview: // -> RqPreview (feed)

		return c.handleRqPreview(seq, x)

	default:

		return fmt.Errorf("invalid messege type %T", m)

	}

}

// subscribe
func (c *Conn) handleSub(seq uint32, sub *msg.Sub) (err error) {

	//

	return
}

// unsubscribe
func (c *Conn) handleUnsub(seq uint32, sub *msg.Unsub) (err error) {

	//

	return
}

// request list of feeds
func (c *Conn) handleRqList(seq uint32, sub *msg.RqList) (err error) {

	if c.n.config.Public == false {
		c.sendErr(seq, ErrNotPublic)
		return
	}

	// TODO

	return
}

func (c *Conn) handleRoot(seq uint32, sub *msg.Root) (err error) {

	//

	return
}

func (c *Conn) handleRootDone(seq uint32, sub *msg.RootDone) (err error) {

	//

	return
}

func (c *Conn) handleRqObject(seq uint32, sub *msg.RqObject) (err error) {

	//

	return
}

func (c *Conn) handleObject(seq uint32, sub *msg.Object) (err error) {

	//

	return
}

func (c *Conn) handleRqPreview(seq uint32, sub *msg.RqPreview) (err error) {

	//

	return
}
