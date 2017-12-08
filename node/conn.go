package node

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/net/conn"

	"github.com/skycoin/cxo/node/msg"
	"github.com/skycoin/cxo/skyobject/registry"
)

// feed of a connection
type connFeed struct {

	// currently filling Root object
	fill *registry.Root

	// a Root that waits to be filled;
	// only one pending Root allowed,
	// if remote peer send new Root to
	// fill, then this pending Root
	// will be repalced with new one,
	// because the node worries about
	// latest Root only, and the node
	// fills one Root per connection-
	// -feed at the same time
	pending *registry.Root

	//

}

// A Conn represent connection of the Node
type Conn struct {
	conn.Connection

	// lock
	mx sync.Mutex

	// is incoming or not
	incoming bool

	// back reference
	n *Node

	// feeds this connection share with peer
	feeds map[cipher.PubKey]*connFeed

	// amount of Root obejcts currently
	// filling by the Conn
	fillingRoots int

	// messege seq number (for request-response)
	seq uint32

	// requests
	reqs map[uint32]<-chan msg.Msg

	// channels (from factory.Connection)
	sendq    chan<- []byte
	receiveq <-chan []byte
	closeq   <-chan struct{}

	// close once
	closeo sync.Once
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

//
// info
//

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

	feeds = make([]cipher.PubKey, len(c.feeds))

	//// TODO ///////////////
	//
	copy(feeds, c.feeds)
	//
	/////////////////////////

	return

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
	return atomic.AddUint32(c.seq, 1)
}

func (c *Conn) sendMsg(seq, rseq uint32, m msg.Msg) {
	c.send(seq, rseq, m.Encode())
}

func (c *Conn) sendRaw(seq, rseq uint32, raw []byte) {

	var raws = make([]byte, 8, 8+len(raw))

	binary.LittleEndian.PutUint32(raws, seq)
	binary.LittleEndian.PutUint32(raws[:4], rseq)

	raws = append(raws, raw)

	select {
	case c.sendq <- raws:
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

	var (
		seq, rseq uint32
		m         msg.Msg
		err       error
	)

	for raw := range c.receiveq {

		// the connection can be closed
		select {
		case <-c.closeq:
			return
		default:
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

	}

}

func (c *Conn) isResponse(rseq uint32) (rq <-chan msg.Msg, ok bool) {
	c.mx.Lock()
	defer c.mx.Unlock()

	return c.reqs[rseq]
}

func (c *Conn) addRequest(seq uint32, rq <-chan msg.Msg) {
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
	case reply <- rq:

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

	case *Sub: // <- Sub (feed)

		return c.handleSub(seq, x)

	case *Unsub: // <- Unsub (feed)

		return c.handleUnsub(seq, x)

	// public server features

	case *RqList: // <- RqList ()

		return c.handleRqList(seq, x)

	// the *List is response and handled outside the handle()

	// root (push and done)

	case *Root: // <- Root (feed, nonce, seq, sig, val)

		return c.handleRoot(seq, x)

	case *RootDone: // -> RD   (feed, nonce, seq)

		return c.handleRootDone(seq, x)

	// obejcts

	case *RqObject: // <- RqO (key, prefetch)

		return c.handleRqObject(seq, x)

	case *Object: // -> O   (val, vals)

		return c.handleObject(seq, x)

	// preview

	case *RqPreview: // -> RqPreview (feed)

		return c.handleRqPreview(seq, x)

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

	if c.n.config.Pings == false {
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
