package nodeManager

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

// Subscriber is a remote node who is subscribed to this local node
type Subscriber struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer

	writerQueue chan []byte

	pubKey *cipher.PubKey

	isQuitting bool
	mu         *sync.RWMutex

	hs *handshakeStatus
}

func newSubscriber() *Subscriber {
	hsStatus := newHandshakeStatus()
	return &Subscriber{
		writerQueue: make(chan []byte),
		hs:          hsStatus,
		mu:          &sync.RWMutex{},
	}
}

// Send enqueues a message to be sent to a Subscriber
func (s *Subscriber) Send(message interface{}) error {
	if s.writerQueue == nil {
		return errors.New("s.writerQueue is nil")
	}
	// transmission is an array of bytes: [type_prefix][length_prefix][body]
	transmission, err := serializeToTransmission(message)
	if err != nil {
		return err
	}
	s.writerQueue <- transmission
	return nil
}

// PubKey returns the public key of the subscriber
func (s *Subscriber) PubKey() cipher.PubKey {
	if s.pubKey == nil {
		return cipher.PubKey{}
	}
	return *s.pubKey
}

// IP returns the remote IP of the node that is subscribed to me
func (s *Subscriber) IP() net.IP {
	addr, err := net.ResolveTCPAddr("tcp", s.conn.RemoteAddr().String())
	if err != nil {
		return net.IP{}
	}
	return addr.IP
}

// Port returns the remote Port of the node that is subscribed to me
func (s *Subscriber) Port() int {
	addr, err := net.ResolveTCPAddr("tcp", s.conn.RemoteAddr().String())
	if err != nil {
		return 0
	}
	return addr.Port
}

func (node *Node) acceptSubscribersFrom(listener net.Listener) {
	node.downstream.listener = listener
	defer node.downstream.listener.Close()
	for {
		if node.quitting {
			return
		}
		// if reached max number of connections, sleep and continue
		if len(node.downstream.subscribers) >= node.config.MaxConnections {
			time.Sleep(time.Millisecond * 200)
			continue
		}
		// Listen for an incoming connection.
		conn, err := node.downstream.listener.Accept()
		if err != nil {
			debug("Error accepting: ", err.Error())
			continue
		}

		debug("onSubscriberConnect")
		// handle connection in a goroutine
		// TODO: use better queueing
		go node.onSubscriberConnect(conn)
	}
}

// someone subscribes to me
func (node *Node) onSubscriberConnect(conn net.Conn) {
	// TODO: set deadline

	// send pubKey, validate my identity
	// get his public key
	// validate his identity
	// check if already connected
	// add to node.downstream.pool
	// call the event callback

	incomingSubscriber := newSubscriber()
	incomingSubscriber.conn = conn
	incomingSubscriber.reader = bufio.NewReader(conn)
	incomingSubscriber.writer = bufio.NewWriter(conn)

	go func(node *Node) {
		select {
		case handshakeError := <-incomingSubscriber.hs.waitForSuccess:
			{
				if handshakeError == nil {
					fmt.Printf(
						"\nNew subscriber from %v, has pubKey %v\n",
						incomingSubscriber.conn.RemoteAddr(),
						incomingSubscriber.pubKey.Hex(),
					)
					debug("adding subscriber to pool")
					err := node.registerSubscriber(incomingSubscriber)
					if err != nil {
						node.TerminateSubscriber(incomingSubscriber, err)
					}
				} else {
					node.TerminateSubscriber(incomingSubscriber, handshakeError)
				}
			}
		case <-time.After(defaultHandshakeTimeout):
			{
				node.TerminateSubscriber(incomingSubscriber, ErrHandshakeTimeout)
			}
		}
	}(node)

	// TODO: decide what needs to be done here in case this function just exits
	//defer

	go func(incomingSubscriber *Subscriber) {
		// the write loop
		debug("server: started write loop")
		for {

			// TODO: if handshake is not done yet, do not send messages other than handshake messages

			if incomingSubscriber.isQuitting {
				return
			}

			transmission, ok := <-incomingSubscriber.writerQueue
			if !ok {
				return
			}

			n, err := incomingSubscriber.conn.Write(transmission)
			if err != nil {
				// TODO: really terminate?
				node.TerminateSubscriber(incomingSubscriber, err)
				return
			}
			debugf("server: written %v", n)
			//time.Sleep(time.Millisecond * 10)
		}
	}(incomingSubscriber)

	// the read loop
	debug("server: started read loop")
	for {

		// TODO: if handshake is not done yet, do not receive messages
		// other than handshake messages

		if incomingSubscriber.isQuitting {
			return
		}

		message, err := node.readDownstreamMessage(incomingSubscriber.reader)
		if err != nil {
			if err == io.EOF {
				node.TerminateSubscriber(incomingSubscriber, err)
				return
			}
			time.Sleep(time.Millisecond * 50)
			continue
			debug("readDownstreamMessage error:", err.Error())
		}
		cc := DownstreamContext{
			Remote: incomingSubscriber,
			Local:  node,
		}
		debug("calling HandleFromDownstream")
		go func() {
			err := message.HandleFromDownstream(&cc)
			if err != nil {
				fmt.Println(err)
			}
		}()
	}

}

