package nodeManager

import (
	"errors"
	"fmt"
	"sync"

	uuid "github.com/satori/go.uuid"
	"github.com/skycoin/skycoin/src/cipher"
)

// ### THE HANDSHAKE ###
// Here are the message structs used to perform the handshake;
// underneath them, are listed the handle methods to
// process them.

// downstream -> upstream (HandleFromDownstream)
type hsm1 struct {
	// The handshake is initiated from downstream; the node wanting to connect
	// sends hsm1 to the node it wants to subscribe to;
	// hsm1 contains the PubKey of the downstream node, and
	// a question that the upstream node will have to answer (sign and return).
	PubKey   cipher.PubKey
	Question uuid.UUID
}

// upstream -> downstream (HandleFromUpstream)
type hsm2 struct {
	// The upstream node replies with its own PubKey and question,
	// and the answer (signature) to the question sent by the downstream node.
	PubKey   cipher.PubKey
	Question uuid.UUID

	Answer cipher.Sig
}

// downstream -> upstream (HandleFromDownstream)
type hsm3 struct {
	// The donwstream node answers to the question of the upstream node.
	Answer cipher.Sig
}

// upstream -> downstream (HandleFromUpstream)
type hsm4 struct {
	// upstream node replies that the answer is correct.
	Success []byte
}

// upstream receives hsm1
func (hsm *hsm1) HandleFromDownstream(cc *DownstreamContext) error {
	cc.Remote.hs.mu.Lock()
	defer cc.Remote.hs.mu.Unlock()

	debug("received hsm1 from downstream")

	if cc.Remote.hs.success {
		return cc.Remote.hs.Fail(fmt.Errorf("handshake from %v already done", cc.Remote.PubKey()))
	}
	if cc.Remote.hs.counter != 0 {
		return cc.Remote.hs.Fail(fmt.Errorf("wrong step number from %v (%v instead of 0)", cc.Remote.PubKey(), cc.Remote.hs.counter))
	}

	if hsm.PubKey == (cipher.PubKey{}) {
		return cc.Remote.hs.Fail(errors.New("PubKey is nil"))
	}
	if hsm.Question == (uuid.UUID{}) {
		return cc.Remote.hs.Fail(errors.New("Question is nil"))
	}

	question := uuid.NewV4()
	cc.Remote.hs.local.asked = question

	// register the pubKey of the downstream node
	cc.Remote.pubKey = &hsm.PubKey

	// hash the question
	hash := cipher.SumSHA256(hsm.Question.Bytes())
	// sign the hash
	answer := cipher.SignHash(hash, *cc.Local.secKey)
	// keep locally track of the sent answer
	cc.Remote.hs.remote.sentAnswer = answer

	// compose the reply message
	reply := hsm2{
		PubKey:   *cc.Local.pubKey,
		Question: question,
		Answer:   answer,
	}

	// send the reply
	err := cc.Remote.Send(reply)
	if err != nil {
		return cc.Remote.hs.Fail(err)
	}
	debug("sent hsm2")

	// increment counter of the handshake process to keep track of the current step
	cc.Remote.hs.counter += 1

	return nil
}

// downstream receives hsm2
func (hsm *hsm2) HandleFromUpstream(cc *UpstreamContext) error {
	cc.Remote.hs.mu.Lock()
	defer cc.Remote.hs.mu.Unlock()

	debug("received hsm2 from upstream")

	if cc.Remote.hs.success {
		return cc.Remote.hs.Fail(fmt.Errorf("handshake from %v already done", cc.Remote.PubKey()))
	}
	if cc.Remote.hs.counter != 1 {
		return cc.Remote.hs.Fail(fmt.Errorf("wrong step number from %v (%v instead of 0)", cc.Remote.PubKey(), cc.Remote.hs.counter))
	}

	if hsm.PubKey == (cipher.PubKey{}) {
		return cc.Remote.hs.Fail(errors.New("PubKey is nil"))
	}
	if hsm.Question == (uuid.UUID{}) {
		return cc.Remote.hs.Fail(errors.New("Question is nil"))
	}

	// hash the question I asked
	myQuestionHash := cipher.SumSHA256(cc.Remote.hs.local.asked.Bytes())
	// verify the answer provided by the upstream node
	err := cipher.VerifySignature(hsm.PubKey, hsm.Answer, myQuestionHash)
	if err != nil {
		return cc.Remote.hs.Fail(err)
	}

	// check whether the upstream node should match a specific PubKey
	if *cc.Remote.pubKey != (cipher.PubKey{}) {
		// check if the upstream node has the desired PubKey
		if *cc.Remote.pubKey != hsm.PubKey {
			return cc.Remote.hs.Fail(fmt.Errorf("PubKeys do not match: want %v, got %v", cc.Remote.pubKey.Hex(), hsm.PubKey.Hex()))
		}
	}
	// register pubKey in cc.Remote
	cc.Remote.pubKey = &hsm.PubKey

	// hash the question received from upstream
	hash := cipher.SumSHA256(hsm.Question.Bytes())
	// sign the hash
	answer := cipher.SignHash(hash, *cc.Local.secKey)
	// keep locally track of the sent answer
	cc.Remote.hs.remote.sentAnswer = answer

	// compose the reply
	reply := hsm3{
		Answer: answer,
	}

	// send the reply
	err = cc.Remote.Send(reply)
	if err != nil {
		return cc.Remote.hs.Fail(err)
	}
	debug("sent hsm3")

	// increment counter of the handshake process to keep track of the current step
	cc.Remote.hs.counter += 1

	return nil
}

