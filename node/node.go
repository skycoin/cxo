package node

import (
	"fmt"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/net/factory"
	discovery "github.com/skycoin/net/skycoin-messenger/factory"

	"github.com/skycoin/cxo/node/log"
	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/cxo/skyobject/registry"
)

// A NodeID represents randomly generated
// unique id that consist of cipher.PubKey
// and related cipher.SecKey. The NodeID
// used to protect nodes against cross
// conenctions (simultaneous conenctions,
// A to B and B to A). Because, a conenction
// is a duplex
type NodeID struct {
	Pk cipher.PubKey
	Sk cipher.SecKey
}

func (n *NodeID) generate() {
	n.Pk, n.Sk = cipher.GenerateKeyPair()
}

// A Node represents network P2P transport
// for CX objects. The node used to receive
// send and retransmit CX obejcts
type Node struct {

	// logger
	log.Logger

	// unique random identifier of the Node
	id NodeID

	// lock
	fmx sync.Mutex

	// related Container
	c *skyobject.Container

	// listen and connect
	tcp *TCPFactory
	udp *UDPFactory

	// discovery
	discovery *discovery.MessengerFactory

	// keep config
	config *Config
}

// ID retursn ID of the Node. See NodeID
// for details
func (n *Node) ID() (id NodeID) {
	return n.id
}

// Config returns Config with which the
// Node was created
func (n *Node) Config() (conf *Config) {

	conf = new(Config)
	*conf = *n.config // copy

	return
}

// Container returns related Container instance
func (n *Node) Container() (c *skyobject.Container) {
	return n.c
}

// Discovery returns related discovery factory. That can be nil
// if this feature is turned off
func (n *Node) Discovery() (discovery *discovery.MessengerFactory) {
	return n.discovery
}

// Publish sends given Root obejct to peers that
// subscribed to feed of the Root. The Publish used
// to publish new Root obejcts
func (n *Node) Publish(r *registry.Root) {

	// TODO

}

// Listen starts listener. The network argument can be "tcp" or "udp".
// IP type will be selected by address (e.g. "127.0.0.1" or "[::1]").
// Provide blank string to listen on all available interfaces with
// arbitrary port asignment. Or use ":0" for the same. The Node
// starts listener inside NewNode using Config. But, it's possible
// to start listening manually using this method
func (n *Node) Listen(network string, address string) (err error) {

	switch network {
	case "tcp":
		return n.listenTCP(address)
	case "udp":
		return n.listenUDP(address)
	}

	return fmt.Errorf("unknown network type %q", network)

}

// LsitenTCP starts TCP listener.
func (n *Node) ListenTCP(address string) (err error) {
	return n.Listen("tcp", address)
}

// ListenUDP starts UDP listener.
func (n *Node) ListenUDP(address string) (err error) {
	return n.Listen("udp", address)
}

// call under lock of the fmx
func (n *Node) createTCPFactory() {

	if n.tcp != nil {
		return // already created
	}

	n.tcp = newTCPFactory(n.acceptedConnectionCallback)

}

func (n *Node) listenTCP(address string) (err error) {
	n.fmx.Lock()
	defer n.fmx.Unlock()

	n.createTCPFactory()

	return n.tcp.Listen(address)

}

// call under lock of the fmx
func (n *Node) createUDPFactory() {

	if n.udp != nil {
		return // alrady created
	}

	n.udp = newUDPFactory(n.acceptedConnectionCallback)

}

func (n *Node) listenUDP(address string) (err error) {
	n.fmx.Lock()
	defer n.fmx.Unlock()

	n.createUDPFactory()

	return n.udp.Listen(address)
}

func (n *Node) acceptedConnectionCallback(c *factory.Connection) {
	// TODO
}