func (node *Node) registerSubscriber(incomingSubscriber *Subscriber) error {
	node.mu.Lock()
	defer node.mu.Unlock()

	if len(node.downstream.subscribers) >= node.config.MaxConnections {
		return fmt.Errorf("reached max connections for downstream; current: %v", len(node.downstream.subscribers))
	}

	if _, alreadyExists := node.downstream.subscribers[*incomingSubscriber.pubKey]; alreadyExists {
		return fmt.Errorf("subscriber already exists: %v", incomingSubscriber.pubKey.Hex())
	}

	node.downstream.subscribers[*incomingSubscriber.pubKey] = incomingSubscriber

	return nil
}

func (node *Node) readDownstreamMessage(connReader *bufio.Reader) (DownstreamMessage, error) {

	messageType, bodyLength, err := readHeader(connReader)
	if err != nil {
		debugf("[readDownstreamMessage] Error parsing header: %v", err.Error())
		return nil, err
	}
	// check message's declared length
	if int(bodyLength) > node.config.MaxMessageLength {
		debugf("[readDownstreamMessage] body too long: %v", int(bodyLength))
		return nil, fmt.Errorf("[downstream] body too long: %v", int(bodyLength))
	}

	debugf("messageType: %v, bodyLength: %v", messageType, bodyLength)

	// check if message has callback
	_, ok := node.downstream.callbacks[messageType]
	if !ok {
		debugf("[downstream] messageType has no callback: '%v'", messageType)
		return nil, fmt.Errorf("[downstream] messageType has no callback: '%v'", messageType)
	}
	debugf("found callback %v", messageType)

	debug("reading body")
	// read message
	bodyBytes, err := readBody(connReader, int(bodyLength))
	if err != nil {
		debugf("[downstream] readBody error: %v", err)
		return nil, fmt.Errorf("[downstream] readBody error: %v", err)
	}

	debug("converting body to message")
	mex, err := node.downstream.convertBodyToMessage(messageType, bodyBytes)
	if err != nil {
		debugf("[downstream] convertBodyToMessage error: %v", err)
		return nil, fmt.Errorf("[downstream] convertBodyToMessage error: %v", err)
	}

	return mex, nil
}

// Event handler that is called after a Connection sends a complete message
func (downstream *downstream) convertBodyToMessage(messageType string, body []byte) (DownstreamMessage, error) {

	t, succ := downstream.callbacks[messageType]
	if !succ {
		debugf("Connection sent unknown message messageType %v", messageType) //string(messageType[:]))
		return nil, fmt.Errorf("Unknown message %s received", messageType)
	}
	debugf("DownstreamMessage type %v is handling it", t)

	var m DownstreamMessage
	var v reflect.Value = reflect.New(t)
	debugf("Giving %d bytes to the decoder", len(body))
	used, err := deserializeBody(body, v)
	if err != nil {
		return nil, err
	}
	if used != len(body) {
		return nil, errors.New("Data buffer was not completely decoded")
	}

	m, succ = (v.Interface()).(DownstreamMessage)
	if !succ {
		// This occurs only when the user registers an interface that does
		// match the Message interface.  They should have known about this
		// earlier via a call to VerifyMessages
		return nil, errors.New("DownstreamMessage obtained from map does not match DownstreamMessage interface")
	}
	return m, nil
}

func (s *Subscriber) JSON() SubscriberJSON {
	return SubscriberJSON{
		IP:     s.IP().String(),
		PubKey: s.PubKey().Hex(),
		Port:   s.Port(),
	}
}

type SubscriberJSON struct {
	IP     string `json:"ip"`
	PubKey string `json:"pubKey"`
	Port   int    `json:"port"`
}

// TerminateSubscriberByID is a shortcut to TerminateSubscriber
func (node *Node) TerminateSubscriberByID(pubKey *cipher.PubKey, reason error) error {
	if pubKey == nil {
		return ErrPubKeyIsNil
	}

	elem, err := node.SubscriberByID(pubKey)
	if err != nil {
		return err
	}

	return node.TerminateSubscriber(elem, reason)
}

func (node *Node) TerminateSubscriber(subscriber *Subscriber, reason error) error {
	fmt.Println("terminating subscriber; reason:", reason)

	// close and cleanup all resources used by the node
	err := subscriber.close()
	if err != nil {
		return err
	}

	// remove node from pool
	err = node.removeSubscriberByID(*subscriber.pubKey)
	if err != nil {
		return err
	}
	return nil
}

func (s *Subscriber) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.isQuitting = true
	closeConn(s.conn)

	return nil
}

func (node *Node) removeSubscriberByID(pubKey cipher.PubKey) error {
	node.mu.Lock()
	defer node.mu.Unlock()

	if _, exists := node.downstream.subscribers[pubKey]; !exists {
		return fmt.Errorf("subscriber does not exist: %v", pubKey.Hex())
	}

	delete(node.downstream.subscribers, pubKey)
	return nil
}