// upstream receives hsm3
func (hsm *hsm3) HandleFromDownstream(cc *DownstreamContext) error {
	cc.Remote.hs.mu.Lock()
	defer cc.Remote.hs.mu.Unlock()

	debug("received hsm3 from downstream")

	if cc.Remote.hs.success {
		return cc.Remote.hs.Fail(fmt.Errorf("handshake from %v already done", cc.Remote.PubKey()))
	}
	if cc.Remote.hs.counter != 1 {
		return cc.Remote.hs.Fail(fmt.Errorf("wrong step number from %v (%v instead of 0)", cc.Remote.PubKey(), cc.Remote.hs.counter))
	}

	if hsm.Answer == (cipher.Sig{}) {
		return cc.Remote.hs.Fail(errors.New("Answer is nil"))
	}

	// hash question I asked
	myQuestionHash := cipher.SumSHA256(cc.Remote.hs.local.asked.Bytes())
	// verify the signature (it should have been signed by the downstream node)
	err := cipher.VerifySignature(*cc.Remote.pubKey, hsm.Answer, myQuestionHash)
	if err != nil {
		return cc.Remote.hs.Fail(err)
	}

	// compose the reply
	reply := hsm4{
		Success: []byte("true"),
	}

	// send reply
	err = cc.Remote.Send(reply)
	if err != nil {
		return cc.Remote.hs.Fail(err)
	}
	debug("sent hsm4")

	// increment counter of the handshake process to keep track of the current step
	cc.Remote.hs.counter += 1

	return cc.Remote.hs.Success()
}

// downstream receives hsm4
func (hsm *hsm4) HandleFromUpstream(cc *UpstreamContext) error {
	cc.Remote.hs.mu.Lock()
	defer cc.Remote.hs.mu.Unlock()

	debug("received hsm4 (success) from upstream")

	if cc.Remote.hs.success {
		return cc.Remote.hs.Fail(fmt.Errorf("hs from %v already done", cc.Remote.PubKey()))
	}
	if cc.Remote.hs.counter != 2 {
		return cc.Remote.hs.Fail(fmt.Errorf("wrong step number from %v (%v instead of 0)", cc.Remote.PubKey(), cc.Remote.hs.counter))
	}

	// increment counter of the handshake process to keep track of the current step
	cc.Remote.hs.counter += 1

	return cc.Remote.hs.Success()
}

func (hs *handshakeStatus) Success() error {
	var err error
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("Unable to send: %v", x)
		}
	}()
	// mark handshake as successfully completed
	hs.success = true

	// send event
	hs.waitForSuccess <- nil
	return err
}

func (hs *handshakeStatus) Fail(reason error) error {
	var err error
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("Unable to send: %v", x)
		}
	}()
	// mark handshake as failed
	hs.success = false

	// send event
	hs.waitForSuccess <- fmt.Errorf("Handshake FAILED: %v ", reason)
	return err
}

type handshakeStatus struct {
	mu *sync.RWMutex

	success        bool
	waitForSuccess chan error

	local struct {
		asked          uuid.UUID
		receivedAnswer cipher.Sig
	}

	remote struct {
		asked      uuid.UUID
		sentAnswer cipher.Sig
	}
	counter uint32
}

func newHandshakeStatus() *handshakeStatus {
	return &handshakeStatus{
		mu:             &sync.RWMutex{},
		waitForSuccess: make(chan error),
	}
}
