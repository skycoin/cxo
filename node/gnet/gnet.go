package gnet

import (
	"net"
	"reflect"
	"time"

	"golang.org/x/net/netutil"
)

const (
	PrefixLength int = 4
)

type Message struct {
	//
}

type conn struct {
	conn net.Conn    // connection
	rq   chan []byte // read queue
	wr   chan []byte // write queue
}

type Config struct {
	// MaxConnections - incoming and outgoing
	MaxConnections int
	// MaxMessageSize is limit of message size to
	// prevent reading any malformed big message
	MaxMessageSize int

	DialTimeout  time.Duration // dial timeout
	ReadTimeout  time.Duration // read timeout
	WriteTimeout time.Duration // write timeout

	PingInterval time.Duration // ping interval

	DebugLogs bool   // print debug logs
	Name      string // name for logs
}

type Pool struct {
	conf  Config                  // configurations
	l     net.Listener            // listener
	reg   map[reflect.Type]string // registery
	rev   map[string]reflect.Type // inverse registery
	conns map[string]*conn        // remote address -> connection
	quit  chan struct{}           // quit
}
