package nodeManager

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
)

// Node is the basic entity capable of accepting connections (subscribers)
// and capable of connecting to other nodes (i.e. creating subscriptions)
type Node struct {
	config *ManagerConfig
	pubKey *cipher.PubKey
	secKey *cipher.SecKey

	mu       *sync.RWMutex
	quitting bool

	upstream   upstream
	downstream downstream
}

// upstream of this node
type upstream struct {
	// I subscribed to them
	subscriptions map[cipher.PubKey]*Subscription
	callbacks     map[string]reflect.Type
}

// downstream of this node
type downstream struct {
	listener net.Listener

	// they subscribed to me
	subscribers map[cipher.PubKey]*Subscriber
	callbacks   map[string]reflect.Type
}

// NewNode generates a new node from scratch.
func (nm *Manager) NewNode() *Node {
	var pubKey cipher.PubKey // a public key
	var secKey cipher.SecKey // a secret key

	pubKey, secKey = cipher.GenerateKeyPair() //generate a keypair

	return &Node{
		config: nm.config,
		pubKey: &pubKey,
		secKey: &secKey,
		mu:     nm.mu,
		upstream: upstream{
			subscriptions: make(map[cipher.PubKey]*Subscription),
			callbacks:     nm.upstreamCallbacks, // reference to the pool of callbacks of the node manager; it's just one shared pool of upstream callbacks
		},
		downstream: downstream{
			subscribers: make(map[cipher.PubKey]*Subscriber),
			callbacks:   nm.downstreamCallbacks, // reference to the pool of callbacks of the node manager; it's just one shared pool of downstream callbacks
		},
	}
}

// NewNodeFromSecKey creates a node with the provided key pair
func (nm *Manager) NewNodeFromSecKey(secKey cipher.SecKey) (*Node, error) {
	var newNode Node

	var pubKey cipher.PubKey = cipher.PubKeyFromSecKey(secKey)

	newNode = Node{
		config: nm.config,
		pubKey: &pubKey,
		secKey: &secKey,
		mu:     nm.mu,
		upstream: upstream{
			subscriptions: make(map[cipher.PubKey]*Subscription),
			callbacks:     nm.upstreamCallbacks, // reference to the pool of callbacks of the node manager; it's just one shared pool upstream callbacks
		},
		downstream: downstream{
			subscribers: make(map[cipher.PubKey]*Subscriber),
			callbacks:   nm.downstreamCallbacks, // reference to the pool of callbacks of the node manager; it's just one shared pool downstream callbacks
		},
	}

	if err := newNode.validate(); err != nil {
		return &Node{}, err
	}

	return &newNode, nil
}

// validate is used to validate a new node created with user-provided
// key pair.
func (node *Node) validate() error {

	if node.pubKey == nil {
		return ErrPubKeyIsNil
	}
	if node.secKey == nil {
		return errors.New("secKey is nil")
	}

	if pk := cipher.PubKeyFromSecKey(*node.secKey); *node.pubKey != pk {
		return errors.New("not matching key pair")
	}

	if *node.pubKey == (cipher.PubKey{}) {
		return ErrPubKeyIsNil
	}
	if *node.secKey == (cipher.SecKey{}) {
		return errors.New("secKey is nil")
	}
	return nil
}

// AddNode is used to add a node to a node manager
func (nm *Manager) AddNode(node *Node) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if len(nm.nodes) >= nm.config.MaxNodes {
		return fmt.Errorf("reache max numer of nodes: %v", nm.config.MaxNodes)
	}

	debug("validating new node")
	if err := node.validate(); err != nil {
		return err
	}

	if _, ok := nm.nodes[*node.pubKey]; ok {
		return errors.New("node with this secKey already exists in this manager")
	}

	node.config = nm.config

	nm.nodes[*node.pubKey] = node

	return nil
}

