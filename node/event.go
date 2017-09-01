package node

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node/msg"
)

type event interface {
	Handle(*Conn)
}

type subscribeEvent struct {
	pk  cipher.PubKey
	err chan error
}

func (s *subscribeEvent) Handle(c *Conn) {
	sub := c.s.src.Subscribe(s.pk)
	if _, err := c.sendRequest(sub.ID, sub, s.err, nil); err != nil {
		s.err <- err
	}
}

func (c *Conn) subscribeEvent(pk cipher.PubKey, err chan error) {
	c.enqueueEvent(&subscribeEvent{pk, err})
}

type unsubscribeEvent struct {
	pk cipher.PubKey
}

func (u *unsubscribeEvent) Handle(c *Conn) {
	c.unsubscribe(u.pk)               //
	c.Send(c.s.src.Unsubscribe(u.pk)) // send Unsubscribe message to peer
}

func (c *Conn) unsubscribeEvent(pk cipher.PubKey) {
	c.enqueueEvent(&unsubscribeEvent{pk})
}

type unsubscribeFromDeletedFeedEvent struct {
	pk   cipher.PubKey
	done chan struct{}
}

func (u *unsubscribeFromDeletedFeedEvent) Handle(c *Conn) {
	defer close(u.done)

	if _, ok := c.subs[u.pk]; ok {
		c.unsubscribe(u.pk) //
		c.Send(c.s.src.Unsubscribe(u.pk))
	}
}

type listOfFeedsEvent struct {
	rspn chan msg.Msg
	err  chan error
}

func (s *listOfFeedsEvent) Handle(c *Conn) {
	rlof := c.s.src.RequestListOfFeeds()
	if _, err := c.sendRequest(rlof.ID, rlof, s.err, s.rspn); err != nil {
		s.err <- err
	}
}

func (c *Conn) listOfFeedsEvent(rspn chan msg.Msg, err chan error) {
	c.enqueueEvent(&listOfFeedsEvent{rspn, err})
}

type resubscribeEvent struct{}

func (resubscribeEvent) Handle(c *Conn) {
	// TODO (kostyarin): resubscribe
	//
	// connection is not created inside the onDial
	// callback and we have a risk to overfill
	// SendQueue and terminate the conn this way
	//
	// so, we can only wait for some message to
	// resubscribe
}

type sendPingEvent struct {
	ttm chan struct{}
}

func (s *sendPingEvent) Handle(c *Conn) {
	ping := c.s.src.Ping()
	c.pings[ping.ID] = s.ttm
	c.Send(ping)
	go c.waitPong(ping.ID, s.ttm)
}

func (c *Conn) waitPong(id msg.ID, ttm chan struct{}) {

	pi := c.s.conf.PingInterval * 2

	tm := time.NewTimer(pi) // x 2
	defer tm.Stop()

	select {
	case <-tm.C:
		c.s.Printf("[WRN] [%s] pong not received after %v, terminate",
			c.Address(), pi)
		c.Close()
	case <-ttm:
		delete(c.pings, id)
		return
	}

}
