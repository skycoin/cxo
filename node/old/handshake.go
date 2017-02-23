package node

import (
	"github.com/satori/go.uuid"
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/skycoin/src/daemon/gnet"
)

func init() {
	// gnet.Message interface
	var (
		_ gnet.Message = &hsQuestion{}
		_ gnet.Message = &hsQuestionAnswer{}
		_ gnet.Message = &hsAnswer{}
		_ gnet.Message = &hsSuccess{}
	)
	// Register handshake related messages
	// (must be non-pointer)
	gnet.RegisterMessage(
		gnet.MessagePrefixFromString("HSQS"),
		hsQuestion{})
	gnet.RegisterMessage(
		gnet.MessagePrefixFromString("HSQA"),
		hsQuestionAnswer{})
	gnet.RegisterMessage(
		gnet.MessagePrefixFromString("HSAN"),
		hsAnswer{})
	gnet.RegisterMessage(
		gnet.MessagePrefixFromString("HSSC"),
		hsSuccess{})

	gnet.VerifyMessages()
}

// hsQuestion is initial handshake message that
// subscriber sends
type hsQuestion struct {
	PubKey   cipher.PubKey
	Question uuid.UUID
}

func (h *hsQuestion) Handle(ctx *gnet.MessageContext, ni interface{}) error {
	return getNodeFromState(ni).hsQuestion(h, ctx)
}

// hsQuestionAnswer is reply for first handshake message.
// It contains answer
type hsQuestionAnswer struct {
	PubKey   cipher.PubKey
	Question uuid.UUID
	Answer   cipher.Sig // answer for hsQuestion.Question
}

func (h *hsQuestionAnswer) Handle(ctx *gnet.MessageContext,
	ni interface{}) error {

	return getNodeFromState(ni).hsQuestionAnswer(h, ctx)
}

// hsAnswer is reply for hsQuestionAnswer
type hsAnswer struct {
	Answer cipher.Sig // answer for hsQuestionAnswer.Question
}

func (h *hsAnswer) Handle(ctx *gnet.MessageContext, ni interface{}) error {
	return getNodeFromState(ni).hsAnswer(h, ctx)
}

// hsSuccess is success report
type hsSuccess struct{}

func (h *hsSuccess) Handle(ctx *gnet.MessageContext, ni interface{}) error {
	return getNodeFromState(ni).hsSuccess(h, ctx)
}

//
// Helpers
//

// getNodeFromState convertes interface{} to Node;
// it panics if given interface is not Node or nil
func getNodeFromState(ni interface{}) (node Node) {
	var ok bool

	if node, ok = ni.(Node); !ok {
		panic("state of pool doesn't set to Node")
	}

	if node == nil {
		panic("state of pool is nil")
	}

	return
}

// handled by feed
func (n *node) hsQuestion(q *hsQuestion, ctx *gnet.MessageContext) error {
	n.Debugf("got handshake question from %v, public key of wich is %s",
		ctx.Conn.Conn.RemoteAddr(),
		q.PubKey.Hex())

	var (
		c  *conn
		ok bool
	)
	n.backmx.RLock()
	defer n.backmx.RUnlock()
	if c, ok = n.back[ctx.Conn]; !ok {
		return ErrUnexpectedHandshake // terminate
	}
	// hsQuestion must be handled by feed
	if !c.isFeed() || !c.isPending() {
		return ErrUnexpectedHandshake // terminate connection
	}
	// check handshake step
	if c.step != hsStepInit {
		return ErrUnexpectedHandshake // terminate connection
	}
	// check handshake message
	if q.PubKey == (cipher.PubKey{}) || q.Question == (uuid.UUID{}) {
		return ErrMalformedHandshake // terminate connection
	}
	c.pub = q.PubKey
	// answer and question
	ctx.Conn.ConnectionPool.SendMessage(ctx.Conn, c.QuestionAnswer(q, n))

	return nil
}

