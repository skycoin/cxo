package nodeManager

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync/atomic"
	"syscall"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/skycoin/skycoin/src/cipher"
)

// Subscription is a remote node who this local node is subscribed to
type Subscription struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer

	writerQueue chan []byte

	pubKey          *cipher.PubKey
	pubKeyMustMatch bool

	hs struct {
		success bool

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
}

func newSubscription() *Subscription {
	return &Subscription{
		writerQueue: make(chan []byte),
	}
}

// Send enqueues a message to be sent to a Subscription
func (s *Subscription) Send(message interface{}) error {
	if s.writerQueue == nil {
		return errors.New("s.writerQueue is nil")
	}
	transmission, err := serializeToTransmission(message)
	if err != nil {
		return err
	}
	s.writerQueue <- transmission
	return nil
}

// PubKey returns the public key of the subscription
func (s *Subscription) PubKey() cipher.PubKey {
	if s.pubKey == nil {
		return cipher.PubKey{}
	}
	return *s.pubKey
}

// IP returns the remote IP of the node I am subscribed to
func (s *Subscription) IP() net.IP {
	addr, err := net.ResolveTCPAddr("tcp", s.conn.RemoteAddr().String())
	if err != nil {
		return net.IP{}
	}
	return addr.IP
}

// Port returns the remote Port of the node I am subscribed to
func (s *Subscription) Port() int {
	addr, err := net.ResolveTCPAddr("tcp", s.conn.RemoteAddr().String())
	if err != nil {
		return 0
	}
	return addr.Port
}

// I sub to someone;
// if the pubKey is blank, any key will be accepted from that IP
func (node *Node) Subscribe(ip string, port uint16, pubKey *cipher.PubKey) error {
	// validate address, port
	// validate pubKey; if is null, then no need to assert the identity of the node we are connecting to
	// try to connect to the node
	// on connect, verify his identity through pubKey
	// then let him verify my identity
	// if all succeeded, add to node.upstream.pool

	var err error

	if *pubKey == (cipher.PubKey{}) {
		fmt.Println("PubKey not specified; the pubKey of node I'm subscribing to will not be matched.")
	}

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", ip, port))
	if err != nil {
		return err
	}

	outgoingSubscription := newSubscription()
	outgoingSubscription.pubKey = pubKey
	outgoingSubscription.reader = bufio.NewReader(outgoingSubscription.conn)
	outgoingSubscription.writer = bufio.NewWriter(outgoingSubscription.conn)

	go func(node *Node) {
		time.Sleep(time.Second * 5)
		if !outgoingSubscription.hs.success {
			println("handshake timeout")
			closeConn(outgoingSubscription.conn)
		}
		debug("adding subscription to pool")
		err := node.registerSubscription(outgoingSubscription)
		if err != nil {
			closeConn(outgoingSubscription.conn)
			println(err)
		}
	}(node)

	debug("connecting to node")
	outgoingSubscription.conn, err = net.Dial("tcp", addr.String())
	if err != nil {
		return fmt.Errorf("Error connectting plain text: %v", err.Error())
	}
	fmt.Printf("\nTrying to connect to %v, pubKey %v\n", outgoingSubscription.conn.RemoteAddr(), pubKey.Hex())

	go func(node *Node) {
		// the write loop
		debug("client: started write loop")
		for {

			// TODO: do not send messages other than handshake messages

			if node.quitting {
				return
			}

			transmission, ok := <-outgoingSubscription.writerQueue
			if !ok {
				debug("client: breaking write loop")
				break
			}

			n, err := outgoingSubscription.conn.Write(transmission)
			if err != nil {
				if err != syscall.EPIPE {
					debugf("client: error while writing: %v", err)
				}
			}
			debugf("client: written %v", n)
			time.Sleep(time.Millisecond * 10)
		}
	}(node)

	go func(node *Node) {
		// the read loop
		debug("client: started read loop")
		for {

			// TODO: do not receive messages other than handshake messages

			if node.quitting {
				closeConn(outgoingSubscription.conn)
				return
			}
			// If the node does not comunicate for timeout, close the connection.
			// Expecting a ping every once in a while
			timeout := 5 * time.Minute
			err = outgoingSubscription.conn.SetReadDeadline(time.Now().Add(timeout))
			if err != nil {
				fmt.Errorf("SetReadDeadline failed: %v", err)
				return
			}

			message, err := node.readUpstreamMessage(outgoingSubscription.conn)
			if err != nil {
				return
			}
			cc := UpstreamContext{
				Remote: outgoingSubscription,
				Local:  node,
			}
			debug("calling HandleFromUpstream")
			go func() {
				err := message.HandleFromUpstream(&cc)
				if err != nil {
					// TODO: if the error is HandshakeFailed, exit
					fmt.Println(err)
				}
			}()
		}
	}(node)

	// initiate handshake
	question := uuid.NewV4()
	outgoingSubscription.hs.local.asked = question
	atomic.AddUint32(&outgoingSubscription.hs.counter, 1)

	message := hsm1{
		PubKey:   *node.pubKey,
		Question: question,
	}
	err = outgoingSubscription.Send(message)
	if err != nil {
		debugf("client: error on hsm1 outgoingSubscription.SendBack: %v", err)
	}

	return nil
}

