package gnet

import (
	"time"
)

type Config struct {
	// MaxConnections - incoming and outgoing
	// together
	MaxConnections int
	// MaxMessageSize is limit of message size to
	// prevent reading any malformed big message
	MaxMessageSize int

	// DialTimeout, ReadTimeout and WriteTimeout are
	// used to read, write and dial with provided
	// timeout. If timeout is zero then no timeout
	// used (no time limit). The ReadTimeout is hard
	// but the WriteTimeout can be x2 greater then
	// provided
	DialTimeout  time.Duration // dial timeout
	ReadTimeout  time.Duration // read timeout
	WriteTimeout time.Duration // write timeout

	// ReadBufferSize and WriteBufferSize are used for
	// buffered reading and writing. If the value is
	// zero then no buffers are used. If the value is
	// negative then default buffer size is used
	ReadBufferSize  int
	WriteBufferSize int

	// ReadQueueSize is size of reading queue. The queue
	// is shared for all connections. All connections
	// read messages and put them to the queue
	ReadQueueSize int
	// WriteQueueSize is size of connection related queue.
	// Every connection has its own write queue
	WriteQueueSize int

	// PingInterval used to send pings every
	// PingInterval time. If the interval is zero
	// then sending of pings is not used. But the
	// interval can't be less then
	// min(ReadTimeout, WriteTimeout). I.e. if
	// a timeout grater then zero, then the
	// interval is greater too
	PingInterval time.Duration // ping interval

	Debug bool   // print debug logs
	Name  string // name for logs (used as prefix)
}
