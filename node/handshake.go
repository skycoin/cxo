package node

import (
	"errors"
	"fmt"
	"time"

	"github.com/skycoin/cxo/node/msg"
)

func (c *Conn) handshake(nodeCloseq <-chan struct{}) (err error) {

	c.n.Debugf(ConnHskPin, "[%s] handshake", c.String())

	if c.incoming == true {
		return c.acceptHandshake(nodeCloseq)
	}

	return c.performHandshake(nodeCloseq)
}

func (c *Conn) sendNodeCloseq(
	raw []byte,
	nodeCloseq <-chan struct{},
) (
	err error,
) {

	select {
	case c.sendq <- raw:
	case <-nodeCloseq:
		err = ErrClosed
	}

	return

}

func (c *Conn) performHandshake(nodeCloseq <-chan struct{}) (err error) {

	c.n.Debugf(ConnHskPin, "[%s] performHandshake", c.String())

	// (1) send Syn
	// (2) receive Ack or Err

	var seq = c.nextSeq()

	err = c.sendNodeCloseq(
		c.encodeMsg(seq, 0, &msg.Syn{
			Protocol: msg.Version,
			NodeID:   c.n.idpk,
		}),
		nodeCloseq,
	)

	if err != nil {
		return
	}

	var (
		rt time.Duration

		tm *time.Timer
		tc <-chan time.Time
	)

	if c.IsTCP() == true {
		rt = c.n.config.TCP.ResponseTimeout
	} else {
		rt = c.n.config.UDP.ResponseTimeout
	}

	if rt > 0 {
		tm = time.NewTimer(rt)
		tc = tm.C

		defer tm.Stop()
	}

	select {

	case raw, ok := <-c.GetChanIn():

		if ok == false {
			return ErrClosed
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

			c.peerID = x.NodeID

			return // ok

		case *msg.Err:

			return errors.New(x.Err)

		default:

			return fmt.Errorf("invalid response type for handshake: %T", m)

		}

	case <-tc:

		return ErrTimeout

	case <-nodeCloseq:

		return ErrClosed

	}

}

func (c *Conn) acceptHandshake(nodeCloseq <-chan struct{}) (err error) {

	c.n.Debugf(ConnHskPin, "[%s] acceptHandshake", c.String())

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
			return ErrClosed
		}

	case <-nodeCloseq:
		return ErrClosed
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

			c.sendNodeCloseq(
				c.encodeMsg(c.nextSeq(), seq, &msg.Err{Err: err.Error()}),
				nodeCloseq,
			)

			return

		}

		c.peerID = x.NodeID

		// (2) send Ack back

		err = c.sendNodeCloseq(
			c.encodeMsg(c.nextSeq(), seq, &msg.Ack{
				NodeID: c.n.idpk,
			}),
			nodeCloseq,
		)

		return

	default:

		return fmt.Errorf(
			"invalid messege type received (expected handshake): %T",
			m,
		)

	}

}