func (node *Node) registerSubscription(outgoingSubscription *Subscription) error {
	node.mu.Lock()
	defer node.mu.Unlock()

	if len(node.upstream.subscriptions) >= node.config.MaxConnections {
		return fmt.Errorf("reached max connections for upstream; current: %v", len(node.upstream.subscriptions))
	}

	if _, alreadyExists := node.upstream.subscriptions[*outgoingSubscription.pubKey]; alreadyExists {
		return fmt.Errorf("subscription already exists: %v", outgoingSubscription.pubKey.Hex())
	}

	node.upstream.subscriptions[*outgoingSubscription.pubKey] = outgoingSubscription

	return nil
}

func (node *Node) UnSubscribeFromPubKey(pubKey *cipher.PubKey) error {
	// check if subscribed to pubKey
	sub, isSubscribed := node.upstream.subscriptions[*pubKey]
	if !isSubscribed {
		return fmt.Errorf("trying to unsubscribe, but not subscribed to %v", pubKey.Hex())
	}

	// close connection
	closeConn(sub.conn)
	// TODO: add a more gentle way of terminating the connection

	node.mu.Lock()
	defer node.mu.Unlock()

	// remove subscription from the pool
	delete(node.upstream.subscriptions, *sub.pubKey)

	return nil
}

func (node *Node) readUpstreamMessage(connReader io.Reader) (UpstreamMessage, error) {

	messageType, bodyLength, err := readHeader(connReader)
	if err != nil {
		debugf("[readUpstreamMessage] Error parsing header: %v", err.Error())
		return nil, err
	}
	// check message's declared length
	if int(bodyLength) > node.config.MaxMessageLength {
		debugf("[readUpstreamMessage] body too long: %v", int(bodyLength))
		return nil, fmt.Errorf("[upstream] body too long: %v", int(bodyLength))
	}

	debugf("messageType: %v, bodyLength: %v", messageType, bodyLength)

	// check if message has callback
	_, ok := node.upstream.callbacks[messageType]
	if !ok {
		debugf("[upstream] messageType has no callback: '%v'", messageType)
		return nil, fmt.Errorf("[upstream] messageType has no callback: '%v'", messageType)
	}
	debugf("found callback %v", messageType)

	debug("reading body")
	// read message
	bodyBytes, err := readBody(connReader, int(bodyLength))
	if err != nil {
		debugf("[upstream] readBody error: %v", err)
		return nil, fmt.Errorf("[upstream] readBody error: %v", err)
	}

	debug("converting body to message")
	mex, err := node.upstream.convertBodyToMessage(messageType, bodyBytes)
	if err != nil {
		debugf("[upstream] convertBodyToMessage error: %v", err)
		return nil, fmt.Errorf("[upstream] convertBodyToMessage error: %v", err)
	}

	return mex, nil
}

// Event handler that is called after a Connection sends a complete message
func (upstream *upstream) convertBodyToMessage(messageType string, body []byte) (UpstreamMessage, error) {

	t, succ := upstream.callbacks[messageType]
	if !succ {
		debugf("Connection sent unknown message messageType %v", messageType) //string(messageType[:]))
		return nil, fmt.Errorf("Unknown message %s received", messageType)
	}
	debugf("UpstreamMessage type %v is handling it", t)

	var m UpstreamMessage
	var v reflect.Value = reflect.New(t)
	debugf("Giving %d bytes to the decoder", len(body))
	used, err := deserializeBody(body, v)
	if err != nil {
		return nil, err
	}
	if used != len(body) {
		return nil, errors.New("Data buffer was not completely decoded")
	}

	m, succ = (v.Interface()).(UpstreamMessage)
	if !succ {
		// This occurs only when the user registers an interface that does
		// match the Message interface.  They should have known about this
		// earlier via a call to VerifyMessages
		return nil, errors.New("UpstreamMessage obtained from map does not match UpstreamMessage interface")
	}
	return m, nil
}

func (s *Subscription) JSON() SubscriptionJSON {
	return SubscriptionJSON{
		IP:     s.IP().String(),
		PubKey: s.PubKey().Hex(),
		Port:   s.Port(),
	}
}

type SubscriptionJSON struct {
	PubKey string `json:"pubKey"`
	IP     string `json:"ip"`
	Port   int    `json:"port"`
}
