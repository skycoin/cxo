package node

import (
	"log"

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

//
// Handshakes
//

// inflow -> feed (question { inflow_pubKey, inflow_question })
// feed -> inflow (qa { feed_pubKey, feed_question, inflow_answer, })
// inflow -> feed (answer { feed_answer })
// feed -> inflow (success { "success" })

//
// handshake processing
//

type hsp struct {
	// steps
	// inflow:
	//   0 - send question
	//   1 - receive qa
	//   2 - receive success
	// feed:
	//   0 - receive question
	//   1 - receive answer
	step     int           // handshake step (from 0)
	question uuid.UUID     // my question
	pub      cipher.PubKey // remote public key
}

// inflow -> feed
// pub is local (inflow's) public key
// qusetion is generated automatically and stored in hsp
func (h *hsp) hsQuestion(n Node) *hsQuestion {

	h.step++ // next step
	h.question = uuid.NewV4()

	return &hsQuestion{
		PubKey:   n.PubKey(), // public key of inflow
		Question: h.question, // inflow->feed question
	}
}

// feed -> inflow
// q is received hsQuestion
// question for remote inflow node generated automatically and stored in hsp
func (h *hsp) hsQuestionAnswer(q *hsQuestion, n Node) *hsQuestionAnswer {

	h.step++ // next step
	h.question = uuid.NewV4()

	return &hsQuestionAnswer{
		PubKey:   n.PubKey(), // public key of feed
		Question: h.question, // feed->inflow question
		Answer: n.Sign(
			cipher.SumSHA256(q.Question.Bytes())), // feed->inflow answer
	}
}

// inflow -> feed
// qa is received hsQuestionAnswer
// answer for received hsQuestionAnswer,
func (h *hsp) hsAnswer(qa *hsQuestionAnswer, n Node) *hsAnswer {

	h.step++ // next step

	return &hsAnswer{
		Answer: n.Sign(
			cipher.SumSHA256(qa.Question.Bytes())), // inflow->feed answer
	}
}

// feed -> inflow
// success report (hsAnswer is correct)
func (*hsp) hsSuccess() *hsSuccess {
	return &hsSuccess{}
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
		log.Panicf("state of pool doesn't set to Node: %T", ni)
	}

	if node == nil {
		log.Panic("state of pool is nil")
	}

	return
}
