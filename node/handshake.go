package node

import (
	uuid "github.com/satori/go.uuid"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"
)

func init() {
	// gnet.Message interface
	var (
		_ gnet.Message = &hsQuest{}
		_ gnet.Message = &hsQuestAnswer{}
		_ gnet.Message = &hsAnswer{}
		_ gnet.Message = &hsSuccess{}
	)
	// Register handshake related messages
	// (must be non-pointer)
	gnet.RegisterMessage(
		gnet.MessagePrefixFromString("HSQS"),
		hsQuest{})
	gnet.RegisterMessage(
		gnet.MessagePrefixFromString("HSQA"),
		hsQuestAnswer{})
	gnet.RegisterMessage(
		gnet.MessagePrefixFromString("HSAN"),
		hsAnswer{})
	gnet.RegisterMessage(
		gnet.MessagePrefixFromString("HSSC"),
		hsSuccess{})

	gnet.VerifyMessages()
}

// outgoing -> question -> incoming (0, 0) (1, 1)
// incoming -> qa       -> outgoing (1, 1) (2, 2)
// outgoing -> answer   -> incoming (2, 2) (3, 3)
// incoming -> success  -> outgoing (3, 3) (10, 10)

type hsQuest struct {
	Pub   cipher.PubKey
	Quest uuid.UUID
}

// handled by feed
func (q *hsQuest) Handle(ctx *gnet.MessageContext,
	ni interface{}) error {

	var (
		n  Node             = ni.(Node)
		gc *gnet.Connection = ctx.Conn
	)

	n.Debugf("[DBG] got HSQS from: %s, public key: %s",
		gc.Addr(),
		q.Pub.Hex())

	// first of all we need to process all incoming connections
	n.(*node).handleIncomingConnections()

	pend, ok := n.pend(gc)
	if !ok {
		n.Print("[WRN] HSQS: connection not in pendings list")
		return ErrUnexpectedHandshake
	}

	// must be incoming connection
	if pend.outgoing {
		n.Print("[ERR] HSQS: got on outgoing connection")
		return ErrUnexpectedHandshake
	}

	if q.Pub == (cipher.PubKey{}) || q.Quest == (uuid.UUID{}) {
		n.Print("[ERR] HSQS: empty message")
		return ErrMalformedHandshake
	}

	if pend.step != 0 {
		n.Print("[ERR] HSQS: invalid handshake step: want 0, got ", pend.step)
		return ErrUnexpectedHandshake
	}

	pend.remotePub = q.Pub

	//
	// send question answer back
	//

	pend.step++
	pend.quest = uuid.NewV4()

	n.Debug("[DBG] HSQS send HSQA")
	gc.ConnectionPool.SendMessage(gc, &hsQuestAnswer{
		Pub:    n.PubKey(),
		Quest:  pend.quest,
		Answer: n.Sign(cipher.SumSHA256(q.Quest.Bytes())),
	})

	return nil
}

type hsQuestAnswer struct {
	Pub    cipher.PubKey
	Quest  uuid.UUID
	Answer cipher.Sig
}

