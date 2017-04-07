package gnet

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/node/log"
)

// pool related constants
const (
	// PrefixLength is length of a message prefix
	PrefixLength int = 4 // message prefix length
)

// pool related errors
var (
	// ErrClosed occurs when the Pool is closed
	ErrClosed = errors.New("the pool was closed")
)

// Message represents interface that every
// message must implement to be handled
type Message interface {
	// Handle called from remote side
	// when the Message received. If
	// the Handle returns an error then
	// the connections will be terminated
	// using the error as reason to the
	// termination
	Handle(ctx MessageContext, user interface{}) (terminate error)
}

type Pool struct {
	sync.RWMutex

	log.Logger

	conf Config // configurations

	// registery
	reg map[reflect.Type]string // registery
	rev map[string]reflect.Type // inverse registery

	// pool
	l       net.Listener         // listener
	dpool   sync.Pool            // reduce gc pressure
	conns   map[string]*conn     // remote address -> connection
	receive chan receivedMessage // receive messages
	sem     chan struct{}        // max connections

	quit chan struct{} // quit
}

// get slice from pool of []byte
func (p *Pool) getSlice(n int) (data []byte) {
	if p.conf.MaxMessageSize == 0 {
		// don't use the pool if there isn't
		// limit for message size
		data = make([]byte, n)
		return
	}
	var i interface{}
	i = p.dpool.Get()
	if i != nil {
		data = i.([]byte)[:n] // get from pool
		return
	}
	data = make([]byte, n, p.conf.MaxMessageSize) // create new instance
	return
}

func (p *Pool) putSlice(data []byte) {
	if p.conf.MaxMessageSize == 0 {
		// don't use the pool if there isn't
		// limit for message size
		return
	}
	if cap(data) != p.conf.MaxMessageSize {
		// drop messages size of which is not
		// equal to conf.MaxMessageSize
		return
	}
	data = data[:0] // reset length
	p.dpool.Put(data)

}

func (p *Pool) encodeMessage(m Message) (data []byte) {
	val := reflect.Indirect(reflect.ValueOf(m))
	prefix, ok := p.reg[val.Type()]
	if !ok {
		p.Panicf("send unregistered message: %T", m)
	}
	em := encoder.Serialize(m)
	data = p.getSlice(4 + PrefixLength + len(em))
	data = append(data, prefix...)
	data = append(data, encoder.SerializeAtomic(uint32(len(em)))...)
	data = append(data, em...)
	return
}

// acquire returns true if number of connections
// latrer then p.conf.MaxConnections
func (p *Pool) acquire() (ok bool) {
	if p.conf.MaxConnections == 0 {
		return true // no limit for connections
	}
	select {
	case p.sem <- struct{}{}:
		ok = true // acquire
	default:
	}
	return
}

// blocking acquire (for l.Accept())
func (p *Pool) acquireBlock() {
	if p.conf.MaxConnections == 0 {
		return // no limit for connections
	}
	p.sem <- struct{}{} // acquire
}

// release connection
func (p *Pool) release() {
	if p.conf.MaxConnections == 0 {
		return // no limit for connections
	}
	<-p.sem // release
}

// BroadcastExcept given message to all connections of the  pool
// except enumerated by 'except' argument
func (p *Pool) BroadcastExcept(m Message, except ...string) {
	//
}
