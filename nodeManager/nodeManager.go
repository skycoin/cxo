package nodeManager

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

// Debugging; of set to true, all debugging messages will be printed out.
var Debugging = false

const defaultMaxNodes = 1000
const defaultMaxConnections = 1000
const defaultMaxMessageLength = 1024 * 256
const defaultIP = "127.0.0.1"
const defaultPort uint16 = 0

// ManagerConfig is the configuration struct of a node manager
type ManagerConfig struct {
	IP   string
	ip   net.IP
	Port uint16

	// max number of nodes
	MaxNodes int
	// max connections per node (x2) per upstream and downstream
	MaxConnections int
	// Messages greater than length are rejected and the sender disconnected
	MaxMessageLength int
}

// Manager is a node manager.
type Manager struct {
	mu *sync.RWMutex

	config *ManagerConfig

	nodes map[cipher.PubKey]*Node

	downstreamCallbacks map[string]reflect.Type
	upstreamCallbacks   map[string]reflect.Type
}

// NewManagerConfig returns an instance of ManagerConfig loaded with default values
func NewManagerConfig() ManagerConfig {
	// set defaults
	newConfig := ManagerConfig{
		IP:               defaultIP,
		Port:             defaultPort,
		MaxNodes:         defaultMaxNodes,
		MaxConnections:   defaultMaxConnections,
		MaxMessageLength: defaultMaxMessageLength,
	}
	return newConfig
}

func (mc *ManagerConfig) validate() error {
	// parse and validate IP address
	mc.ip = net.ParseIP(mc.IP)
	if mc.ip == nil {
		return fmt.Errorf("cannot parse IP: %v", mc.IP)
	}
	return nil
}

// NewManager returns a new manager
func NewManager(config ManagerConfig) (*Manager, error) {
	// validate config
	err := config.validate()
	if err != nil {
		return &Manager{}, err
	}

	newManager := Manager{
		nodes:               make(map[cipher.PubKey]*Node),
		config:              &config,
		downstreamCallbacks: make(map[string]reflect.Type),
		upstreamCallbacks:   make(map[string]reflect.Type),
		mu:                  &sync.RWMutex{},
	}

	// REGISTER HANDSHAKE MESSAGES
	// TODO: register standard messages in the node manager, not for every node.
	// register messages that this node can receive from downstream
	newManager.registerDownstreamMessage(hsm1{})
	newManager.registerDownstreamMessage(hsm3{})

	// register messages that this node can receive from upstream
	newManager.registerUpstreamMessage(hsm2{})
	newManager.registerUpstreamMessage(hsm4{})

	return &newManager, nil
}

func (nm *Manager) registerDownstreamMessage(msg interface{}) {
	registerMessage("DownstreamMessage", nm.mu, nm.downstreamCallbacks, msg)
}
func (nm *Manager) registerUpstreamMessage(msg interface{}) {
	registerMessage("UpstreamMessage", nm.mu, nm.upstreamCallbacks, msg)
}

func (nm *Manager) debugInit() error {
	newNode := nm.NewNode()
	err := nm.AddNode(newNode)
	if err != nil {
		return err
	}

	// register standard messages
	debug("registering message")
	// TODO: register only those used by each

	// register messages that this node can receive from downstream
	newNode.RegisterDownstreamMessage(hsm1{})
	newNode.RegisterDownstreamMessage(hsm3{})

	// register messages that this node can receive from upstream
	newNode.RegisterUpstreamMessage(hsm2{})
	newNode.RegisterUpstreamMessage(hsm4{})

	err = newNode.Start()
	if err != nil {
		return err
	}

	time.Sleep(time.Second * 1)
	debug("connecting to node")

	subscribeToPubKey := newNode.PubKey()
	err = newNode.Subscribe("127.0.0.1", uint16(45455), &subscribeToPubKey)
	if err != nil {
		return err
	}

	return nil
}

// Nodes returns all the nodes registered in the NodeManager
func (nm *Manager) Nodes() []*Node {
	nodeList := []*Node{}
	for pubKey, node := range nm.nodes {
		fmt.Printf("\nid %v, node %v\n", pubKey.Hex(), node.PubKey().Hex())
		nodeList = append(nodeList, node)
	}
	return nodeList
}

// NodeByID returns a Node by its pubKey
func (nm *Manager) NodeByID(pubKey cipher.PubKey) (*Node, error) {
	node, ok := nm.nodes[pubKey]
	if !ok {
		return &Node{}, errors.New("Node not found")
	}
	return node, nil
}

func (nm *Manager) Shutdown() error {
	// TODO: Shutdown gracefully every component (starting from the level most at the bottom)
	return nil
}
