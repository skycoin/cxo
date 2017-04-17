// Package transport contains common transporting types and
// interfaces
package transport

import (
	"strings"

	"github.com/skycoin/cxo/node/pool/registry"
)

// A Schema is connection schema such as "tcp", "tls+tcp", etc
type Schema string

// SplitSchemaAddress split given string by "://"
func SplitSchemaAddress(address string) (schema Schema, addr string) {
	if ss := strings.Split(address, "://"); len(ss) == 2 {
		schema, addr = Schema(ss[0]), ss[1]
	} else if len(ss) == 1 {
		schema = Schema(ss[0])
	}
	return
}

// MessageContext represents received message with
// conection from which the message received
type MessageContext struct {
	Message registry.Message // received message
	Conn    Conn             // connection from wich the message received
}

// A Transport represents transporting interface
type Transport interface {
	// Schema of the Transport
	Schema() Schema

	// Initialize the transport, providing
	// registry with filters, and shared receiver
	// channel for incoming messages
	Initialize(reg *registry.Registry, receiver chan<- MessageContext)

	// Dial to given address
	Dial(address string) (c Conn, err error)
	// Listen on given address
	Listen(address string) (l Listener, err error)
}

// A Listener represents listener interface.
// All methods of the listener are thread safe
type Listener interface {
	// Accept connection
	Accept() (c Conn, err error)
	// Address the Listener listening on
	// with schema
	Address() string
	// Close the Listener
	Close() (err error)
}

// A Conn represents connection interface.
// All methods of the connection are thread-safe
type Conn interface {
	// Address returns address of remote node
	// with schema. For example tcp://127.0.0.1:9987
	Address() string
	// Send given message to remote node. The method
	// returns immediately
	Send(m registry.Message) (err error)
	// SendRaw sends encoded message with
	// length and prefix to remote node. The
	// method returns immediately
	SendRaw(p []byte) (err error)
	// Close the connection
	Close() (err error)

	// IsIncoming returns true if the connection
	// accepted by a listener. If the connection
	// created using Dial method then the
	// IsIncoming method returns false
	IsIncoming() bool

	//
	// Attach any user-provided value to the connection
	//

	Value() (value interface{}) // get the value
	SetValue(value interface{}) // set the value
}
