package node

import (
	"net"
	"net/rpc"
)

type rpc struct {
	log    Logger
	events chan interface{}
	reply  chan interface{}
	done   chan struct{}
	quit   chan struct{}

	conf *Config
}

func newRpc(c *Config, log Logger, quit chan struct{}) *rpc {
	return &rpc{
		log:    log,
		events: make(chan interface{}, c.RPCEventsChanSize),
		done:   make(chan struct{}),
		conf:   c,
		quit:   quit,
	}
}

func (r *rpc) Start(quit chan struct{}) (err error) {
	if err = net.Listen("tcp", r.conf.RPCAddress); err != nil {
		return
	}
}

func (r *rpc) Connect(address string) error {
	ce := newConnectEvent(address)
	r.events <- ce
	return <-ce.err
}

func (r *rpc) Disconenct(address string) error {
	de := newDisconnectEvent(address)
	r.events <- de
	return <-de.err
}

func (r *rpc) List() ([]string, error) {
	le := newListEvent()
	r.events <- le
	select {
	case li := <-le.list:
		return li, nil
	case err := <-le.err:
		return nil, err
	case <-quit:
		return nil, ErrClosed
	}
}

// connecting

type connectEvent struct {
	address string
	err     chan error
}

func newConnectEvent(address string) connectEvent {
	return connectEvent{
		address,
		make(chan error),
	}
}

// disconnect

type disconnectEvent struct {
	address string
	err     chan error
}

func newDisconnectEvent(address string) disconnectEvent {
	return disconnectEvent{
		address,
		make(chan error),
	}
}

// list event

type listEvent struct {
	list chan []string
	err  chan error
}

func newListEvent() listEvent {
	return listEvent{
		list: make(chan []string),
		err:  make(chan error),
	}
}