// handled by inflow
func (n *node) hsQuestionAnswer(qa *hsQuestionAnswer,
	ctx *gnet.MessageContext) error {

	n.Debugf(
		"got handshake question-answer from %v, public key of wich is %s",
		ctx.Conn.Conn.RemoteAddr(),
		qa.PubKey.Hex())

	var (
		c  *conn
		ok bool

		err error
	)
	n.backmx.RLock()
	defer n.backmx.RUnlock()
	if c, ok = n.back[ctx.Conn]; !ok {
		return ErrUnexpectedHandshake // terminate
	}
	// hsQuestionAnswer must be handled by inflow
	if c.isFeed() || !c.isPending() {
		return ErrUnexpectedHandshake // terminate connection
	}
	// check handshake step
	if c.step != hsStepSendQuestion {
		return ErrUnexpectedHandshake // terminate connection
	}
	// check handshake message
	if qa.PubKey == (cipher.PubKey{}) ||
		qa.Question == (uuid.UUID{}) ||
		qa.Answer == (cipher.Sig{}) {
		return ErrMalformedHandshake // terminate connection
	}
	// desired public key
	if c.pub != (cipher.PubKey{}) && qa.PubKey != c.pub {
		return ErrPubKeyMismatch // terminate
	}
	c.pub = qa.PubKey
	//
	// verify answer
	err = cipher.VerifySignature(
		qa.PubKey,
		qa.Answer,
		cipher.SumSHA256(c.quest.Bytes()))
	if err != nil {
		return gnet.DisconnectReason(err) // terminate connection
	}
	// answer and question
	ctx.Conn.ConnectionPool.SendMessage(ctx.Conn, c.Answer(qa, n))

	return nil
}

// handled by feed
func (n *node) hsAnswer(a *hsAnswer, ctx *gnet.MessageContext) error {

	n.Debugf("got handshake answer from %v",
		ctx.Conn.Conn.RemoteAddr())

	var (
		c  *conn
		ok bool

		err error
	)
	n.backmx.RLock()
	defer n.backmx.RUnlock()
	if c, ok = n.back[ctx.Conn]; !ok {
		return ErrUnexpectedHandshake // terminate
	}
	// hsAnswer must be handled by feed
	if !c.isFeed() || !c.isPending() {
		return ErrUnexpectedHandshake // terminate connection
	}
	// check handshake step
	if c.step != hsStepSendQA {
		return ErrUnexpectedHandshake // terminate connection
	}
	// check handshake message
	if a.Answer == (cipher.Sig{}) {
		return ErrMalformedHandshake // terminate connection
	}
	// verify answer
	err = cipher.VerifySignature(
		c.pub,    // remote public key
		a.Answer, // received answer
		cipher.SumSHA256(c.quest.Bytes())) // my question hash
	if err != nil {
		return gnet.DisconnectReason(err) // terminate connection
	}
	// do we already have the public key?
	for _, bc := range n.back {
		if !bc.isFeed() || bc.isPending() || bc.pub != c.pub {
			continue
		}
		// we alrady have a subscriber with the same public key
		return ErrAlreadyConnected
	}
	// established
	c.step = hsStepSuccess
	close(c.done)
	// success, yay!
	ctx.Conn.ConnectionPool.SendMessage(ctx.Conn, c.Success(n))

	return nil
}

// handled by inflow
func (n *node) hsSuccess(s *hsSuccess, ctx *gnet.MessageContext) error {

	n.Debugf("got handshake success from %v",
		ctx.Conn.Conn.RemoteAddr())

	var (
		c  *conn
		ok bool
	)
	n.backmx.RLock()
	defer n.backmx.RUnlock()
	if c, ok = n.back[ctx.Conn]; !ok {
		return ErrUnexpectedHandshake // terminate
	}
	// hsSucces must be handled by inflow
	if c.isFeed() || !c.isPending() {
		return ErrUnexpectedHandshake // terminate connection
	}
	// check handshake step
	if c.step != 3 {
		return ErrUnexpectedHandshake // terminate connection
	}
	// do we already have the public key?
	for _, bc := range n.back {
		if bc.isFeed() || bc.isPending() || bc.pub != c.pub {
			continue
		}
		// we alrady have a subscription with the same public key
		return ErrAlreadyConnected
	}
	// established
	c.step = hsStepSuccess
	close(c.done)
	n.Debugf("handshake success: %v",
		ctx.Conn.Conn.RemoteAddr())
	// success, yay!
	// no more handshake messages

	return nil
}