// Start is used to start a node
func (node *Node) Start() error {

	var listener net.Listener
	var err error

	// let the kernel assign a free port where to listen to
	listener, err = net.Listen("tcp", fmt.Sprintf("%v:%v", node.config.ip.String(), node.config.Port))
	if err != nil {
		return fmt.Errorf("Error listening plain text:", err.Error())
	}
	fmt.Printf("Listening (plain) on %v, pubKey %v\n", listener.Addr().String(), node.pubKey.Hex())

	// start listening for connections from downstream
	go node.acceptSubscribersFrom(listener)
	return nil
}

func (node *Node) close() error {
	node.mu.Lock()
	defer node.mu.Unlock()

	node.quitting = true

	// terminate all subscribers
	for _, subscriber := range node.downstream.subscribers {
		err := node.TerminateSubscriber(subscriber, errors.New("closing node"))
		if err != nil {
			// TODO:
		}
	}

	// terminate all subscriptions
	for _, subscription := range node.upstream.subscriptions {
		err := node.TerminateSubscription(subscription, errors.New("closing node"))
		if err != nil {
			// TODO:
		}
	}

	return nil
}

// PubKey returns by value the pubKey of the node
func (node *Node) PubKey() cipher.PubKey {
	if node.pubKey == nil {
		return cipher.PubKey{}
	}
	return *node.pubKey
}

// IP returns the local IP at which this node is listening to for incoming connections
func (n *Node) IP() net.IP {
	if n.downstream.listener == nil {
		return net.IP{}
	}
	addr, err := net.ResolveTCPAddr("tcp", n.downstream.listener.Addr().String())
	if err != nil {
		return net.IP{}
	}
	return addr.IP
}

// Port returns the local Port at which this node is listening to for incoming connections
func (n *Node) Port() int {
	if n.downstream.listener == nil {
		return 0
	}
	addr, err := net.ResolveTCPAddr("tcp", n.downstream.listener.Addr().String())
	if err != nil {
		return 0
	}
	return addr.Port
}

// BroadcastToSubscribers sends a message to all subscribers
func (n *Node) BroadcastToSubscribers(message interface{}) error {
	// comunicate to all subscribers that I have new data
	subs := n.Subscribers()
	for _, subscriber := range subs {
		subscriber.Send(message)
	}
	return nil
}

// Subscribers returns the current nodes subscribed to this node
func (n *Node) Subscribers() []*Subscriber {
	subscriberList := []*Subscriber{}
	for _, subscriber := range n.downstream.subscribers {
		subscriberList = append(subscriberList, subscriber)
	}
	return subscriberList
}

// Subscriptions returns the current nodes this node is subscribed to
func (n *Node) Subscriptions() []*Subscription {
	subscriptionList := []*Subscription{}
	for _, subscription := range n.upstream.subscriptions {
		subscriptionList = append(subscriptionList, subscription)
	}
	return subscriptionList
}

// SubscriberByID returns a Subscriber node by its pubKey
func (n *Node) SubscriberByID(pubKey *cipher.PubKey) (*Subscriber, error) {
	if pubKey == nil {
		return &Subscriber{}, ErrPubKeyIsNil
	}
	subscriber, ok := n.downstream.subscribers[*pubKey]
	if !ok {
		return &Subscriber{}, errors.New("Subscriber not found")
	}
	return subscriber, nil
}

// SubscriptionByID returns a Subscription node by its pubKey
func (n *Node) SubscriptionByID(pubKey *cipher.PubKey) (*Subscription, error) {
	if pubKey == nil {
		return &Subscription{}, ErrPubKeyIsNil
	}
	subscription, ok := n.upstream.subscriptions[*pubKey]
	if !ok {
		return &Subscription{}, errors.New("Subscription not found")
	}
	return subscription, nil
}

// SubscriptionByID returns a Subscription node by its pubKey
func (n *Node) JSON() NodeJSON {
	return NodeJSON{
		IP:     n.IP().String(),
		PubKey: n.PubKey().Hex(),
		Port:   n.Port(),
	}
}

type NodeJSON struct {
	IP     string `json:"ip"`
	PubKey string `json:"pubKey"`
	Port   int    `json:"port"`
}
