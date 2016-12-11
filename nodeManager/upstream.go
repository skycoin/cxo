package nodeManager

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"
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

	pubKey *cipher.PubKey

	isQuitting bool
	mu         *sync.RWMutex

	hs *handshakeStatus
}

func newSubscription() *Subscription {
	hsStatus := newHandshakeStatus()
	return &Subscription{
		writerQueue: make(chan []byte),
		hs:          hsStatus,
		mu:          &sync.RWMutex{},
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
func (node *Node) Subscribe(ip string, port uint16, desiredPubKey *cipher.PubKey) error {
	// validate address, port
	// validate desiredPubKey; if is null, then no need to assert the identity of the node we are connecting to
	// try to connect to the node
	// on connect, verify his identity through desiredPubKey
	// then let him verify my identity
	// if all succeeded, add to node.upstream.pool

	var err error

	if *desiredPubKey == (cipher.PubKey{}) {
		fmt.Println("PubKey not specified; the pubKey of node I'm subscribing to will not be matched.")
	}

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", ip, port))
	if err != nil {
		return err
	}

	outgoingSubscription := newSubscription()
	outgoingSubscription.pubKey = desiredPubKey
	outgoingSubscription.reader = bufio.NewReader(outgoingSubscription.conn)
	outgoingSubscription.writer = bufio.NewWriter(outgoingSubscription.conn)

	go func(node *Node) {
		select {
		case handshakeError := <-outgoingSubscription.hs.waitForSuccess:
			{
				if handshakeError == nil {
					fmt.Printf(
						"\nConnected to %v, has pubKey %v\n",
						outgoingSubscription.conn.RemoteAddr(),
						outgoingSubscription.pubKey.Hex(),
					)
					debug("adding subscription to pool")
					err := node.registerSubscription(outgoingSubscription)
					if err != nil {
						node.TerminateSubscription(outgoingSubscription, err)
					}
				} else {
					node.TerminateSubscription(outgoingSubscription, handshakeError)
				}
			}
		case <-time.After(defaultHandshakeTimeout):
			{
				node.TerminateSubscription(outgoingSubscription, ErrHandshakeTimeout)
			}
		}
	}(node)

	debug("connecting to node")
	outgoingSubscription.conn, err = net.Dial("tcp", addr.String())
	if err != nil {
		return fmt.Errorf("Error connectting plain text: %v", err.Error())
	}

	desiredPubKeyIsAny := *desiredPubKey == (cipher.PubKey{})
	if desiredPubKeyIsAny {
		fmt.Printf("\nTrying to connect to %v, ANY pubKey\n", outgoingSubscription.conn.RemoteAddr())
	} else {
		fmt.Printf("\nTrying to connect to %v, MUST have pubKey %v\n", outgoingSubscription.conn.RemoteAddr(), desiredPubKey.Hex())
	}

	go func(outgoingSubscription *Subscription) {
		// the write loop
		debug("client: started write loop")
		for {

			// TODO: do not send messages other than handshake messages

			if outgoingSubscription.isQuitting {
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
			//time.Sleep(time.Millisecond * 10)
		}
	}(outgoingSubscription)

	go func(node *Node) {
		// the read loop
		debug("client: started read loop")
		for {

			// TODO: do not receive messages other than handshake messages

			if outgoingSubscription.isQuitting {
				return
			}

			message, err := node.readUpstreamMessage(outgoingSubscription.conn)
			if err != nil {
				if err == io.EOF {
					node.TerminateSubscription(outgoingSubscription, err)
					return
				}
				debug("readUpstreamMessage error:", err.Error())
				continue
			}
			cc := UpstreamContext{
				Remote: outgoingSubscription,
				Local:  node,
			}
			debug("calling HandleFromUpstream")
			go func() {
				err := message.HandleFromUpstream(&cc)
				if err != nil {
					fmt.Println(err)
				}
			}()
		}
	}(node)

	return node.initHandshakeWith(outgoingSubscription)
}

func (node *Node) initHandshakeWith(outgoingSubscription *Subscription) error {
	// initiate handshake
	question := uuid.NewV4()
	outgoingSubscription.hs.local.asked = question
	atomic.AddUint32(&outgoingSubscription.hs.counter, 1)

	message := hsm1{
		PubKey:   *node.pubKey,
		Question: question,
	}
	return outgoingSubscription.Send(message)
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

func (node *Node) TerminateSubscriptionByID(pubKey *cipher.PubKey, reason error) error {
	if pubKey == nil {
		return ErrPubKeyIsNil
	}

	elem, err := node.SubscriptionByID(pubKey)
	if err != nil {
		return err
	}

	return node.TerminateSubscription(elem, reason)
}

func (node *Node) TerminateSubscription(subscription *Subscription, reason error) error {
	fmt.Println("terminating subscription; reason:", reason)

	// close and cleanup all resources used by the node
	err := subscription.close()
	if err != nil {
		return err
	}

	// remove node from pool
	err = node.removeSubscriptionByID(*subscription.pubKey)
	if err != nil {
		return err
	}
	return nil
}

func (s *Subscription) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.isQuitting = true
	closeConn(s.conn)

	return nil
}

func (node *Node) removeSubscriptionByID(pubKey cipher.PubKey) error {
	node.mu.Lock()
	defer node.mu.Unlock()

	if _, exists := node.upstream.subscriptions[pubKey]; !exists {
		return fmt.Errorf("subscription does not exist: %v", pubKey.Hex())
	}

	delete(node.upstream.subscriptions, pubKey)
	return nil
}