func (qa *hsQuestAnswer) Handle(ctx *gnet.MessageContext,
	ni interface{}) error {

	var (
		n  Node             = ni.(Node)
		gc *gnet.Connection = ctx.Conn

		err error
	)

	n.Debugf("[DBG] got HSQA from: %s, public key: %s",
		gc.Addr(),
		qa.Pub.Hex())

	pend, ok := ni.(Node).pend(gc)
	if !ok {
		// may be it was deleted by handshake timeout
		n.Print("[WRN] HSQA: connection not in pendings list")
		return ErrUnexpectedHandshake
	}

	// must be outgoing connection
	if !pend.outgoing {
		n.Print("[ERR] HSQA: got on incoming connection")
		return ErrUnexpectedHandshake
	}

	if qa.Pub == (cipher.PubKey{}) ||
		qa.Quest == (uuid.UUID{}) ||
		qa.Answer == (cipher.Sig{}) {
		n.Print("[ERR] HSQA: empty message")
		return ErrMalformedHandshake
	}

	if pend.step != 1 {
		n.Print("[ERR] HSQA: invalid handshake step: want 1, got ", pend.step)
		return ErrUnexpectedHandshake
	}

	// desired public key
	if pend.remotePub != (cipher.PubKey{}) && qa.Pub != pend.remotePub {
		n.Print("[ERR] HSQA: public key is not desired")
		return ErrPubKeyMismatch
	}

	pend.remotePub = qa.Pub

	// verify answer
	err = cipher.VerifySignature(
		qa.Pub,
		qa.Answer,
		cipher.SumSHA256(pend.quest.Bytes()))
	if err != nil {
		n.Print("[ERR] HSQA: bad answer")
		return gnet.DisconnectReason(err) // terminate connection
	}

	//
	// send answer back
	//

	pend.step++

	n.Debug("[DBG] HSQA send HSAN")
	gc.ConnectionPool.SendMessage(gc, &hsAnswer{
		Answer: n.Sign(cipher.SumSHA256(qa.Quest.Bytes())),
	})

	return nil
}

type hsAnswer struct {
	Answer cipher.Sig
}

func (a *hsAnswer) Handle(ctx *gnet.MessageContext,
	ni interface{}) error {

	var (
		n  Node             = ni.(Node)
		gc *gnet.Connection = ctx.Conn

		err error
	)

	n.Debug("[DBG] got HSAN from: ", gc.Addr())

	pend, ok := ni.(Node).pend(gc)
	if !ok {
		// may be it was deleted by handshake timeout
		n.Print("[WRN] HSAN: connection not in pendings list")
		return ErrUnexpectedHandshake
	}

	// must be incoming connection
	if pend.outgoing {
		n.Print("[ERR] HSAN: got on outging connection")
		return ErrUnexpectedHandshake
	}

	if a.Answer == (cipher.Sig{}) {
		n.Print("[ERR] HSAN: empty message")
		return ErrMalformedHandshake
	}

	if pend.step != 1 {
		n.Print("[ERR] HSAN: invalid handshake step: want 1, got ", pend.step)
		return ErrUnexpectedHandshake
	}

	// verify answer
	err = cipher.VerifySignature(
		pend.remotePub,
		a.Answer,
		cipher.SumSHA256(pend.quest.Bytes()))
	if err != nil {
		n.Print("[ERR] HSAN: bad answer")
		return gnet.DisconnectReason(err) // terminate connection
	}

	//
	// check public key and add

	if err = n.addIncoming(gc, pend.remotePub); err != nil {
		n.Print("[ERR] HSAN already have incoming connection to %s",
			pend.remotePub.Hex())
		return err
	}

	//
	// send success back
	//

	n.Debug("[DBG] HSAN send HSSC")
	gc.ConnectionPool.SendMessage(gc, &hsSuccess{})

	return nil
}

type hsSuccess struct{}

func (h *hsSuccess) Handle(ctx *gnet.MessageContext,
	ni interface{}) error {

	var (
		n  Node             = ni.(Node)
		gc *gnet.Connection = ctx.Conn

		err error
	)

	n.Debug("[DBG] got HSSC from: ", gc.Addr())

	pend, ok := ni.(Node).pend(gc)
	if !ok {
		// may be it was deleted by handshake timeout
		n.Print("[WRN] HSSC: connection not in pendings list")
		return ErrUnexpectedHandshake
	}

	// must be outgoing connection
	if !pend.outgoing {
		n.Print("[ERR] HSSC: got on incoming connection")
		return ErrUnexpectedHandshake
	}

	if pend.step != 2 {
		n.Print("[ERR] HSSC: invalid handshake step: want 2, got ", pend.step)
		return ErrUnexpectedHandshake
	}

	if err = n.addOutgoing(gc, pend.remotePub); err != nil {
		n.Print("[ERR] HSAN already have outgoing connection to ",
			pend.remotePub.Hex())
		return err
	}

	return nil
}
