package node

import (
	"fmt"

	"github.com/satori/go.uuid"

	"github.com/skycoin/skycoin/src/cipher"
)

type hsStep int

const (
	hsStepInit         hsStep = 0
	hsStepSendQuestion hsStep = 1
	hsStepSendQA       hsStep = 2
	hsStepSendAnswer   hsStep = 3

	hsStepSuccess hsStep = 4
)

type conn struct {
	feed  bool
	pub   cipher.PubKey
	quest uuid.UUID
	step  hsStep
	done  chan struct{}
}

func (c *conn) String() string {
	var (
		feed string = "inflow side"
		step string = "established"
		done string
	)
	if c.feed {
		feed = "feed side"
	}
	if c.step != hsStepSuccess {
		step = "pending"
	}
	select {
	case <-c.done:
		done = "done"
	default:
		done = "not done"
	}
	return fmt.Sprintf("conn{%s, %s, %s}", feed, step, done)
}

func (c *conn) isFeed() bool {
	return c.feed
}

func (c *conn) isPending() bool {
	return c.step != hsStepSuccess
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

// inflow -> feed
// pub is local (inflow's) public key
// qusetion is generated automatically and stored in conn
func (c *conn) Question(n Node) *hsQuestion {

	n.Debug("send handshake question")

	c.step = hsStepSendQuestion
	c.quest = uuid.NewV4()

	return &hsQuestion{
		PubKey:   n.PubKey(), // public key of inflow
		Question: c.quest,    // inflow->feed question
	}
}

// feed -> inflow
// q is received hsQuestion
// question for remote inflow node generated automatically and stored in conn
func (c *conn) QuestionAnswer(q *hsQuestion, n Node) *hsQuestionAnswer {

	n.Debug("send handshake qa")

	c.step = hsStepSendQA
	c.quest = uuid.NewV4()

	return &hsQuestionAnswer{
		PubKey:   n.PubKey(), // public key of feed
		Question: c.quest,    // feed->inflow question
		Answer: n.Sign(
			cipher.SumSHA256(q.Question.Bytes())), // feed->inflow answer
	}
}

// inflow -> feed
// qa is received hsQuestionAnswer
// answer for received hsQuestionAnswer,
func (c *conn) Answer(qa *hsQuestionAnswer, n Node) *hsAnswer {

	n.Debug("send handshake answer")

	c.step = hsStepSendAnswer

	return &hsAnswer{
		Answer: n.Sign(
			cipher.SumSHA256(qa.Question.Bytes())), // inflow->feed answer
	}
}

// feed -> inflow
// success report (hsAnswer is correct)
func (c *conn) Success(n Node) *hsSuccess {
	n.Debug("send handshake success")

	c.step = hsStepSuccess // that is hsStepSuccess

	return &hsSuccess{}
}
