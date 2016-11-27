package nodeManager

import (
	"errors"
	"fmt"
	"sync/atomic"

	uuid "github.com/satori/go.uuid"
	"github.com/skycoin/skycoin/src/cipher"
)

// ### THE HANDSHAKE ###
// here are the message structs used to perform the handshake;
// underneath them, are listed the handle methods to
// process them.

// downstream -> upstream (HandleFromDownstream)
type hsm1 struct {
	PubKey   cipher.PubKey
	Question uuid.UUID
}

// upstream -> downstream (HandleFromUpstream)
type hsm2 struct {
	PubKey   cipher.PubKey
	Question uuid.UUID

	Answer cipher.Sig
}

// downstream -> upstream (HandleFromDownstream)
type hsm3 struct {
	// signature value of handshakeStep1.Question
	Answer cipher.Sig
}

// upstream -> downstream (HandleFromUpstream)
type hsm4 struct {
	Success []byte
}

func (hsm *hsm1) HandleFromDownstream(cc *DownstreamContext) error {

	debug("received hsm1")

	if cc.Remote.hs.success {
		return fmt.Errorf("handshake from %v already done", cc.Remote.PubKey())
	}
	if cc.Remote.hs.counter != 0 {
		return fmt.Errorf("wrong step number from %v (%v instead of 0)", cc.Remote.PubKey(), cc.Remote.hs.counter)
	}

	if hsm.PubKey == (cipher.PubKey{}) {
		return errors.New("PubKey is nil")
	}
	if hsm.Question == (uuid.UUID{}) {
		return errors.New("Question is nil")
	}

	question := uuid.NewV4()
	cc.Remote.hs.local.asked = question
	cc.Remote.pubKey = &hsm.PubKey

	// hash the question
	hash := cipher.SumSHA256(hsm.Question.Bytes())
	// sign the hash
	answer := cipher.SignHash(hash, *cc.Local.secKey)
	cc.Remote.hs.remote.sentAnswer = answer

	reply := hsm2{
		PubKey:   *cc.Local.pubKey,
		Question: question,
		Answer:   answer,
	}

	err := cc.Remote.Send(reply)
	if err != nil {
		return err
	}
	debug("sent hsm2")

	atomic.AddUint32(&cc.Remote.hs.counter, 1)

	return nil
}

func (hsm *hsm2) HandleFromUpstream(cc *UpstreamContext) error {

	debug("received hsm2")

	if cc.Remote.hs.success {
		return fmt.Errorf("handshake from %v already done", cc.Remote.PubKey())
	}
	if cc.Remote.hs.counter != 1 {
		return fmt.Errorf("wrong step number from %v (%v instead of 0)", cc.Remote.PubKey(), cc.Remote.hs.counter)
	}

	if hsm.PubKey == (cipher.PubKey{}) {
		return errors.New("PubKey is nil")
	}
	if hsm.Question == (uuid.UUID{}) {
		return errors.New("Question is nil")
	}

	// hash the question
	myQuestionHash := cipher.SumSHA256(cc.Remote.hs.local.asked.Bytes())
	// verify the answer
	err := cipher.VerifySignedHash(hsm.Answer, myQuestionHash)
	if err != nil {
		return err
	}

	if *cc.Remote.pubKey != (cipher.PubKey{}) {
		if cc.Remote.pubKey != &hsm.PubKey {
			return fmt.Errorf("PubKeys do not match: want %v, got %v", cc.Remote.pubKey.Hex(), hsm.PubKey.Hex())
		}
	}

	cc.Remote.pubKey = &hsm.PubKey

	// hash the question
	hash := cipher.SumSHA256(hsm.Question.Bytes())
	// sign the hash
	answer := cipher.SignHash(hash, *cc.Local.secKey)
	cc.Remote.hs.remote.sentAnswer = answer

	reply := hsm3{
		Answer: answer,
	}

	err = cc.Remote.Send(reply)
	if err != nil {
		return err
	}
	debug("sent hsm3")

	atomic.AddUint32(&cc.Remote.hs.counter, 1)

	return nil
}

func (hsm *hsm3) HandleFromDownstream(cc *DownstreamContext) error {

	debug("received hsm3")

	if cc.Remote.hs.success {
		return fmt.Errorf("handshake from %v already done", cc.Remote.PubKey())
	}
	if cc.Remote.hs.counter != 1 {
		return fmt.Errorf("wrong step number from %v (%v instead of 0)", cc.Remote.PubKey(), cc.Remote.hs.counter)
	}

	if hsm.Answer == (cipher.Sig{}) {
		return errors.New("Answer is nil")
	}

	// hash the question
	myQuestionHash := cipher.SumSHA256(cc.Remote.hs.local.asked.Bytes())
	err := cipher.VerifySignedHash(hsm.Answer, myQuestionHash)
	if err != nil {
		return err
	}

	reply := hsm4{
		Success: []byte("true"),
	}
	cc.Remote.hs.success = true

	fmt.Printf("\nNew subscriber from %v, pubKey %v\n", cc.Remote.conn.RemoteAddr(), cc.Remote.pubKey.Hex())

	err = cc.Remote.Send(reply)
	if err != nil {
		return err
	}
	debug("sent hsm4")

	atomic.AddUint32(&cc.Remote.hs.counter, 1)

	return nil
}

func (hsm *hsm4) HandleFromUpstream(cc *UpstreamContext) error {

	debug("received hsm4 (success)")

	fmt.Printf("\nConnected to %v, pubKey %v\n", cc.Remote.conn.RemoteAddr(), cc.Remote.pubKey.Hex())

	if cc.Remote.hs.success {
		return fmt.Errorf("handshake from %v already done", cc.Remote.PubKey())
	}
	if cc.Remote.hs.counter != 2 {
		return fmt.Errorf("wrong step number from %v (%v instead of 0)", cc.Remote.PubKey(), cc.Remote.hs.counter)
	}

	atomic.AddUint32(&cc.Remote.hs.counter, 1)
	cc.Remote.hs.success = true

	return nil
}
