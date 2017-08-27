package node

import (
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
	if _, err := c.sendRequest(sub.ID, sub); err != nil {
		s.err <- err
	}
}

func (c *Conn) subscribeEvent(pk cipher.PubKey, err chan error) {
	c.events <- &subscribeEvent{pk, err}
}

type unsubscribeEvent struct {
	pk cipher.PubKey
}

func (u *unsubscribeEvent) Handle(c *Conn) {
	c.s.delConnFromFeed(c, u.pk)
	delete(c.subs, u.pk)
	c.Send(c.s.src.Unsubscribe(u.pk))
}

func (c *Conn) unsubscribeEvent(pk cipher.PubKey) {
	c.events <- &unsubscribeEvent{pk, err}
}

type listOfFeedsEvent struct {
	rspn chan msg.Msg
	err  chan error
}

func (s *listOfFeedsEvent) Handle(c *Conn) {
	rlof := c.s.src.RequestListOfFeeds()
	if _, err := c.sendRequest(rlof.ID, rlof); err != nil {
		s.err <- err
	}
}

func (c *Conn) listOfFeedsEvent(rpsn chan msg.Msg, err chan error) {
	c.events <- &listOfFeedsEvent{pk, err}
}
