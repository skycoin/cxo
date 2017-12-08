package node

import (
	"errors"
	"fmt"
	"time"

	"github.com/skycoin/cxo/node/msg"
)

func (c *Conn) handshake(nodeQuiting <-chan struct{}) (err error) {

	if c.incoming == true {
		return c.acceptHandshake(nodeQuiting)
	}

	return c.performHandshake(nodeQuiting)
}

func (c *Conn) sendWithQuiting(raw []byte, quiting <-chan struct{}) {

	select {
	case c.sendq <- raw:
	case <-quiting:
	}

}

func (c *Conn) performHandshake(nodeQuiting <-chan struct{}) (err error) {

	// (1) send Syn
	// (2) receive Ack or Err

	var seq = c.nextSeq()

	c.sendWithQuiting(
		c.encodeMsg(seq, 0, &msg.Syn{
			Protocol: msg.Version,
			NodePk:   c.n.id.Pk,
			NodeSk:   c.n.id.Sk,
		}),
		nodeQuiting,
	)

	var (
		tm *time.Timer
		tc <-chan time.Time
	)

	if rt := c.n.config.ResponseTimeout; rt > 0 {
		tm = time.NewTimer(rt)
		tc = tm.C

		defer tm.Stop()
	}

	select {

	case raw, ok := <-c.GetChanIn():

		if ok == false {
			return
		}

		var (
			rseq uint32
			m    msg.Msg
		)

		if _, rseq, m, err = c.decodeRaw(raw); err != nil {
			return
		}

		if rseq != seq {
			return errors.New("invlaid resposne for handshake: wrong seq")
		}

		switch x := m.(type) {

		case *msg.Ack:

			c.peerID.Pk = x.NodePk
			c.peerID.Sk = x.NodeSk

			return

		case *msg.Err:

			return errors.New(x.Err)

		default:

			return fmt.Errorf("invalid response type for handshake: %T", m)

		}

	case <-tc:

		return ErrTimeout

	case <-nodeQuiting:

		return

	}

}

func (c *Conn) acceptHandshake(nodeQuiting <-chan struct{}) (err error) {

	// (1) receive the Syn
	// (2) send the Ack or Err

	// (1)

	var (
		raw []byte
		ok  bool
	)

	select {
	case raw, ok = <-c.GetChanIn():

		if ok == false {
			return
		}

	case <-nodeQuiting:
		return
	}

	var (
		seq uint32
		m   msg.Msg
	)

	if seq, _, m, err = c.decodeRaw(raw); err != nil {
		return
	}

	switch x := m.(type) {

	case *msg.Syn:

		if x.Protocol != msg.Version {

			err = fmt.Errorf("incompatible protocol version: %d, want %d",
				x.Protocol,
				msg.Version)

			// send Err back

			c.sendWithQuiting(
				c.encodeMsg(c.nextSeq(), seq, &msg.Err{err.Error()}),
				nodeQuiting,
			)

			return

		}

		c.peerID.Pk = x.NodePk
		c.peerID.Sk = x.NodeSk

		// (2) send Ack back

		c.sendWithQuiting(
			c.encodeMsg(c.nextSeq(), seq, &msg.Ack{
				NodePk: c.n.id.Pk,
				NodeSk: c.n.id.Sk,
			}),
			nodeQuiting,
		)

		return

	default:

		return fmt.Errorf(
			"invalid messege type received (expected handshake): %T",
			m,
		)

	}

}
